package rag

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
)

// --- Adaptive Scoring Tests ---

func TestAdaptiveScore_Neutral(t *testing.T) {
	// New doc (0 successes, 0 failures) should have multiplier = 1.0.
	score := adaptiveScore(0.8, 0, 0)
	if score < 0.799 || score > 0.801 {
		t.Errorf("neutral doc should have score ≈ 0.8, got %f", score)
	}
}

func TestAdaptiveScore_SuccessBoosted(t *testing.T) {
	neutral := adaptiveScore(0.8, 0, 0)
	boosted := adaptiveScore(0.8, 10, 0)
	if boosted <= neutral {
		t.Errorf("10 successes should boost score: neutral=%f, boosted=%f", neutral, boosted)
	}
}

func TestAdaptiveScore_FailurePenalized(t *testing.T) {
	neutral := adaptiveScore(0.8, 0, 0)
	penalized := adaptiveScore(0.8, 0, 10)
	if penalized >= neutral {
		t.Errorf("10 failures should penalize score: neutral=%f, penalized=%f", neutral, penalized)
	}
}

func TestAdaptiveScore_MixedSignals(t *testing.T) {
	// 10 successes + 5 failures should still be above neutral.
	neutral := adaptiveScore(0.8, 0, 0)
	mixed := adaptiveScore(0.8, 10, 5)
	if mixed <= neutral {
		t.Errorf("10 successes + 5 failures should be above neutral: neutral=%f, mixed=%f", neutral, mixed)
	}
}

func TestAdaptiveScore_FloorPreventsZero(t *testing.T) {
	// Even with massive failures, score should never be zero.
	score := adaptiveScore(0.8, 0, 10000)
	if score <= 0 {
		t.Errorf("floor should prevent zero score, got %f", score)
	}
}

func TestAdaptiveScore_ZeroCosine(t *testing.T) {
	// Zero cosine should stay zero regardless of scoring.
	score := adaptiveScore(0.0, 100, 0)
	if score != 0 {
		t.Errorf("zero cosine should stay zero, got %f", score)
	}
}

func TestAdaptiveScore_LogDampening(t *testing.T) {
	// Going from 10 to 100 successes should NOT give 10x boost.
	score10 := adaptiveScore(1.0, 10, 0)
	score100 := adaptiveScore(1.0, 100, 0)
	ratio := score100 / score10
	if ratio > 2.0 {
		t.Errorf("log dampening should prevent runaway: 10s=%f, 100s=%f, ratio=%f", score10, score100, ratio)
	}
}

// --- UpdateScore Tests ---

func TestUpdateScore_Success(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "check memory", Vector: []float32{0.9, 0.1, 0.0}})
	s.Add(Document{Text: "check disk", Vector: []float32{0.0, 0.1, 0.9}})

	// Query close to "check memory".
	updated := s.UpdateScore([]float32{0.8, 0.2, 0.0}, true)
	if !updated {
		t.Fatal("UpdateScore should return true when a match is found")
	}
	if s.docs[0].SuccessCount != 1 {
		t.Errorf("expected SuccessCount=1, got %d", s.docs[0].SuccessCount)
	}
	if s.docs[1].SuccessCount != 0 {
		t.Errorf("non-matching doc should be unchanged, got SuccessCount=%d", s.docs[1].SuccessCount)
	}
}

func TestUpdateScore_Failure(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "check memory", Vector: []float32{0.9, 0.1, 0.0}})

	updated := s.UpdateScore([]float32{0.8, 0.2, 0.0}, false)
	if !updated {
		t.Fatal("UpdateScore should return true")
	}
	if s.docs[0].FailureCount != 1 {
		t.Errorf("expected FailureCount=1, got %d", s.docs[0].FailureCount)
	}
}

func TestUpdateScore_EmptyStore(t *testing.T) {
	s := NewStore()
	updated := s.UpdateScore([]float32{1, 2, 3}, true)
	if updated {
		t.Error("empty store should return false")
	}
}

