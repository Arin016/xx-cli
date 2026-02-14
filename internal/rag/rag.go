package rag

import (
	"context"
	"fmt"
	"strings"
)

const (
	// DefaultTopK is how many documents to retrieve per query.
	// 5 gives enough context without bloating the prompt.
	DefaultTopK = 5

	// MinScore is the minimum cosine similarity to include a result.
	// Below this threshold, the document isn't relevant enough.
	MinScore = 0.3
)

// Retrieve takes a user's natural language query, embeds it, searches the
// vector store, and returns a formatted context string ready to inject
// into the system prompt.
//
// This is the main entry point for RAG — called before every AI translation.
func Retrieve(ctx context.Context, query string) (string, error) {
	// Load the vector store from disk.
	store := NewStore()
	if err := store.Load(); err != nil {
		// If no index exists, return empty context (graceful degradation).
		return "", nil
	}

	// Embed the user's query into a vector.
	embedder := NewEmbedClient()
	queryVec, err := embedder.Embed(ctx, query)
	if err != nil {
		// If embedding fails, continue without RAG context.
		return "", nil
	}

	// Search for the most relevant documents (no category filter — search everything).
	results := store.Search(queryVec, DefaultTopK, "")

	// Filter out low-relevance results.
	var relevant []SearchResult
	for _, r := range results {
		if r.Score >= MinScore {
			relevant = append(relevant, r)
		}
	}

	if len(relevant) == 0 {
		return "", nil
	}

	// Format results into a context block for the system prompt.
	return formatContext(relevant), nil
}

// formatContext turns search results into a string that gets injected
// into the AI's system prompt. The format is designed to be clear and
// concise so the LLM can use it effectively.
func formatContext(results []SearchResult) string {
	var sb strings.Builder
	sb.WriteString("\nRelevant knowledge (use this to pick the right command):\n")
	for _, r := range results {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", r.Doc.Source, r.Doc.Text))
	}
	return sb.String()
}
