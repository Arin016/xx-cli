package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/arin/xx-cli/internal/ai"
	"github.com/arin/xx-cli/internal/config"
	"github.com/arin/xx-cli/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var wtfCmd = &cobra.Command{
	Use:   "wtf [error message]",
	Short: "Diagnose an error message and suggest a fix",
	Long: `Paste an error message and get an instant diagnosis with a fix.

Examples:
  xx wtf "EACCES: permission denied"
  xx wtf "fatal: not a git repository"
  xx wtf "command not found: node"
  some-command 2>&1 | xx wtf`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		var errorMsg string
		if len(args) > 0 {
			errorMsg = strings.Join(args, " ")
		} else {
			info, _ := os.Stdin.Stat()
			if (info.Mode() & os.ModeCharDevice) == 0 {
				data, _ := io.ReadAll(os.Stdin)
				errorMsg = strings.TrimSpace(string(data))
			}
		}

		if errorMsg == "" {
			return fmt.Errorf("please provide an error message\n\nUsage: xx wtf \"error message\"\n   or: some-command 2>&1 | xx wtf")
		}

		if len(errorMsg) > 4000 {
			errorMsg = errorMsg[:4000] + "\n... (truncated)"
		}

		client := ai.NewClient(cfg)

		red := color.New(color.FgRed, color.Bold)
		red.Fprintf(os.Stderr, "\n  🔍 Diagnosis\n\n")

		stream := client.DiagnoseStream(cmd.Context(), errorMsg)
		_, err = ui.RenderStream(os.Stdout, stream, "  ")
		if err != nil {
			return fmt.Errorf("diagnosis failed: %w", err)
		}

		return nil
	},
}
