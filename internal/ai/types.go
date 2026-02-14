// Package ai provides the AI client and provider interface for xx-cli.
// Types in this file are shared across the client and provider implementations.
package ai

// Intent constants define how xx should handle the AI's response.
const (
	IntentQuery    = "query"    // User is asking a question — auto-run, summarize output.
	IntentExecute  = "execute"  // User wants an action — confirm before running.
	IntentDisplay  = "display"  // User wants to see data — auto-run, show raw output.
	IntentWorkflow = "workflow" // User wants a multi-step pipeline — confirm once, run sequentially.
)

// Result is the structured response from the AI translation.
type Result struct {
	Command     string   `json:"command"`
	Explanation string   `json:"explanation"`
	Intent      string   `json:"intent"`
	Steps       []Step   `json:"steps,omitempty"` // Populated when intent is "workflow".
	RAGContext  string   `json:"-"`               // Injected RAG knowledge (not from JSON, for debug/verbose output).
}

// Step is a single command in a multi-step workflow.
type Step struct {
	Command     string `json:"command"`
	Explanation string `json:"explanation"`
}

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	Role    string
	Content string
}

// ollamaRequest is the request body sent to the Ollama API.
type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Format   string          `json:"format,omitempty"`
	Options  ollamaOptions   `json:"options"`
}

// ollamaMessage is a single message in the Ollama chat format.
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ollamaOptions controls generation parameters.
type ollamaOptions struct {
	Temperature float64 `json:"temperature"`
}

// ollamaResponse is the response body from the Ollama API.
type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
}
// ollamaStreamChunk is a single line from Ollama's streaming NDJSON response.
type ollamaStreamChunk struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
}
