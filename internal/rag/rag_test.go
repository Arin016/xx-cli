package rag

import (
	"fmt"
	"path/filepath"
	"testing"
)

// --- Cosine Similarity Tests ---

func TestCosineSimilarity_IdenticalVectors(t *testing.T) {
	a := []float32{1, 2, 3}
	score := cosineSimilarity(a, a)
	if score < 0.999 {
		t.Errorf("identical vectors should have similarity ~1.0, got %f", score)
	}
}

func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	score := cosineSimilarity(a, b)
	if score > 0.001 {
		t.Errorf("orthogonal vectors should have similarity ~0.0, got %f", score)
	}
}

func TestCosineSimilarity_SimilarVectors(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{1.1, 2.1, 3.1}
	score := cosineSimilarity(a, b)
	if score < 0.99 {
		t.Errorf("similar vectors should have high similarity, got %f", score)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	score := cosineSimilarity(a, b)
	if score != 0 {
		t.Errorf("different length vectors should return 0, got %f", score)
	}
}

func TestCosineSimilarity_EmptyVectors(t *testing.T) {
	score := cosineSimilarity([]float32{}, []float32{})
	if score != 0 {
		t.Errorf("empty vectors should return 0, got %f", score)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}
	score := cosineSimilarity(a, b)
	if score != 0 {
		t.Errorf("zero vector should return 0, got %f", score)
	}
}

// --- Store Tests ---

func TestStore_AddAndLen(t *testing.T) {
	s := NewStore()
	if s.Len() != 0 {
		t.Errorf("new store should be empty, got %d", s.Len())
	}

	s.Add(Document{Text: "test", Source: "test", Category: "test", Vector: []float32{1, 2, 3}})
	if s.Len() != 1 {
		t.Errorf("store should have 1 doc, got %d", s.Len())
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := NewStore()
	s.Add(Document{
		Text:     "check memory on macOS",
		Source:   "builtin",
		Category: "memory",
		Vector:   []float32{0.1, 0.2, 0.3, 0.4, 0.5},
	})
	s.Add(Document{
		Text:     "kill a process",
		Source:   "builtin",
		Category: "process",
		Vector:   []float32{0.5, 0.4, 0.3, 0.2, 0.1},
	})

	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if s2.Len() != 2 {
		t.Fatalf("expected 2 docs, got %d", s2.Len())
	}
	if s2.docs[0].Text != "check memory on macOS" {
		t.Errorf("first doc text mismatch: %s", s2.docs[0].Text)
	}
	if s2.docs[1].Category != "process" {
		t.Errorf("second doc category mismatch: %s", s2.docs[1].Category)
	}
	if len(s2.docs[0].Vector) != 5 || s2.docs[0].Vector[0] != 0.1 {
		t.Errorf("first doc vector mismatch: %v", s2.docs[0].Vector)
	}
}

func TestStore_Search_TopK(t *testing.T) {
	s := NewStore()
	// Add 3 docs with known vectors.
	s.Add(Document{Text: "memory info", Source: "builtin", Category: "memory", Vector: []float32{0.9, 0.1, 0.0}})
	s.Add(Document{Text: "network info", Source: "builtin", Category: "network", Vector: []float32{0.0, 0.9, 0.1}})
	s.Add(Document{Text: "disk info", Source: "builtin", Category: "disk", Vector: []float32{0.1, 0.0, 0.9}})

	// Query vector is close to "memory info".
	query := []float32{0.8, 0.2, 0.0}
	results := s.Search(query, 2, "")

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Doc.Text != "memory info" {
		t.Errorf("top result should be 'memory info', got '%s'", results[0].Doc.Text)
	}
}

func TestStore_Search_CategoryFilter(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "memory info", Source: "builtin", Category: "memory", Vector: []float32{0.9, 0.1}})
	s.Add(Document{Text: "network info", Source: "builtin", Category: "network", Vector: []float32{0.8, 0.2}})

	// Search only in "network" category.
	results := s.Search([]float32{0.9, 0.1}, 10, "network")

	if len(results) != 1 {
		t.Fatalf("expected 1 result with category filter, got %d", len(results))
	}
	if results[0].Doc.Category != "network" {
		t.Errorf("result should be in 'network' category, got '%s'", results[0].Doc.Category)
	}
}

