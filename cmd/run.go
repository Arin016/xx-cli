package cmd

import (
	"fmt"
	"io"
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

	// Check if there's piped input from stdin.
	stdinData := readStdin()
	if stdinData != "" {
		// Piped input → analyze mode, not command translation.
		sp := ui.NewSpinner("Analyzing...")
		sp.Start()
		answer, err := client.Analyze(cmd.Context(), prompt, stdinData)
		sp.Stop()
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}
		green := color.New(color.FgGreen)
		green.Printf("\n  %s\n\n", answer)
		return nil
	}

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
	if result.Intent == ai.IntentWorkflow && len(result.Steps) > 0 {
		// Show the full workflow plan.
		yellow := color.New(color.FgYellow, color.Bold)
		yellow.Fprintf(os.Stderr, "\n  📋 Workflow (%d steps):\n\n", len(result.Steps))
		for i, step := range result.Steps {
			cyan.Fprintf(os.Stderr, "  %d. %s\n", i+1, step.Command)
			if step.Explanation != "" {
				dim.Fprintf(os.Stderr, "     %s\n", step.Explanation)
			}
		}
		fmt.Fprintln(os.Stderr)
	} else if showCommand {
		cyan.Fprintf(os.Stderr, "\n  → %s\n", result.Command)
		if result.Explanation != "" {
			dim.Fprintf(os.Stderr, "  %s\n", result.Explanation)
		}
		fmt.Fprintln(os.Stderr)
	}

	if dryRun {
		return nil
	}

	// Workflow intent — multi-step pipeline.
	if result.Intent == ai.IntentWorkflow && len(result.Steps) > 0 {
		return runWorkflow(cmd, client, result, prompt)
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
			// Smart retry: ask AI to diagnose and suggest a fix.
			retryCmd, retryErr := smartRetry(cmd, client, prompt, result.Command, output)
			if retryErr == nil && retryCmd != "" {
				cyan.Fprintf(os.Stderr, "\n  🔧 Suggested fix:\n")
				cyan.Fprintf(os.Stderr, "  → %s\n\n", retryCmd)
				if promptRetry() {
					sp4 := ui.NewSpinner("Retrying...")
					sp4.Start()
					retryOutput, retryExecErr := executor.Run(retryCmd)
					sp4.Stop()
					_ = history.Save(history.Entry{
						Prompt:  prompt + " (retry)",
						Command: retryCmd,
						Output:  retryOutput,
						Success: retryExecErr == nil,
					})
					if retryExecErr == nil {
						green := color.New(color.FgGreen)
						green.Fprintf(os.Stderr, "\n  ✓ Done.\n\n")
					} else {
						red.Fprintf(os.Stderr, "\n  ✗ Retry also failed: %v\n\n", retryExecErr)
					}
				}
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

// readStdin reads piped input if available.
func readStdin() string {
	info, err := os.Stdin.Stat()
	if err != nil {
		return ""
	}
	// Check if data is being piped in (not a terminal).
	if (info.Mode() & os.ModeCharDevice) != 0 {
		return ""
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	// Limit to 4000 chars to keep prompt reasonable.
	if len(s) > 4000 {
		s = s[:4000] + "\n... (truncated)"
	}
	return s
}

// smartRetry asks the AI to diagnose a failed command and suggest a fix.
func smartRetry(cmd *cobra.Command, client *ai.Client, prompt, failedCmd, errorOutput string) (string, error) {
	sp := ui.NewSpinner("Diagnosing...")
	sp.Start()
	fix, err := client.SmartRetry(cmd.Context(), prompt, failedCmd, errorOutput)
	sp.Stop()
	return fix, err
}

func promptRetry() bool {
	yellow := color.New(color.FgYellow)
	yellow.Fprint(os.Stderr, "  Retry? [y/N] ")
	var response string
	fmt.Scanln(&response)
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// runWorkflow executes a multi-step pipeline, confirming once then running each step sequentially.
func runWorkflow(cmd *cobra.Command, client *ai.Client, result *ai.Result, prompt string) error {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	cyan := color.New(color.FgCyan, color.Bold)
	dim := color.New(color.FgHiBlack)

	if !yolo {
		yellow := color.New(color.FgYellow)
		yellow.Fprint(os.Stderr, "  Run all? [y/N] ")
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Fprintln(os.Stderr, "Aborted.")
			return nil
		}
		fmt.Fprintln(os.Stderr)
	}

	var allOutput strings.Builder
	for i, step := range result.Steps {
		label := fmt.Sprintf("Step %d/%d", i+1, len(result.Steps))
		sp := ui.NewSpinner(label + ": " + step.Command)
		sp.Start()
		output, err := executor.Run(step.Command)
		sp.Stop()

		_ = history.Save(history.Entry{
			Prompt:  prompt,
			Command: step.Command,
			Output:  output,
			Success: err == nil,
		})

		if err != nil {
			red.Fprintf(os.Stderr, "  ✗ Step %d: %s\n", i+1, step.Command)
			dim.Fprintf(os.Stderr, "    %v\n", err)
			if output != "" {
				dim.Fprintf(os.Stderr, "    %s\n", strings.TrimSpace(output))
			}
			fmt.Fprintln(os.Stderr)
			red.Fprintf(os.Stderr, "  Workflow stopped at step %d.\n\n", i+1)
			return nil
		}

		cyan.Fprintf(os.Stderr, "  ✓ Step %d: ", i+1)
		green.Fprintf(os.Stderr, "%s\n", step.Command)
		allOutput.WriteString(output)
	}

	fmt.Fprintln(os.Stderr)
	green.Fprintf(os.Stderr, "  ✓ All %d steps completed.\n\n", len(result.Steps))
	return nil
}