func TestUpdateScore_LowSimilarity(t *testing.T) {
	s := NewStore()
	// Orthogonal vectors — similarity < 0.5 threshold.
	s.Add(Document{Text: "a", Vector: []float32{1, 0, 0}})

	updated := s.UpdateScore([]float32{0, 1, 0}, true)
	if updated {
		t.Error("should not update when best match is below 0.5 threshold")
	}
}

func TestUpdateScore_MultipleUpdates(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "check memory", Vector: []float32{0.9, 0.1}})

	for i := 0; i < 5; i++ {
		s.UpdateScore([]float32{0.9, 0.1}, true)
	}
	for i := 0; i < 3; i++ {
		s.UpdateScore([]float32{0.9, 0.1}, false)
	}

	if s.docs[0].SuccessCount != 5 {
		t.Errorf("expected 5 successes, got %d", s.docs[0].SuccessCount)
	}
	if s.docs[0].FailureCount != 3 {
		t.Errorf("expected 3 failures, got %d", s.docs[0].FailureCount)
	}
}

// --- Scoring Persistence Tests ---

func TestScoringFields_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := NewStore()
	s.Add(Document{
		Text:         "check memory",
		Source:       "builtin",
		Category:     "memory",
		Vector:       []float32{0.1, 0.2, 0.3},
		SuccessCount: 7,
		FailureCount: 2,
	})
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s2.docs[0].SuccessCount != 7 {
		t.Errorf("SuccessCount not persisted: got %d", s2.docs[0].SuccessCount)
	}
	if s2.docs[0].FailureCount != 2 {
		t.Errorf("FailureCount not persisted: got %d", s2.docs[0].FailureCount)
	}
}

func TestScoringFields_AppendPreservesScores(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := NewStore()
	s.Add(Document{
		Text: "existing", Source: "builtin", Category: "memory",
		Vector: []float32{0.1, 0.2}, SuccessCount: 5, FailureCount: 1,
	})
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := s.Append(Document{
		Text: "new", Source: "history", Category: "disk",
		Vector: []float32{0.3, 0.4}, SuccessCount: 0, FailureCount: 0,
	}); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s2.docs[0].SuccessCount != 5 {
		t.Errorf("original doc scores corrupted: SuccessCount=%d", s2.docs[0].SuccessCount)
	}
	if s2.docs[1].SuccessCount != 0 {
		t.Errorf("appended doc should have 0 successes, got %d", s2.docs[1].SuccessCount)
	}
}

func TestSearch_AdaptiveScoringAffectsRanking(t *testing.T) {
	s := NewStore()
	// Two docs with identical cosine similarity to the query,
	// but different success counts.
	s.Add(Document{
		Text: "reliable command", Source: "history", Category: "general",
		Vector: []float32{0.9, 0.1}, SuccessCount: 10, FailureCount: 0,
	})
	s.Add(Document{
		Text: "unreliable command", Source: "history", Category: "general",
		Vector: []float32{0.9, 0.1}, SuccessCount: 0, FailureCount: 10,
	})

	results := s.Search([]float32{0.9, 0.1}, 2, "")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Doc.Text != "reliable command" {
		t.Errorf("reliable doc should rank first, got %q", results[0].Doc.Text)
	}
	if results[0].Score <= results[1].Score {
		t.Errorf("reliable doc should have higher score: %f vs %f", results[0].Score, results[1].Score)
	}
}

// --- LRU Cache Tests ---

func TestEmbedClient_CacheStructure(t *testing.T) {
	client := NewEmbedClient()
	if client.cache == nil {
		t.Fatal("cache should be initialized")
	}
	if client.cacheOrder == nil {
		t.Fatal("cacheOrder should be initialized")
	}
}

func TestLRU_PutAndGet(t *testing.T) {
	client := NewEmbedClient()
	vec := []float32{1.0, 2.0, 3.0}
	client.putLRU("test-key", vec)

	cached, ok := client.cache["test-key"]
	if !ok {
		t.Fatal("key should be in cache after put")
	}
	if len(cached.vector) != 3 || cached.vector[0] != 1.0 {
		t.Errorf("cached vector mismatch: %v", cached.vector)
	}
}