func TestStore_Search_EmptyStore(t *testing.T) {
	s := NewStore()
	results := s.Search([]float32{1, 2, 3}, 5, "")
	if len(results) != 0 {
		t.Errorf("empty store should return 0 results, got %d", len(results))
	}
}

// --- Indexer Tests ---

func TestOsCommandDocs_NotEmpty(t *testing.T) {
	docs := osCommandDocs()
	if len(docs) == 0 {
		t.Error("osCommandDocs should return at least some entries")
	}
	for _, doc := range docs {
		if doc.Text == "" {
			t.Error("document text should not be empty")
		}
		if doc.Source != "builtin" {
			t.Errorf("OS docs should have source 'builtin', got '%s'", doc.Source)
		}
		if doc.Category == "" {
			t.Error("document category should not be empty")
		}
	}
}

func TestCategorizeCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		expected string
	}{
		{"git push origin main", "git"},
		{"docker ps -a", "docker"},
		{"brew install node", "packages"},
		{"vm_stat", "memory"},
		{"lsof -i :3000", "network"},
		{"ps aux | grep chrome", "process"},
		{"df -h", "disk"},
		{"find . -name '*.go'", "files"},
		{"echo hello", "general"},
	}

	for _, tt := range tests {
		got := categorizeCommand(tt.cmd)
		if got != tt.expected {
			t.Errorf("categorizeCommand(%q) = %q, want %q", tt.cmd, got, tt.expected)
		}
	}
}

// --- Format Tests ---

func TestFormatContext(t *testing.T) {
	results := []SearchResult{
		{Doc: Document{Text: "use vm_stat for memory", Source: "builtin"}, Score: 0.9},
		{Doc: Document{Text: "user said: run tests → make test", Source: "learned"}, Score: 0.7},
	}

	output := formatContext(results)
	if output == "" {
		t.Error("formatContext should not return empty string")
	}
	if !contains(output, "vm_stat") {
		t.Error("formatted context should contain document text")
	}
	if !contains(output, "[builtin]") {
		t.Error("formatted context should contain source tag")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Append Tests ---

func TestStore_Append_ToExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	// Create an initial store with 2 docs and save it.
	s := NewStore()
	s.Add(Document{Text: "doc one", Source: "builtin", Category: "general", Vector: []float32{0.1, 0.2, 0.3}})
	s.Add(Document{Text: "doc two", Source: "builtin", Category: "memory", Vector: []float32{0.4, 0.5, 0.6}})
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Append a third doc.
	newDoc := Document{Text: "doc three", Source: "history", Category: "network", Vector: []float32{0.7, 0.8, 0.9}}
	if err := s.Append(newDoc); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// In-memory store should have 3 docs.
	if s.Len() != 3 {
		t.Errorf("in-memory store should have 3 docs, got %d", s.Len())
	}

	// Reload from disk and verify.
	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load after Append failed: %v", err)
	}
	if s2.Len() != 3 {
		t.Errorf("reloaded store should have 3 docs, got %d", s2.Len())
	}

	// Verify the appended doc's content.
	found := false
	for i := 0; i < s2.Len(); i++ {
		if s2.docs[i].Text == "doc three" && s2.docs[i].Source == "history" && s2.docs[i].Category == "network" {
			found = true
			if len(s2.docs[i].Vector) != 3 || s2.docs[i].Vector[0] != 0.7 {
				t.Errorf("appended doc vector mismatch: %v", s2.docs[i].Vector)
			}
		}
	}
	if !found {
		t.Error("appended document not found after reload")
	}
}

func TestStore_Append_ToNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	// Append to a store that doesn't exist yet — should fall back to Save.
	s := NewStore()
	doc := Document{Text: "first doc", Source: "history", Category: "general", Vector: []float32{1.0, 2.0}}
	if err := s.Append(doc); err != nil {
		t.Fatalf("Append to non-existent file failed: %v", err)
	}

	// Reload and verify.
	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s2.Len() != 1 {
		t.Errorf("store should have 1 doc, got %d", s2.Len())
	}
}

