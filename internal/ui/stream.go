// Package ui — stream.go provides a helper to render streaming AI tokens
// to the terminal with a leading prefix and proper formatting.
package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/arin/xx-cli/internal/ai"
)

// RenderStream reads tokens from a StreamDelta channel and writes them
// to w in real-time. It prepends prefix to the first token (e.g. "  ")
// for indentation. Returns the full concatenated text and any error.
func RenderStream(w io.Writer, ch <-chan ai.StreamDelta, prefix string) (string, error) {
	var full strings.Builder
	first := true

	for delta := range ch {
		if delta.Err != nil {
			return full.String(), delta.Err
		}
		if delta.Done {
			break
		}
		if delta.Token == "" {
			continue
		}

		if first {
			fmt.Fprint(w, prefix)
			first = false
		}

		fmt.Fprint(w, delta.Token)
		full.WriteString(delta.Token)
	}

	// Ensure we end with a newline.
	if full.Len() > 0 && !strings.HasSuffix(full.String(), "\n") {
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)

	return strings.TrimSpace(full.String()), nil
}
