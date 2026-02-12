// Package ai handles communication with Ollama's local API to translate
// natural language prompts into executable shell commands.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/arin/xx-cli/internal/config"
	projctx "github.com/arin/xx-cli/internal/context"
)

const (
	ollamaAPIURL = "http://localhost:11434/api/chat"
	timeout      = 60 * time.Second
)

// Client communicates with the Ollama API.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) chat(ctx context.Context, messages []ollamaMessage, jsonMode bool) (string, error) {
	reqBody := ollamaRequest{
		Model:    c.cfg.Model,
		Messages: messages,
		Stream:   false,
		Options:  ollamaOptions{Temperature: 0.1},
	}
	if jsonMode {
		reqBody.Format = "json"
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ollamaAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not reach Ollama at %s — is it running? (start with: ollama serve)", ollamaAPIURL)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		body := string(respBody)
		if strings.Contains(body, "model") && strings.Contains(body, "not found") {
			return "", fmt.Errorf("model %q not found — run: ollama pull %s", c.cfg.Model, c.cfg.Model)
		}
		return "", fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, body)
	}
	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}
	return strings.TrimSpace(ollamaResp.Message.Content), nil
}

func (c *Client) Translate(ctx context.Context, prompt string) (*Result, error) {
	messages := []ollamaMessage{
		{Role: "system", Content: buildSystemPrompt()},
		{Role: "user", Content: prompt},
	}
	rawText, err := c.chat(ctx, messages, true)
	if err != nil {
		return nil, err
	}
	if rawText == "" {
		return nil, fmt.Errorf("no response from AI")
	}
	var result Result
	if err := json.Unmarshal([]byte(rawText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI output: %w\nRaw: %s", err, rawText)
	}
	if result.Command == "" && result.Intent != IntentWorkflow {
		return nil, fmt.Errorf("AI returned an empty command")
	}
	switch result.Intent {
	case IntentQuery, IntentExecute, IntentDisplay:
	case IntentWorkflow:
		if len(result.Steps) == 0 {
			// AI said workflow but gave no steps — fall back to single execute.
			result.Intent = IntentExecute
		}
	default:
		result.Intent = IntentDisplay
	}

	// If the AI chained commands with && despite instructions, auto-split into a workflow.
	if result.Intent != IntentWorkflow && result.Command != "" && isChainedCommand(result.Command) {
		steps := splitChainedCommand(result.Command)
		if len(steps) > 1 {
			result.Intent = IntentWorkflow
			result.Steps = steps
			result.Explanation = "Multi-step workflow"
		}
	}

	return &result, nil
}

func (c *Client) Summarize(ctx context.Context, userPrompt, command, output string, success bool) (string, error) {
	status := "succeeded"
	if !success {
		status = "failed"
	}
	messages := []ollamaMessage{
		{Role: "system", Content: "You are a helpful CLI assistant. Interpret command output and give a short, friendly, human-readable answer. Be concise (1-3 sentences). Answer the user's question directly. Don't show raw output. Use plain language."},
		{Role: "user", Content: fmt.Sprintf("I asked: %q\nCommand: %s\nStatus: %s\nOutput:\n%s", userPrompt, command, status, truncate(output, 2000))},
	}
	summary, err := c.chat(ctx, messages, false)
	if err != nil {
		return "", err
	}
	return summary, nil
}

// Explain takes a shell command and returns a plain English explanation.
func (c *Client) Explain(ctx context.Context, command string) (string, error) {
	messages := []ollamaMessage{
		{Role: "system", Content: "You are a shell command expert. Explain the given command in plain English. Break down each flag and argument. Be concise but thorough. Use simple language a junior developer would understand. Do not use markdown."},
		{Role: "user", Content: command},
	}
	return c.chat(ctx, messages, false)
}

// Analyze interprets piped input data based on the user's question.
func (c *Client) Analyze(ctx context.Context, question, data string) (string, error) {
	messages := []ollamaMessage{
		{Role: "system", Content: "You are a helpful assistant that analyzes data and answers questions about it. Be concise and direct. Give clear, actionable answers. Don't repeat the input data back. Use plain language."},
		{Role: "user", Content: fmt.Sprintf("Question: %s\n\nData:\n%s", question, truncate(data, 4000))},
	}
	return c.chat(ctx, messages, false)
}

// Chat sends a conversational message with full history for context.
// History is capped to the last 20 messages to stay within the model's context window.
func (c *Client) Chat(ctx context.Context, history []ChatMessage) (string, error) {
	proj := projctx.Detect()

	systemMsg := fmt.Sprintf(`You are xx, a friendly and knowledgeable terminal assistant. You help users with shell commands, system administration, programming, and general tech questions.

Environment:
- OS: %s
- Architecture: %s
- Shell: %s
%s

Personality:
- Be friendly, casual, and helpful — like a senior dev friend.
- Give concise answers. Don't over-explain unless asked.
- When suggesting commands, show the command and briefly explain what it does.
- If the user seems stuck, guide them step by step.
- You can chat about anything tech-related, not just commands.
- Keep responses short and conversational. No walls of text.`,
		runtime.GOOS, runtime.GOARCH, detectShell(), proj.Summary())

	messages := []ollamaMessage{
		{Role: "system", Content: systemMsg},
	}

	// Keep only the last 20 messages to avoid exceeding the context window.
	trimmed := history
	const maxHistory = 20
	if len(trimmed) > maxHistory {
		trimmed = trimmed[len(trimmed)-maxHistory:]
	}

	for _, m := range trimmed {
		messages = append(messages, ollamaMessage{Role: m.Role, Content: m.Content})
	}

	return c.chat(ctx, messages, false)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n... (truncated)"
}

func buildSystemPrompt() string {
	proj := projctx.Detect()
	projectContext := proj.Summary()

	return fmt.Sprintf(`You are a shell command expert. Translate natural language into shell commands.

Environment:
- OS: %s
- Architecture: %s
- Shell: %s
%s

Rules:
1. Return JSON with one of these formats:

   Single command:
   {"command": "the shell command", "explanation": "what it does", "intent": "query|execute|display"}

   Multi-step workflow (use ONLY when the request needs 2+ sequential commands):
   {"command": "", "explanation": "overall description", "intent": "workflow", "steps": [{"command": "first command", "explanation": "what step 1 does"}, {"command": "second command", "explanation": "what step 2 does"}]}

   Intent meanings:
   - "query": user asks a question (is X running?, how much RAM?)
   - "execute": user wants a single action (kill Slack, delete files)
   - "display": user wants to see data (show disk usage, list files)
   - "workflow": user wants multiple sequential steps (commit and push, clean build and test)

2. IMPORTANT: "command" must always be a string, never an array.
3. Use "workflow" when the request clearly involves 2 or more distinct commands that must run in order. Examples: "commit and push", "clean build and run tests", "stop server and restart".
4. NEVER chain multiple commands with && or ; — if you need multiple commands, use "workflow" intent with "steps". Pipes (|) within a single command are fine.
5. For workflows, each step's "command" must be a complete standalone shell command string.
5. Command must be valid for the user's OS and shell.
6. Prefer simple, common commands.
7. Use the safest variant for destructive ops.
8. No sudo unless explicitly asked.
9. Use project context to pick the right tool (e.g. "run tests" -> "go test ./..." in a Go project, "npm test" in Node).
10. Use git context (branch, diff, recent commits) to generate accurate git commands and meaningful commit messages.
11. When the user wants to navigate/go to a directory by name, use: find ~ -maxdepth 5 -type d -iname "*<name>*" 2>/dev/null | head -10. Set intent to "display".
12. Return valid JSON only. No extra text.`,
		runtime.GOOS, runtime.GOARCH, detectShell(), projectContext)
}

// isChainedCommand checks if a command contains && or ; separators
// (but not inside quotes or as part of a pipe).
func isChainedCommand(cmd string) bool {
	return strings.Contains(cmd, " && ") || strings.Contains(cmd, " ; ")
}

// splitChainedCommand breaks a "cmd1 && cmd2 && cmd3" string into individual steps.
func splitChainedCommand(cmd string) []Step {
	// Split on && first, then ; as fallback.
	var parts []string
	if strings.Contains(cmd, " && ") {
		parts = strings.Split(cmd, " && ")
	} else {
		parts = strings.Split(cmd, " ; ")
	}

	var steps []Step
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			steps = append(steps, Step{Command: p})
		}
	}
	return steps
}

func detectShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	if shell := os.Getenv("SHELL"); shell != "" {
		// Return just the base name (e.g. "/bin/zsh" → "zsh").
		parts := strings.Split(shell, "/")
		return parts[len(parts)-1]
	}
	return "sh"
}
