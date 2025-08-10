// Package main provides the entry point for the Go pre-commit system
package main

import (
	"fmt"
	"os"

	"github.com/mrz1836/go-pre-commit/cmd/go-pre-commit/cmd"
)

// Build variables - set by ldflags during build
var (
	version   = "dev"
	commit    = "none"    //nolint:gochecknoglobals // Required for ldflags injection at build time
	buildDate = "unknown" //nolint:gochecknoglobals // Required for ldflags injection at build time
)

func main() {
	os.Exit(run())
}

// run executes the main application logic and returns the exit code.
// This function is separated from main() to enable testing.
func run() int {
	// Create CLI application with dependency injection
	app := cmd.NewCLIApp(version, commit, buildDate)
	builder := cmd.NewCommandBuilder(app)

	// Execute the root command
	if err := builder.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
