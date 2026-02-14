package rag

import (
	"context"
	"fmt"
	"strings"
	"time"
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

// NearDuplicateThreshold is the cosine similarity above which two vectors
// are considered "the same knowledge". 0.95 is high enough to catch
// "check disk space" vs "show disk usage" but won't merge unrelated commands.
const NearDuplicateThreshold float32 = 0.95

// LearnFromSuccess embeds a successful prompt+command pair and appends it
// to the vector store — but only if no near-duplicate already exists.
//
// This runs in a background goroutine after every successful command execution.
// It's the core of the online learning loop:
//   user runs command → command succeeds → embed & store → future queries benefit
//
// It enforces its own 5-second timeout so it never blocks the process from
// exiting. If Ollama is slow or unreachable, we just abandon this learning
// opportunity — the user never waits.
//
// Errors are silently ignored — this must never degrade the user experience.
func LearnFromSuccess(ctx context.Context, prompt, command, category string) {
	// Hard timeout: if the whole operation (embed + load + dedup + append)
	// takes longer than 5s, bail. Typical time is ~300ms.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Compose the text exactly like historyDocs() does, so the embeddings
	// are in the same semantic space and dedup works correctly.
	text := fmt.Sprintf("'%s' was successfully executed as: %s", prompt, command)

	// Embed the text.
	embedder := NewEmbedClient()
	vec, err := embedder.Embed(ctx, text)
	if err != nil {
		return // Silent failure — user never sees this.
	}

	// Load the current store to check for duplicates.
	store := NewStore()
	if err := store.Load(); err != nil {
		return // No store yet — skip (user hasn't run 'xx index').
	}

	// Semantic dedup: if a very similar vector already exists, skip.
	if store.HasNearDuplicate(vec, NearDuplicateThreshold) {
		return
	}

	// Append the new document (O(1) write).
	doc := Document{
		Text:     text,
		Source:   "history",
		Category: category,
	}
	doc.Vector = vec
	_ = store.Append(doc) // Silent failure.
}

// RecordFeedback updates the adaptive relevance score for the document
// most similar to the user's query. Called after command execution to
// provide a reinforcement signal — success boosts the doc, failure penalizes it.
//
// This is the feedback loop that makes retrieval quality improve over time:
//   query → retrieve docs → execute command → success/failure → update scores
//
// Like LearnFromSuccess, this runs in a background subprocess with a 5s timeout.
// Errors are silently ignored.
func RecordFeedback(ctx context.Context, prompt string, success bool) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Embed the original prompt to find which doc was most relevant.
	embedder := NewEmbedClient()
	vec, err := embedder.Embed(ctx, prompt)
	if err != nil {
		return
	}

	store := NewStore()
	if err := store.Load(); err != nil {
		return
	}

	// Update the score of the best-matching document.
	if !store.UpdateScore(vec, success) {
		return // No relevant doc found.
	}

	// Persist the updated store.
	_ = store.Save()
}

