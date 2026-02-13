package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/arin/xx-cli/internal/ai"
	"github.com/arin/xx-cli/internal/config"
	"github.com/arin/xx-cli/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var diffStaged bool

var diffExplainCmd = &cobra.Command{
	Use:   "diff-explain",
	Short: "Explain your current git diff in plain English",
	Long: `Reads your current git diff and generates a human-readable summary.
Great for writing PR descriptions or commit messages.

Examples:
  xx diff-explain            # Explain unstaged changes
  xx diff-explain --staged   # Explain staged changes`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		// Get the git diff.
		gitArgs := []string{"diff", "--no-color"}
		if diffStaged {
			gitArgs = append(gitArgs, "--staged")
		}
		gitCmd := exec.Command("git", gitArgs...)
		out, err := gitCmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get git diff — are you in a git repo?")
		}

		diff := strings.TrimSpace(string(out))
		if diff == "" {
			label := "unstaged"
			if diffStaged {
				label = "staged"
			}
			fmt.Printf("  No %s changes found.\n", label)
			return nil
		}

		// Cap diff size for the AI prompt.
		if len(diff) > 6000 {
			diff = diff[:6000] + "\n... (truncated)"
		}

		client := ai.NewClient(cfg)
		sp := ui.NewSpinner("Analyzing diff...")
		sp.Start()
		explanation, err := client.DiffExplain(cmd.Context(), diff)
		sp.Stop()

		if err != nil {
			return fmt.Errorf("diff explanation failed: %w", err)
		}

		cyan := color.New(color.FgCyan, color.Bold)
		cyan.Fprintf(os.Stderr, "\n  📝 Diff Summary\n\n")
		fmt.Printf("  %s\n\n", explanation)
		return nil
	},
}

func init() {
	diffExplainCmd.Flags().BoolVar(&diffStaged, "staged", false, "Explain staged changes instead of unstaged")
}
