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

	"github.com/mrz1836/go-pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/shared"
	"github.com/mrz1836/go-pre-commit/internal/tools"
)

// LintCheck runs golangci-lint directly or via build tools
type LintCheck struct {
	sharedCtx *shared.Context
	config    *config.Config
	timeout   time.Duration
	buildTags []string
}

// NewLintCheck creates a new lint check
func NewLintCheck() *LintCheck {
	return &LintCheck{
		sharedCtx: shared.NewContext(),
		config:    nil,              // Config not available in basic constructor
		timeout:   60 * time.Second, // 60 second timeout for lint
	}
}

// NewLintCheckWithSharedContext creates a new lint check with shared context
func NewLintCheckWithSharedContext(sharedCtx *shared.Context) *LintCheck {
	return &LintCheck{
		sharedCtx: sharedCtx,
		config:    nil, // Config not available in this constructor
		timeout:   60 * time.Second,
	}
}

// NewLintCheckWithConfig creates a new lint check with shared context and custom timeout
func NewLintCheckWithConfig(sharedCtx *shared.Context, cfg *config.Config, timeout time.Duration) *LintCheck {
	return &LintCheck{
		sharedCtx: sharedCtx,
		config:    cfg,
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

	// Check for build tags from environment variable
	if envTags := os.Getenv("GO_PRE_COMMIT_BUILD_TAGS"); envTags != "" {
		c.buildTags = strings.Split(envTags, ",")
		for i, tag := range c.buildTags {
			c.buildTags[i] = strings.TrimSpace(tag)
		}
	}

	// Ensure golangci-lint is installed
	if err := tools.EnsureInstalled(ctx, "golangci-lint"); err != nil {
		return prerrors.NewToolExecutionError(
			"golangci-lint",
			err.Error(),
			"Failed to install golangci-lint. You can install it manually with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
		)
	}

	// Run golangci-lint directly
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

// runDirectLint runs golangci-lint directly on files
func (c *LintCheck) runDirectLint(ctx context.Context, files []string) error {
	// Tool installation is already handled in Run(), so we can proceed directly

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

	// Build golangci-lint command arguments
	args := []string{"run", "--new-from-rev=HEAD~1"}

	// Add build tags if configured
	if len(c.buildTags) > 0 {
		args = append(args, "--build-tags", strings.Join(c.buildTags, ","))
	}

	// Determine the target directory and working directory
	targetDir := filepath.Join(repoRoot, dir)
	workingDir := repoRoot
	lintTarget := targetDir

	// Check if the target directory contains a Go module
	if isGoModule(targetDir) {
		// If target directory is a Go module, run golangci-lint from within it
		// This ensures proper module resolution and dependency handling
		workingDir = targetDir
		lintTarget = "./..."
	} else {
		// Check if this directory is within a Go module in a subdirectory
		moduleRoot := findGoModuleRoot(targetDir, repoRoot)
		if moduleRoot != "" && moduleRoot != repoRoot {
			// Calculate the relative path from module root to target directory
			relPath, err := filepath.Rel(moduleRoot, targetDir)
			if err == nil {
				workingDir = moduleRoot
				lintTarget = "./" + relPath
			}
		} else if moduleRoot == "" {
			// No module found via directory walking - check GO_SUM_FILE configuration
			if c.config != nil {
				moduleDirFromConfig := c.config.GetModuleDir()
				var configModuleDir string
				if moduleDirFromConfig != "" {
					configModuleDir = filepath.Join(repoRoot, moduleDirFromConfig)
				} else {
					configModuleDir = repoRoot
				}

				// If config points to a valid Go module, use it
				if isGoModule(configModuleDir) {
					workingDir = configModuleDir
					relPath, err := filepath.Rel(configModuleDir, targetDir)
					if err == nil && !strings.HasPrefix(relPath, "..") {
						// Target is within the configured module
						lintTarget = "./" + relPath
					} else {
						// Target is outside the configured module - skip linting
						return nil
					}
				} else {
					// No valid Go module found - skip linting
					return nil
				}
			} else {
				// No config and no module found - skip linting
				// Go files outside of modules can't be properly linted by golangci-lint
				return nil
			}
		}
	}

	args = append(args, lintTarget)

	cmd := exec.CommandContext(ctx, "golangci-lint", args...)
	cmd.Dir = workingDir

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

		// Check if it's build constraints issue
		if strings.Contains(output, "build constraints exclude all Go files") {
			return c.handleBuildConstraintsError(ctx, repoRoot, dir, output)
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

// handleBuildConstraintsError handles the case where build constraints exclude all Go files
func (c *LintCheck) handleBuildConstraintsError(ctx context.Context, repoRoot, dir, originalOutput string) error {
	// Get all Go files in the directory
	dirPath := filepath.Join(repoRoot, dir)
	dirFiles, err := filepath.Glob(filepath.Join(dirPath, "*.go"))
	if err != nil {
		return prerrors.NewToolExecutionError(
			"golangci-lint run",
			originalOutput,
			"Failed to scan directory for Go files with build constraints.",
		)
	}

	// Detect build tags in the files
	buildTags := detectBuildTags(dirFiles)
	if len(buildTags) == 0 {
		return prerrors.NewToolExecutionError(
			"golangci-lint run",
			originalOutput,
			"Build constraints exclude all Go files. Consider adding 'build-tags' to your .golangci.json configuration.",
		)
	}

	// Retry with detected build tags
	retryArgs := []string{"run", "--new-from-rev=HEAD~1", "--build-tags", strings.Join(buildTags, ",")}
	retryArgs = append(retryArgs, filepath.Join(repoRoot, dir))

	retryCmd := exec.CommandContext(ctx, "golangci-lint", retryArgs...) //nolint:gosec // Command arguments are validated
	retryCmd.Dir = repoRoot

	var retryStdout, retryStderr bytes.Buffer
	retryCmd.Stdout = &retryStdout
	retryCmd.Stderr = &retryStderr

	if err := retryCmd.Run(); err == nil {
		// Success with auto-detected build tags
		return nil
	}

	// Still failing, provide helpful error with detected tags
	retryOutput := retryStdout.String() + retryStderr.String()

	// Check if the retry attempt shows linting issues (success case)
	if strings.Contains(retryOutput, ".go:") && strings.Contains(retryOutput, ":") {
		formattedOutput := FormatLintErrors(retryOutput)
		return &prerrors.CheckError{
			Err:        prerrors.ErrLintingIssues,
			Message:    formattedOutput,
			Suggestion: fmt.Sprintf("Fix the linting issues shown above. Run 'golangci-lint run --build-tags %s %s' to see full details.", strings.Join(buildTags, ","), dir),
			Command:    fmt.Sprintf("golangci-lint run --build-tags %s %s", strings.Join(buildTags, ","), dir),
			Output:     formattedOutput,
		}
	}

	return prerrors.NewToolExecutionError(
		"golangci-lint run",
		originalOutput,
		fmt.Sprintf("Build constraints exclude all Go files. Detected build tags: %v. Consider adding these to your .golangci.json configuration:\n\"build-tags\": %q",
			buildTags, buildTags),
	)
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

// detectBuildTags scans files for build constraints and returns unique tags
func detectBuildTags(files []string) []string {
	tagSet := make(map[string]bool)

	for _, file := range files {
		content, err := os.ReadFile(file) //nolint:gosec // File paths are validated by caller
		if err != nil {
			continue // Skip files we can't read
		}

		lines := strings.Split(string(content), "\n")
		// Only check the first 10 lines for build constraints
		maxLines := len(lines)
		if maxLines > 10 {
			maxLines = 10
		}

		for i := 0; i < maxLines; i++ {
			line := strings.TrimSpace(lines[i])

			// Check for //go:build constraints (Go 1.17+)
			if strings.HasPrefix(line, "//go:build ") {
				tags := extractTagsFromConstraint(line)
				for _, tag := range tags {
					tagSet[tag] = true
				}
			}
			// Check for // +build constraints (legacy)
			if strings.HasPrefix(line, "// +build ") {
				tags := extractTagsFromLegacyConstraint(line)
				for _, tag := range tags {
					tagSet[tag] = true
				}
			}
		}
	}

	// Convert to slice and filter out operators
	var tags []string
	for tag := range tagSet {
		// Filter out operators and empty strings
		if tag != "" && tag != "!" && tag != "||" && tag != "&&" && tag != "(" && tag != ")" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// extractTagsFromConstraint extracts build tags from //go:build constraint
func extractTagsFromConstraint(line string) []string {
	// Remove //go:build prefix
	constraint := strings.TrimSpace(strings.TrimPrefix(line, "//go:build"))

	// Split on operators and whitespace
	var tags []string
	words := strings.FieldsFunc(constraint, func(r rune) bool {
		return r == '(' || r == ')' || r == '&' || r == '|' || r == '!' || r == ' ' || r == '\t'
	})

	for _, word := range words {
		word = strings.TrimSpace(word)
		if word != "" && word != "&&" && word != "||" {
			tags = append(tags, word)
		}
	}

	return tags
}

// extractTagsFromLegacyConstraint extracts build tags from // +build constraint
func extractTagsFromLegacyConstraint(line string) []string {
	// Remove // +build prefix
	constraint := strings.TrimSpace(strings.TrimPrefix(line, "// +build"))

	// Split on whitespace and commas
	var tags []string
	words := strings.FieldsFunc(constraint, func(r rune) bool {
		return r == ' ' || r == '\t' || r == ','
	})

	for _, word := range words {
		word = strings.TrimSpace(word)
		// Remove ! prefix for negative constraints
		word = strings.TrimPrefix(word, "!")
		if word != "" {
			tags = append(tags, word)
		}
	}

	return tags
}

// isGoModule checks if a directory contains a go.mod file, indicating it's a Go module
func isGoModule(dir string) bool {
	goModPath := filepath.Join(dir, "go.mod")
	_, err := os.Stat(goModPath)
	return err == nil
}

// findGoModuleRoot finds the nearest Go module root by walking up the directory tree from targetDir
// Returns the module root path, or empty string if no module is found within repoRoot
func findGoModuleRoot(targetDir, repoRoot string) string {
	current := targetDir

	for {
		// Check if current directory contains go.mod
		if isGoModule(current) {
			return current
		}

		// Move up one directory
		parent := filepath.Dir(current)

		// Stop if we've reached the repo root or filesystem root
		if parent == current || !strings.HasPrefix(parent, repoRoot) {
			break
		}

		current = parent
	}

	return ""
}

// installGolangciLint installs golangci-lint using the official installation script
// installGolangciLint is no longer needed - handled by tools.EnsureInstalled
