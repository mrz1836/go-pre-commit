// Package main provides the entry point for the Go pre-commit system
package main

import (
	"fmt"
	"os"

	"github.com/mrz1836/go-pre-commit/cmd/go-pre-commit/cmd"
)

// version information - set by ldflags during build
var (
	version   = "dev"
	commit    = "none"    //nolint:gochecknoglobals // Build var
	buildDate = "unknown" //nolint:gochecknoglobals // Build var
)

func main() {
	os.Exit(run())
}

// run executes the main application logic and returns the exit code.
// This function is separated from main() to enable testing.
func run() int {
	// Set version information for the root command
	cmd.SetVersionInfo(version, commit, buildDate)

	// Execute the root command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
