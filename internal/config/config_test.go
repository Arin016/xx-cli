package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_DefaultModel(t *testing.T) {
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", t.TempDir())
	defer os.Setenv("HOME", origHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Model != defaultModel {
		t.Errorf("expected default model %q, got %q", defaultModel, cfg.Model)
	}
}

func TestLoad_ModelFromEnv(t *testing.T) {
	t.Setenv(envKeyModel, "llama3.1:latest")

	origHome := os.Getenv("HOME")
	t.Setenv("HOME", t.TempDir())
	defer os.Setenv("HOME", origHome)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.Model != "llama3.1:latest" {
		t.Errorf("expected model from env, got: %s", cfg.Model)
	}
}

func TestLoad_NeverErrors(t *testing.T) {
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", t.TempDir())
	defer os.Setenv("HOME", origHome)

	_, err := Load()
	if err != nil {
		t.Fatalf("Load should never error for Ollama setup, got: %v", err)
	}
}

func TestDir_ReturnsNonEmpty(t *testing.T) {
	dir := Dir()
	if dir == "" {
		t.Fatal("Dir should return a non-empty path")
	}
}

func TestDir_ContainsDirName(t *testing.T) {
	dir := Dir()
	if !strings.Contains(dir, ".xx-cli") {
		t.Errorf("Dir should contain '.xx-cli', got: %s", dir)
	}
}

func TestSetModel(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	err := SetModel("llama3.1:latest")
	if err != nil {
		t.Fatalf("SetModel failed: %v", err)
	}

	cfg, _ := Load()
	if cfg.Model != "llama3.1:latest" {
		t.Errorf("expected model 'llama3.1:latest', got %q", cfg.Model)
	}
}

func TestSetAPIKey(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	err := SetAPIKey("test-key-123")
	if err != nil {
		t.Fatalf("SetAPIKey failed: %v", err)
	}

	cfg, _ := Load()
	if cfg.APIKey != "test-key-123" {
		t.Errorf("expected API key 'test-key-123', got %q", cfg.APIKey)
	}
}

func TestSetModel_PreservesAPIKey(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	SetAPIKey("my-key")
	SetModel("custom-model")

	cfg, _ := Load()
	if cfg.APIKey != "my-key" {
		t.Errorf("SetModel should preserve API key, got %q", cfg.APIKey)
	}
	if cfg.Model != "custom-model" {
		t.Errorf("expected model 'custom-model', got %q", cfg.Model)
	}
}

func TestLoad_EmptyModelFallsBackToDefault(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Write a config with empty model.
	os.MkdirAll(filepath.Join(tmpDir, ".xx-cli"), 0o700)
	os.WriteFile(filepath.Join(tmpDir, ".xx-cli", "config.json"), []byte(`{"model":""}`), 0o600)

	cfg, _ := Load()
	if cfg.Model != defaultModel {
		t.Errorf("empty model should fall back to default, got %q", cfg.Model)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	os.MkdirAll(filepath.Join(tmpDir, ".xx-cli"), 0o700)
	os.WriteFile(filepath.Join(tmpDir, ".xx-cli", "config.json"), []byte(`not json`), 0o600)

	// Should not error — gracefully falls back to defaults.
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load should not error on invalid JSON: %v", err)
	}
	if cfg.Model != defaultModel {
		t.Errorf("expected default model on invalid JSON, got %q", cfg.Model)
	}
}
