// Package stats provides structured observability for xx-cli.
// It tracks per-command metrics (AI latency, execution time, intent,
// success/failure) and persists them to ~/.xx-cli/stats.json.
package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/arin/xx-cli/internal/config"
)

const fileName = "stats.json"

// Record is a single instrumented command execution.
type Record struct {
	Timestamp   time.Time     `json:"timestamp"`
	Prompt      string        `json:"prompt"`
	Command     string        `json:"command,omitempty"`
	Intent      string        `json:"intent"`
	AILatency   time.Duration `json:"ai_latency_ms"`
	ExecLatency time.Duration `json:"exec_latency_ms,omitempty"`
	Success     bool          `json:"success"`
	Subcommand  string        `json:"subcommand,omitempty"` // "run", "explain", "chat", etc.
}

// Summary is the aggregated stats dashboard.
type Summary struct {
	TotalCommands   int            `json:"total_commands"`
	SuccessRate     float64        `json:"success_rate"`
	AvgAILatencyMs  int64          `json:"avg_ai_latency_ms"`
	AvgExecLatencyMs int64         `json:"avg_exec_latency_ms"`
	IntentBreakdown map[string]int `json:"intent_breakdown"`
	SubcmdBreakdown map[string]int `json:"subcmd_breakdown"`
	TopCommands     []CommandCount `json:"top_commands"`
	TodayCount      int            `json:"today_count"`
	ThisWeekCount   int            `json:"this_week_count"`
}

// CommandCount pairs a command with its usage count.
type CommandCount struct {
	Command string `json:"command"`
	Count   int    `json:"count"`
}

var fileMu sync.Mutex

func statsPath() string {
	return filepath.Join(config.Dir(), fileName)
}

// Save appends a new record to the stats file.
func Save(r Record) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	r.Timestamp = time.Now()
	// Store durations as milliseconds for readability.
	r.AILatency = r.AILatency / time.Millisecond
	r.ExecLatency = r.ExecLatency / time.Millisecond

	records, _ := loadAll()
	records = append(records, r)

	// Cap at 1000 records.
	if len(records) > 1000 {
		records = records[len(records)-1000:]
	}

	if err := os.MkdirAll(config.Dir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statsPath(), data, 0o600)
}

// LoadAll returns all stored records.
func LoadAll() ([]Record, error) {
	return loadAll()
}

func loadAll() ([]Record, error) {
	data, err := os.ReadFile(statsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// Summarize computes aggregated stats from all records.
func Summarize() (*Summary, error) {
	records, err := loadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return &Summary{
			IntentBreakdown: map[string]int{},
			SubcmdBreakdown: map[string]int{},
		}, nil
	}

	s := &Summary{
		TotalCommands:   len(records),
		IntentBreakdown: map[string]int{},
		SubcmdBreakdown: map[string]int{},
	}

	var totalAI, totalExec int64
	var execCount int
	var successCount int
	cmdFreq := map[string]int{}
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	weekAgo := now.AddDate(0, 0, -7)

	for _, r := range records {
		if r.Success {
			successCount++
		}
		totalAI += int64(r.AILatency)
		if r.ExecLatency > 0 {
			totalExec += int64(r.ExecLatency)
			execCount++
		}
		if r.Intent != "" {
			s.IntentBreakdown[r.Intent]++
		}
		if r.Subcommand != "" {
			s.SubcmdBreakdown[r.Subcommand]++
		}
		if r.Command != "" {
			cmdFreq[r.Command]++
		}
		if r.Timestamp.After(today) {
			s.TodayCount++
		}
		if r.Timestamp.After(weekAgo) {
			s.ThisWeekCount++
		}
	}

	s.SuccessRate = float64(successCount) / float64(len(records)) * 100
	s.AvgAILatencyMs = totalAI / int64(len(records))
	if execCount > 0 {
		s.AvgExecLatencyMs = totalExec / int64(execCount)
	}

	// Top 5 commands by frequency.
	s.TopCommands = topN(cmdFreq, 5)

	return s, nil
}

func topN(freq map[string]int, n int) []CommandCount {
	var all []CommandCount
	for cmd, count := range freq {
		all = append(all, CommandCount{Command: cmd, Count: count})
	}
	// Simple selection sort for small N.
	for i := 0; i < len(all) && i < n; i++ {
		maxIdx := i
		for j := i + 1; j < len(all); j++ {
			if all[j].Count > all[maxIdx].Count {
				maxIdx = j
			}
		}
		all[i], all[maxIdx] = all[maxIdx], all[i]
	}
	if len(all) > n {
		all = all[:n]
	}
	return all
}
