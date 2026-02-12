// Package history manages the command history for xx-cli.
// History is stored as a JSON file in the user's config directory.
package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/arin/xx-cli/internal/config"
)

const (
	fileName   = "history.json"
	maxEntries = 500
)

// fileMu guards concurrent access to the history file.
var fileMu sync.Mutex

// Entry represents a single history record.
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Prompt    string    `json:"prompt"`
	Command   string    `json:"command"`
	Output    string    `json:"output,omitempty"`
	Success   bool      `json:"success"`
}

func historyPath() string {
	return filepath.Join(config.Dir(), fileName)
}

// Save appends a new entry to the history file.
func Save(entry Entry) error {
	fileMu.Lock()
	defer fileMu.Unlock()

	entry.Timestamp = time.Now()

	entries, _ := loadAll()
	entries = append(entries, entry)

	// Trim to max entries, keeping the most recent.
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}

	if err := os.MkdirAll(config.Dir(), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(historyPath(), data, 0o600)
}

// Load returns the most recent n history entries.
func Load(limit int) ([]Entry, error) {
	entries, err := loadAll()
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	return entries, nil
}

func loadAll() ([]Entry, error) {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}
