package gotools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/shared"
)

// LintCheck runs golangci-lint directly or via build tools
type LintCheck struct {
	sharedCtx *shared.Context
	timeout   time.Duration
}

// NewLintCheck creates a new lint check
func NewLintCheck() *LintCheck {
	return &LintCheck{
		sharedCtx: shared.NewContext(),
		timeout:   60 * time.Second, // 60 second timeout for lint
	}
}

// NewLintCheckWithSharedContext creates a new lint check with shared context
func NewLintCheckWithSharedContext(sharedCtx *shared.Context) *LintCheck {
	return &LintCheck{
		sharedCtx: sharedCtx,
		timeout:   60 * time.Second,
	}
}

// NewLintCheckWithConfig creates a new lint check with shared context and custom timeout
func NewLintCheckWithConfig(sharedCtx *shared.Context, timeout time.Duration) *LintCheck {
	return &LintCheck{
		sharedCtx: sharedCtx,
		timeout:   timeout,
	}
}

// Name returns the name of the check
func (c *LintCheck) Name() string {
	return "lint"
}

// Description returns a brief description of the check
func (c *LintCheck) Description() string {
	return "Run golangci-lint"
}

// Metadata returns comprehensive metadata about the check
func (c *LintCheck) Metadata() interface{} {
	return CheckMetadata{
		Name:              "lint",
		Description:       "Run golangci-lint to check code quality and style",
		FilePatterns:      []string{"*.go"},
		EstimatedDuration: 10 * time.Second,
		Dependencies:      []string{"lint"}, // tool or build target
		DefaultTimeout:    c.timeout,
		Category:          "linting",
		RequiresFiles:     true,
	}
}

// Run executes the lint check
func (c *LintCheck) Run(ctx context.Context, files []string) error {
	// Early return if no files to process
	if len(files) == 0 {
		return nil
	}

	// Prefer direct golangci-lint execution for pure Go implementation
	err := c.runDirectLint(ctx, files)

	// Only fall back to build tool if direct execution failed and build target exists
	if err != nil && c.sharedCtx.HasMagexTarget(ctx, "lint") {
		// Try magex lint as fallback
		err = c.runMagexLint(ctx)
	}

	return err
}

// FilterFiles filters to only Go files
func (c *LintCheck) FilterFiles(files []string) []string {
	var filtered []string
	for _, file := range files {
		if strings.HasSuffix(file, ".go") {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// runMagexLint runs magex lint with proper error handling
func (c *LintCheck) runMagexLint(ctx context.Context) error {
	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Add timeout for magex command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "magex", "lint")
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Combine stdout and stderr for analysis
		output := stdout.String() + stderr.String()

		// Check if it's a context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolExecutionError(
				"magex lint",
				output,
				fmt.Sprintf("Lint check timed out after %v. Consider increasing GO_PRE_COMMIT_LINT_TIMEOUT or run 'magex lint' manually to see detailed output.", c.timeout),
			)
		}

		// Parse the error for better context
		if strings.Contains(output, "command not found") || strings.Contains(output, "unknown command") {
			return prerrors.NewMagexTargetNotFoundError(
				"lint",
				"Create a 'lint' target in your magex configuration or disable linting with GO_PRE_COMMIT_ENABLE_LINT=false",
			)
		}

		if strings.Contains(output, "golangci-lint") && strings.Contains(output, "not found") {
			return prerrors.NewToolNotFoundError(
				"golangci-lint",
				"Install golangci-lint: 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest' or add an install target to your build configuration",
			)
		}

		// Try to determine if this is linting issues vs. actual failure
		if strings.Contains(output, "level=error") ||
			strings.Contains(output, "ERRO") ||
			(strings.Contains(output, ".go:") && strings.Contains(output, ":")) {
			// This looks like linting issues, not a tool failure
			// Extract and format specific lint errors for better visibility
			formattedOutput := FormatLintErrors(output)
			// For lint errors, return the formatted output as the error message
			return &prerrors.CheckError{
				Err:        prerrors.ErrLintingIssues,
				Message:    formattedOutput,
				Suggestion: "Fix the linting issues shown above. Run 'magex lint' to see full details and 'golangci-lint run --help' for configuration options.",
				Command:    "magex lint",
				Output:     formattedOutput,
			}
		}

		// Generic failure
		return prerrors.NewToolExecutionError(
			"magex lint",
			output,
			"Run 'magex lint' manually to see detailed error output. Check your build configuration and golangci-lint settings.",
		)
	}

	return nil
}

