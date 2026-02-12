package cmd

import (
	"fmt"

	"github.com/arin/xx-cli/internal/history"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var historyLimit int

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show command history",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := history.Load(historyLimit)
		if err != nil {
			return fmt.Errorf("failed to load history: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No history yet.")
			return nil
		}

		cyan := color.New(color.FgCyan)
		dim := color.New(color.FgHiBlack)
		red := color.New(color.FgRed)
		green := color.New(color.FgGreen)

		for i, e := range entries {
			dim.Printf("[%s] ", e.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("%s ", e.Prompt)
			cyan.Printf("→ %s ", e.Command)
			if e.Success {
				green.Println("✓")
			} else {
				red.Println("✗")
			}
			if i < len(entries)-1 {
				fmt.Println()
			}
		}
		return nil
	},
}

func init() {
	historyCmd.Flags().IntVarP(&historyLimit, "limit", "n", 20, "Number of history entries to show")
}
