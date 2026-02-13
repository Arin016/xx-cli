package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/arin/xx-cli/internal/ai"
	"github.com/arin/xx-cli/internal/config"
	"github.com/arin/xx-cli/internal/history"
	"github.com/arin/xx-cli/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var recapCmd = &cobra.Command{
	Use:   "recap",
	Short: "Summarize what you did today (standup-ready)",
	Long: `Generate a standup-ready summary of your terminal activity.
Reads your command history, groups by project and time, and produces
a concise recap powered by AI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		// Load all history entries from today.
		entries, err := history.Load(0)
		if err != nil {
			return fmt.Errorf("failed to load history: %w", err)
		}

		today := time.Now().Truncate(24 * time.Hour)
		var todayEntries []history.Entry
		for _, e := range entries {
			if e.Timestamp.After(today) {
				todayEntries = append(todayEntries, e)
			}
		}

		if len(todayEntries) == 0 {
			fmt.Println("  No commands recorded today. Go do something first.")
			return nil
		}

		// Build a summary of raw history for the AI.
		var sb strings.Builder
		for _, e := range todayEntries {
			status := "✓"
			if !e.Success {
				status = "✗"
			}
			sb.WriteString(fmt.Sprintf("[%s] %s → %s %s\n",
				e.Timestamp.Format("15:04"),
				e.Prompt, e.Command, status))
		}

		client := ai.NewClient(cfg)

		cyan := color.New(color.FgCyan, color.Bold)
		cyan.Fprintf(cmd.ErrOrStderr(), "\n  📋 Today's Recap\n\n")

		stream := client.RecapStream(cmd.Context(), sb.String(), len(todayEntries))
		_, err = ui.RenderStream(os.Stdout, stream, "  ")
		if err != nil {
			return fmt.Errorf("recap failed: %w", err)
		}

		return nil
	},
}
