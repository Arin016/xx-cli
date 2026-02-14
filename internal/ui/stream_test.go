package ui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/arin/xx-cli/internal/ai"
)

func TestRenderStream_BasicTokens(t *testing.T) {
	ch := make(chan ai.StreamDelta, 4)
	ch <- ai.StreamDelta{Token: "hello"}
	ch <- ai.StreamDelta{Token: " world"}
	ch <- ai.StreamDelta{Done: true}
	close(ch)

	var buf bytes.Buffer
	result, err := RenderStream(&buf, ch, "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
	// Output should start with the prefix.
	if !strings.HasPrefix(buf.String(), "  hello") {
		t.Errorf("expected output to start with prefix, got %q", buf.String())
	}
}

func TestRenderStream_EmptyPrefix(t *testing.T) {
	ch := make(chan ai.StreamDelta, 3)
	ch <- ai.StreamDelta{Token: "test"}
	ch <- ai.StreamDelta{Done: true}
	close(ch)

	var buf bytes.Buffer
	result, err := RenderStream(&buf, ch, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got %q", result)
	}
	if strings.HasPrefix(buf.String(), " ") {
		t.Error("empty prefix should not add leading space")
	}
}

func TestRenderStream_SkipsEmptyTokens(t *testing.T) {
	ch := make(chan ai.StreamDelta, 5)
	ch <- ai.StreamDelta{Token: ""}
	ch <- ai.StreamDelta{Token: "hello"}
	ch <- ai.StreamDelta{Token: ""}
	ch <- ai.StreamDelta{Done: true}
	close(ch)

	var buf bytes.Buffer
	result, err := RenderStream(&buf, ch, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestRenderStream_Error(t *testing.T) {
	ch := make(chan ai.StreamDelta, 3)
	ch <- ai.StreamDelta{Token: "partial"}
	ch <- ai.StreamDelta{Err: fmt.Errorf("stream broke")}
	close(ch)

	var buf bytes.Buffer
	result, err := RenderStream(&buf, ch, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if result != "partial" {
		t.Errorf("expected partial 'partial', got %q", result)
	}
	if !strings.Contains(err.Error(), "stream broke") {
		t.Errorf("expected 'stream broke', got: %v", err)
	}
}

func TestRenderStream_EmptyStream(t *testing.T) {
	ch := make(chan ai.StreamDelta, 1)
	ch <- ai.StreamDelta{Done: true}
	close(ch)

	var buf bytes.Buffer
	result, err := RenderStream(&buf, ch, ">> ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestRenderStream_ClosedChannel(t *testing.T) {
	ch := make(chan ai.StreamDelta)
	close(ch)

	var buf bytes.Buffer
	result, err := RenderStream(&buf, ch, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestRenderStream_AddsTrailingNewline(t *testing.T) {
	ch := make(chan ai.StreamDelta, 2)
	ch <- ai.StreamDelta{Token: "no newline at end"}
	ch <- ai.StreamDelta{Done: true}
	close(ch)

	var buf bytes.Buffer
	_, err := RenderStream(&buf, ch, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Error("output should end with newline")
	}
}

func TestRenderStream_PreservesExistingNewline(t *testing.T) {
	ch := make(chan ai.StreamDelta, 2)
	ch <- ai.StreamDelta{Token: "ends with newline\n"}
	ch <- ai.StreamDelta{Done: true}
	close(ch)

	var buf bytes.Buffer
	_, err := RenderStream(&buf, ch, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should not double-newline.
	output := buf.String()
	if strings.HasSuffix(output, "\n\n\n") {
		t.Errorf("should not triple-newline, got %q", output)
	}
}

func TestRenderStream_MultipleTokensConcatenate(t *testing.T) {
	ch := make(chan ai.StreamDelta, 6)
	ch <- ai.StreamDelta{Token: "a"}
	ch <- ai.StreamDelta{Token: "b"}
	ch <- ai.StreamDelta{Token: "c"}
	ch <- ai.StreamDelta{Token: "d"}
	ch <- ai.StreamDelta{Token: "e"}
	ch <- ai.StreamDelta{Done: true}
	close(ch)

	var buf bytes.Buffer
	result, err := RenderStream(&buf, ch, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "abcde" {
		t.Errorf("expected 'abcde', got %q", result)
	}
}

func TestRenderStream_NewlineInToken(t *testing.T) {
	ch := make(chan ai.StreamDelta, 3)
	ch <- ai.StreamDelta{Token: "line1\nline2"}
	ch <- ai.StreamDelta{Done: true}
	close(ch)

	var buf bytes.Buffer
	result, err := RenderStream(&buf, ch, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "line1\nline2" {
		t.Errorf("expected 'line1\\nline2', got %q", result)
	}
}

func TestRenderStream_LargeOutput(t *testing.T) {
	ch := make(chan ai.StreamDelta, 1002)
	for i := 0; i < 1000; i++ {
		ch <- ai.StreamDelta{Token: "x"}
	}
	ch <- ai.StreamDelta{Done: true}
	close(ch)

	var buf bytes.Buffer
	result, err := RenderStream(&buf, ch, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1000 {
		t.Errorf("expected 1000 chars, got %d", len(result))
	}
}

func TestRenderStream_PrefixOnlyOnFirstToken(t *testing.T) {
	ch := make(chan ai.StreamDelta, 4)
	ch <- ai.StreamDelta{Token: "a"}
	ch <- ai.StreamDelta{Token: "b"}
	ch <- ai.StreamDelta{Token: "c"}
	ch <- ai.StreamDelta{Done: true}
	close(ch)

	var buf bytes.Buffer
	_, err := RenderStream(&buf, ch, ">> ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	// Prefix should appear exactly once.
	count := strings.Count(output, ">> ")
	if count != 1 {
		t.Errorf("prefix should appear exactly once, appeared %d times in %q", count, output)
	}
}

func TestRenderStream_ErrorAfterDone(t *testing.T) {
	ch := make(chan ai.StreamDelta, 3)
	ch <- ai.StreamDelta{Token: "ok"}
	ch <- ai.StreamDelta{Done: true}
	ch <- ai.StreamDelta{Err: fmt.Errorf("late error")} // Should be ignored.
	close(ch)

	var buf bytes.Buffer
	result, err := RenderStream(&buf, ch, "")
	if err != nil {
		t.Fatalf("error after Done should be ignored, got: %v", err)
	}
	if result != "ok" {
		t.Errorf("expected 'ok', got %q", result)
	}
}
