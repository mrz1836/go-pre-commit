package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/git"
	"github.com/mrz1836/go-pre-commit/internal/output"
	"github.com/mrz1836/go-pre-commit/internal/runner"
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
  eof          - Ensure files end with newline
  fumpt        - Format code with gofumpt
  gitleaks     - Scan for secrets and credentials in code
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
	formatter := cb.newFormatter(cfg)

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
	filesToCheck, err := selectFilesToCheck(runConfig, repoRoot, formatter)
	if err != nil {
		return err
	}

	if len(filesToCheck) == 0 {
		formatter.Info("No files to check")
		return nil
	}

	// Create runner and configure options
	r := runner.New(cfg, repoRoot)
	opts := buildRunnerOptions(runConfig, args, filesToCheck, formatter)

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

// newFormatter builds an output formatter honoring the CLI color flags
// (--no-color, --color) with a fallback to the configured color preference.
func (cb *CommandBuilder) newFormatter(cfg *config.Config) *output.Formatter {
	if cb.app.config.NoColor {
		// --no-color flag takes highest priority
		return output.NewWithColorMode(output.ColorNever)
	}

	// Use --color flag or auto-detect
	switch cb.app.config.ColorMode {
	case colorModeAlways:
		return output.NewWithColorMode(output.ColorAlways)
	case colorModeNever:
		return output.NewWithColorMode(output.ColorNever)
	case colorModeAuto:
		return output.NewWithColorMode(output.ColorAuto)
	default:
		// Default to auto mode with config override
		if !cfg.UI.ColorOutput {
			return output.NewWithColorMode(output.ColorNever)
		}
		return output.NewWithColorMode(output.ColorAuto)
	}
}

// selectFilesToCheck resolves the set of files to run checks against based on
// the run configuration: explicit files, all repository files, or staged files.
func selectFilesToCheck(runConfig RunConfig, repoRoot string, formatter *output.Formatter) ([]string, error) {
	switch {
	case len(runConfig.Files) > 0:
		// Specific files provided
		return runConfig.Files, nil
	case runConfig.AllFiles:
		// All files in repository
		files, err := git.NewRepository(repoRoot).GetAllFiles()
		if err != nil {
			formatter.Error("Failed to get all files: %v", err)
			return nil, fmt.Errorf("failed to get all files: %w", err)
		}
		return files, nil
	default:
		// Staged files (default)
		files, err := git.NewRepository(repoRoot).GetStagedFiles()
		if err != nil {
			formatter.Error("Failed to get staged files: %v", err)
			return nil, fmt.Errorf("failed to get staged files: %w", err)
		}
		return files, nil
	}
}

// buildRunnerOptions assembles runner options from the run configuration,
// wiring up the progress callback and resolving which checks to run.
func buildRunnerOptions(runConfig RunConfig, args, filesToCheck []string, formatter *output.Formatter) runner.Options {
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
	switch {
	case len(args) > 0:
		// Specific check requested as positional argument
		opts.OnlyChecks = []string{args[0]}
	case len(runConfig.OnlyChecks) > 0:
		// --only flag
		opts.OnlyChecks = runConfig.OnlyChecks
	case len(runConfig.SkipChecks) > 0:
		// --skip flag
		opts.SkipChecks = runConfig.SkipChecks
	}

	return opts
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
		if isProgressLine(line) {
			continue
		}

		if isKeyErrorLine(line, goErrorRegex) {
			// Clean up ANSI color codes for cleaner display
			errorLines = append(errorLines, stripANSI(line))

			// Limit to first 10 error lines for non-verbose mode
			if len(errorLines) >= 10 {
				break
			}
		}
	}

	return errorLines
}

// isProgressLine reports whether a line is informational progress/status output
// that should be skipped when extracting error lines.
func isProgressLine(line string) bool {
	return strings.HasPrefix(line, "Running") ||
		strings.HasPrefix(line, "Checking") ||
		strings.HasPrefix(line, "Analyzing") ||
		strings.Contains(line, "files linted")
}

