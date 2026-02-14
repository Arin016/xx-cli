package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/arin/xx-cli/internal/rag"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Build the RAG knowledge index for smarter command suggestions",
	Long: `Builds a local vector store by embedding OS command knowledge, your learned
corrections, and command history. This enables semantic search at query time
so xx picks the right command more often.

Run this once after install, and again whenever you want to refresh the index
(e.g. after teaching xx new corrections with 'xx learn').

Requires the nomic-embed-text model:
  ollama pull nomic-embed-text`,
	RunE: func(cmd *cobra.Command, args []string) error {
		start := time.Now()
		cyan := color.New(color.FgCyan)
		green := color.New(color.FgGreen)

		cyan.Println("🔍 Building knowledge index...")
		fmt.Println()

		embedder := rag.NewEmbedClient()
		indexer := rag.NewIndexer(embedder)

		err := indexer.IndexAll(context.Background(), func(msg string) {
			fmt.Println("  " + msg)
		})
		if err != nil {
			return fmt.Errorf("indexing failed: %w", err)
		}

		fmt.Println()
		green.Printf("Done in %s\n", time.Since(start).Round(time.Millisecond))
		return nil
	},
}
