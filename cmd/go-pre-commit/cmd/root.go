package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// CLIApp holds the application state and configuration
type CLIApp struct {
	version   string
	commit    string
	buildDate string
	config    *AppConfig
}

// AppConfig holds global application configuration
type AppConfig struct {
	Verbose   bool
	NoColor   bool
	ColorMode string // "auto", "always", "never"
}

// NewCLIApp creates a new CLI application instance
func NewCLIApp(version, commit, buildDate string) *CLIApp {
	return &CLIApp{
		version:   version,
		commit:    commit,
		buildDate: buildDate,
		config:    &AppConfig{},
	}
}

// CommandBuilder creates cobra commands with dependency injection
type CommandBuilder struct {
	app *CLIApp
}

// NewCommandBuilder creates a new command builder
func NewCommandBuilder(app *CLIApp) *CommandBuilder {
	return &CommandBuilder{app: app}
}

// BuildRootCmd creates the root command
func (cb *CommandBuilder) BuildRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "go-pre-commit",
		Short: "Go Pre-commit System - Fast, Go-native git pre-commit checks",
		Long: `Go Pre-commit System is a high-performance, Go-native replacement
for traditional pre-commit frameworks. It provides fast, parallel execution
of code quality checks with zero Python dependencies.

Key features:
  - Lightning fast parallel execution
  - Zero runtime dependencies (single binary)
  - Environment-based configuration via .github/env/ (modular) or .github/.env.base (legacy)
  - Seamless CI/CD integration
  - Direct tool execution without build system dependencies`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			// Get flags and set in app config
			cb.app.config.Verbose, _ = cmd.Flags().GetBool("verbose")
			cb.app.config.NoColor, _ = cmd.Flags().GetBool("no-color")
			cb.app.config.ColorMode, _ = cmd.Flags().GetString("color")
			cb.initConfig()
		},
	}

	// Set version information
	cmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", cb.app.version, cb.app.commit, cb.app.buildDate)
	cmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
`)

	// Add persistent flags
	cmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
	cmd.PersistentFlags().Bool("no-color", false, "Disable colored output (same as --color=never)")
	cmd.PersistentFlags().String("color", "auto", "Control color output: auto, always, never")

	return cmd
}

// Execute runs the root command using the provided CLI app
func (cb *CommandBuilder) Execute() error {
	rootCmd := cb.BuildRootCmd()

	// Add subcommands
	rootCmd.AddCommand(cb.BuildInstallCmd())
	rootCmd.AddCommand(cb.BuildRunCmd())
	rootCmd.AddCommand(cb.BuildUninstallCmd())
	rootCmd.AddCommand(cb.BuildStatusCmd())
	rootCmd.AddCommand(cb.BuildUpgradeCmd())
	rootCmd.AddCommand(cb.BuildPluginCmd())

	return rootCmd.Execute()
}

// Execute runs the default CLI application (legacy compatibility function)
func Execute() error {
	// This is a temporary shim - main.go will be updated to use the new pattern
	app := NewCLIApp("dev", "unknown", "unknown")
	builder := NewCommandBuilder(app)
	return builder.Execute()
}

// SetVersionInfo is kept for backward compatibility with tests
func SetVersionInfo(_, _, _ string) {
	// This is a no-op function for test compatibility
	// The new architecture handles version info through dependency injection
	// Tests that need version info should use the new CLIApp directly
}

// ResetCommand resets the command for testing - will be refactored with new test architecture
func ResetCommand() {
	// This function will be updated when we refactor the test architecture
	// For now, it's a no-op since we use dependency injection
}

// initConfig initializes configuration using the app config
func (cb *CommandBuilder) initConfig() {
	// Handle color configuration with priority:
	// 1. --no-color flag (highest priority)
	// 2. --color flag
	// 3. Environment variables and auto-detection (handled in formatter)
	if cb.app.config.NoColor {
		color.NoColor = true
	} else {
		// Let the formatter handle the smart detection
		// We'll update the formatter creation in run.go to use the color mode
		switch cb.app.config.ColorMode {
		case "never":
			color.NoColor = true
		case "always":
			color.NoColor = false
		case "auto":
			// Check NO_COLOR environment variable for auto mode
			if os.Getenv("NO_COLOR") != "" {
				color.NoColor = true
			} else {
				color.NoColor = false
			}
		default:
			// Default to auto mode - check NO_COLOR env var
			if os.Getenv("NO_COLOR") != "" {
				color.NoColor = true
			} else {
				color.NoColor = false
			}
		}
	}

	// Set up paths relative to repository root
	// The binary should be run from the repository root or have access to .github/
	if !hasGitHubConfig(".") {
		// Try to find the repository root
		cwd, _ := os.Getwd()
		for cwd != "/" && cwd != "" {
			if hasGitHubConfig(cwd) {
				_ = os.Chdir(cwd)
				break
			}
			cwd = filepath.Dir(cwd)
		}
	}
}

// hasGitHubConfig checks if the given directory contains GitHub config files
// (modular .github/env/ directory or legacy .github/.env.base)
func hasGitHubConfig(dir string) bool {
	// Check modular config first (preferred)
	if info, err := os.Stat(filepath.Join(dir, ".github", "env")); err == nil && info.IsDir() {
		return true
	}
	// Fall back to legacy config
	if _, err := os.Stat(filepath.Join(dir, ".github", ".env.base")); err == nil {
		return true
	}
	return false
}

// Helper functions for consistent output - these will be updated to use app config
// For now keeping them as legacy functions for backward compatibility
func printSuccess(format string, args ...interface{}) {
	if !color.NoColor {
		color.Green("✓ " + fmt.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "✓ %s\n", fmt.Sprintf(format, args...))
	}
}

func printError(format string, args ...interface{}) {
	if !color.NoColor {
		color.Red("✗ " + fmt.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "✗ %s\n", fmt.Sprintf(format, args...))
	}
}

func printInfo(format string, args ...interface{}) {
	if !color.NoColor {
		color.Blue("ℹ " + fmt.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "ℹ %s\n", fmt.Sprintf(format, args...)) // #nosec G705 - output goes to stdout, not a web response
	}
}

func printWarning(format string, args ...interface{}) {
	if !color.NoColor {
		color.Yellow("⚠ " + fmt.Sprintf(format, args...))
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "⚠ %s\n", fmt.Sprintf(format, args...))
	}
}
