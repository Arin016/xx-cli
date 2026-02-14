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

func TestRun_MultiLineOutput(t *testing.T) {
	output, err := Run("echo -e 'line1\nline2\nline3'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "line1") || !strings.Contains(output, "line3") {
		t.Errorf("expected multi-line output, got: %s", output)
	}
}

func TestRun_StderrCapture(t *testing.T) {
	output, err := Run("echo error >&2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "error") {
		t.Errorf("expected stderr captured, got: %s", output)
	}
}

func TestRun_EmptyCommand(t *testing.T) {
	output, err := Run("")
	// Empty command behavior varies by shell, but shouldn't panic.
	_ = output
	_ = err
}

func TestRun_ExitCode(t *testing.T) {
	_, err := Run("exit 42")
	if err == nil {
		t.Fatal("expected error for non-zero exit code")
	}
}

func TestExpandHome(t *testing.T) {
	expanded := expandHome("~/Documents")
	if strings.HasPrefix(expanded, "~") {
		t.Errorf("expected ~ to be expanded, got: %s", expanded)
	}
	if !strings.Contains(expanded, "Documents") {
		t.Errorf("expected path to contain 'Documents', got: %s", expanded)
	}
}

func TestExpandHome_NoTilde(t *testing.T) {
	path := "/usr/local/bin"
	expanded := expandHome(path)
	if expanded != path {
		t.Errorf("path without ~ should be unchanged, got: %s", expanded)
	}
}

func TestShellAndFlag(t *testing.T) {
	shell, flag := shellAndFlag()
	if shell == "" {
		t.Error("shell should not be empty")
	}
	if flag == "" {
		t.Error("flag should not be empty")
	}
}

func TestIsCdCommand_EdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"  cd  ", true},
		{"  cd /tmp  ", true},
		{"CD /tmp", false}, // Case-sensitive.
		{"cdr", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isCdCommand(tt.input); got != tt.want {
			t.Errorf("isCdCommand(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestExtractCdTarget_EdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"cd   /tmp  ", "/tmp"},
		{"cd", "~"},
		{"cd .", "."},
		{"cd ..", ".."},
	}
	for _, tt := range tests {
		if got := extractCdTarget(tt.input); got != tt.want {
			t.Errorf("extractCdTarget(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRun_CdWithTilde(t *testing.T) {
	output, err := Run("cd ~")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "__XX_CD__:") {
		t.Errorf("expected cd marker, got: %s", output)
	}
	// Should have expanded the tilde.
	if strings.Contains(output, "~") {
		t.Errorf("expected tilde to be expanded, got: %s", output)
	}
}
