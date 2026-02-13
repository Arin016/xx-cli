package cmd

import (
	"fmt"

	"github.com/arin/xx-cli/internal/learn"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var learnCmd = &cobra.Command{
	Use:   "learn <prompt> <correct-command>",
	Short: "Teach xx the correct command for a phrase",
	Long: `Store a correction so xx learns your preferred commands.

Examples:
  xx learn "run tests" "make test"
  xx learn "deploy" "./scripts/deploy.sh"
  xx learn "lint" "golangci-lint run ./..."

View all learned corrections:
  xx learn --list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		listFlag, _ := cmd.Flags().GetBool("list")

		if listFlag {
			corrections, err := learn.LoadAll()
			if err != nil {
				return fmt.Errorf("failed to load corrections: %w", err)
			}
			if len(corrections) == 0 {
				fmt.Println("  No learned corrections yet.")
				return nil
			}
			cyan := color.New(color.FgCyan)
			dim := color.New(color.FgHiBlack)
			fmt.Println()
			for _, c := range corrections {
				cyan.Printf("  \"%s\"", c.Prompt)
				dim.Printf(" → ")
				fmt.Printf("%s\n", c.Command)
			}
			fmt.Println()
			return nil
		}

		if len(args) != 2 {
			return fmt.Errorf("expected 2 arguments: xx learn \"prompt\" \"command\"\n\nExample: xx learn \"run tests\" \"make test\"")
		}

		correction := learn.Correction{
			Prompt:  args[0],
			Command: args[1],
		}
		if err := learn.Save(correction); err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}

		green := color.New(color.FgGreen)
		green.Printf("\n  ✓ Learned: \"%s\" → %s\n\n", args[0], args[1])
		return nil
	},
}

func init() {
	learnCmd.Flags().Bool("list", false, "Show all learned corrections")
}
