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

	// cacheMaxSize is the maximum number of embeddings to cache in memory.
	// 100 entries × 768 floats × 4 bytes = ~300KB — negligible memory cost.
	cacheMaxSize = 100
)

// EmbedClient generates vector embeddings via Ollama's local API.
// It includes an in-memory LRU cache that eliminates redundant API calls
// for repeated queries (e.g., "is chrome running" asked 5 times).
type EmbedClient struct {
	model      string
	apiURL     string
	httpClient *http.Client
	cache      map[string]cachedEmbedding
	cacheOrder []string // LRU order: oldest at front, newest at back.
}

// cachedEmbedding stores a cached vector with its key for LRU tracking.
type cachedEmbedding struct {
	vector []float32
}

// NewEmbedClient creates an embedding client that talks to local Ollama.
func NewEmbedClient() *EmbedClient {
	return &EmbedClient{
		model:      EmbedModel,
		apiURL:     embedURL,
		httpClient: &http.Client{Timeout: embedTimeout},
		cache:      make(map[string]cachedEmbedding),
		cacheOrder: make([]string, 0, cacheMaxSize),
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
//
// Uses an in-memory LRU cache to avoid redundant Ollama API calls.
// Cache hit: 0ms. Cache miss: ~200ms (Ollama round-trip).
func (e *EmbedClient) Embed(ctx context.Context, text string) ([]float32, error) {
	// Check cache first.
	if cached, ok := e.cache[text]; ok {
		// Move to back of LRU order (most recently used).
		e.touchLRU(text)
		return cached.vector, nil
	}

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

	// Store in cache.
	e.putLRU(text, result.Embedding)

	return result.Embedding, nil
}

// touchLRU moves a key to the back of the LRU order (most recently used).
func (e *EmbedClient) touchLRU(key string) {
	for i, k := range e.cacheOrder {
		if k == key {
			e.cacheOrder = append(e.cacheOrder[:i], e.cacheOrder[i+1:]...)
			e.cacheOrder = append(e.cacheOrder, key)
			return
		}
	}
}

// putLRU adds a new entry to the cache, evicting the oldest if at capacity.
func (e *EmbedClient) putLRU(key string, vec []float32) {
	if len(e.cache) >= cacheMaxSize {
		// Evict the oldest entry (front of the order slice).
		oldest := e.cacheOrder[0]
		e.cacheOrder = e.cacheOrder[1:]
		delete(e.cache, oldest)
	}
	e.cache[key] = cachedEmbedding{vector: vec}
	e.cacheOrder = append(e.cacheOrder, key)
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
