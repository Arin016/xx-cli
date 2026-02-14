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

// Save writes all documents to disk in a compact binary format.
//
// Binary format (all little-endian):
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

	// Write document count.
	if err := binary.Write(f, binary.LittleEndian, uint32(len(s.docs))); err != nil {
		return err
	}

	for _, doc := range s.docs {
		// Write text.
		if err := writeString(f, doc.Text); err != nil {
			return err
		}
		// Write source.
		if err := writeString(f, doc.Source); err != nil {
			return err
		}
		// Write category.
		if err := writeString(f, doc.Category); err != nil {
			return err
		}
		// Write vector.
		if err := binary.Write(f, binary.LittleEndian, uint32(len(doc.Vector))); err != nil {
			return err
		}
		if err := binary.Write(f, binary.LittleEndian, doc.Vector); err != nil {
			return err
		}
	}

	return nil
}

// Load reads the binary vector store from disk into memory.
func (s *Store) Load() error {
	f, err := os.Open(storePath())
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("vector store not found — run 'xx index' first")
		}
		return fmt.Errorf("failed to open vector store: %w", err)
	}
	defer f.Close()

	var count uint32
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return fmt.Errorf("failed to read document count: %w", err)
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

		s.docs[i] = Document{
			Text:     text,
			Source:   source,
			Category: category,
			Vector:   vec,
		}
	}

	return nil
}

// Search finds the top-K most similar documents to the query vector.
// It computes cosine similarity between the query and every document,
// then returns the highest-scoring results sorted by score descending.
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

		score := cosineSimilarity(queryVec, doc.Vector)
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
// Binary layout reminder:
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

	// Read current document count from the first 4 bytes.
	var count uint32
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return fmt.Errorf("failed to read document count: %w", err)
	}

	// Seek to end of file to append the new document.
	if _, err := f.Seek(0, 2); err != nil {
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	// Write the document in the same binary format as Save().
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

	// Seek back to byte 0 and overwrite the count with count+1.
	if _, err := f.Seek(0, 0); err != nil {
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

