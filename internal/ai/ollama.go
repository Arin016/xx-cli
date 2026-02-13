package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultOllamaURL = "http://localhost:11434/api/chat"
	defaultTimeout   = 60 * time.Second
)

// OllamaProvider implements Provider for the Ollama local API.
type OllamaProvider struct {
	model      string
	apiURL     string
	httpClient *http.Client
}

// NewOllamaProvider creates a provider that talks to a local Ollama instance.
func NewOllamaProvider(model string) *OllamaProvider {
	return &OllamaProvider{
		model:      model,
		apiURL:     defaultOllamaURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

// Complete sends messages to Ollama and returns the response text.
func (o *OllamaProvider) Complete(ctx context.Context, messages []Message, jsonMode bool) (string, error) {
	// Convert provider-agnostic messages to Ollama format.
	ollamaMsgs := make([]ollamaMessage, len(messages))
	for i, m := range messages {
		ollamaMsgs[i] = ollamaMessage{Role: m.Role, Content: m.Content}
	}

	reqBody := ollamaRequest{
		Model:    o.model,
		Messages: ollamaMsgs,
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not reach Ollama at %s — is it running? (start with: ollama serve)", o.apiURL)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := string(respBody)
		if strings.Contains(errMsg, "model") && strings.Contains(errMsg, "not found") {
			return "", fmt.Errorf("model %q not found — run: ollama pull %s", o.model, o.model)
		}
		return "", fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, errMsg)
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return strings.TrimSpace(ollamaResp.Message.Content), nil
}

// CompleteStream sends messages to Ollama with streaming enabled and returns
// a channel that emits tokens as they arrive. The channel is closed when the
// response is complete. This implements the StreamingProvider interface.
func (o *OllamaProvider) CompleteStream(ctx context.Context, messages []Message) <-chan StreamDelta {
	ch := make(chan StreamDelta)

	go func() {
		defer close(ch)

		ollamaMsgs := make([]ollamaMessage, len(messages))
		for i, m := range messages {
			ollamaMsgs[i] = ollamaMessage{Role: m.Role, Content: m.Content}
		}

		reqBody := ollamaRequest{
			Model:    o.model,
			Messages: ollamaMsgs,
			Stream:   true,
			Options:  ollamaOptions{Temperature: 0.1},
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			ch <- StreamDelta{Err: fmt.Errorf("failed to marshal request: %w", err)}
			return
		}

		// Use a client without timeout — streaming can take a while and
		// the context handles cancellation.
		streamClient := &http.Client{}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.apiURL, bytes.NewReader(body))
		if err != nil {
			ch <- StreamDelta{Err: fmt.Errorf("failed to create request: %w", err)}
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := streamClient.Do(req)
		if err != nil {
			ch <- StreamDelta{Err: fmt.Errorf("could not reach Ollama at %s — is it running? (start with: ollama serve)", o.apiURL)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			errMsg := string(respBody)
			if strings.Contains(errMsg, "model") && strings.Contains(errMsg, "not found") {
				ch <- StreamDelta{Err: fmt.Errorf("model %q not found — run: ollama pull %s", o.model, o.model)}
			} else {
				ch <- StreamDelta{Err: fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, errMsg)}
			}
			return
		}

		// Ollama streams newline-delimited JSON objects.
		// Each chunk: {"message":{"role":"assistant","content":"token"},"done":false}
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var chunk ollamaStreamChunk
			if err := json.Unmarshal(line, &chunk); err != nil {
				ch <- StreamDelta{Err: fmt.Errorf("failed to parse stream chunk: %w", err)}
				return
			}

			if chunk.Message.Content != "" {
				ch <- StreamDelta{Token: chunk.Message.Content}
			}

			if chunk.Done {
				ch <- StreamDelta{Done: true}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- StreamDelta{Err: fmt.Errorf("stream read error: %w", err)}
		}
	}()

	return ch
}
