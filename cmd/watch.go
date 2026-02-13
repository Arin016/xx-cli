package cmd

import (
	"fmt"
	"os"
	"os/signal"
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

		var lastOutput string
		tick := time.NewTicker(time.Duration(watchInterval) * time.Second)
		defer tick.Stop()

		// Run immediately, then on each tick.
		runWatch := func() {
			output, _ := executor.Run(result.Command)
			output = strings.TrimSpace(output)
			now := time.Now().Format("15:04:05")

			if lastOutput == "" {
				// First run.
				dim.Fprintf(os.Stderr, "  [%s] ", now)
				green.Fprintf(os.Stderr, "Initial: ")
				fmt.Fprintf(os.Stderr, "%s\n", summarizeOutput(output))
			} else if output != lastOutput {
				// Status changed.
				yellow.Fprintf(os.Stderr, "  [%s] ⚠ CHANGED: ", now)
				fmt.Fprintf(os.Stderr, "%s\n", summarizeOutput(output))
				fmt.Fprint(os.Stderr, "\a") // Terminal bell.
			} else {
				dim.Fprintf(os.Stderr, "  [%s] No change\n", now)
			}
			lastOutput = output
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

func init() {
	watchCmd.Flags().IntVar(&watchInterval, "interval", 10, "Polling interval in seconds")
}
