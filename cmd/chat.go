package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/arin/xx-cli/internal/ai"
	"github.com/arin/xx-cli/internal/config"
	"github.com/arin/xx-cli/internal/ui"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long: `Start a conversational session with xx. Ask questions,
get guided through tasks, and have context carry over between messages.

Type 'exit' or 'quit' to end the session.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}

		client := ai.NewClient(cfg)
		cyan := color.New(color.FgCyan, color.Bold)
		dim := color.New(color.FgHiBlack)
		green := color.New(color.FgGreen)

		fmt.Fprintln(os.Stderr)
		cyan.Fprintln(os.Stderr, "  xx chat")
		dim.Fprintln(os.Stderr, "  Your friendly terminal buddy. Ask me anything.")
		dim.Fprintf(os.Stderr, "  Type 'exit' to quit.\n\n")

		scanner := bufio.NewScanner(os.Stdin)
		var history []ai.ChatMessage

		for {
			green.Fprint(os.Stderr, "  you → ")
			if !scanner.Scan() {
				break
			}

			input := strings.TrimSpace(scanner.Text())
			if input == "" {
				continue
			}
			if input == "exit" || input == "quit" || input == "bye" {
				dim.Fprintf(os.Stderr, "\n  Later! 👋\n\n")
				break
			}

			history = append(history, ai.ChatMessage{Role: "user", Content: input})

			sp := ui.NewSpinner("Thinking...")
			sp.Start()
			reply, err := client.Chat(cmd.Context(), history)
			sp.Stop()

			if err != nil {
				fmt.Fprintf(os.Stderr, "  Error: %v\n\n", err)
				continue
			}

			history = append(history, ai.ChatMessage{Role: "assistant", Content: reply})

			cyan.Fprintf(os.Stderr, "  xx → ")
			fmt.Fprintf(os.Stderr, "%s\n\n", reply)
		}

		return nil
	},
}
