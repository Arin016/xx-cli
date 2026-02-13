package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/arin/xx-cli/internal/stats"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show usage statistics and performance metrics",
	Long: `Display a dashboard of your xx usage: command counts, success rates,
AI response times, most-used commands, and intent breakdown.

Data is collected automatically and stored locally in ~/.xx-cli/stats.json.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		summary, err := stats.Summarize()
		if err != nil {
			return fmt.Errorf("failed to load stats: %w", err)
		}

		cyan := color.New(color.FgCyan, color.Bold)
		green := color.New(color.FgGreen)
		yellow := color.New(color.FgYellow)
		dim := color.New(color.FgHiBlack)

		cyan.Fprintf(os.Stderr, "\n  📊 xx stats\n\n")

		if summary.TotalCommands == 0 {
			dim.Fprintln(os.Stderr, "  No data yet. Use xx for a while and come back.")
			fmt.Fprintln(os.Stderr)
			return nil
		}

		// Overview
		green.Fprintf(os.Stderr, "  Commands:  ")
		fmt.Fprintf(os.Stderr, "%d total", summary.TotalCommands)
		dim.Fprintf(os.Stderr, "  (%d today, %d this week)\n", summary.TodayCount, summary.ThisWeekCount)

		green.Fprintf(os.Stderr, "  Success:   ")
		if summary.SuccessRate >= 90 {
			fmt.Fprintf(os.Stderr, "%.0f%%\n", summary.SuccessRate)
		} else {
			yellow.Fprintf(os.Stderr, "%.0f%%\n", summary.SuccessRate)
		}

		// Latency
		green.Fprintf(os.Stderr, "  AI time:   ")
		fmt.Fprintf(os.Stderr, "%dms avg\n", summary.AvgAILatencyMs)
		if summary.AvgExecLatencyMs > 0 {
			green.Fprintf(os.Stderr, "  Exec time: ")
			fmt.Fprintf(os.Stderr, "%dms avg\n", summary.AvgExecLatencyMs)
		}

		// Intent breakdown
		if len(summary.IntentBreakdown) > 0 {
			fmt.Fprintln(os.Stderr)
			cyan.Fprintln(os.Stderr, "  Intent Breakdown")
			for intent, count := range summary.IntentBreakdown {
				pct := float64(count) / float64(summary.TotalCommands) * 100
				bar := strings.Repeat("█", int(pct/5))
				dim.Fprintf(os.Stderr, "  %-10s ", intent)
				fmt.Fprintf(os.Stderr, "%s %d (%.0f%%)\n", bar, count, pct)
			}
		}

		// Subcommand breakdown
		if len(summary.SubcmdBreakdown) > 0 {
			fmt.Fprintln(os.Stderr)
			cyan.Fprintln(os.Stderr, "  Subcommands")
			for sub, count := range summary.SubcmdBreakdown {
				dim.Fprintf(os.Stderr, "  %-14s ", sub)
				fmt.Fprintf(os.Stderr, "%d\n", count)
			}
		}

		// Top commands
		if len(summary.TopCommands) > 0 {
			fmt.Fprintln(os.Stderr)
			cyan.Fprintln(os.Stderr, "  Top Commands")
			for i, tc := range summary.TopCommands {
				cmd := tc.Command
				if len(cmd) > 50 {
					cmd = cmd[:50] + "..."
				}
				dim.Fprintf(os.Stderr, "  %d. ", i+1)
				fmt.Fprintf(os.Stderr, "%s ", cmd)
				dim.Fprintf(os.Stderr, "(%dx)\n", tc.Count)
			}
		}

		fmt.Fprintln(os.Stderr)
		return nil
	},
}
