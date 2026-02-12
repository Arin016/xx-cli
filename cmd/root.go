package cmd

import (
	"github.com/spf13/cobra"
)

var (
	dryRun  bool
	yolo    bool
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "xx [natural language command]",
	Short: "A natural language CLI assistant",
	Long: `xx translates plain English into shell commands using AI.

Examples:
  xx kill the Slack app
  xx show me disk usage
  xx find all .log files larger than 100mb
  xx take me to my downloads folder
  xx --yolo compress this folder

Note: Avoid special shell characters like ? or * in your prompt.
      Use quotes if needed: xx "is slack running?"`,
	RunE:                       run,
	SilenceUsage:               true,
	SilenceErrors:              true,
	DisableFlagParsing:         false,
	TraverseChildren:           true,
	SuggestionsMinimumDistance: 1,
}

func init() {
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show the generated command without executing it")
	rootCmd.Flags().BoolVar(&yolo, "yolo", false, "Execute without confirmation prompt")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show the generated command for all intents")

	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(explainCmd)
	rootCmd.AddCommand(chatCmd)
}

// Execute is the entry point called from main.
func Execute() error {
	return rootCmd.Execute()
}