func TestLRU_Eviction(t *testing.T) {
	client := NewEmbedClient()

	// Fill cache to capacity.
	for i := 0; i < cacheMaxSize; i++ {
		client.putLRU(fmt.Sprintf("key-%d", i), []float32{float32(i)})
	}

	if len(client.cache) != cacheMaxSize {
		t.Fatalf("cache should be at capacity: %d", len(client.cache))
	}

	// Add one more — should evict key-0 (oldest).
	client.putLRU("overflow", []float32{999})

	if len(client.cache) != cacheMaxSize {
		t.Errorf("cache should still be at capacity after eviction: %d", len(client.cache))
	}
	if _, ok := client.cache["key-0"]; ok {
		t.Error("key-0 should have been evicted (oldest)")
	}
	if _, ok := client.cache["overflow"]; !ok {
		t.Error("overflow key should be in cache")
	}
}

func TestLRU_TouchPreventsEviction(t *testing.T) {
	client := NewEmbedClient()

	// Fill cache.
	for i := 0; i < cacheMaxSize; i++ {
		client.putLRU(fmt.Sprintf("key-%d", i), []float32{float32(i)})
	}

	// Touch key-0 (move it to back of LRU).
	client.touchLRU("key-0")

	// Add one more — should evict key-1 (now the oldest), not key-0.
	client.putLRU("overflow", []float32{999})

	if _, ok := client.cache["key-0"]; !ok {
		t.Error("key-0 should NOT have been evicted (was touched)")
	}
	if _, ok := client.cache["key-1"]; ok {
		t.Error("key-1 should have been evicted (oldest after touch)")
	}
}

// --- Benchmarks ---
// Run with: go test ./internal/rag/ -bench=. -benchmem

func BenchmarkCosineSimilarity_768dim(b *testing.B) {
	a := makeRandomVector(768)
	c := makeRandomVector(768)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cosineSimilarity(a, c)
	}
}

func BenchmarkSearch_100docs(b *testing.B) {
	s := makeStoreWithDocs(100, 768)
	query := makeRandomVector(768)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Search(query, 5, "")
	}
}

func BenchmarkSearch_1000docs(b *testing.B) {
	s := makeStoreWithDocs(1000, 768)
	query := makeRandomVector(768)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Search(query, 5, "")
	}
}

func BenchmarkSearch_10000docs(b *testing.B) {
	s := makeStoreWithDocs(10000, 768)
	query := makeRandomVector(768)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Search(query, 5, "")
	}
}

func BenchmarkSaveAndLoad_100docs(b *testing.B) {
	tmpDir := b.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := makeStoreWithDocs(100, 768)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Save()
		s2 := NewStore()
		_ = s2.Load()
	}
}

func BenchmarkSaveAndLoad_1000docs(b *testing.B) {
	tmpDir := b.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := makeStoreWithDocs(1000, 768)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Save()
		s2 := NewStore()
		_ = s2.Load()
	}
}

func BenchmarkAppend_768dim(b *testing.B) {
	tmpDir := b.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := makeStoreWithDocs(100, 768)
	_ = s.Save()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := Document{
			Text:     fmt.Sprintf("bench-doc-%d", i),
			Source:   "bench",
			Category: "general",
			Vector:   makeRandomVector(768),
		}
		_ = s.Append(doc)
	}
}

func BenchmarkHasNearDuplicate_100docs(b *testing.B) {
	s := makeStoreWithDocs(100, 768)
	query := makeRandomVector(768)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.HasNearDuplicate(query, 0.95)
	}
}

func BenchmarkAdaptiveScore(b *testing.B) {
	for i := 0; i < b.N; i++ {
		adaptiveScore(0.85, 10, 3)
	}
}

// --- Helpers ---

func makeRandomVector(dim int) []float32 {
	vec := make([]float32, dim)
	for i := range vec {
		vec[i] = rand.Float32()
	}
	return vec
}

func makeStoreWithDocs(n, dim int) *Store {
	s := NewStore()
	for i := 0; i < n; i++ {
		s.Add(Document{
			Text:     fmt.Sprintf("doc-%d", i),
			Source:   "bench",
			Category: "general",
			Vector:   makeRandomVector(dim),
		})
	}
	return s
}
