package gotools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mrz1836/go-pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/shared"
)

// Static errors for linting compliance
var (
	// ErrModTidyDiffNotSupported is returned when go mod tidy -diff flag is not supported
	ErrModTidyDiffNotSupported = errors.New("go mod tidy -diff not supported")

	// ErrModTidyDiffFailed is returned when go mod tidy -diff command fails
	ErrModTidyDiffFailed = errors.New("go mod tidy -diff failed")
)

// ModTidyCheck ensures go.mod and go.sum are tidy
type ModTidyCheck struct {
	sharedCtx *shared.Context
	config    *config.Config
	timeout   time.Duration
}

// NewModTidyCheck creates a new mod tidy check
func NewModTidyCheck() *ModTidyCheck {
	return &ModTidyCheck{
		sharedCtx: shared.NewContext(),
		config:    nil,              // Config not available in basic constructor
		timeout:   30 * time.Second, // 30 second timeout for mod tidy
	}
}

// NewModTidyCheckWithSharedContext creates a new mod tidy check with shared context
func NewModTidyCheckWithSharedContext(sharedCtx *shared.Context) *ModTidyCheck {
	return &ModTidyCheck{
		sharedCtx: sharedCtx,
		config:    nil, // Config not available in this constructor
		timeout:   30 * time.Second,
	}
}

// NewModTidyCheckWithConfig creates a new mod tidy check with shared context and custom timeout
func NewModTidyCheckWithConfig(sharedCtx *shared.Context, cfg *config.Config, timeout time.Duration) *ModTidyCheck {
	return &ModTidyCheck{
		sharedCtx: sharedCtx,
		config:    cfg,
		timeout:   timeout,
	}
}

// Name returns the name of the check
func (c *ModTidyCheck) Name() string {
	return "mod-tidy"
}

// Description returns a brief description of the check
func (c *ModTidyCheck) Description() string {
	return "Ensure go.mod and go.sum are tidy"
}

// Metadata returns comprehensive metadata about the check
func (c *ModTidyCheck) Metadata() interface{} {
	return CheckMetadata{
		Name:              "mod-tidy",
		Description:       "Ensure go.mod and go.sum are up to date and tidy",
		FilePatterns:      []string{"*.go", "go.mod", "go.sum"},
		EstimatedDuration: 5 * time.Second,
		Dependencies:      []string{"mod-tidy"}, // tool or build target
		DefaultTimeout:    c.timeout,
		Category:          "dependencies",
		RequiresFiles:     false, // Can run even with no staged files
	}
}

// Run executes the mod tidy check
func (c *ModTidyCheck) Run(ctx context.Context, files []string) error {
	// Early return if no files to process
	if len(files) == 0 {
		return nil
	}

	// Run go mod tidy directly (no tools to install, it's built into go)
	return c.runDirectModTidy(ctx, files)
}

// FilterFiles filters to only go.mod and go.sum files or when .go files change
func (c *ModTidyCheck) FilterFiles(files []string) []string {
	var hasGoMod, hasGoFiles bool
	var filtered []string

	for _, file := range files {
		// Check for go.mod/go.sum changes
		if file == "go.mod" || file == "go.sum" || strings.HasSuffix(file, "/go.mod") || strings.HasSuffix(file, "/go.sum") {
			hasGoMod = true
			filtered = append(filtered, file)
		}
		// Check for .go file changes
		if strings.HasSuffix(file, ".go") {
			hasGoFiles = true
		}
	}

	// If we have go.mod/go.sum changes, run on those
	if hasGoMod {
		return filtered
	}

	// If we have .go files but no go.mod/go.sum changes, still run mod-tidy
	// because imports might have changed. Return the actual .go files so
	// findGoModuleRoot can locate the correct module for each file.
	if hasGoFiles {
		var goFiles []string
		for _, file := range files {
			if strings.HasSuffix(file, ".go") {
				goFiles = append(goFiles, file)
			}
		}
		return goFiles
	}

	// No relevant files
	return []string{}
}

