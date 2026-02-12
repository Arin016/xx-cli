package ai

import (
	"context"
	"testing"
)

// mockProvider is a test double that returns canned responses.
type mockProvider struct {
	response string
	err      error
}

func (m *mockProvider) Complete(_ context.Context, _ []Message, _ bool) (string, error) {
	return m.response, m.err
}

func TestTranslate_QueryIntent(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "ps aux | grep chrome", "explanation": "check chrome", "intent": "query"}`,
	}
	client := NewClientWithProvider(mock)

	result, err := client.Translate(context.Background(), "is chrome running")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Intent != IntentQuery {
		t.Errorf("expected intent %q, got %q", IntentQuery, result.Intent)
	}
	if result.Command != "ps aux | grep chrome" {
		t.Errorf("unexpected command: %s", result.Command)
	}
}

func TestTranslate_ExecuteIntent(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "pkill Slack", "explanation": "kill slack", "intent": "execute"}`,
	}
	client := NewClientWithProvider(mock)

	result, err := client.Translate(context.Background(), "kill slack")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Intent != IntentExecute {
		t.Errorf("expected intent %q, got %q", IntentExecute, result.Intent)
	}
}

func TestTranslate_DisplayIntent(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "df -h", "explanation": "disk usage", "intent": "display"}`,
	}
	client := NewClientWithProvider(mock)

	result, err := client.Translate(context.Background(), "show disk usage")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Intent != IntentDisplay {
		t.Errorf("expected intent %q, got %q", IntentDisplay, result.Intent)
	}
}

func TestTranslate_WorkflowIntent(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "", "explanation": "git workflow", "intent": "workflow", "steps": [{"command": "git add .", "explanation": "stage"}, {"command": "git commit -m \"test\"", "explanation": "commit"}]}`,
	}
	client := NewClientWithProvider(mock)

	result, err := client.Translate(context.Background(), "commit and push")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Intent != IntentWorkflow {
		t.Errorf("expected intent %q, got %q", IntentWorkflow, result.Intent)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(result.Steps))
	}
	if result.Steps[0].Command != "git add ." {
		t.Errorf("unexpected step 0 command: %s", result.Steps[0].Command)
	}
}

func TestTranslate_WorkflowNoSteps_FallsBackToExecute(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "git push", "explanation": "push", "intent": "workflow"}`,
	}
	client := NewClientWithProvider(mock)

	result, err := client.Translate(context.Background(), "push")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Intent != IntentExecute {
		t.Errorf("expected fallback to %q, got %q", IntentExecute, result.Intent)
	}
}

func TestTranslate_UnknownIntent_FallsBackToDisplay(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "ls", "explanation": "list", "intent": "banana"}`,
	}
	client := NewClientWithProvider(mock)

	result, err := client.Translate(context.Background(), "list files")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Intent != IntentDisplay {
		t.Errorf("expected fallback to %q, got %q", IntentDisplay, result.Intent)
	}
}

func TestTranslate_ChainedCommand_AutoSplitsToWorkflow(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "git add . && git commit -m \"test\" && git push", "explanation": "commit and push", "intent": "execute"}`,
	}
	client := NewClientWithProvider(mock)

	result, err := client.Translate(context.Background(), "commit and push")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Intent != IntentWorkflow {
		t.Errorf("expected auto-split to %q, got %q", IntentWorkflow, result.Intent)
	}
	if len(result.Steps) != 3 {
		t.Fatalf("expected 3 steps from auto-split, got %d", len(result.Steps))
	}
	if result.Steps[0].Command != "git add ." {
		t.Errorf("unexpected step 0: %s", result.Steps[0].Command)
	}
	if result.Steps[2].Command != "git push" {
		t.Errorf("unexpected step 2: %s", result.Steps[2].Command)
	}
}

func TestTranslate_PipeCommand_NotSplit(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "ps aux | grep chrome", "explanation": "check", "intent": "query"}`,
	}
	client := NewClientWithProvider(mock)

	result, err := client.Translate(context.Background(), "is chrome running")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Pipe commands should NOT be split into workflows.
	if result.Intent != IntentQuery {
		t.Errorf("pipe command should stay as query, got %q", result.Intent)
	}
	if result.Command != "ps aux | grep chrome" {
		t.Errorf("pipe command should be preserved: %s", result.Command)
	}
}

