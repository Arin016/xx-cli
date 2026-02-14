package rag

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/arin/xx-cli/internal/config"
)

const storeFileName = "vectors.bin"
// Flush deletes the vector store file from disk. This is the nuclear option
// for fixing a poisoned index — wipe everything and rebuild from scratch
// with `xx index`. Returns nil if the file doesn't exist (already clean).
func (s *Store) Flush() error {
	path := storePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete vector store: %w", err)
	}
	s.docs = nil
	return nil
}

// Document is a single entry in the vector store.
// It holds the original text, its embedding vector, and metadata.
type Document struct {
	// Text is the original content (e.g. "vm_stat — show virtual memory statistics").
	Text string
	// Source identifies where this doc came from: "tldr", "learned", "history".
	Source string
	// Category groups docs for pre-filtering: "memory", "network", "git", "files", etc.
	Category string
	// Vector is the embedding — a slice of 768 float32s.
	Vector []float32
	// SuccessCount tracks how many times this doc led to a successful command.
	// Used by adaptive relevance scoring to boost reliable docs.
	SuccessCount int32
	// FailureCount tracks how many times this doc led to a failed command.
	// Used by adaptive relevance scoring to penalize unreliable docs.
	FailureCount int32
}

// SearchResult is a document matched by similarity search, with its score.
type SearchResult struct {
	Doc   Document
	Score float32 // Cosine similarity: 1.0 = identical, 0.0 = unrelated.
}

// Store is an in-memory vector store backed by a binary file on disk.
// On load, it reads the entire file into memory. On save, it writes everything.
// This is fine for our scale (~3-4K docs = ~10MB on disk).
type Store struct {
	docs []Document
}

// NewStore creates an empty store.
func NewStore() *Store {
	return &Store{}
}

// Add inserts a document into the store (in memory only — call Save to persist).
func (s *Store) Add(doc Document) {
	s.docs = append(s.docs, doc)
}

// Len returns the number of documents in the store.
func (s *Store) Len() int {
	return len(s.docs)
}

// storePath returns the full path to the binary vector file.
// It's a variable so tests can override it.
var storePath = func() string {
	return filepath.Join(config.Dir(), storeFileName)
}

// storeFormatVersion is the current binary format version.
// v1: original format (no version header, no scoring fields)
// v2: added version header + SuccessCount/FailureCount per document
const storeFormatVersion uint32 = 2

// Save writes all documents to disk in a compact binary format.
//
// Binary format v2 (all little-endian):
//   [4 bytes] format version (uint32) — always 2
//   [4 bytes] number of documents (uint32)
//   For each document:
//     [4 bytes] text length (uint32)
//     [N bytes] text (UTF-8)
//     [4 bytes] source length (uint32)
//     [N bytes] source (UTF-8)
//     [4 bytes] category length (uint32)
//     [N bytes] category (UTF-8)
//     [4 bytes] vector dimension (uint32)
//     [dim*4 bytes] vector (float32 array)
//     [4 bytes] success count (int32)
//     [4 bytes] failure count (int32)
//
// Why binary instead of JSON? A 768-dim float32 vector is 3KB in binary
// but ~6KB in JSON (decimal text). For 4K docs that's 12MB vs 24MB.
// Binary is also faster to parse — no string→float conversion.
func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(storePath()), 0o700); err != nil {
		return err
	}

	f, err := os.Create(storePath())
	if err != nil {
		return fmt.Errorf("failed to create vector store: %w", err)
	}
	defer f.Close()

	// Write format version.
	if err := binary.Write(f, binary.LittleEndian, storeFormatVersion); err != nil {
		return err
	}

	// Write document count.
	if err := binary.Write(f, binary.LittleEndian, uint32(len(s.docs))); err != nil {
		return err
	}

	for _, doc := range s.docs {
		if err := writeDoc(f, doc); err != nil {
			return err
		}
	}

	return nil
}

