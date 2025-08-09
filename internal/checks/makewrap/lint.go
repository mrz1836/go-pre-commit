package makewrap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/shared"
)

// LintCheck runs golangci-lint via make
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
		Dependencies:      []string{"lint"}, // make target
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

	// Check if make lint is available
	if c.sharedCtx.HasMakeTarget(ctx, "lint") {
		// Run make lint with timeout
		return c.runMakeLint(ctx)
	}

	// Fall back to direct golangci-lint if available
	return c.runDirectLint(ctx, files)
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

// runMakeLint runs make lint with proper error handling
func (c *LintCheck) runMakeLint(ctx context.Context) error {
	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Add timeout for make command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "make", "lint")
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
				"make lint",
				output,
				fmt.Sprintf("Lint check timed out after %v. Consider increasing GO_PRE_COMMIT_LINT_TIMEOUT or run 'make lint' manually to see detailed output.", c.timeout),
			)
		}

		// Parse the error for better context
		if strings.Contains(output, "No rule to make target") {
			return prerrors.NewMakeTargetNotFoundError(
				"lint",
				"Create a 'lint' target in your Makefile or disable linting with GO_PRE_COMMIT_ENABLE_LINT=false",
			)
		}

		if strings.Contains(output, "golangci-lint") && strings.Contains(output, "not found") {
			return prerrors.NewToolNotFoundError(
				"golangci-lint",
				"Install golangci-lint: 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest' or add an install target to your Makefile",
			)
		}

		// Try to determine if this is linting issues vs. actual failure
		if strings.Contains(output, "level=error") ||
			strings.Contains(output, "ERRO") ||
			(strings.Contains(output, ".go:") && strings.Contains(output, ":")) {
			// This looks like linting issues, not a tool failure
			// Extract and format specific lint errors for better visibility
			formattedOutput := formatLintErrors(output)
			// For lint errors, return the formatted output as the error message
			return &prerrors.CheckError{
				Err:        prerrors.ErrLintingIssues,
				Message:    formattedOutput,
				Suggestion: "Fix the linting issues shown above. Run 'make lint' to see full details and 'golangci-lint run --help' for configuration options.",
				Command:    "make lint",
				Output:     formattedOutput,
			}
		}

		// Generic failure
		return prerrors.NewToolExecutionError(
			"make lint",
			output,
			"Run 'make lint' manually to see detailed error output. Check your Makefile and golangci-lint configuration.",
		)
	}

	return nil
}

// runDirectLint runs golangci-lint directly on files
func (c *LintCheck) runDirectLint(ctx context.Context, files []string) error {
	// Check if golangci-lint is available
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		return prerrors.NewToolNotFoundError(
			"golangci-lint",
			"Install golangci-lint: 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest'",
		)
	}

	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Build absolute paths
	absFiles := make([]string, len(files))
	for i, file := range files {
		absFiles[i] = filepath.Join(repoRoot, file)
	}

	// Add timeout for golangci-lint command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Run golangci-lint
	args := append([]string{"run", "--new-from-rev=HEAD~1"}, absFiles...)
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
				fmt.Sprintf("Lint check timed out after %v. Consider increasing GO_PRE_COMMIT_LINT_TIMEOUT or running on fewer files.", c.timeout),
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
			formattedOutput := formatLintErrors(output)
			// For lint errors, return the formatted output as the error message
			return &prerrors.CheckError{
				Err:        prerrors.ErrLintingIssues,
				Message:    formattedOutput,
				Suggestion: "Fix the linting issues shown above. Run 'golangci-lint run' to see full details.",
				Command:    "golangci-lint run",
				Output:     formattedOutput,
			}
		}

		// Generic failure
		return prerrors.NewToolExecutionError(
			"golangci-lint run",
			output,
			"Run 'golangci-lint run' manually to see detailed error output. Check your configuration and file paths.",
		)
	}

	return nil
}

// formatLintErrors extracts and formats specific lint violations for clearer display
func formatLintErrors(output string) string {
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
			cleanLine := stripANSIColors(line)

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

// stripANSIColors removes ANSI color codes from a string
func stripANSIColors(s string) string {
	// Remove ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}
