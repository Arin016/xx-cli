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
