package config

import (
	"os"
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
