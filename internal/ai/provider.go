package ai

import "context"

// Message is a provider-agnostic chat message.
type Message struct {
	Role    string // "system", "user", or "assistant"
	Content string
}

// Provider is the interface that any AI backend must implement.
// This abstraction allows swapping between Ollama, OpenAI, Groq, etc.
// without changing any business logic in Client.
type Provider interface {
	// Complete sends a list of messages and returns the assistant's response text.
	// If jsonMode is true, the provider should request structured JSON output.
	Complete(ctx context.Context, messages []Message, jsonMode bool) (string, error)
}