func TestStore_Append_MultipleAppends(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := NewStore()
	s.Add(Document{Text: "initial", Source: "builtin", Category: "general", Vector: []float32{0.1}})
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Append 5 more docs.
	for i := 0; i < 5; i++ {
		doc := Document{
			Text:     fmt.Sprintf("appended-%d", i),
			Source:   "history",
			Category: "general",
			Vector:   []float32{float32(i) * 0.1},
		}
		if err := s.Append(doc); err != nil {
			t.Fatalf("Append %d failed: %v", i, err)
		}
	}

	// Reload and verify count.
	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s2.Len() != 6 {
		t.Errorf("store should have 6 docs (1 initial + 5 appended), got %d", s2.Len())
	}
}

// --- HasNearDuplicate Tests ---

func TestHasNearDuplicate_IdenticalVector(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "test", Source: "test", Category: "test", Vector: []float32{1, 2, 3}})

	if !s.HasNearDuplicate([]float32{1, 2, 3}, 0.95) {
		t.Error("identical vector should be detected as near-duplicate")
	}
}

func TestHasNearDuplicate_SimilarVector(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "test", Source: "test", Category: "test", Vector: []float32{1, 2, 3}})

	// Very similar vector — should be above 0.95 threshold.
	if !s.HasNearDuplicate([]float32{1.01, 2.01, 3.01}, 0.95) {
		t.Error("very similar vector should be detected as near-duplicate")
	}
}

func TestHasNearDuplicate_DifferentVector(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "test", Source: "test", Category: "test", Vector: []float32{1, 0, 0}})

	// Orthogonal vector — should NOT be a near-duplicate.
	if s.HasNearDuplicate([]float32{0, 1, 0}, 0.95) {
		t.Error("orthogonal vector should not be detected as near-duplicate")
	}
}

func TestHasNearDuplicate_EmptyStore(t *testing.T) {
	s := NewStore()
	if s.HasNearDuplicate([]float32{1, 2, 3}, 0.95) {
		t.Error("empty store should never have near-duplicates")
	}
}


// --- Additional Store Tests ---

func TestStore_Load_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "nonexistent.bin") }
	defer func() { storePath = origStorePath }()

	s := NewStore()
	err := s.Load()
	if err == nil {
		t.Fatal("Load on missing file should return error")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestStore_SaveAndLoad_EmptyStore(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := NewStore()
	if err := s.Save(); err != nil {
		t.Fatalf("Save empty store failed: %v", err)
	}

	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load empty store failed: %v", err)
	}
	if s2.Len() != 0 {
		t.Errorf("expected 0 docs, got %d", s2.Len())
	}
}

func TestStore_SaveAndLoad_LargeVector(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	// Simulate a real 768-dim embedding.
	vec := make([]float32, 768)
	for i := range vec {
		vec[i] = float32(i) * 0.001
	}

	s := NewStore()
	s.Add(Document{Text: "real embedding test", Source: "builtin", Category: "memory", Vector: vec})
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s2.Len() != 1 {
		t.Fatalf("expected 1 doc, got %d", s2.Len())
	}
	if len(s2.docs[0].Vector) != 768 {
		t.Errorf("expected 768-dim vector, got %d", len(s2.docs[0].Vector))
	}
	if s2.docs[0].Vector[100] != vec[100] {
		t.Errorf("vector element mismatch at index 100: got %f, want %f", s2.docs[0].Vector[100], vec[100])
	}
}

func TestStore_SaveAndLoad_EmptyFields(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := NewStore()
	s.Add(Document{Text: "", Source: "", Category: "", Vector: []float32{1.0}})
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s2.Len() != 1 {
		t.Fatalf("expected 1 doc, got %d", s2.Len())
	}
	if s2.docs[0].Text != "" || s2.docs[0].Source != "" || s2.docs[0].Category != "" {
		t.Error("empty fields should round-trip correctly")
	}
}

