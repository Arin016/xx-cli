package history

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".xx-cli"), 0o700)
	return func() { os.Setenv("HOME", origHome) }
}

func TestSave_SingleEntry(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	err := Save(Entry{Prompt: "list files", Command: "ls -la", Output: "file1\nfile2", Success: true})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	entries, err := Load(10)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Prompt != "list files" {
		t.Errorf("expected prompt 'list files', got %q", entries[0].Prompt)
	}
	if entries[0].Command != "ls -la" {
		t.Errorf("expected command 'ls -la', got %q", entries[0].Command)
	}
	if !entries[0].Success {
		t.Error("expected success=true")
	}
	if entries[0].Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestSave_MultipleEntries(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		Save(Entry{Prompt: "test", Command: "echo test", Success: true})
	}

	entries, err := Load(100)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}

func TestSave_TrimsToMaxEntries(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	// Save more than maxEntries (500).
	for i := 0; i < 510; i++ {
		Save(Entry{Prompt: "test", Command: "echo test", Success: true})
	}

	entries, err := Load(0) // Load all.
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(entries) > maxEntries {
		t.Errorf("expected at most %d entries, got %d", maxEntries, len(entries))
	}
}

func TestLoad_WithLimit(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	for i := 0; i < 20; i++ {
		Save(Entry{Prompt: "test", Command: "echo test", Success: true})
	}

	entries, err := Load(5)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 entries with limit, got %d", len(entries))
	}
}

func TestLoad_NoFile(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	entries, err := Load(10)
	if err != nil {
		t.Fatalf("Load on missing file should not error: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries, got %v", entries)
	}
}

func TestSave_FailedEntry(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	Save(Entry{Prompt: "bad cmd", Command: "nonexistent", Output: "not found", Success: false})

	entries, _ := Load(10)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Success {
		t.Error("expected success=false")
	}
	if entries[0].Output != "not found" {
		t.Errorf("expected output 'not found', got %q", entries[0].Output)
	}
}

func TestSave_EmptyOutput(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	Save(Entry{Prompt: "kill app", Command: "pkill app", Output: "", Success: true})

	entries, _ := Load(10)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Output != "" {
		t.Errorf("expected empty output, got %q", entries[0].Output)
	}
}

func TestLoad_ZeroLimit(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		Save(Entry{Prompt: "test", Command: "echo", Success: true})
	}

	// limit=0 should return all.
	entries, _ := Load(0)
	if len(entries) != 10 {
		t.Errorf("expected 10 entries with limit=0, got %d", len(entries))
	}
}
