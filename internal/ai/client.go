// Package ai provides the AI client that translates natural language
// into shell commands. It uses a Provider interface for the AI backend,
// keeping business logic (prompt engineering, intent classification,
// workflow splitting) decoupled from any specific API.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/arin/xx-cli/internal/config"
	projctx "github.com/arin/xx-cli/internal/context"
	"github.com/arin/xx-cli/internal/learn"
)

// Client orchestrates AI interactions. It builds prompts, parses responses,
// classifies intents, and manages conversation context. The actual API
// communication is delegated to a Provider.
type Client struct {
	provider Provider
}

// NewClient creates a Client with the appropriate provider based on config.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		provider: NewOllamaProvider(cfg.Model),
	}
}

// NewClientWithProvider creates a Client with a custom provider.
// Useful for testing or alternative backends (OpenAI, Groq, etc.).
func NewClientWithProvider(p Provider) *Client {
	return &Client{provider: p}
}

// Translate converts a natural language prompt into a structured Result
// containing the shell command, explanation, and intent classification.
func (c *Client) Translate(ctx context.Context, prompt string) (*Result, error) {
	messages := []Message{
		{Role: "system", Content: buildSystemPrompt()},
		{Role: "user", Content: prompt},
	}
	rawText, err := c.provider.Complete(ctx, messages, true)
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

	// Validate and normalize intent.
	switch result.Intent {
	case IntentQuery, IntentExecute, IntentDisplay:
	case IntentWorkflow:
		if len(result.Steps) == 0 {
			result.Intent = IntentExecute
		}
	default:
		result.Intent = IntentDisplay
	}

	// Safety net: if the AI chained commands with && despite instructions,
	// auto-split into a proper workflow.
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

// Summarize interprets command output and returns a human-friendly answer.
func (c *Client) Summarize(ctx context.Context, userPrompt, command, output string, success bool) (string, error) {
	status := "succeeded"
	if !success {
		status = "failed"
	}
	messages := []Message{
		{Role: "system", Content: "You are a helpful CLI assistant. Interpret command output and give a short, friendly, human-readable answer. Be concise (1-3 sentences). Answer the user's question directly. Don't show raw output. Use plain language."},
		{Role: "user", Content: fmt.Sprintf("I asked: %q\nCommand: %s\nStatus: %s\nOutput:\n%s", userPrompt, command, status, truncate(output, 2000))},
	}
	return c.provider.Complete(ctx, messages, false)
}

// Explain takes a shell command and returns a plain English explanation.
func (c *Client) Explain(ctx context.Context, command string) (string, error) {
	messages := []Message{
		{Role: "system", Content: "You are a shell command expert. Explain the given command in plain English. Break down each flag and argument. Be concise but thorough. Use simple language a junior developer would understand. Do not use markdown."},
		{Role: "user", Content: command},
	}
	return c.provider.Complete(ctx, messages, false)
}

// Analyze interprets piped input data based on the user's question.
func (c *Client) Analyze(ctx context.Context, question, data string) (string, error) {
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant that analyzes data and answers questions about it. Be concise and direct. Give clear, actionable answers. Don't repeat the input data back. Use plain language."},
		{Role: "user", Content: fmt.Sprintf("Question: %s\n\nData:\n%s", question, truncate(data, 4000))},
	}
	return c.provider.Complete(ctx, messages, false)
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

	messages := []Message{
		{Role: "system", Content: systemMsg},
	}

	// Keep only the last 20 messages to avoid exceeding the context window.
	trimmed := history
	const maxHistory = 20
	if len(trimmed) > maxHistory {
		trimmed = trimmed[len(trimmed)-maxHistory:]
	}
	for _, m := range trimmed {
		messages = append(messages, Message{Role: m.Role, Content: m.Content})
	}

	return c.provider.Complete(ctx, messages, false)
}

// Recap generates a standup-ready summary from today's command history.
func (c *Client) Recap(ctx context.Context, historyData string, count int) (string, error) {
	messages := []Message{
		{Role: "system", Content: "You are a productivity assistant. Given a log of terminal commands from today, generate a concise standup-ready summary. Group related commands by project or task. Mention key actions (builds, deploys, git operations, debugging). Use bullet points. Be concise — this should be copy-pasteable into a standup message. Don't list every command, summarize the work."},
		{Role: "user", Content: fmt.Sprintf("Here are my %d commands from today:\n\n%s", count, historyData)},
	}
	return c.provider.Complete(ctx, messages, false)
}

// Diagnose takes an error message and returns a diagnosis with a suggested fix.
func (c *Client) Diagnose(ctx context.Context, errorMsg string) (string, error) {
	messages := []Message{
		{Role: "system", Content: "You are a senior DevOps engineer and debugging expert. Given an error message, explain what went wrong in plain English, why it happened, and give the exact command to fix it. Be concise and actionable. Format: 1) What happened 2) Why 3) Fix command. No markdown."},
		{Role: "user", Content: errorMsg},
	}
	return c.provider.Complete(ctx, messages, false)
}

// DiffExplain takes a git diff and returns a human-readable summary.
func (c *Client) DiffExplain(ctx context.Context, diff string) (string, error) {
	messages := []Message{
		{Role: "system", Content: "You are a code reviewer. Given a git diff, write a concise summary of what changed and why it matters. Group changes by file or feature. This should be useful as a PR description or commit message. Be specific about what was added, removed, or modified. No markdown. Keep it under 10 lines."},
		{Role: "user", Content: diff},
	}
	return c.provider.Complete(ctx, messages, false)
}

// SmartRetry analyzes a failed command and suggests a corrected version.
func (c *Client) SmartRetry(ctx context.Context, userPrompt, failedCmd, errorOutput string) (string, error) {
	messages := []Message{
		{Role: "system", Content: "You are a shell expert. A command failed. Analyze the error and return ONLY the corrected command — nothing else. No explanation, no quotes, just the fixed command on a single line. If you can't determine a fix, return an empty string."},
		{Role: "user", Content: fmt.Sprintf("User wanted: %s\nFailed command: %s\nError output:\n%s", userPrompt, failedCmd, truncate(errorOutput, 2000))},
	}
	fix, err := c.provider.Complete(ctx, messages, false)
	if err != nil {
		return "", err
	}
	// Clean up — the AI sometimes wraps in backticks or quotes.
	fix = strings.TrimSpace(fix)
	fix = strings.Trim(fix, "`\"'")
	return fix, nil
}

// --- Helper functions ---

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
6. Command must be valid for the user's OS and shell.
7. Prefer simple, common commands.
8. Use the safest variant for destructive ops.
9. No sudo unless explicitly asked.
10. Use project context to pick the right tool (e.g. "run tests" -> "go test ./..." in a Go project, "npm test" in Node).
11. Use git context (branch, diff, recent commits) to generate accurate git commands and meaningful commit messages.
12. When the user wants to navigate/go to a directory by name, use: find ~ -maxdepth 5 -type d -iname "*<name>*" 2>/dev/null | head -10. Set intent to "display".
13. Return valid JSON only. No extra text.%s`,
		runtime.GOOS, runtime.GOARCH, detectShell(), projectContext, learn.FewShotPrompt())
}

// isChainedCommand checks if a command contains && or ; separators.
func isChainedCommand(cmd string) bool {
	return strings.Contains(cmd, " && ") || strings.Contains(cmd, " ; ")
}

// splitChainedCommand breaks a "cmd1 && cmd2 && cmd3" string into individual steps.
func splitChainedCommand(cmd string) []Step {
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
		parts := strings.Split(shell, "/")
		return parts[len(parts)-1]
	}
	return "sh"
}
