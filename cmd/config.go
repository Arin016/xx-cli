package cmd

import (
	"fmt"

	"github.com/arin/xx-cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage xx-cli configuration",
}

var setKeyCmd = &cobra.Command{
	Use:   "set-key <api-key>",
	Short: "Set your Groq API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.SetAPIKey(args[0]); err != nil {
			return fmt.Errorf("failed to save API key: %w", err)
		}
		fmt.Println("API key saved successfully.")
		return nil
	},
}

var setModelCmd = &cobra.Command{
	Use:   "set-model <model-name>",
	Short: "Set the Groq model (default: llama-3.3-70b-versatile)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.SetModel(args[0]); err != nil {
			return fmt.Errorf("failed to save model: %w", err)
		}
		fmt.Printf("Model set to %s.\n", args[0])
		return nil
	},
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Printf("Model:      %s\n", cfg.Model)
		fmt.Printf("API Key:    %s...%s\n", cfg.APIKey[:4], cfg.APIKey[len(cfg.APIKey)-4:])
		fmt.Printf("Config Dir: %s\n", config.Dir())
		return nil
	},
}

func init() {
	configCmd.AddCommand(setKeyCmd)
	configCmd.AddCommand(setModelCmd)
	configCmd.AddCommand(showCmd)
}
