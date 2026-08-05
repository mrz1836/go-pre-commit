// Package main provides the entry point for the Go pre-commit system
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mrz1836/go-pre-commit/cmd/go-pre-commit/cmd"
)

func main() {
	os.Exit(run())
}

// run executes the main application logic and returns the exit code.
// This function is separated from main() to enable testing.
func run() int {
	// Create build information using the new pattern
	buildInfo := NewBuildInfo()

	// Get version and add modified suffix if there are uncommitted changes
	version := buildInfo.Version()
	if buildInfo.IsModified() && !strings.HasSuffix(version, "-dirty") {
		version += "-dirty"
	}

	// Create CLI application with dependency injection
	app := cmd.NewCLIApp(version, buildInfo.Commit(), buildInfo.BuildDate())
	builder := cmd.NewCommandBuilder(app)

	// Execute the root command
	if err := builder.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
