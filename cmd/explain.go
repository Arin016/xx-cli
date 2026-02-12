package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/arin/xx-cli/internal/ai"
	"github.com/arin/xx-cli/internal/config"
	"github.com/arin/xx-cli/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var explainCmd = &cobra.Command{
	Use:   "explain <command>",
	Short: "Explain a shell command in plain English",
	Long: `Explain what a shell command does in plain English.

Examples:
  xx explain "tar -xzf archive.tar.gz"
  xx explain "find / -name '*.log' -size +100M"
  xx explain "awk '{print $1}' file.txt"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		command := strings.Join(args, " ")
		client := ai.NewClient(cfg)

		sp := ui.NewSpinner("Thinking...")
		sp.Start()

		explanation, err := client.Explain(cmd.Context(), command)
		sp.Stop()

		if err != nil {
			return fmt.Errorf("explanation failed: %w", err)
		}

		cyan := color.New(color.FgCyan, color.Bold)
		cyan.Fprintf(os.Stderr, "\n  %s\n\n", command)
		fmt.Printf("  %s\n\n", explanation)
		return nil
	},
}

func init() {
	// registered in root.go
}
