// Package rag implements a local Retrieval-Augmented Generation pipeline.
// It embeds documents using Ollama, stores vectors on disk, and retrieves
// relevant context at query time to improve AI accuracy.
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// EmbedModel is the Ollama model used for generating embeddings.
	// nomic-embed-text produces 768-dimensional vectors and runs locally.
	EmbedModel = "nomic-embed-text"

	embedURL     = "http://localhost:11434/api/embeddings"
	embedTimeout = 30 * time.Second
)

// EmbedClient generates vector embeddings via Ollama's local API.
type EmbedClient struct {
	model      string
	apiURL     string
	httpClient *http.Client
}

// NewEmbedClient creates an embedding client that talks to local Ollama.
func NewEmbedClient() *EmbedClient {
	return &EmbedClient{
		model:      EmbedModel,
		apiURL:     embedURL,
		httpClient: &http.Client{Timeout: embedTimeout},
	}
}

// embedRequest is the JSON body sent to Ollama's /api/embeddings endpoint.
type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// embedResponse is the JSON body returned by Ollama's /api/embeddings endpoint.
type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// Embed converts a single text string into a vector of floats.
// This is the fundamental operation — every piece of text we want to search
// over must be embedded first. The returned slice has exactly 768 elements
// (the dimensionality of nomic-embed-text).
func (e *EmbedClient) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody := embedRequest{
		Model:  e.model,
		Prompt: text,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach Ollama for embeddings — is it running? (start with: ollama serve)")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embed response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API error (status %d): %s\nHint: run 'ollama pull %s' if the model is missing", resp.StatusCode, string(respBody), e.model)
	}

	var result embedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse embed response: %w", err)
	}

	if len(result.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned — model may not support embeddings")
	}

	return result.Embedding, nil
}

// EmbedBatch embeds multiple texts and returns their vectors in the same order.
// Used during indexing when we need to embed hundreds of documents.
// Ollama doesn't have a native batch endpoint, so we call Embed() in sequence.
// This is fine for our scale (~3-4K docs, one-time indexing operation).
func (e *EmbedClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d/%d: %w", i+1, len(texts), err)
		}
		vectors[i] = vec
	}
	return vectors, nil
}