// runDirectLint runs golangci-lint directly on files
func (c *LintCheck) runDirectLint(ctx context.Context, files []string) error {
	// Check if golangci-lint is available, install if not
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		// Try to install golangci-lint using the official install script
		// This is the preferred installation method as it handles platform specifics
		if installErr := c.installGolangciLint(ctx); installErr != nil {
			return prerrors.NewToolNotFoundError(
				"golangci-lint",
				fmt.Sprintf("Failed to auto-install golangci-lint: %v\nTry manually: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin", installErr),
			)
		}

		// Verify installation succeeded
		if _, err := exec.LookPath("golangci-lint"); err != nil {
			return prerrors.NewToolNotFoundError(
				"golangci-lint",
				"golangci-lint was installed but not found in PATH. Ensure $(go env GOPATH)/bin is in your PATH",
			)
		}
	}

	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Group files by directory
	filesByDir := make(map[string][]string)
	for _, file := range files {
		dir := filepath.Dir(file)
		filesByDir[dir] = append(filesByDir[dir], file)
	}

	// If all files are in the same directory, use the optimized single-directory path
	if len(filesByDir) == 1 {
		return c.runLintOnFiles(ctx, repoRoot, files)
	}

	// For multiple directories, run golangci-lint on each directory
	// This avoids the "named files must all be in one directory" error
	var allErrors []string
	var hasLintingIssues bool
	var hasToolFailure bool

	for dir := range filesByDir {
		// Run golangci-lint on the directory containing the files
		err = c.runLintOnDirectory(ctx, repoRoot, dir)
		if err != nil {
			// Check if it's actual linting issues vs tool failure
			var checkErr *prerrors.CheckError
			if errors.As(err, &checkErr) && errors.Is(checkErr.Err, prerrors.ErrLintingIssues) {
				hasLintingIssues = true
				allErrors = append(allErrors, fmt.Sprintf("Directory %s:\n%s", dir, checkErr.Message))
			} else {
				hasToolFailure = true
				allErrors = append(allErrors, fmt.Sprintf("Directory %s: %v", dir, err))
			}
		}
	}

	// If there were any errors, aggregate and return them
	if len(allErrors) > 0 {
		combinedErrors := strings.Join(allErrors, "\n\n")

		if hasLintingIssues && !hasToolFailure {
			// All errors are linting issues
			return &prerrors.CheckError{
				Err:        prerrors.ErrLintingIssues,
				Message:    combinedErrors,
				Suggestion: "Fix the linting issues shown above. Run 'golangci-lint run' on each directory to see full details.",
				Command:    "golangci-lint run",
				Output:     combinedErrors,
			}
		}

		// There were tool failures
		return prerrors.NewToolExecutionError(
			"golangci-lint run",
			combinedErrors,
			"Run 'golangci-lint run' manually on each directory to see detailed error output.",
		)
	}

	return nil
}

// runLintOnFiles runs golangci-lint on the directory containing the files (all in the same directory)
// This ensures golangci-lint has the full package context for typecheck to work correctly
func (c *LintCheck) runLintOnFiles(ctx context.Context, repoRoot string, files []string) error {
	// Since all files are guaranteed to be in the same directory (checked in caller),
	// run golangci-lint on the directory to ensure full package context
	dir := filepath.Dir(files[0])
	return c.runLintOnDirectory(ctx, repoRoot, dir)
}