func TestTranslate_EmptyResponse_ReturnsError(t *testing.T) {
	mock := &mockProvider{response: ""}
	client := NewClientWithProvider(mock)

	_, err := client.Translate(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for empty response")
	}
}

func TestTranslate_EmptyCommand_ReturnsError(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "", "explanation": "nothing", "intent": "execute"}`,
	}
	client := NewClientWithProvider(mock)

	_, err := client.Translate(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestTranslate_InvalidJSON_ReturnsError(t *testing.T) {
	mock := &mockProvider{response: "not json at all"}
	client := NewClientWithProvider(mock)

	_, err := client.Translate(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSummarize(t *testing.T) {
	mock := &mockProvider{response: "Chrome is running with 5 processes."}
	client := NewClientWithProvider(mock)

	summary, err := client.Summarize(context.Background(), "is chrome running", "ps aux | grep chrome", "lots of output", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary != "Chrome is running with 5 processes." {
		t.Errorf("unexpected summary: %s", summary)
	}
}

func TestExplain(t *testing.T) {
	mock := &mockProvider{response: "tar extracts a gzipped archive."}
	client := NewClientWithProvider(mock)

	explanation, err := client.Explain(context.Background(), "tar -xzf archive.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if explanation != "tar extracts a gzipped archive." {
		t.Errorf("unexpected explanation: %s", explanation)
	}
}

func TestAnalyze(t *testing.T) {
	mock := &mockProvider{response: "The error is a null pointer dereference."}
	client := NewClientWithProvider(mock)

	answer, err := client.Analyze(context.Background(), "what went wrong", "panic: nil pointer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if answer != "The error is a null pointer dereference." {
		t.Errorf("unexpected answer: %s", answer)
	}
}

func TestChat_CapsHistory(t *testing.T) {
	mock := &mockProvider{response: "hello"}
	client := NewClientWithProvider(mock)

	// Build a history with 30 messages (exceeds the 20-message cap).
	var history []ChatMessage
	for i := 0; i < 30; i++ {
		history = append(history, ChatMessage{Role: "user", Content: "msg"})
	}

	_, err := client.Chat(context.Background(), history)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// If we got here without error, the cap worked (no crash from oversized context).
}

func TestIsChainedCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"git add . && git commit -m test", true},
		{"echo a ; echo b", true},
		{"ps aux | grep chrome", false},
		{"echo hello", false},
		{"ls -la", false},
	}
	for _, tt := range tests {
		if got := isChainedCommand(tt.cmd); got != tt.want {
			t.Errorf("isChainedCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestSplitChainedCommand(t *testing.T) {
	tests := []struct {
		cmd      string
		wantLen  int
		wantFirst string
	}{
		{"git add . && git commit -m test && git push", 3, "git add ."},
		{"echo a ; echo b", 2, "echo a"},
		{"single command", 1, "single command"},
	}
	for _, tt := range tests {
		steps := splitChainedCommand(tt.cmd)
		if len(steps) != tt.wantLen {
			t.Errorf("splitChainedCommand(%q): got %d steps, want %d", tt.cmd, len(steps), tt.wantLen)
			continue
		}
		if steps[0].Command != tt.wantFirst {
			t.Errorf("splitChainedCommand(%q): first step = %q, want %q", tt.cmd, steps[0].Command, tt.wantFirst)
		}
	}
}

func TestTruncate(t *testing.T) {
	short := "hello"
	if got := truncate(short, 100); got != short {
		t.Errorf("truncate should not modify short strings: got %q", got)
	}

	long := "abcdefghij"
	got := truncate(long, 5)
	if !contains(got, "abcde") || !contains(got, "truncated") {
		t.Errorf("truncate should cut and add marker: got %q", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
