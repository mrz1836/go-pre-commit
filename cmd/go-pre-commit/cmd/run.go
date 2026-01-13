package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mrz1836/go-pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/git"
	"github.com/mrz1836/go-pre-commit/internal/output"
	"github.com/mrz1836/go-pre-commit/internal/runner"
	"github.com/spf13/cobra"
)

// RunConfig holds configuration for the run command
type RunConfig struct {
	AllFiles            bool
	Files               []string
	SkipChecks          []string
	OnlyChecks          []string
	Parallel            int
	FailFast            bool
	ShowVersion         bool
	GracefulDegradation bool
	ShowProgress        bool
	Quiet               bool
	DebugTimeout        bool
}

// BuildRunCmd creates the run command
func (cb *CommandBuilder) BuildRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [check-name] [flags] [files...]",
		Short: "Run pre-commit checks",
		Long: `Run pre-commit checks on your code.

By default, runs all enabled checks on files staged for commit.
You can specify individual checks to run, or provide specific files to check.

Available checks:
  ai_detection - Detect AI attribution in code and commit messages
  eof          - Ensure files end with newline
  fmt          - Format code with go fmt
  fumpt        - Format code with gofumpt
  gitleaks     - Scan for secrets and credentials in code
  goimports    - Format code and manage imports
  lint         - Run golangci-lint
  mod-tidy     - Ensure go.mod and go.sum are tidy
  whitespace   - Fix trailing whitespace`,
		Example: `  # Run all checks on staged files
  go-pre-commit run

  # Run specific check on staged files
  go-pre-commit run lint

  # Run all checks on all files
  go-pre-commit run --all-files

  # Run checks on specific files
  go-pre-commit run --files main.go,utils.go

  # Skip specific checks
  go-pre-commit run --skip lint,fumpt

  # Run only specific checks
  go-pre-commit run --only whitespace,eof`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flags and create config
			config := RunConfig{}
			var err error

			config.AllFiles, err = cmd.Flags().GetBool("all-files")
			if err != nil {
				return err
			}

			config.Files, err = cmd.Flags().GetStringSlice("files")
			if err != nil {
				return err
			}

			config.SkipChecks, err = cmd.Flags().GetStringSlice("skip")
			if err != nil {
				return err
			}

			config.OnlyChecks, err = cmd.Flags().GetStringSlice("only")
			if err != nil {
				return err
			}

			config.Parallel, err = cmd.Flags().GetInt("parallel")
			if err != nil {
				return err
			}

			config.FailFast, err = cmd.Flags().GetBool("fail-fast")
			if err != nil {
				return err
			}

			config.ShowVersion, err = cmd.Flags().GetBool("show-checks")
			if err != nil {
				return err
			}

			config.GracefulDegradation, err = cmd.Flags().GetBool("graceful")
			if err != nil {
				return err
			}

			config.ShowProgress, err = cmd.Flags().GetBool("progress")
			if err != nil {
				return err
			}

			config.Quiet, err = cmd.Flags().GetBool("quiet")
			if err != nil {
				return err
			}

			config.DebugTimeout, err = cmd.Flags().GetBool("debug-timeout")
			if err != nil {
				return err
			}

			return cb.runChecksWithConfig(config, cmd, args)
		},
	}

	// Add flags
	cmd.Flags().BoolP("all-files", "a", false, "Run on all files in the repository")
	cmd.Flags().StringSliceP("files", "f", nil, "Specific files to check")
	cmd.Flags().StringSlice("skip", nil, "Skip specific checks")
	cmd.Flags().StringSlice("only", nil, "Run only specific checks")
	cmd.Flags().IntP("parallel", "p", 0, "Number of parallel workers (0 = auto)")
	cmd.Flags().Bool("fail-fast", false, "Stop on first check failure")
	cmd.Flags().Bool("show-checks", false, "Show available checks and exit")
	cmd.Flags().Bool("graceful", false, "Skip checks that can't run instead of failing")
	cmd.Flags().Bool("progress", true, "Show progress indicators during execution")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress progress messages, show only errors and results")
	cmd.Flags().Bool("debug-timeout", false, "Enable detailed timeout debugging information")

	return cmd
}

