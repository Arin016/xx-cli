// Package ui provides terminal UI helpers.
package ui

import (
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// Spinner wraps a terminal spinner for loading states.
type Spinner struct {
	s *spinner.Spinner
}

// NewSpinner creates a spinner with the given message.
func NewSpinner(msg string) *Spinner {
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Suffix = "  " + msg
	s.Color("cyan")
	return &Spinner{s: s}
}

// Start begins the spinner animation.
func (sp *Spinner) Start() {
	sp.s.Start()
}

// Stop halts the spinner and clears the line.
func (sp *Spinner) Stop() {
	sp.s.Stop()
}

// Success stops the spinner and prints a green check.
func (sp *Spinner) Success(msg string) {
	sp.s.Stop()
	green := color.New(color.FgGreen)
	green.Fprintf(os.Stderr, "  ✓ %s\n", msg)
}

// Fail stops the spinner and prints a red cross.
func (sp *Spinner) Fail(msg string) {
	sp.s.Stop()
	red := color.New(color.FgRed)
	red.Fprintf(os.Stderr, "  ✗ %s\n", msg)
}
