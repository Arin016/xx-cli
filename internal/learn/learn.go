// Package learn manages user corrections that improve AI accuracy over time.
// Corrections are stored as JSON in ~/.xx-cli/learned.json and injected
// as few-shot examples into the system prompt.
package learn

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/arin/xx-cli/internal/config"
)

const fileName = "learned.json"

// Correction maps a natural language prompt to the correct command.
type Correction struct {
	Prompt  string `json:"prompt"`
	Command string `json:"command"`
}

func learnedPath() string {
	return filepath.Join(config.Dir(), fileName)
}

// Save stores a new correction.
func Save(c Correction) error {
	corrections, _ := LoadAll()

	// Update existing correction for the same prompt, or append.
	found := false
	for i, existing := range corrections {
		if existing.Prompt == c.Prompt {
			corrections[i].Command = c.Command
			found = true
			break
		}
	}
	if !found {
		corrections = append(corrections, c)
	}

	if err := os.MkdirAll(config.Dir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(corrections, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(learnedPath(), data, 0o600)
}

// LoadAll returns all stored corrections.
func LoadAll() ([]Correction, error) {
	data, err := os.ReadFile(learnedPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var corrections []Correction
	if err := json.Unmarshal(data, &corrections); err != nil {
		return nil, err
	}
	return corrections, nil
}

// FewShotPrompt returns a string of learned examples for injection into the system prompt.
func FewShotPrompt() string {
	corrections, err := LoadAll()
	if err != nil || len(corrections) == 0 {
		return ""
	}

	var sb []byte
	sb = append(sb, "\nUser corrections (always prefer these over your own judgment):\n"...)
	for _, c := range corrections {
		sb = append(sb, "- \""+c.Prompt+"\" → "+c.Command+"\n"...)
	}
	return string(sb)
}
