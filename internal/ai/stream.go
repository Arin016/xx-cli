// Package ai — stream.go provides the streaming interface and helpers.
// Streaming lets tokens appear in real-time as the AI generates them,
// replacing the spinner → wall-of-text pattern with a smooth, incremental UX.
package ai

import "context"

// StreamDelta represents a single chunk from a streaming AI response.
type StreamDelta struct {
	// Token is the text fragment. Empty string is valid (heartbeat).
	Token string
	// Done is true when the stream is complete.
	Done bool
	// Err is non-nil if the stream encountered an error.
	Err error
}

// StreamingProvider extends Provider with token-by-token streaming.
// Providers that don't support streaming can omit this interface —
// the Client will fall back to Complete() automatically.
type StreamingProvider interface {
	Provider
	// CompleteStream sends messages and returns a channel that emits tokens
	// as they arrive. The channel is closed when the response is complete.
	// If jsonMode is true, the provider should request structured JSON output
	// (streaming is typically only used for free-text responses).
	CompleteStream(ctx context.Context, messages []Message) <-chan StreamDelta
}

// collectStream reads all tokens from a stream channel and returns the
// concatenated result. Useful for testing and fallback paths.
func collectStream(ch <-chan StreamDelta) (string, error) {
	var result string
	for delta := range ch {
		if delta.Err != nil {
			return result, delta.Err
		}
		result += delta.Token
	}
	return result, nil
}