// runDirectModTidy runs go mod tidy directly on modules
func (c *ModTidyCheck) runDirectModTidy(ctx context.Context, files []string) error {
	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Group files by their module directory
	modulesByDir := make(map[string][]string)
	for _, file := range files {
		dir := filepath.Dir(file)
		// Find the Go module root for this file
		moduleRoot := findGoModuleRoot(filepath.Join(repoRoot, dir), repoRoot)
		if moduleRoot == "" {
			// Try to check if the file is go.mod or go.sum itself
			if strings.HasSuffix(file, "go.mod") || strings.HasSuffix(file, "go.sum") {
				moduleRoot = filepath.Join(repoRoot, dir)
			}
		}
		if moduleRoot != "" {
			modulesByDir[moduleRoot] = append(modulesByDir[moduleRoot], file)
		}
	}

	// If no modules found, use GO_SUM_FILE to determine primary module location
	if len(modulesByDir) == 0 {
		// Determine the module directory from GO_SUM_FILE configuration
		var moduleDir string
		if c.config != nil {
			moduleDirFromConfig := c.config.GetModuleDir()
			if moduleDirFromConfig != "" {
				moduleDir = filepath.Join(repoRoot, moduleDirFromConfig)
			} else {
				moduleDir = repoRoot
			}
		} else {
			// Fallback to repo root if config is not available
			moduleDir = repoRoot
		}

		// Only run if the directory contains a go.mod file
		if isGoModule(moduleDir) {
			return c.runModTidyOnModule(ctx, moduleDir, repoRoot)
		}

		// No Go modules found - skip gracefully (not an error)
		return nil
	}

	// Run mod tidy on each module
	var allErrors []string
	for moduleDir := range modulesByDir {
		err := c.runModTidyOnModule(ctx, moduleDir, repoRoot)
		if err != nil {
			relPath, _ := filepath.Rel(repoRoot, moduleDir)
			if relPath == "" {
				relPath = "."
			}

			// Check if this is a CheckError with detailed output
			var checkErr *prerrors.CheckError
			if errors.As(err, &checkErr) && checkErr.Output != "" {
				// Include the detailed output in a structured format
				errorMsg := fmt.Sprintf("Module %s needs tidying:\n%s", relPath, checkErr.Output)
				allErrors = append(allErrors, errorMsg)
			} else {
				// Fallback to simple format for other error types
				allErrors = append(allErrors, fmt.Sprintf("Module %s: %v", relPath, err))
			}
		}
	}

	// If there were any errors, aggregate and return them
	if len(allErrors) > 0 {
		combinedErrors := strings.Join(allErrors, "\n\n")
		return prerrors.NewToolExecutionError(
			"go mod tidy (in one or more modules)",
			combinedErrors,
			"Fix the mod tidy issues shown above. Run 'go mod tidy' in each module directory.",
		)
	}

	return nil
}

// runModTidyOnModule runs go mod tidy on a specific module directory
func (c *ModTidyCheck) runModTidyOnModule(ctx context.Context, moduleDir, repoRoot string) error {
	// Calculate relative path for display
	relPath, _ := filepath.Rel(repoRoot, moduleDir)
	if relPath == "" {
		relPath = "."
	}

	// Try to use go mod tidy -diff first (Go 1.21+)
	diffErr := c.checkModTidyDiff(ctx, moduleDir, repoRoot)
	if diffErr != nil {
		// Check if it's because -diff is not supported
		if !strings.Contains(diffErr.Error(), "not supported") {
			// -diff is supported but found issues, return the error
			return diffErr
		}
		// -diff not supported, fall back to old method
	} else {
		// -diff succeeded, no changes needed
		return nil
	}

	// Fall back to running go mod tidy and checking for changes
	// Add timeout for go mod tidy command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = moduleDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stdout.String() + stderr.String()

		// Check if it's a context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolExecutionError(
				fmt.Sprintf("go mod tidy (in %s)", relPath),
				output,
				fmt.Sprintf("Mod tidy timed out after %v in module '%s'. Consider increasing GO_PRE_COMMIT_MOD_TIDY_TIMEOUT.", c.timeout, relPath),
			)
		}

		if strings.Contains(output, "no go.mod file") {
			return prerrors.NewToolExecutionError(
				fmt.Sprintf("go mod tidy (in %s)", relPath),
				output,
				fmt.Sprintf("No go.mod file found in module '%s'. Initialize a Go module with 'go mod init <module-name>'.", relPath),
			)
		}

		if strings.Contains(output, "network") || strings.Contains(output, "timeout") {
			return prerrors.NewToolExecutionError(
				fmt.Sprintf("go mod tidy (in %s)", relPath),
				output,
				fmt.Sprintf("Network error downloading modules in '%s'. Check your internet connection and proxy settings.", relPath),
			)
		}

		if strings.Contains(output, "checksum mismatch") {
			return prerrors.NewToolExecutionError(
				fmt.Sprintf("go mod tidy (in %s)", relPath),
				output,
				fmt.Sprintf("Module checksum verification failed in '%s'. Run 'go clean -modcache' and try again.", relPath),
			)
		}

		if strings.Contains(output, "not found") {
			return prerrors.NewToolExecutionError(
				fmt.Sprintf("go mod tidy (in %s)", relPath),
				output,
				fmt.Sprintf("Module dependencies not found in '%s'. Check that all imported modules exist and are accessible.", relPath),
			)
		}

		// Generic failure
		return prerrors.NewToolExecutionError(
			fmt.Sprintf("go mod tidy (in %s)", relPath),
			output,
			fmt.Sprintf("Run 'cd %s && go mod tidy' manually to see detailed error output.", relPath),
		)
	}

	// Check if there are uncommitted changes
	return c.checkUncommittedChanges(ctx, moduleDir, repoRoot)
}

