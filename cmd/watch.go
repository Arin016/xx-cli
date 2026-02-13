package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/arin/xx-cli/internal/ai"
	"github.com/arin/xx-cli/internal/config"
	"github.com/arin/xx-cli/internal/executor"
	"github.com/arin/xx-cli/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var watchInterval int

var watchCmd = &cobra.Command{
	Use:   "watch [query]",
	Short: "Monitor something and alert when it changes",
	Long: `Poll a query every N seconds and alert when the status changes.

Examples:
  xx watch is my server still running
  xx watch is port 3000 in use
  xx watch --interval 5 is docker running`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		prompt := strings.Join(args, " ")
		client := ai.NewClient(cfg)
		cyan := color.New(color.FgCyan, color.Bold)
		dim := color.New(color.FgHiBlack)
		yellow := color.New(color.FgYellow, color.Bold)
		green := color.New(color.FgGreen)

		// First, translate the prompt to a command.
		sp := ui.NewSpinner("Setting up watch...")
		sp.Start()
		result, err := client.Translate(cmd.Context(), prompt)
		sp.Stop()
		if err != nil {
			return fmt.Errorf("failed to translate: %w", err)
		}

		cyan.Fprintf(os.Stderr, "\n  👁 Watching: %s\n", prompt)
		dim.Fprintf(os.Stderr, "  Command: %s\n", result.Command)
		dim.Fprintf(os.Stderr, "  Interval: %ds (Ctrl+C to stop)\n\n", watchInterval)

		// Catch Ctrl+C for clean exit.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		var lastStable string // normalized output for change detection
		var lastRaw string    // raw output for display
		tick := time.NewTicker(time.Duration(watchInterval) * time.Second)
		defer tick.Stop()

		// Run immediately, then on each tick.
		runWatch := func() {
			raw, _ := executor.Run(result.Command)
			output := filterWatchNoise(raw, result.Command)
			stable := normalizeForComparison(output)
			now := time.Now().Format("15:04:05")

			if lastStable == "" {
				// First run.
				dim.Fprintf(os.Stderr, "  [%s] ", now)
				green.Fprintf(os.Stderr, "Initial: ")
				fmt.Fprintf(os.Stderr, "%s\n", summarizeOutput(output))
			} else if stable != lastStable {
				// Status changed (meaningful difference, not just PID/CPU jitter).
				yellow.Fprintf(os.Stderr, "  [%s] ⚠ CHANGED: ", now)
				fmt.Fprintf(os.Stderr, "%s\n", summarizeOutput(output))
				fmt.Fprint(os.Stderr, "\a") // Terminal bell.
			} else {
				dim.Fprintf(os.Stderr, "  [%s] No change\n", now)
			}
			lastStable = stable
			lastRaw = output
			_ = lastRaw // available for future verbose mode
		}

		runWatch()
		for {
			select {
			case <-tick.C:
				runWatch()
			case <-sigCh:
				fmt.Fprintf(os.Stderr, "\n  Stopped watching.\n\n")
				return nil
			}
		}
	},
}

// summarizeOutput returns the first line or a truncated version of the output.
func summarizeOutput(output string) string {
	if output == "" {
		return "(no output)"
	}
	lines := strings.Split(output, "\n")
	first := strings.TrimSpace(lines[0])
	if len(first) > 120 {
		first = first[:120] + "..."
	}
	if len(lines) > 1 {
		first += fmt.Sprintf(" (+%d lines)", len(lines)-1)
	}
	return first
}
// filterWatchNoise removes lines from command output that reference the watch
// process itself or transient grep subshells. Without this, commands like
// "ps aux | grep X" trigger false CHANGED alerts every tick because the PID
// and CPU stats of the xx-watch process fluctuate.
func filterWatchNoise(raw, command string) string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		lower := strings.ToLower(line)
		// Skip lines referencing the watch process or grep itself.
		if strings.Contains(lower, "xx watch") ||
			strings.Contains(lower, "xx-cli watch") {
			continue
		}
		// Classic "grep -v grep" pattern: skip the grep subprocess line.
		if strings.Contains(lower, "grep") && strings.Contains(lower, command) {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}
// normalizeForComparison strips volatile numeric fields (PIDs, CPU%, memory,
// timestamps) from command output so that only meaningful content changes
// trigger alerts. Without this, commands like "ps aux | grep X" report false
// changes every tick because stats like CPU% and RSS fluctuate constantly.
var numericFieldRe = regexp.MustCompile(`\b\d[\d.:]*\b`)

func normalizeForComparison(output string) string {
	// Replace all numeric tokens with a placeholder.
	normalized := numericFieldRe.ReplaceAllString(output, "N")
	// Collapse whitespace so column alignment shifts don't matter.
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	return strings.TrimSpace(normalized)
}

func init() {
	watchCmd.Flags().IntVar(&watchInterval, "interval", 10, "Polling interval in seconds")
}