// isKeyErrorLine reports whether a trimmed, non-empty output line represents a
// meaningful error worth surfacing to the user.
func isKeyErrorLine(line string, goErrorRegex *regexp.Regexp) bool {
	switch {
	// Go file errors (file.go:line:col: message)
	case goErrorRegex.MatchString(line):
		return true
	// Whitespace issues
	case strings.Contains(line, "trailing whitespace"),
		strings.Contains(line, "mixed spaces and tabs"):
		return true
	// General error indicators
	case strings.Contains(line, "error:"),
		strings.Contains(line, "Error:"),
		strings.Contains(line, "ERRO"),
		strings.Contains(line, "level=error"),
		strings.Contains(line, "✗"):
		return true
	// Module tidy issues
	case strings.Contains(line, "go.mod") && strings.Contains(line, "not tidy"):
		return true
	// Module path indicators for multi-module errors
	case strings.HasPrefix(line, "Module ") && (strings.Contains(line, "needs tidying") || strings.Contains(line, ":")):
		return true
	// Diff output from go mod tidy -diff
	case strings.HasPrefix(line, "diff "),
		strings.HasPrefix(line, "--- "),
		strings.HasPrefix(line, "+++ "),
		strings.HasPrefix(line, "@@"),
		strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"),
		strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
		return true
	default:
		return false
	}
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
		{"eof", "Ensure files end with newline", cfg.Checks.EOF},
		{"fumpt", "Format code with gofumpt", cfg.Checks.Fumpt},
		{"gitleaks", "Scan for secrets and credentials in code", cfg.Checks.Gitleaks},
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

	// Display each check result, collecting failures for the error summary
	var failedChecks []runner.CheckResult
	for _, result := range results.CheckResults {
		if !result.Success {
			failedChecks = append(failedChecks, result)
		}
		displayCheckResult(formatter, result, quietMode, verboseMode)
	}

	displayResultSummary(formatter, results, quietMode)
	displayErrorSummary(formatter, failedChecks)
}

// displayCheckResult renders a single check result: success, graceful skip, or
// failure (with key error lines and a remediation suggestion).
func displayCheckResult(formatter *output.Formatter, result runner.CheckResult, quietMode, verboseMode bool) {
	if result.Success {
		if quietMode {
			return
		}
		if result.CanSkip && result.Suggestion != "" {
			// This was a gracefully skipped check
			formatter.Warning("%s - %s", result.Name, result.Error)
			formatter.SuggestAction(result.Suggestion)
			return
		}
		// Normal success - always show duration inline
		formatter.Success("%s completed successfully (%s)", result.Name, formatter.Duration(result.Duration))
		if verboseMode && len(result.Files) > 0 {
			formatter.Detail("Files: %s", formatter.FormatFileList(result.Files, 3))
		}
		return
	}

	// Failed check - always show duration inline
	formatter.Error("%s failed (%s)", result.Name, formatter.Duration(result.Duration))

	if verboseMode && len(result.Files) > 0 {
		formatter.Detail("Files: %s", formatter.FormatFileList(result.Files, 3))
	}

	// Show error message
	if result.Error != "" {
		formatter.Detail("Error: %s", result.Error)
	}

	// Always show command output for failures to make errors visible,
	// even without verbose mode.
	if result.Output != "" {
		errorLines := extractKeyErrorLines(result.Output)
		switch {
		case len(errorLines) > 0:
			for _, line := range errorLines {
				formatter.Detail("  %s", line)
			}
			if !verboseMode && len(errorLines) >= 10 {
				formatter.Detail("  ... (run with --verbose for full output)")
			}
		case verboseMode:
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

// displayResultSummary prints the execution-statistics summary line, colored by
// outcome. It is skipped in quiet mode when everything passed.
func displayResultSummary(formatter *output.Formatter, results *runner.Results, quietMode bool) {
	if quietMode && results.Failed == 0 {
		return
	}

	formatter.Subheader("Summary")
	stats := formatter.FormatExecutionStats(results.Passed, results.Failed, results.Skipped, results.TotalDuration, results.TotalFiles)
	switch {
	case results.Failed > 0:
		formatter.Error(stats)
	case results.Skipped > 0:
		formatter.Warning(stats)
	default:
		formatter.Success(stats)
	}
}

// displayErrorSummary prints a consolidated view of all failed checks with their
// key error lines and fix suggestions. It is a no-op when nothing failed.
func displayErrorSummary(formatter *output.Formatter, failedChecks []runner.CheckResult) {
	if len(failedChecks) == 0 {
		return
	}

	formatter.Header("ERRORS FOUND")
	formatter.Error("%d check(s) failed with the following errors:", len(failedChecks))
	formatter.Detail("") // Empty line for spacing

	for _, check := range failedChecks {
		formatter.Error("━ %s ━", check.Name)

		// Show the specific errors for this check
		switch {
		case check.Output != "":
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
		case check.Error != "":
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

// resetRunFlags is no longer needed - flags are handled through dependency injection
