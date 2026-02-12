package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/arin/xx-cli/internal/ai"
	"github.com/arin/xx-cli/internal/config"
	"github.com/arin/xx-cli/internal/executor"
	"github.com/arin/xx-cli/internal/history"
	"github.com/arin/xx-cli/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please provide a natural language command\n\nUsage: xx <your request>\nExample: xx kill the Slack app")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	prompt := strings.Join(args, " ")
	client := ai.NewClient(cfg)

	sp := ui.NewSpinner("Thinking...")
	sp.Start()

	result, err := client.Translate(cmd.Context(), prompt)
	sp.Stop()

	if err != nil {
		return fmt.Errorf("AI translation failed: %w", err)
	}

	// Show command only for execute intent, dry-run, or verbose mode.
	cyan := color.New(color.FgCyan, color.Bold)
	dim := color.New(color.FgHiBlack)

	showCommand := verbose || dryRun || result.Intent == ai.IntentExecute
	if showCommand {
		cyan.Fprintf(os.Stderr, "\n  → %s\n", result.Command)
		if result.Explanation != "" {
			dim.Fprintf(os.Stderr, "  %s\n", result.Explanation)
		}
		fmt.Fprintln(os.Stderr)
	}

	if dryRun {
		return nil
	}

	// Only confirm on execute (state-changing) commands.
	if !yolo && result.Intent == ai.IntentExecute {
		if !promptConfirmation() {
			fmt.Fprintln(os.Stderr, "Aborted.")
			return nil
		}
	}

	sp2 := ui.NewSpinner("Running...")
	sp2.Start()
	output, execErr := executor.Run(result.Command)
	sp2.Stop()
	success := execErr == nil

	_ = history.Save(history.Entry{
		Prompt:  prompt,
		Command: result.Command,
		Output:  output,
		Success: success,
	})

	switch result.Intent {
	case ai.IntentQuery:
		sp3 := ui.NewSpinner("Summarizing...")
		sp3.Start()
		summary, sErr := client.Summarize(cmd.Context(), prompt, result.Command, output, success)
		sp3.Stop()
		if sErr != nil {
			fmt.Print(output)
		} else {
			green := color.New(color.FgGreen)
			green.Printf("\n  %s\n\n", summary)
		}

	case ai.IntentExecute:
		if success {
			green := color.New(color.FgGreen)
			green.Fprintf(os.Stderr, "\n  ✓ Done.\n\n")
		} else {
			red := color.New(color.FgRed)
			red.Fprintf(os.Stderr, "\n  ✗ Failed: %v\n\n", execErr)
			if output != "" {
				dim.Fprintf(os.Stderr, "  %s\n", output)
			}
		}

	default:
		if output != "" {
			fmt.Print(output)
		}
	}

	if execErr != nil && result.Intent != ai.IntentExecute {
		return fmt.Errorf("command failed: %w", execErr)
	}

	return nil
}

func promptConfirmation() bool {
	yellow := color.New(color.FgYellow)
	yellow.Fprint(os.Stderr, "Execute? [y/N] ")

	var response string
	fmt.Scanln(&response)
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
