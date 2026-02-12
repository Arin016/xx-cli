package executor

import (
	"strings"
	"testing"
)

func TestRun_SimpleCommand(t *testing.T) {
	output, err := Run("echo hello")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got := strings.TrimSpace(output); got != "hello" {
		t.Errorf("expected 'hello', got '%s'", got)
	}
}

func TestRun_FailingCommand(t *testing.T) {
	_, err := Run("false")
	if err == nil {
		t.Fatal("expected error for failing command")
	}
}

func TestRun_CdCommand(t *testing.T) {
	output, err := Run("cd /tmp")
	if err != nil {
		t.Fatalf("expected no error for cd, got: %v", err)
	}
	if !strings.Contains(output, "__XX_CD__:") {
		t.Errorf("expected cd marker, got: %s", output)
	}
}

func TestIsCdCommand(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"cd", true},
		{"cd /tmp", true},
		{"cd ~/Downloads", true},
		{"echo cd", false},
		{"cdr something", false},
	}

	for _, tt := range tests {
		if got := isCdCommand(tt.input); got != tt.want {
			t.Errorf("isCdCommand(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestExtractCdTarget(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"cd", "~"},
		{"cd /tmp", "/tmp"},
		{"cd ~/Downloads", "~/Downloads"},
	}

	for _, tt := range tests {
		if got := extractCdTarget(tt.input); got != tt.want {
			t.Errorf("extractCdTarget(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