// Load reads the binary vector store from disk into memory.
// Supports both v1 (legacy, no version header) and v2 (with scoring fields).
func (s *Store) Load() error {
	f, err := os.Open(storePath())
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("vector store not found — run 'xx index' first")
		}
		return fmt.Errorf("failed to open vector store: %w", err)
	}
	defer f.Close()

	// Read the first uint32 — could be a version number (v2+) or a doc count (v1).
	var firstWord uint32
	if err := binary.Read(f, binary.LittleEndian, &firstWord); err != nil {
		return fmt.Errorf("failed to read store header: %w", err)
	}

	var count uint32
	version := uint32(1) // Default: legacy format.

	if firstWord == storeFormatVersion {
		// v2 format: first word is version, second word is doc count.
		version = firstWord
		if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
			return fmt.Errorf("failed to read document count: %w", err)
		}
	} else {
		// v1 format: first word IS the doc count (no version header).
		count = firstWord
	}

	s.docs = make([]Document, count)
	for i := uint32(0); i < count; i++ {
		text, err := readString(f)
		if err != nil {
			return err
		}
		source, err := readString(f)
		if err != nil {
			return err
		}
		category, err := readString(f)
		if err != nil {
			return err
		}

		var dim uint32
		if err := binary.Read(f, binary.LittleEndian, &dim); err != nil {
			return err
		}
		vec := make([]float32, dim)
		if err := binary.Read(f, binary.LittleEndian, vec); err != nil {
			return err
		}

		doc := Document{
			Text:     text,
			Source:   source,
			Category: category,
			Vector:   vec,
		}

		// v2: read scoring fields.
		if version >= 2 {
			if err := binary.Read(f, binary.LittleEndian, &doc.SuccessCount); err != nil {
				return err
			}
			if err := binary.Read(f, binary.LittleEndian, &doc.FailureCount); err != nil {
				return err
			}
		}

		s.docs[i] = doc
	}

	return nil
}

// Search finds the top-K most similar documents to the query vector.
// It computes cosine similarity between the query and every document,
// then applies adaptive relevance scoring based on success/failure history,
// and returns the highest-scoring results sorted by final score descending.
//
// Adaptive scoring formula:
//   finalScore = cosine * (1 + ln(1 + successes) - 0.5 * ln(1 + failures))
//
// This is a lightweight bandit-style signal:
//   - A doc with 10 successes and 0 failures: cosine * 3.4 (boosted)
//   - A doc with 0 successes and 5 failures: cosine * 0.19 (penalized)
//   - A brand new doc (0/0): cosine * 1.0 (neutral — no bias)
//
// The log dampening prevents any single document from dominating forever.
//
// If category is non-empty, only documents matching that category are searched.
// This is the "hybrid retrieval" optimization — pre-filter by category,
// then do vector search on the smaller subset.
func (s *Store) Search(queryVec []float32, topK int, category string) []SearchResult {
	var results []SearchResult

	for _, doc := range s.docs {
		// Category pre-filter: skip docs that don't match.
		if category != "" && doc.Category != category {
			continue
		}

		cosine := cosineSimilarity(queryVec, doc.Vector)
		score := adaptiveScore(cosine, doc.SuccessCount, doc.FailureCount)

		// Source boost: builtin entries are curated, high-quality knowledge.
		// Give them a 20% edge so auto-learned garbage doesn't drown them out.
		// Learned corrections get a 10% boost since the user explicitly taught them.
		switch doc.Source {
		case "builtin":
			score *= 1.20
		case "learned":
			score *= 1.10
		}

		results = append(results, SearchResult{Doc: doc, Score: score})
	}

	// Sort by score descending (highest similarity first).
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top-K.
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results
}

// adaptiveScore applies the reinforcement signal to a cosine similarity score.
//
// Formula: cosine * (1 + ln(1 + successes) - 0.5 * ln(1 + failures))
//
// The multiplier is always >= some small positive value because ln(1) = 0,
// so a doc with 0 successes and many failures gets heavily penalized but
// never goes fully negative (the cosine itself can be negative though).
//
// Interview talking points:
//   - Multi-armed bandit analogy: explore new docs vs exploit known-good ones
//   - Log dampening prevents runaway scores (10 successes ≈ 3.4x, not 10x)
//   - Cold-start: new docs start at 1.0 multiplier (neutral)
//   - Same principle as Reddit's ranking and HN's story scoring
func adaptiveScore(cosine float32, successes, failures int32) float32 {
	multiplier := 1.0 + math.Log(1.0+float64(successes)) - 0.5*math.Log(1.0+float64(failures))
	if multiplier < 0.01 {
		multiplier = 0.01 // Floor: never fully zero out a doc.
	}
	return cosine * float32(multiplier)
}

// cosineSimilarity computes the cosine of the angle between two vectors.
//
// Formula: cos(A,B) = (A·B) / (|A| × |B|)
//
// Where A·B is the dot product (sum of element-wise products),
// and |A| is the magnitude (sqrt of sum of squares).
//
// Returns a value between -1 and 1:
//   1.0 = identical direction (same meaning)
//   0.0 = orthogonal (unrelated)
//  -1.0 = opposite direction (opposite meaning)
//
// In practice, embedding vectors are always positive, so scores
// typically range from 0.0 to 1.0.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, magA, magB float32
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}

	denom := float32(math.Sqrt(float64(magA))) * float32(math.Sqrt(float64(magB)))
	if denom == 0 {
		return 0
	}

	return dot / denom
}

// writeString writes a length-prefixed UTF-8 string to a binary file.
func writeString(f *os.File, s string) error {
	b := []byte(s)
	if err := binary.Write(f, binary.LittleEndian, uint32(len(b))); err != nil {
		return err
	}
	_, err := f.Write(b)
	return err
}