// runLintOnDirectory runs golangci-lint on a specific directory
func (c *LintCheck) runLintOnDirectory(ctx context.Context, repoRoot, dir string) error {
	// Add timeout for golangci-lint command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Run golangci-lint on the directory with --new-from-rev flag to only check changed files
	args := []string{"run", "--new-from-rev=HEAD~1", filepath.Join(repoRoot, dir)}
	cmd := exec.CommandContext(ctx, "golangci-lint", args...) //nolint:gosec // Command arguments are validated
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stdout.String() + stderr.String()

		// Check if it's a context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolExecutionError(
				"golangci-lint run",
				output,
				fmt.Sprintf("Lint check timed out after %v. Consider increasing GO_PRE_COMMIT_LINT_TIMEOUT.", c.timeout),
			)
		}

		// Check if it's configuration issues
		if strings.Contains(output, "config") && (strings.Contains(output, "error") || strings.Contains(output, "failed")) {
			return prerrors.NewToolExecutionError(
				"golangci-lint run",
				output,
				"Fix golangci-lint configuration issues. Check your .golangci.yml file or run 'golangci-lint config path'.",
			)
		}

		// Check if it's actual linting issues vs tool failure
		if strings.Contains(output, ".go:") && strings.Contains(output, ":") {
			// This looks like linting issues, not a tool failure
			// Extract and format specific lint errors for better visibility
			formattedOutput := FormatLintErrors(output)
			// For lint errors, return the formatted output as the error message
			return &prerrors.CheckError{
				Err:        prerrors.ErrLintingIssues,
				Message:    formattedOutput,
				Suggestion: fmt.Sprintf("Fix the linting issues shown above. Run 'golangci-lint run %s' to see full details.", dir),
				Command:    fmt.Sprintf("golangci-lint run %s", dir),
				Output:     formattedOutput,
			}
		}

		// Generic failure
		return prerrors.NewToolExecutionError(
			"golangci-lint run",
			output,
			fmt.Sprintf("Run 'golangci-lint run %s' manually to see detailed error output.", dir),
		)
	}

	return nil
}

// FormatLintErrors extracts and formats specific lint violations for clearer display
// Exported for testing purposes
func FormatLintErrors(output string) string {
	var result strings.Builder
	lines := strings.Split(output, "\n")
	errorCount := 0

	// Track unique errors to avoid duplicates
	seenErrors := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for Go file error patterns (file.go:line:col: message)
		// Example: internal/git/files.go:89:2: ineffectual assignment to err (ineffassign)
		if strings.Contains(line, ".go:") && strings.Contains(line, ":") {
			// Clean up ANSI codes if present
			cleanLine := StripANSIColors(line)

			// Avoid duplicate errors
			if !seenErrors[cleanLine] {
				seenErrors[cleanLine] = true
				if errorCount > 0 {
					result.WriteString("\n")
				}
				result.WriteString(cleanLine)
				errorCount++
			}
		}
	}

	// If we found specific errors, return them
	if errorCount > 0 {
		header := fmt.Sprintf("Found %d linting issue(s):\n", errorCount)
		return header + result.String()
	}

	// Otherwise return the original output
	return output
}

// StripANSIColors removes ANSI color codes from a string
// Exported for testing purposes
func StripANSIColors(s string) string {
	// Remove ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

// installGolangciLint installs golangci-lint using the official installation script
func (c *LintCheck) installGolangciLint(ctx context.Context) error {
	// Get GOPATH to determine installation directory
	goCmd := exec.CommandContext(ctx, "go", "env", "GOPATH")
	gopathBytes, err := goCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get GOPATH: %w", err)
	}
	gopath := strings.TrimSpace(string(gopathBytes))
	if gopath == "" {
		// Fallback to default GOPATH
		gopath = filepath.Join(os.Getenv("HOME"), "go")
	}

	installDir := filepath.Join(gopath, "bin")

	// Download and run the installation script
	// Using sh -c to pipe the curl output to sh
	installScript := fmt.Sprintf(
		"curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b %s",
		installDir,
	)

	cmd := exec.CommandContext(ctx, "sh", "-c", installScript) //nolint:gosec // installScript is constructed from constants and validated paths
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installation failed: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}
