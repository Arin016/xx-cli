package stats

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDir(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	os.MkdirAll(filepath.Join(dir, ".xx-cli"), 0o700)
	return func() { os.Setenv("HOME", origHome) }
}

func TestSaveAndLoadAll(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	err := Save(Record{
		Prompt:    "is chrome running",
		Command:   "ps aux | grep chrome",
		Intent:    "query",
		AILatency: 500 * time.Millisecond,
		Success:   true,
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	records, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Prompt != "is chrome running" {
		t.Errorf("unexpected prompt: %s", records[0].Prompt)
	}
}

func TestSave_MultipleRecords(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		Save(Record{Prompt: "test", Intent: "query", AILatency: 100 * time.Millisecond, Success: true})
	}

	records, _ := LoadAll()
	if len(records) != 5 {
		t.Fatalf("expected 5 records, got %d", len(records))
	}
}

func TestSummarize_Empty(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	s, err := Summarize()
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}
	if s.TotalCommands != 0 {
		t.Errorf("expected 0 commands, got %d", s.TotalCommands)
	}
}

func TestSummarize_WithData(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	Save(Record{Prompt: "a", Command: "ls", Intent: "display", AILatency: 200 * time.Millisecond, ExecLatency: 50 * time.Millisecond, Success: true, Subcommand: "run"})
	Save(Record{Prompt: "b", Command: "ls", Intent: "display", AILatency: 300 * time.Millisecond, ExecLatency: 60 * time.Millisecond, Success: true, Subcommand: "run"})
	Save(Record{Prompt: "c", Command: "pkill x", Intent: "execute", AILatency: 400 * time.Millisecond, Success: false, Subcommand: "run"})

	s, err := Summarize()
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}
	if s.TotalCommands != 3 {
		t.Errorf("expected 3 commands, got %d", s.TotalCommands)
	}
	if s.SuccessRate < 66 || s.SuccessRate > 67 {
		t.Errorf("expected ~66%% success rate, got %.0f%%", s.SuccessRate)
	}
	if s.IntentBreakdown["display"] != 2 {
		t.Errorf("expected 2 display intents, got %d", s.IntentBreakdown["display"])
	}
	if s.IntentBreakdown["execute"] != 1 {
		t.Errorf("expected 1 execute intent, got %d", s.IntentBreakdown["execute"])
	}
	if s.SubcmdBreakdown["run"] != 3 {
		t.Errorf("expected 3 run subcommands, got %d", s.SubcmdBreakdown["run"])
	}
	if len(s.TopCommands) == 0 {
		t.Fatal("expected top commands")
	}
	if s.TopCommands[0].Command != "ls" || s.TopCommands[0].Count != 2 {
		t.Errorf("expected top command 'ls' with count 2, got %+v", s.TopCommands[0])
	}
	if s.TodayCount != 3 {
		t.Errorf("expected 3 today, got %d", s.TodayCount)
	}
	if s.ThisWeekCount != 3 {
		t.Errorf("expected 3 this week, got %d", s.ThisWeekCount)
	}
}

func TestSummarize_TopN(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	// Create 7 different commands with varying frequencies.
	for i := 0; i < 10; i++ {
		Save(Record{Command: "ls", Intent: "display", AILatency: 100 * time.Millisecond, Success: true})
	}
	for i := 0; i < 5; i++ {
		Save(Record{Command: "ps aux", Intent: "query", AILatency: 100 * time.Millisecond, Success: true})
	}
	for i := 0; i < 3; i++ {
		Save(Record{Command: "df -h", Intent: "display", AILatency: 100 * time.Millisecond, Success: true})
	}
	Save(Record{Command: "whoami", Intent: "query", AILatency: 100 * time.Millisecond, Success: true})
	Save(Record{Command: "uname", Intent: "query", AILatency: 100 * time.Millisecond, Success: true})
	Save(Record{Command: "date", Intent: "query", AILatency: 100 * time.Millisecond, Success: true})
	Save(Record{Command: "uptime", Intent: "query", AILatency: 100 * time.Millisecond, Success: true})

	s, _ := Summarize()
	if len(s.TopCommands) != 5 {
		t.Fatalf("expected 5 top commands, got %d", len(s.TopCommands))
	}
	// First should be "ls" with 10.
	if s.TopCommands[0].Command != "ls" {
		t.Errorf("expected top command 'ls', got %q", s.TopCommands[0].Command)
	}
	if s.TopCommands[0].Count != 10 {
		t.Errorf("expected count 10, got %d", s.TopCommands[0].Count)
	}
}

