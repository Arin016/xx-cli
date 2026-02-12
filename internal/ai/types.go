// Package ai provides types for the Ollama API client.
package ai

// Intent constants define how xx should handle the AI's response.
const (
	IntentQuery   = "query"   // User is asking a question — auto-run, summarize output.
	IntentExecute = "execute" // User wants an action — confirm before running.
	IntentDisplay = "display" // User wants to see data — auto-run, show raw output.
)

// Result is the structured response from the AI translation.
type Result struct {
	Command     string `json:"command"`
	Explanation string `json:"explanation"`
	Intent      string `json:"intent"`
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
