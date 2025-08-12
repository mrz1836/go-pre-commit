// Package main provides the entry point for the Go pre-commit system
package main

import (
	"fmt"
	"os"

	"github.com/mrz1836/go-pre-commit/cmd/go-pre-commit/cmd"
)

// BuildInfo holds build-time information that gets injected via ldflags
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

// getBuildInfo returns build information from version constants
func getBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}
}

func main() {
	os.Exit(run())
}

// run executes the main application logic and returns the exit code.
// This function is separated from main() to enable testing.
func run() int {
	// Get build information
	buildInfo := getBuildInfo()

	// Create CLI application with dependency injection
	app := cmd.NewCLIApp(buildInfo.Version, buildInfo.Commit, buildInfo.BuildDate)
	builder := cmd.NewCommandBuilder(app)

	// Execute the root command
	if err := builder.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