// writeDoc writes a single document in v2 binary format.
func writeDoc(f *os.File, doc Document) error {
	if err := writeString(f, doc.Text); err != nil {
		return err
	}
	if err := writeString(f, doc.Source); err != nil {
		return err
	}
	if err := writeString(f, doc.Category); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(len(doc.Vector))); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, doc.Vector); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, doc.SuccessCount); err != nil {
		return err
	}
	if err := binary.Write(f, binary.LittleEndian, doc.FailureCount); err != nil {
		return err
	}
	return nil
}

// readString reads a length-prefixed UTF-8 string from a binary file.
func readString(f *os.File) (string, error) {
	var length uint32
	if err := binary.Read(f, binary.LittleEndian, &length); err != nil {
		return "", err
	}
	b := make([]byte, length)
	if _, err := f.Read(b); err != nil {
		return "", err
	}
	return string(b), nil
}

// Append writes a single document to the end of the binary store file
// and updates the document count header — O(1) instead of O(n) full rewrite.
//
// Binary layout (v2):
//   [4 bytes] format version (uint32)
//   [4 bytes] doc count (uint32)  ← we update this in-place
//   [... existing docs ...]
//   [new doc appended here]
//
// This is the write-behind pattern: the user's command finishes instantly,
// and we persist the new knowledge in the background.
func (s *Store) Append(doc Document) error {
	path := storePath()

	// If the file doesn't exist yet, fall back to a full Save.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		s.docs = append(s.docs, doc)
		return s.Save()
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open vector store for append: %w", err)
	}
	defer f.Close()

	// Read version and count. The count is at byte 4 in v2 (after version header)
	// or byte 0 in v1 (no version header).
	var firstWord uint32
	if err := binary.Read(f, binary.LittleEndian, &firstWord); err != nil {
		return fmt.Errorf("failed to read store header: %w", err)
	}

	var count uint32
	countOffset := int64(0) // Where the count lives in the file.

	if firstWord == storeFormatVersion {
		// v2: version at byte 0, count at byte 4.
		countOffset = 4
		if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
			return fmt.Errorf("failed to read document count: %w", err)
		}
	} else {
		// v1: count at byte 0. We can't append v2 docs to a v1 file cleanly,
		// so fall back to a full rewrite in v2 format.
		s.docs = append(s.docs, doc)
		// Reload existing docs from the v1 file first.
		f.Close()
		old := NewStore()
		if err := old.Load(); err == nil {
			// Merge: old docs + new doc (already appended to s.docs).
			s.docs = append(old.docs, doc)
		}
		return s.Save()
	}

	// Seek to end of file to append the new document.
	if _, err := f.Seek(0, 2); err != nil {
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	// Write the document in v2 binary format.
	if err := writeDoc(f, doc); err != nil {
		return err
	}

	// Seek to the count offset and overwrite with count+1.
	if _, err := f.Seek(countOffset, 0); err != nil {
		return fmt.Errorf("failed to seek to header: %w", err)
	}
	if err := binary.Write(f, binary.LittleEndian, count+1); err != nil {
		return fmt.Errorf("failed to update document count: %w", err)
	}

	// Also update the in-memory store so subsequent operations see the new doc.
	s.docs = append(s.docs, doc)

	return nil
}

// HasNearDuplicate checks if any document in the store has cosine similarity
// above the given threshold with the provided vector.
//
// Used to prevent bloat during auto-learning: if the user runs "check disk space"
// ten times, we only store it once. Semantic dedup, not string dedup — so
// "check disk" and "show disk usage" are recognized as near-duplicates.
func (s *Store) HasNearDuplicate(vec []float32, threshold float32) bool {
	for _, doc := range s.docs {
		if cosineSimilarity(vec, doc.Vector) > threshold {
			return true
		}
	}
	return false
}

// UpdateScore finds the document most similar to the given vector and
// increments its success or failure count. This is the reinforcement signal
// that makes adaptive scoring work over time.
//
// The update is applied in-memory and then the entire store is persisted.
// This is O(n) for the search + O(n) for the save, but it only runs in
// background subprocesses so the user never waits.
//
// Returns true if a matching document was found and updated.
func (s *Store) UpdateScore(vec []float32, success bool) bool {
	if len(s.docs) == 0 {
		return false
	}

	// Find the most similar document.
	bestIdx := -1
	bestScore := float32(-1)
	for i, doc := range s.docs {
		score := cosineSimilarity(vec, doc.Vector)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}

	// Only update if the match is reasonably relevant (> 0.5).
	if bestIdx < 0 || bestScore < 0.5 {
		return false
	}

	if success {
		s.docs[bestIdx].SuccessCount++
	} else {
		s.docs[bestIdx].FailureCount++
	}

	return true
}