func TestStore_SaveAndLoad_UnicodeText(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := NewStore()
	s.Add(Document{Text: "日本語テスト 🚀 émojis", Source: "builtin", Category: "general", Vector: []float32{0.5}})
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s2.docs[0].Text != "日本語テスト 🚀 émojis" {
		t.Errorf("unicode text mismatch: %s", s2.docs[0].Text)
	}
}

func TestStore_Search_TopKZero_ReturnsAll(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "a", Source: "b", Category: "c", Vector: []float32{1, 0}})
	s.Add(Document{Text: "d", Source: "e", Category: "f", Vector: []float32{0, 1}})
	s.Add(Document{Text: "g", Source: "h", Category: "i", Vector: []float32{1, 1}})

	results := s.Search([]float32{1, 0}, 0, "")
	if len(results) != 3 {
		t.Errorf("topK=0 should return all docs, got %d", len(results))
	}
}

func TestStore_Search_ScoreOrdering(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "exact match", Source: "b", Category: "c", Vector: []float32{1, 0, 0}})
	s.Add(Document{Text: "partial match", Source: "b", Category: "c", Vector: []float32{0.7, 0.7, 0}})
	s.Add(Document{Text: "no match", Source: "b", Category: "c", Vector: []float32{0, 0, 1}})

	results := s.Search([]float32{1, 0, 0}, 3, "")
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// Scores should be strictly descending.
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not sorted: score[%d]=%f > score[%d]=%f", i, results[i].Score, i-1, results[i-1].Score)
		}
	}
	if results[0].Doc.Text != "exact match" {
		t.Errorf("first result should be 'exact match', got %q", results[0].Doc.Text)
	}
}

func TestStore_Search_NonMatchingCategory(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "a", Source: "b", Category: "memory", Vector: []float32{1, 0}})
	s.Add(Document{Text: "b", Source: "b", Category: "disk", Vector: []float32{0, 1}})

	results := s.Search([]float32{1, 0}, 10, "network")
	if len(results) != 0 {
		t.Errorf("non-matching category should return 0 results, got %d", len(results))
	}
}

func TestStore_Append_PreservesOriginalDocs(t *testing.T) {
	tmpDir := t.TempDir()
	origStorePath := storePath
	storePath = func() string { return filepath.Join(tmpDir, "vectors.bin") }
	defer func() { storePath = origStorePath }()

	s := NewStore()
	s.Add(Document{Text: "original", Source: "builtin", Category: "memory", Vector: []float32{0.1, 0.2}})
	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := s.Append(Document{Text: "appended", Source: "history", Category: "disk", Vector: []float32{0.3, 0.4}}); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	s2 := NewStore()
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if s2.docs[0].Text != "original" || s2.docs[0].Source != "builtin" {
		t.Errorf("original doc corrupted after append: %+v", s2.docs[0])
	}
	if s2.docs[1].Text != "appended" || s2.docs[1].Source != "history" {
		t.Errorf("appended doc incorrect: %+v", s2.docs[1])
	}
}

func TestHasNearDuplicate_LowThreshold(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "test", Source: "test", Category: "test", Vector: []float32{1, 0, 0}})

	// With a very low threshold, even somewhat different vectors match.
	if !s.HasNearDuplicate([]float32{0.8, 0.6, 0}, 0.5) {
		t.Error("low threshold should match moderately similar vectors")
	}
}

func TestHasNearDuplicate_HighThreshold(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "test", Source: "test", Category: "test", Vector: []float32{1, 2, 3}})

	// With threshold=1.0, only exact identical vectors match.
	// Cosine similarity of identical vectors is ~0.9999, not exactly 1.0 due to float precision.
	if s.HasNearDuplicate([]float32{1.05, 2.05, 3.05}, 1.0) {
		t.Error("threshold=1.0 should not match slightly different vectors")
	}
}

