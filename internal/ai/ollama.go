package ai

import (
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