// checkModTidyDiff uses go mod tidy -diff to check if changes would be made (Go 1.21+)
func (c *ModTidyCheck) checkModTidyDiff(ctx context.Context, moduleDir, repoRoot string) error {
	// Calculate relative path for display
	relPath, _ := filepath.Rel(repoRoot, moduleDir)
	if relPath == "" {
		relPath = "."
	}

	// Add timeout for go mod tidy -diff command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "mod", "tidy", "-diff")
	cmd.Dir = moduleDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stderr.String()

		// Check if -diff flag is not supported (older Go versions)
		if strings.Contains(output, "unknown flag") || strings.Contains(output, "flag provided but not defined") {
			// Return an error to indicate we should fall back to the old method
			return ErrModTidyDiffNotSupported
		}

		// Check if it's a context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolExecutionError(
				fmt.Sprintf("go mod tidy -diff (in %s)", relPath),
				output,
				fmt.Sprintf("Mod tidy check timed out after %v in module '%s'. Consider increasing GO_PRE_COMMIT_MOD_TIDY_TIMEOUT.", c.timeout, relPath),
			)
		}

		// Handle other errors that are not diff-related
		if strings.Contains(output, "no go.mod file") {
			return prerrors.NewToolExecutionError(
				fmt.Sprintf("go mod tidy -diff (in %s)", relPath),
				output,
				fmt.Sprintf("No go.mod file found in module '%s'. Initialize a Go module with 'go mod init <module-name>'.", relPath),
			)
		}

		if strings.Contains(output, "network") || strings.Contains(output, "timeout") {
			return prerrors.NewToolExecutionError(
				fmt.Sprintf("go mod tidy -diff (in %s)", relPath),
				output,
				fmt.Sprintf("Network error downloading modules in '%s'. Check your internet connection and proxy settings.", relPath),
			)
		}

		// For go mod tidy -diff, exit code 1 with diff output in stdout is expected
		// when changes need to be made. Only treat it as an error if no diff output.
		diffOutput := stdout.String()
		if diffOutput == "" {
			// No diff output but command failed - this is a real error
			return prerrors.NewToolExecutionError(
				fmt.Sprintf("go mod tidy -diff (in %s)", relPath),
				output,
				fmt.Sprintf("Module dependencies check failed in '%s'. See error details above.", relPath),
			)
		}
		// This is the expected case - go mod tidy -diff found differences
		// Continue to process the diff output below
	}

	// If there's any diff output (excluding warnings), it means changes would be made
	if diffOutput := stdout.String(); diffOutput != "" {
		// Filter out go warnings which are not actual diffs
		lines := strings.Split(diffOutput, "\n")
		var actualDiffs []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// Skip empty lines and go warnings
			if line == "" || strings.HasPrefix(line, "go: warning:") {
				continue
			}
			actualDiffs = append(actualDiffs, line)
		}

		// If there are actual diffs (not just warnings), return error with detailed diff
		if len(actualDiffs) > 0 {
			diffDetails := strings.Join(actualDiffs, "\n")
			return prerrors.NewToolExecutionError(
				fmt.Sprintf("go mod tidy -diff (in %s)", relPath),
				diffDetails,
				fmt.Sprintf("go.mod or go.sum are not tidy in module '%s'. Run 'cd %s && go mod tidy' to update dependencies.", relPath, relPath),
			)
		}
	}

	return nil
}

// checkUncommittedChanges checks if go mod tidy made any changes
// This is a fallback method when go mod tidy -diff is not available
func (c *ModTidyCheck) checkUncommittedChanges(ctx context.Context, moduleDir, repoRoot string) error {
	// Calculate relative path for display
	relPath, _ := filepath.Rel(repoRoot, moduleDir)
	if relPath == "" {
		relPath = "."
	}

	// Add short timeout for git diff
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get relative paths to go.mod and go.sum from the module directory
	goModPath := filepath.Join(moduleDir, "go.mod")
	goSumPath := filepath.Join(moduleDir, "go.sum")

	// Check for new untracked files (like go.sum created for the first time)
	statusCmd := exec.CommandContext(ctx, "git", "status", "--porcelain", goModPath, goSumPath) //nolint:gosec // paths are controlled
	statusCmd.Dir = moduleDir

	var statusOutput bytes.Buffer
	statusCmd.Stdout = &statusOutput
	statusCmd.Stderr = &statusOutput

	if err := statusCmd.Run(); err != nil {
		return fmt.Errorf("failed to check git status in module '%s': %w", relPath, err)
	}

	// If there are any changes or new files, that's an error
	if statusOutput.Len() > 0 {
		return prerrors.NewToolExecutionError(
			fmt.Sprintf("git status (in %s)", relPath),
			statusOutput.String(),
			fmt.Sprintf("go.mod or go.sum were modified by 'go mod tidy' in module '%s'. Commit these changes to proceed. Run 'cd %s && git add go.mod go.sum && git commit -m \"Update module dependencies\"'.", relPath, relPath),
		)
	}

	return nil
}