func TestLoadAll_NoFile(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	records, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll on missing file should not error: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil records, got %v", records)
	}
}

func TestSave_CapsAt1000(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	for i := 0; i < 1010; i++ {
		Save(Record{Prompt: "test", Intent: "query", AILatency: 100 * time.Millisecond, Success: true})
	}

	records, _ := LoadAll()
	if len(records) > 1000 {
		t.Errorf("expected at most 1000 records, got %d", len(records))
	}
}

func TestSummarize_AvgLatency(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	Save(Record{Prompt: "a", Intent: "query", AILatency: 200 * time.Millisecond, ExecLatency: 100 * time.Millisecond, Success: true})
	Save(Record{Prompt: "b", Intent: "query", AILatency: 400 * time.Millisecond, ExecLatency: 300 * time.Millisecond, Success: true})

	s, _ := Summarize()
	// AI latency: (200+400)/2 = 300ms, but stored as ms already so (200+400)/2 = 300.
	// Actually Save divides by time.Millisecond, so 200ms/ms = 200, 400ms/ms = 400.
	// Average = (200+400)/2 = 300.
	if s.AvgAILatencyMs != 300 {
		t.Errorf("expected avg AI latency 300ms, got %d", s.AvgAILatencyMs)
	}
	if s.AvgExecLatencyMs != 200 {
		t.Errorf("expected avg exec latency 200ms, got %d", s.AvgExecLatencyMs)
	}
}

func TestSummarize_AllFailed(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	Save(Record{Prompt: "a", Intent: "execute", AILatency: 100 * time.Millisecond, Success: false})
	Save(Record{Prompt: "b", Intent: "execute", AILatency: 100 * time.Millisecond, Success: false})

	s, _ := Summarize()
	if s.SuccessRate != 0 {
		t.Errorf("expected 0%% success rate, got %.0f%%", s.SuccessRate)
	}
}

func TestSummarize_AllSucceeded(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	Save(Record{Prompt: "a", Intent: "query", AILatency: 100 * time.Millisecond, Success: true})
	Save(Record{Prompt: "b", Intent: "query", AILatency: 100 * time.Millisecond, Success: true})

	s, _ := Summarize()
	if s.SuccessRate != 100 {
		t.Errorf("expected 100%% success rate, got %.0f%%", s.SuccessRate)
	}
}

func TestSummarize_NoExecLatency(t *testing.T) {
	cleanup := setupTestDir(t)
	defer cleanup()

	// Records with zero exec latency (e.g. explain, chat).
	Save(Record{Prompt: "a", Intent: "query", AILatency: 200 * time.Millisecond, Success: true})

	s, _ := Summarize()
	if s.AvgExecLatencyMs != 0 {
		t.Errorf("expected 0 avg exec latency when no exec, got %d", s.AvgExecLatencyMs)
	}
}

func TestTopN_LessThanN(t *testing.T) {
	freq := map[string]int{"ls": 5, "ps": 3}
	result := topN(freq, 10)
	if len(result) != 2 {
		t.Errorf("expected 2 results when fewer than N, got %d", len(result))
	}
	if result[0].Command != "ls" || result[0].Count != 5 {
		t.Errorf("expected top command 'ls' with count 5, got %+v", result[0])
	}
}

func TestTopN_Empty(t *testing.T) {
	result := topN(map[string]int{}, 5)
	if len(result) != 0 {
		t.Errorf("expected 0 results for empty map, got %d", len(result))
	}
}

func TestTopN_ExactlyN(t *testing.T) {
	freq := map[string]int{"a": 3, "b": 2, "c": 1}
	result := topN(freq, 3)
	if len(result) != 3 {
		t.Errorf("expected 3 results, got %d", len(result))
	}
	if result[0].Count < result[1].Count || result[1].Count < result[2].Count {
		t.Error("results should be sorted by count descending")
	}
}
