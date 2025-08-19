// Package gotools provides pre-commit checks that run Go tools directly
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
	"github.com/mrz1836/go-pre-commit/internal/tools"
)

// FumptCheck runs gofumpt directly or via build tools
type FumptCheck struct {
	sharedCtx *shared.Context
	timeout   time.Duration
	config    *config.Config
	autoStage bool
}

// NewFumptCheck creates a new fumpt check
func NewFumptCheck() *FumptCheck {
	return &FumptCheck{
		sharedCtx: shared.NewContext(),
		timeout:   30 * time.Second, // 30 second timeout for fumpt
		config:    nil,
		autoStage: false,
	}
}

// NewFumptCheckWithSharedContext creates a new fumpt check with shared context
func NewFumptCheckWithSharedContext(sharedCtx *shared.Context) *FumptCheck {
	return &FumptCheck{
		sharedCtx: sharedCtx,
		timeout:   30 * time.Second,
		config:    nil,
		autoStage: false,
	}
}

// NewFumptCheckWithConfig creates a new fumpt check with shared context and custom timeout
func NewFumptCheckWithConfig(sharedCtx *shared.Context, timeout time.Duration) *FumptCheck {
	return &FumptCheck{
		sharedCtx: sharedCtx,
		timeout:   timeout,
		config:    nil,
		autoStage: false,
	}
}

// NewFumptCheckWithFullConfig creates a new fumpt check with full configuration including auto-stage
func NewFumptCheckWithFullConfig(sharedCtx *shared.Context, cfg *config.Config) *FumptCheck {
	timeout := 30 * time.Second
	autoStage := false

	if cfg != nil {
		timeout = time.Duration(cfg.CheckTimeouts.Fumpt) * time.Second
		autoStage = cfg.CheckBehaviors.FumptAutoStage
	}

	return &FumptCheck{
		sharedCtx: sharedCtx,
		timeout:   timeout,
		config:    cfg,
		autoStage: autoStage,
	}
}

// Name returns the name of the check
func (c *FumptCheck) Name() string {
	return "fumpt"
}

// Description returns a brief description of the check
func (c *FumptCheck) Description() string {
	return "Format Go code with gofumpt"
}

// Metadata returns comprehensive metadata about the check
func (c *FumptCheck) Metadata() interface{} {
	return CheckMetadata{
		Name:              "fumpt",
		Description:       "Format Go code with gofumpt (stricter gofmt)",
		FilePatterns:      []string{"*.go"},
		EstimatedDuration: 3 * time.Second,
		Dependencies:      []string{"fumpt"}, // tool or build target
		DefaultTimeout:    c.timeout,
		Category:          "formatting",
		RequiresFiles:     true,
	}
}

// Run executes the fumpt check
func (c *FumptCheck) Run(ctx context.Context, files []string) error {
	// Early return if no files to process
	if len(files) == 0 {
		return nil
	}

	// Get list of files before formatting for auto-stage detection
	var modifiedFiles []string
	if c.autoStage {
		modifiedFiles = files // Track the files we're formatting
	}

	// Ensure gofumpt is installed
	if err := tools.EnsureInstalled(ctx, "gofumpt"); err != nil {
		return prerrors.NewToolExecutionError(
			"gofumpt",
			err.Error(),
			"Failed to install gofumpt. You can install it manually with: go install mvdan.cc/gofumpt@latest",
		)
	}

	// Run gofumpt directly
	err := c.runDirectFumpt(ctx, files)

	// If formatting succeeded and auto-stage is enabled, stage the modified files
	if err == nil && c.autoStage && len(modifiedFiles) > 0 {
		if stageErr := c.stageFiles(ctx, modifiedFiles); stageErr != nil {
			// Log warning but don't fail the check
			// The formatting was successful, staging is a convenience feature
			return fmt.Errorf("formatting completed but auto-staging failed: %w", stageErr)
		}
	}

	return err
}

// FilterFiles filters to only Go files
func (c *FumptCheck) FilterFiles(files []string) []string {
	var filtered []string
	for _, file := range files {
		if strings.HasSuffix(file, ".go") {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// runDirectFumpt runs gofumpt directly on files
func (c *FumptCheck) runDirectFumpt(ctx context.Context, files []string) error {
	// Tool installation is already handled in Run(), so we can proceed directly

	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Build absolute paths
	absFiles := make([]string, len(files))
	for i, file := range files {
		absFiles[i] = filepath.Join(repoRoot, file)
	}

	// Add timeout for gofumpt command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Run gofumpt
	args := append([]string{"-w"}, absFiles...)
	cmd := exec.CommandContext(ctx, "gofumpt", args...) //nolint:gosec // Command arguments are validated
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stdout.String() + stderr.String()

		// Check if it's a context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolExecutionError(
				"gofumpt",
				output,
				fmt.Sprintf("Fumpt timed out after %v. Consider running on fewer files or increasing GO_PRE_COMMIT_FUMPT_TIMEOUT.", c.timeout),
			)
		}

		if strings.Contains(output, "permission denied") {
			return prerrors.NewToolExecutionError(
				"gofumpt",
				output,
				"Permission denied. Check file permissions and ensure you have write access to all Go files.",
			)
		}

		if strings.Contains(output, "syntax error") || strings.Contains(output, "invalid Go syntax") {
			return prerrors.NewToolExecutionError(
				"gofumpt",
				output,
				"Go syntax errors prevent formatting. Fix syntax errors in your Go files before running fumpt.",
			)
		}

		// Generic failure
		return prerrors.NewToolExecutionError(
			"gofumpt",
			output,
			"Run 'gofumpt -w <files>' manually to see detailed error output.",
		)
	}

	return nil
}

// stageFiles adds modified files to git staging area
func (c *FumptCheck) stageFiles(ctx context.Context, files []string) error {
	if ctx == nil {
		return prerrors.ErrNilContext
	}

	if len(files) == 0 {
		return nil
	}

	// Build git add command with all modified files
	args := append([]string{"add"}, files...)
	cmd := exec.CommandContext(ctx, "git", args...) //nolint:gosec // Command arguments are controlled

	// Get repository root to run git command from correct location
	if repoRoot, err := c.sharedCtx.GetRepoRoot(ctx); err == nil {
		cmd.Dir = repoRoot
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w (stderr: %s)", err, stderr.String())
	}

	return nil
}
