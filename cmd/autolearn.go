package cmd

import (
	"github.com/arin/xx-cli/internal/rag"
	"github.com/spf13/cobra"
)

// autoLearnCmd is a hidden subcommand that runs auto-learning in a detached
// subprocess. When a command succeeds, run.go spawns `xx _learn <prompt> <command> <category>`
// as a background process. This process embeds the prompt+command, checks for
// near-duplicates, and appends to the vector store — then exits.
//
// The parent process never waits for this. If it fails, nobody notices.
// This is the same pattern as `git maintenance run --detach`.
var autoLearnCmd = &cobra.Command{
	Use:    "_learn",
	Hidden: true, // Not user-facing — internal plumbing only.
	Args:   cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := args[0]
		command := args[1]
		category := args[2]
		rag.LearnFromSuccess(cmd.Context(), prompt, command, category)
		return nil
	},
}
