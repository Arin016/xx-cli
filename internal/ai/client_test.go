package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// --- Mock providers ---

// mockProvider is a test double that returns canned responses (non-streaming).
type mockProvider struct {
	response string
	err      error
	// Track calls for verification.
	calls    int
	lastMsgs []Message
	lastJSON bool
}

func (m *mockProvider) Complete(_ context.Context, msgs []Message, jsonMode bool) (string, error) {
	m.calls++
	m.lastMsgs = msgs
	m.lastJSON = jsonMode
	return m.response, m.err
}

// mockStreamProvider implements both Provider and StreamingProvider.
type mockStreamProvider struct {
	mockProvider
	tokens []string // Tokens to emit one by one.
	streamErr error // Error to emit mid-stream.
}

func (m *mockStreamProvider) CompleteStream(_ context.Context, msgs []Message) <-chan StreamDelta {
	m.calls++
	m.lastMsgs = msgs
	ch := make(chan StreamDelta)
	go func() {
		defer close(ch)
		for _, tok := range m.tokens {
			ch <- StreamDelta{Token: tok}
		}
		if m.streamErr != nil {
			ch <- StreamDelta{Err: m.streamErr}
			return
		}
		ch <- StreamDelta{Done: true}
	}()
	return ch
}

// --- Translate tests ---

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
	if !mock.lastJSON {
		t.Error("Translate should request JSON mode")
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

func TestTranslate_SemicolonChain_AutoSplitsToWorkflow(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "echo a ; echo b", "explanation": "two commands", "intent": "execute"}`,
	}
	client := NewClientWithProvider(mock)

	result, err := client.Translate(context.Background(), "echo a then b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Intent != IntentWorkflow {
		t.Errorf("expected auto-split to %q, got %q", IntentWorkflow, result.Intent)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(result.Steps))
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

func TestTranslate_ProviderError_Propagates(t *testing.T) {
	mock := &mockProvider{err: fmt.Errorf("connection refused")}
	client := NewClientWithProvider(mock)

	_, err := client.Translate(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error from provider")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("expected provider error, got: %v", err)
	}
}

func TestTranslate_WorkflowEmptyCommand_Allowed(t *testing.T) {
	mock := &mockProvider{
		response: `{"command": "", "explanation": "multi", "intent": "workflow", "steps": [{"command": "echo hi", "explanation": "greet"}]}`,
	}
	client := NewClientWithProvider(mock)

	result, err := client.Translate(context.Background(), "greet")
	if err != nil {
		t.Fatalf("workflow with empty command but steps should succeed: %v", err)
	}
	if result.Intent != IntentWorkflow {
		t.Errorf("expected workflow, got %q", result.Intent)
	}
}

// --- Non-streaming client method tests ---

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
	if mock.lastJSON {
		t.Error("Summarize should NOT request JSON mode")
	}
}

func TestSummarize_FailedCommand(t *testing.T) {
	mock := &mockProvider{response: "The command failed because the file was not found."}
	client := NewClientWithProvider(mock)

	summary, err := client.Summarize(context.Background(), "find config", "find / -name config", "no such file", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify the "failed" status is passed to the AI.
	found := false
	for _, m := range mock.lastMsgs {
		if strings.Contains(m.Content, "failed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'failed' status in messages")
	}
	if summary == "" {
		t.Error("expected non-empty summary")
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

func TestExplain_Error(t *testing.T) {
	mock := &mockProvider{err: fmt.Errorf("model not loaded")}
	client := NewClientWithProvider(mock)

	_, err := client.Explain(context.Background(), "ls")
	if err == nil {
		t.Fatal("expected error")
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

func TestAnalyze_TruncatesLongInput(t *testing.T) {
	mock := &mockProvider{response: "truncated analysis"}
	client := NewClientWithProvider(mock)

	longData := strings.Repeat("x", 5000)
	_, err := client.Analyze(context.Background(), "what is this", longData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify the data was truncated in the message.
	for _, m := range mock.lastMsgs {
		if strings.Contains(m.Content, "truncated") {
			return // Found truncation marker.
		}
	}
	t.Error("expected data to be truncated at 4000 chars")
}

func TestChat_CapsHistory(t *testing.T) {
	mock := &mockProvider{response: "hello"}
	client := NewClientWithProvider(mock)

	var history []ChatMessage
	for i := 0; i < 30; i++ {
		history = append(history, ChatMessage{Role: "user", Content: fmt.Sprintf("msg %d", i)})
	}

	_, err := client.Chat(context.Background(), history)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// System message + 20 trimmed history messages = 21 total.
	if len(mock.lastMsgs) != 21 {
		t.Errorf("expected 21 messages (1 system + 20 history), got %d", len(mock.lastMsgs))
	}
	// The first history message should be msg 10 (30 - 20 = 10).
	if !strings.Contains(mock.lastMsgs[1].Content, "msg 10") {
		t.Errorf("expected trimmed history to start at msg 10, got: %s", mock.lastMsgs[1].Content)
	}
}

func TestChat_EmptyHistory(t *testing.T) {
	mock := &mockProvider{response: "hi there"}
	client := NewClientWithProvider(mock)

	reply, err := client.Chat(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reply != "hi there" {
		t.Errorf("unexpected reply: %s", reply)
	}
	// Should have just the system message.
	if len(mock.lastMsgs) != 1 {
		t.Errorf("expected 1 system message, got %d", len(mock.lastMsgs))
	}
}

func TestRecap(t *testing.T) {
	mock := &mockProvider{response: "Built and tested AI provider."}
	client := NewClientWithProvider(mock)

	recap, err := client.Recap(context.Background(), "[14:00] test → go test ./... ✓", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recap != "Built and tested AI provider." {
		t.Errorf("unexpected recap: %s", recap)
	}
}

func TestDiagnose(t *testing.T) {
	mock := &mockProvider{response: "1) Permission denied 2) Root owns dir 3) sudo chown"}
	client := NewClientWithProvider(mock)

	diagnosis, err := client.Diagnose(context.Background(), "EACCES: permission denied")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(diagnosis, "Permission denied") {
		t.Errorf("unexpected diagnosis: %s", diagnosis)
	}
}

func TestDiffExplain(t *testing.T) {
	mock := &mockProvider{response: "Added streaming support to AI client."}
	client := NewClientWithProvider(mock)

	explanation, err := client.DiffExplain(context.Background(), "diff --git a/client.go...")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if explanation != "Added streaming support to AI client." {
		t.Errorf("unexpected explanation: %s", explanation)
	}
}

func TestSmartRetry(t *testing.T) {
	mock := &mockProvider{response: "pip3 install tensorflow"}
	client := NewClientWithProvider(mock)

	fix, err := client.SmartRetry(context.Background(), "install tensorflow", "pip install tensorflow", "ERROR: not found")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fix != "pip3 install tensorflow" {
		t.Errorf("unexpected fix: %s", fix)
	}
}

func TestSmartRetry_StripsBackticks(t *testing.T) {
	mock := &mockProvider{response: "`pip3 install tensorflow`"}
	client := NewClientWithProvider(mock)

	fix, err := client.SmartRetry(context.Background(), "install tf", "pip install tf", "error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fix != "pip3 install tensorflow" {
		t.Errorf("expected backticks stripped, got: %s", fix)
	}
}

func TestSmartRetry_StripsQuotes(t *testing.T) {
	mock := &mockProvider{response: `"brew install node"`}
	client := NewClientWithProvider(mock)

	fix, err := client.SmartRetry(context.Background(), "install node", "apt install node", "not found")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fix != "brew install node" {
		t.Errorf("expected quotes stripped, got: %s", fix)
	}
}

func TestSmartRetry_EmptyFix(t *testing.T) {
	mock := &mockProvider{response: "   "}
	client := NewClientWithProvider(mock)

	fix, err := client.SmartRetry(context.Background(), "do thing", "thing", "error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fix != "" {
		t.Errorf("expected empty fix for whitespace response, got: %q", fix)
	}
}

// --- Streaming tests ---

func TestStreamOrFallback_UsesStreamingProvider(t *testing.T) {
	mock := &mockStreamProvider{
		tokens: []string{"hello", " ", "world"},
	}
	client := NewClientWithProvider(mock)

	ch := client.streamOrFallback(context.Background(), []Message{
		{Role: "user", Content: "test"},
	})

	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestStreamOrFallback_FallsBackToComplete(t *testing.T) {
	// mockProvider does NOT implement StreamingProvider.
	mock := &mockProvider{response: "full response"}
	client := NewClientWithProvider(mock)

	ch := client.streamOrFallback(context.Background(), []Message{
		{Role: "user", Content: "test"},
	})

	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "full response" {
		t.Errorf("expected 'full response', got %q", result)
	}
}

func TestStreamOrFallback_FallbackError(t *testing.T) {
	mock := &mockProvider{err: fmt.Errorf("provider down")}
	client := NewClientWithProvider(mock)

	ch := client.streamOrFallback(context.Background(), []Message{
		{Role: "user", Content: "test"},
	})

	_, err := collectStream(ch)
	if err == nil {
		t.Fatal("expected error from fallback")
	}
	if !strings.Contains(err.Error(), "provider down") {
		t.Errorf("expected 'provider down', got: %v", err)
	}
}

func TestStreamOrFallback_StreamError(t *testing.T) {
	mock := &mockStreamProvider{
		tokens:    []string{"partial"},
		streamErr: fmt.Errorf("stream interrupted"),
	}
	client := NewClientWithProvider(mock)

	ch := client.streamOrFallback(context.Background(), []Message{
		{Role: "user", Content: "test"},
	})

	result, err := collectStream(ch)
	if err == nil {
		t.Fatal("expected stream error")
	}
	if result != "partial" {
		t.Errorf("expected partial result 'partial', got %q", result)
	}
	if !strings.Contains(err.Error(), "stream interrupted") {
		t.Errorf("expected 'stream interrupted', got: %v", err)
	}
}

func TestExplainStream_WithStreamingProvider(t *testing.T) {
	mock := &mockStreamProvider{
		tokens: []string{"tar ", "extracts ", "files"},
	}
	client := NewClientWithProvider(mock)

	ch := client.ExplainStream(context.Background(), "tar -xzf archive.tar.gz")
	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "tar extracts files" {
		t.Errorf("expected 'tar extracts files', got %q", result)
	}
}

func TestExplainStream_FallbackToComplete(t *testing.T) {
	mock := &mockProvider{response: "tar extracts a gzipped archive."}
	client := NewClientWithProvider(mock)

	ch := client.ExplainStream(context.Background(), "tar -xzf archive.tar.gz")
	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "tar extracts a gzipped archive." {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestSummarizeStream(t *testing.T) {
	mock := &mockStreamProvider{
		tokens: []string{"Chrome ", "is ", "running."},
	}
	client := NewClientWithProvider(mock)

	ch := client.SummarizeStream(context.Background(), "is chrome running", "ps aux", "output", true)
	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Chrome is running." {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestAnalyzeStream(t *testing.T) {
	mock := &mockStreamProvider{
		tokens: []string{"The ", "error ", "is ", "OOM."},
	}
	client := NewClientWithProvider(mock)

	ch := client.AnalyzeStream(context.Background(), "what happened", "killed: OOM")
	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "The error is OOM." {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestChatStream(t *testing.T) {
	mock := &mockStreamProvider{
		tokens: []string{"Use ", "df -h"},
	}
	client := NewClientWithProvider(mock)

	history := []ChatMessage{
		{Role: "user", Content: "how to check disk space"},
	}
	ch := client.ChatStream(context.Background(), history)
	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Use df -h" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestChatStream_CapsHistory(t *testing.T) {
	mock := &mockStreamProvider{
		tokens: []string{"ok"},
	}
	client := NewClientWithProvider(mock)

	var history []ChatMessage
	for i := 0; i < 30; i++ {
		history = append(history, ChatMessage{Role: "user", Content: fmt.Sprintf("msg %d", i)})
	}

	ch := client.ChatStream(context.Background(), history)
	_, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// System + 20 trimmed = 21.
	if len(mock.lastMsgs) != 21 {
		t.Errorf("expected 21 messages, got %d", len(mock.lastMsgs))
	}
}

func TestDiagnoseStream(t *testing.T) {
	mock := &mockStreamProvider{
		tokens: []string{"1) ", "Permission ", "denied"},
	}
	client := NewClientWithProvider(mock)

	ch := client.DiagnoseStream(context.Background(), "EACCES")
	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "1) Permission denied" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestRecapStream(t *testing.T) {
	mock := &mockStreamProvider{
		tokens: []string{"Built ", "AI ", "provider."},
	}
	client := NewClientWithProvider(mock)

	ch := client.RecapStream(context.Background(), "history data", 5)
	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Built AI provider." {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestDiffExplainStream(t *testing.T) {
	mock := &mockStreamProvider{
		tokens: []string{"Added ", "streaming."},
	}
	client := NewClientWithProvider(mock)

	ch := client.DiffExplainStream(context.Background(), "diff content")
	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Added streaming." {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestCollectStream_EmptyChannel(t *testing.T) {
	ch := make(chan StreamDelta)
	close(ch)

	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestCollectStream_OnlyDone(t *testing.T) {
	ch := make(chan StreamDelta, 1)
	ch <- StreamDelta{Done: true}
	close(ch)

	result, err := collectStream(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestCollectStream_ErrorMidStream(t *testing.T) {
	ch := make(chan StreamDelta, 3)
	ch <- StreamDelta{Token: "hello"}
	ch <- StreamDelta{Err: fmt.Errorf("broken")}
	ch <- StreamDelta{Token: "world"} // Should not be reached.
	close(ch)

	result, err := collectStream(ch)
	if err == nil {
		t.Fatal("expected error")
	}
	if result != "hello" {
		t.Errorf("expected partial 'hello', got %q", result)
	}
}

// --- Helper function tests ---

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
		{"echo '&&' something", false}, // && inside quotes, not a real chain.
		{"", false},
	}
	for _, tt := range tests {
		if got := isChainedCommand(tt.cmd); got != tt.want {
			t.Errorf("isChainedCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestSplitChainedCommand(t *testing.T) {
	tests := []struct {
		cmd       string
		wantLen   int
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
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"hello", 100, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hello\n... (truncated)"},
		{"", 10, ""},
	}
	for _, tt := range tests {
		got := truncate(tt.input, tt.maxLen)
		if tt.maxLen >= len(tt.input) {
			if got != tt.input {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.input)
			}
		} else {
			if !strings.Contains(got, "truncated") {
				t.Errorf("truncate(%q, %d) should contain 'truncated', got %q", tt.input, tt.maxLen, got)
			}
		}
	}
}

func TestDetectShell(t *testing.T) {
	shell := detectShell()
	if shell == "" {
		t.Error("detectShell should return a non-empty string")
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	prompt := buildSystemPrompt()
	if prompt == "" {
		t.Fatal("buildSystemPrompt should return a non-empty string")
	}
	if !strings.Contains(prompt, "query") {
		t.Error("system prompt should mention intent types")
	}
	if !strings.Contains(prompt, "workflow") {
		t.Error("system prompt should mention workflow intent")
	}
	if !strings.Contains(prompt, "JSON") {
		t.Error("system prompt should mention JSON format")
	}
}