func TestHasNearDuplicate_MultipleDocs(t *testing.T) {
	s := NewStore()
	s.Add(Document{Text: "a", Source: "b", Category: "c", Vector: []float32{1, 0, 0}})
	s.Add(Document{Text: "d", Source: "e", Category: "f", Vector: []float32{0, 1, 0}})
	s.Add(Document{Text: "g", Source: "h", Category: "i", Vector: []float32{0, 0, 1}})

	// Should match the second doc.
	if !s.HasNearDuplicate([]float32{0, 1, 0}, 0.95) {
		t.Error("should find near-duplicate matching second doc")
	}
	// Should not match any doc.
	if s.HasNearDuplicate([]float32{0.577, 0.577, 0.577}, 0.95) {
		t.Error("equidistant vector should not match any single doc at 0.95")
	}
}

// --- Additional Cosine Similarity Tests ---

func TestCosineSimilarity_NegativeValues(t *testing.T) {
	a := []float32{1, -1, 1}
	b := []float32{-1, 1, -1}
	score := cosineSimilarity(a, b)
	if score > -0.99 {
		t.Errorf("opposite vectors should have similarity ~-1.0, got %f", score)
	}
}

func TestCosineSimilarity_SingleElement(t *testing.T) {
	a := []float32{5}
	b := []float32{3}
	score := cosineSimilarity(a, b)
	if score < 0.999 {
		t.Errorf("parallel single-element vectors should have similarity ~1.0, got %f", score)
	}
}

// --- Additional Categorize Tests ---

func TestCategorizeCommand_CaseInsensitive(t *testing.T) {
	if categorizeCommand("GIT push") != "git" {
		t.Error("categorizeCommand should be case-insensitive for git")
	}
	if categorizeCommand("DOCKER ps") != "docker" {
		t.Error("categorizeCommand should be case-insensitive for docker")
	}
}

func TestCategorizeCommand_AptYum(t *testing.T) {
	if categorizeCommand("apt install vim") != "packages" {
		t.Error("apt should categorize as packages")
	}
	if categorizeCommand("yum install vim") != "packages" {
		t.Error("yum should categorize as packages")
	}
}

func TestCategorizeCommand_MemoryVariants(t *testing.T) {
	if categorizeCommand("sysctl hw.memsize") != "memory" {
		t.Error("memsize should categorize as memory")
	}
	if categorizeCommand("free -h") != "memory" {
		t.Error("free should categorize as memory")
	}
}

// --- Format Context Tests ---

func TestFormatContext_SingleResult(t *testing.T) {
	results := []SearchResult{
		{Doc: Document{Text: "use df -h", Source: "builtin"}, Score: 0.8},
	}
	output := formatContext(results)
	if !contains(output, "[builtin]") {
		t.Error("should contain source tag")
	}
	if !contains(output, "df -h") {
		t.Error("should contain document text")
	}
}

func TestFormatContext_MultipleSourceTypes(t *testing.T) {
	results := []SearchResult{
		{Doc: Document{Text: "builtin doc", Source: "builtin"}, Score: 0.9},
		{Doc: Document{Text: "history doc", Source: "history"}, Score: 0.8},
		{Doc: Document{Text: "learned doc", Source: "learned"}, Score: 0.7},
	}
	output := formatContext(results)
	if !contains(output, "[builtin]") || !contains(output, "[history]") || !contains(output, "[learned]") {
		t.Error("should contain all source types")
	}
}

// --- Indexer Doc Count Tests ---

func TestMacosCommandDocs_Count(t *testing.T) {
	docs := macosCommandDocs()
	if len(docs) < 40 {
		t.Errorf("expected at least 40 macOS docs, got %d", len(docs))
	}
}

func TestLinuxCommandDocs_Count(t *testing.T) {
	docs := linuxCommandDocs()
	if len(docs) < 5 {
		t.Errorf("expected at least 5 Linux docs, got %d", len(docs))
	}
}

func TestOsCommandDocs_AllHaveCategories(t *testing.T) {
	for _, docs := range [][]Document{macosCommandDocs(), linuxCommandDocs()} {
		for _, doc := range docs {
			if doc.Category == "" {
				t.Errorf("doc %q has empty category", doc.Text)
			}
			if doc.Source != "builtin" {
				t.Errorf("doc %q has source %q, expected 'builtin'", doc.Text, doc.Source)
			}
		}
	}
}
