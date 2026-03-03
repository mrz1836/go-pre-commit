// Package main provides the entry point for the Go pre-commit system
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mrz1836/go-pre-commit/cmd/go-pre-commit/cmd"
	"github.com/mrz1836/go-pre-commit/internal/update"
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

	// Start background update check (non-blocking)
	// This runs early in the CLI lifecycle to maximize time for the network request
	ctx := context.Background()
	updateChan := update.StartBackgroundCheck(ctx, version)

	// Create CLI application with dependency injection
	app := cmd.NewCLIApp(version, buildInfo.Commit(), buildInfo.BuildDate())
	app.SetUpdateChan(updateChan)
	builder := cmd.NewCommandBuilder(app)

	// Execute the root command
	if err := builder.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