func (cb *CommandBuilder) runChecksWithConfig(runConfig RunConfig, _ *cobra.Command, args []string) error {
	// Load configuration first
	cfg, err := config.Load()
	if err != nil {
		// Use basic formatter for this error since config failed to load
		formatter := output.NewDefault()
		formatter.Error("Failed to load configuration: %v", err)
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create output formatter with config-based color settings
	var formatter *output.Formatter
	if cb.app.config.NoColor {
		// --no-color flag takes highest priority
		formatter = output.NewWithColorMode(output.ColorNever)
	} else {
		// Use --color flag or auto-detect
		switch cb.app.config.ColorMode {
		case "always":
			formatter = output.NewWithColorMode(output.ColorAlways)
		case "never":
			formatter = output.NewWithColorMode(output.ColorNever)
		case "auto":
			formatter = output.NewWithColorMode(output.ColorAuto)
		default:
			// Default to auto mode with config override
			if !cfg.UI.ColorOutput {
				formatter = output.NewWithColorMode(output.ColorNever)
			} else {
				formatter = output.NewWithColorMode(output.ColorAuto)
			}
		}
	}

	// Check if pre-commit system is enabled
	if !cfg.Enabled {
		formatter.Warning("Pre-commit system is disabled in configuration (ENABLE_GO_PRE_COMMIT=false)")
		return nil
	}

	// Get repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		formatter.Error("Failed to find git repository: %v", err)
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	// If show-checks flag is set, display available checks and exit
	if runConfig.ShowVersion {
		return showAvailableChecks(cfg, formatter)
	}

	// Determine which files to check
	var filesToCheck []string
	if len(runConfig.Files) > 0 {
		// Specific files provided
		filesToCheck = runConfig.Files
	} else if runConfig.AllFiles {
		// All files in repository
		repo := git.NewRepository(repoRoot)
		filesToCheck, err = repo.GetAllFiles()
		if err != nil {
			formatter.Error("Failed to get all files: %v", err)
			return fmt.Errorf("failed to get all files: %w", err)
		}
	} else {
		// Staged files (default)
		repo := git.NewRepository(repoRoot)
		filesToCheck, err = repo.GetStagedFiles()
		if err != nil {
			formatter.Error("Failed to get staged files: %v", err)
			return fmt.Errorf("failed to get staged files: %w", err)
		}
	}

	if len(filesToCheck) == 0 {
		formatter.Info("No files to check")
		return nil
	}

	// Create runner
	r := runner.New(cfg, repoRoot)

	// Configure runner options
	opts := runner.Options{
		Files:               filesToCheck,
		Parallel:            runConfig.Parallel,
		FailFast:            runConfig.FailFast,
		GracefulDegradation: runConfig.GracefulDegradation,
		DebugTimeout:        runConfig.DebugTimeout,
	}

	// Set up progress callback if progress is enabled and not in quiet mode
	if runConfig.ShowProgress && !runConfig.Quiet {
		opts.ProgressCallback = func(checkName, status string, duration time.Duration) {
			durationStr := formatter.Duration(duration)
			switch status {
			case "running":
				formatter.Progress("Running %s check...", checkName)
			case "passed":
				formatter.Success("%s check passed (%s)", checkName, durationStr)
			case "failed":
				formatter.Error("%s check failed (%s)", checkName, durationStr)
			case "skipped":
				formatter.Warning("%s check skipped (%s)", checkName, durationStr)
			}
		}
	}

	// Handle check selection
	if len(args) > 0 {
		// Specific check requested as positional argument
		opts.OnlyChecks = []string{args[0]}
	} else if len(runConfig.OnlyChecks) > 0 {
		// --only flag
		opts.OnlyChecks = runConfig.OnlyChecks
	} else if len(runConfig.SkipChecks) > 0 {
		// --skip flag
		opts.SkipChecks = runConfig.SkipChecks
	}

	// Show initial information (unless in quiet mode)
	if cb.app.config.Verbose && !runConfig.Quiet {
		formatter.Info("Running checks on %s", formatter.FormatFileList(filesToCheck, 3))
		if opts.Parallel > 0 {
			formatter.Info("Using %d parallel workers", opts.Parallel)
		}
		if runConfig.GracefulDegradation {
			formatter.Info("Graceful degradation enabled - missing tools will be skipped")
		}
	}

	// Run checks
	results, err := r.Run(context.Background(), opts)
	if err != nil {
		formatter.Error("Failed to run checks: %v", err)
		return fmt.Errorf("failed to run checks: %w", err)
	}

	// Display results
	displayEnhancedResults(formatter, results, runConfig.Quiet, cb.app.config.Verbose)

	// Return error if any checks failed (unless they were gracefully skipped)
	if results.Failed > 0 {
		return fmt.Errorf("%w: %d", prerrors.ErrChecksFailed, results.Failed)
	}

	if results.Passed > 0 {
		formatter.Success("All checks passed! %s",
			formatter.FormatExecutionStats(results.Passed, results.Failed, results.Skipped, results.TotalDuration, results.TotalFiles))
	}

	return nil
}

// extractKeyErrorLines parses command output and extracts the most important error lines
func extractKeyErrorLines(output string) []string {
	var errorLines []string
	lines := strings.Split(output, "\n")

	// Compile regex once outside the loop
	goErrorRegex := regexp.MustCompile(`\w+\.go:\d+:\d+:`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip progress/status lines
		if strings.HasPrefix(line, "Running") ||
			strings.HasPrefix(line, "Checking") ||
			strings.HasPrefix(line, "Analyzing") ||
			strings.Contains(line, "files linted") {
			continue
		}

		// Look for actual error patterns
		isError := false

		// Go file errors (file.go:line:col: message)
		if goErrorRegex.MatchString(line) {
			isError = true
		}

		// Whitespace issues
		if strings.Contains(line, "trailing whitespace") ||
			strings.Contains(line, "mixed spaces and tabs") {
			isError = true
		}

		// General error indicators
		if strings.Contains(line, "error:") ||
			strings.Contains(line, "Error:") ||
			strings.Contains(line, "ERRO") ||
			strings.Contains(line, "level=error") ||
			strings.Contains(line, "✗") {
			isError = true
		}

		// Module tidy issues
		if strings.Contains(line, "go.mod") && strings.Contains(line, "not tidy") {
			isError = true
		}

		// Module path indicators for multi-module errors
		if strings.HasPrefix(line, "Module ") && (strings.Contains(line, "needs tidying") || strings.Contains(line, ":")) {
			isError = true
		}

		// Diff output from go mod tidy -diff
		if strings.HasPrefix(line, "diff ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") ||
			strings.HasPrefix(line, "@@") ||
			(strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++")) ||
			(strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---")) {
			isError = true
		}

		if isError {
			// Clean up ANSI color codes for cleaner display
			cleanLine := stripANSI(line)
			errorLines = append(errorLines, cleanLine)

			// Limit to first 10 error lines for non-verbose mode
			if len(errorLines) >= 10 {
				break
			}
		}
	}

	return errorLines
}

// stripANSI removes ANSI color codes from a string
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

func showAvailableChecks(cfg *config.Config, formatter *output.Formatter) error {
	formatter.Header("Available Checks")

	checks := []struct {
		name        string
		description string
		enabled     bool
	}{
		{"ai_detection", "Detect AI attribution in code and commit messages", cfg.Checks.AIDetection},
		{"eof", "Ensure files end with newline", cfg.Checks.EOF},
		{"fmt", "Format code with go fmt", cfg.Checks.Fmt},
		{"fumpt", "Format code with gofumpt", cfg.Checks.Fumpt},
		{"gitleaks", "Scan for secrets and credentials in code", cfg.Checks.Gitleaks},
		{"goimports", "Format code and manage imports", cfg.Checks.Goimports},
		{"lint", "Run golangci-lint", cfg.Checks.Lint},
		{"mod-tidy", "Ensure go.mod and go.sum are tidy", cfg.Checks.ModTidy},
		{"whitespace", "Fix trailing whitespace", cfg.Checks.Whitespace},
	}

	for _, check := range checks {
		if check.enabled {
			formatter.Success("%-12s %s", check.name, check.description)
		} else {
			formatter.Detail("%-12s %s (disabled)", check.name, check.description)
		}
	}

	return nil
}

func displayEnhancedResults(formatter *output.Formatter, results *runner.Results, quietMode, verboseMode bool) {
	// In quiet mode, skip the header and only show failures
	if !quietMode {
		formatter.Header("Check Results")
	}

	// Collect failed checks for error summary
	var failedChecks []runner.CheckResult

	// Display each check result
	for _, result := range results.CheckResults {
		if result.Success {
			if !quietMode {
				if result.CanSkip && result.Suggestion != "" {
					// This was a gracefully skipped check
					formatter.Warning("%s - %s", result.Name, result.Error)
					if result.Suggestion != "" {
						formatter.SuggestAction(result.Suggestion)
					}
				} else {
					// Normal success - always show duration inline
					formatter.Success("%s completed successfully (%s)", result.Name, formatter.Duration(result.Duration))
					if verboseMode && len(result.Files) > 0 {
						formatter.Detail("Files: %s", formatter.FormatFileList(result.Files, 3))
					}
				}
			}
		} else {
			// Failed check - add to failed list for summary
			failedChecks = append(failedChecks, result)

			// Failed check - always show duration inline
			formatter.Error("%s failed (%s)", result.Name, formatter.Duration(result.Duration))

			if verboseMode && len(result.Files) > 0 {
				formatter.Detail("Files: %s", formatter.FormatFileList(result.Files, 3))
			}

			// Show error message
			if result.Error != "" {
				formatter.Detail("Error: %s", result.Error)
			}

			// Always show command output for failures to make errors visible
			// This is the key change - show actual errors even without verbose mode
			if result.Output != "" {
				// Parse and show key error lines
				errorLines := extractKeyErrorLines(result.Output)
				if len(errorLines) > 0 {
					for _, line := range errorLines {
						formatter.Detail("  %s", line)
					}
					if !verboseMode && len(errorLines) >= 10 {
						formatter.Detail("  ... (run with --verbose for full output)")
					}
				} else if verboseMode {
					// Fall back to full output in verbose mode
					formatter.Subheader("Command Output")
					formatter.CodeBlock(result.Output)
				}
			}

			// Show actionable suggestion
			if result.Suggestion != "" {
				formatter.SuggestAction(result.Suggestion)
			}
		}
	}

	// Summary (skip in quiet mode if all passed)
	if !quietMode || results.Failed > 0 {
		formatter.Subheader("Summary")
		stats := formatter.FormatExecutionStats(results.Passed, results.Failed, results.Skipped, results.TotalDuration, results.TotalFiles)
		if results.Failed > 0 {
			formatter.Error(stats)
		} else if results.Skipped > 0 {
			formatter.Warning(stats)
		} else {
			formatter.Success(stats)
		}
	}

	// Error Summary - show consolidated view of all failures
	if len(failedChecks) > 0 {
		formatter.Header("ERRORS FOUND")
		formatter.Error("%d check(s) failed with the following errors:", len(failedChecks))
		formatter.Detail("") // Empty line for spacing

		for _, check := range failedChecks {
			formatter.Error("━ %s ━", check.Name)

			// Show the specific errors for this check
			if check.Output != "" {
				errorLines := extractKeyErrorLines(check.Output)
				if len(errorLines) > 0 {
					for i, line := range errorLines {
						if i < 5 { // Show up to 5 error lines per check in summary
							formatter.Detail("  %s", line)
						}
					}
					if len(errorLines) > 5 {
						formatter.Detail("  ... and %d more errors", len(errorLines)-5)
					}
				} else {
					// No specific errors extracted, show generic message
					formatter.Detail("  %s", check.Error)
				}
			} else if check.Error != "" {
				formatter.Detail("  %s", check.Error)
			}

			// Show how to fix
			if check.Suggestion != "" {
				formatter.SuggestAction(fmt.Sprintf("To fix: %s", check.Suggestion))
			}
			formatter.Detail("") // Empty line for spacing
		}

		// Overall guidance
		formatter.Info("Run with --verbose flag to see full error details")
		formatter.Info("Fix the errors above and run the checks again")
	}
}

// resetRunFlags is no longer needed - flags are handled through dependency injection
