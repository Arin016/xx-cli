package learn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	// Override the config dir for testing.
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	// Create the .xx-cli directory.
	os.MkdirAll(filepath.Join(dir, ".xx-cli"), 0o700)
	return dir, func() {
		os.Setenv("HOME", origHome)
	}
}

func TestSaveAndLoadAll(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	err := Save(Correction{Prompt: "run tests", Command: "make test"})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	corrections, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(corrections) != 1 {
		t.Fatalf("expected 1 correction, got %d", len(corrections))
	}
	if corrections[0].Prompt != "run tests" {
		t.Errorf("expected prompt 'run tests', got %q", corrections[0].Prompt)
	}
	if corrections[0].Command != "make test" {
		t.Errorf("expected command 'make test', got %q", corrections[0].Command)
	}
}

func TestSave_UpdatesExisting(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	Save(Correction{Prompt: "deploy", Command: "old-deploy.sh"})
	Save(Correction{Prompt: "deploy", Command: "new-deploy.sh"})

	corrections, _ := LoadAll()
	if len(corrections) != 1 {
		t.Fatalf("expected 1 correction (updated), got %d", len(corrections))
	}
	if corrections[0].Command != "new-deploy.sh" {
		t.Errorf("expected updated command, got %q", corrections[0].Command)
	}
}

func TestSave_MultipleCorrections(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	Save(Correction{Prompt: "run tests", Command: "make test"})
	Save(Correction{Prompt: "deploy", Command: "./deploy.sh"})
	Save(Correction{Prompt: "lint", Command: "golangci-lint run"})

	corrections, _ := LoadAll()
	if len(corrections) != 3 {
		t.Fatalf("expected 3 corrections, got %d", len(corrections))
	}
}

func TestLoadAll_EmptyFile(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	// No file exists yet.
	corrections, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll on missing file should not error: %v", err)
	}
	if corrections != nil {
		t.Errorf("expected nil corrections, got %v", corrections)
	}
}

func TestFewShotPrompt_Empty(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	prompt := FewShotPrompt()
	if prompt != "" {
		t.Errorf("expected empty prompt with no corrections, got %q", prompt)
	}
}

func TestFewShotPrompt_WithCorrections(t *testing.T) {
	_, cleanup := setupTestDir(t)
	defer cleanup()

	Save(Correction{Prompt: "run tests", Command: "make test"})
	Save(Correction{Prompt: "deploy", Command: "./deploy.sh"})

	prompt := FewShotPrompt()
	if !strings.Contains(prompt, "run tests") {
		t.Error("few-shot prompt should contain 'run tests'")
	}
	if !strings.Contains(prompt, "make test") {
		t.Error("few-shot prompt should contain 'make test'")
	}
	if !strings.Contains(prompt, "deploy") {
		t.Error("few-shot prompt should contain 'deploy'")
	}
	if !strings.Contains(prompt, "User corrections") {
		t.Error("few-shot prompt should have header")
	}
}
