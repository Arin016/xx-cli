package main

import (
	"fmt"
	"os"

	"github.com/arin/xx-cli/cmd"
)

// version is set at build time via -ldflags.
var version = "dev"

func main() {
	cmd.SetVersion(version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
