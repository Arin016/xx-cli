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

const (
	IntentQuery   = "query"
	IntentExecute = "execute"
	IntentDisplay = "display"
)

type Result struct {
	Command     string `json:"command"`
	Explanation string `json:"explanation"`
	Intent      string `json:"intent"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   string          `json:"format"`
	Options  ollamaOptions   `json:"options"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
}

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
	format := ""
	if jsonMode {
		format = "json"
	}
	reqBody := ollamaRequest{
		Model: c.cfg.Model, Messages: messages,
		Stream: false, Format: format,
		Options: ollamaOptions{Temperature: 0.1},
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
		return "", fmt.Errorf("is Ollama running? %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API status %d: %s", resp.StatusCode, string(respBody))
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
	if result.Command == "" {
		return nil, fmt.Errorf("AI returned an empty command")
	}
	switch result.Intent {
	case IntentQuery, IntentExecute, IntentDisplay:
	default:
		result.Intent = IntentDisplay
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

// ChatMessage is a public type for conversation history.
type ChatMessage struct {
	Role    string
	Content string
}

// Chat sends a conversational message with full history for context.
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
	for _, m := range history {
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
1. Return JSON: {"command": "...", "explanation": "...", "intent": "query|execute|display"}
   - "query": user asks a question (is X running?, how much RAM?)
   - "execute": user wants an action (kill Slack, delete files)
   - "display": user wants to see data (show disk usage, list files)
2. Command must be valid for the user's OS and shell.
3. Prefer simple, common commands.
4. Use the safest variant for destructive ops.
5. No sudo unless explicitly asked.
6. Use project context to pick the right tool (e.g. "run tests" -> "go test ./..." in a Go project, "npm test" in Node).
7. When the user wants to navigate/go to a directory by name, use: find ~ -maxdepth 5 -type d -iname "*<name>*" 2>/dev/null | head -10. Set intent to "display" so the user sees the matching paths.
8. Return valid JSON only. No extra text.`,
		runtime.GOOS, runtime.GOARCH, detectShell(), projectContext)
}

func detectShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "zsh"
}
