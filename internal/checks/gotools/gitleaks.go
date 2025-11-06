package gotools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mrz1836/go-pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/shared"
	"github.com/mrz1836/go-pre-commit/internal/tools"
)

// GitleaksCheck runs gitleaks to scan for secrets and credentials
type GitleaksCheck struct {
	sharedCtx *shared.Context
	timeout   time.Duration
	config    *config.Config
}

// NewGitleaksCheck creates a new gitleaks check
func NewGitleaksCheck() *GitleaksCheck {
	return &GitleaksCheck{
		sharedCtx: shared.NewContext(),
		timeout:   60 * time.Second, // 60 second timeout for gitleaks
		config:    nil,
	}
}

// NewGitleaksCheckWithSharedContext creates a new gitleaks check with shared context
func NewGitleaksCheckWithSharedContext(sharedCtx *shared.Context) *GitleaksCheck {
	return &GitleaksCheck{
		sharedCtx: sharedCtx,
		timeout:   60 * time.Second,
		config:    nil,
	}
}

// NewGitleaksCheckWithConfig creates a new gitleaks check with shared context and custom timeout
func NewGitleaksCheckWithConfig(sharedCtx *shared.Context, timeout time.Duration) *GitleaksCheck {
	return &GitleaksCheck{
		sharedCtx: sharedCtx,
		timeout:   timeout,
		config:    nil,
	}
}

// NewGitleaksCheckWithFullConfig creates a new gitleaks check with full configuration
func NewGitleaksCheckWithFullConfig(sharedCtx *shared.Context, cfg *config.Config) *GitleaksCheck {
	timeout := 60 * time.Second

	if cfg != nil {
		timeout = time.Duration(cfg.CheckTimeouts.Gitleaks) * time.Second
	}

	return &GitleaksCheck{
		sharedCtx: sharedCtx,
		timeout:   timeout,
		config:    cfg,
	}
}

// Name returns the name of the check
func (c *GitleaksCheck) Name() string {
	return "gitleaks"
}

// Description returns a brief description of the check
func (c *GitleaksCheck) Description() string {
	return "Scan for secrets and credentials in code"
}

// Metadata returns comprehensive metadata about the check
func (c *GitleaksCheck) Metadata() interface{} {
	return CheckMetadata{
		Name:              "gitleaks",
		Description:       "Scan for secrets and credentials in code using gitleaks",
		FilePatterns:      []string{"*"}, // Scan all files for secrets
		EstimatedDuration: 5 * time.Second,
		Dependencies:      []string{"gitleaks"}, // Auto-installed via binary download
		DefaultTimeout:    c.timeout,
		Category:          "security",
		RequiresFiles:     true,
	}
}

// Run executes the gitleaks check
func (c *GitleaksCheck) Run(ctx context.Context, files []string) error {
	// Early return if no files to process
	if len(files) == 0 {
		return nil
	}

	// Ensure gitleaks is installed (auto-install if needed)
	if err := tools.EnsureInstalled(ctx, "gitleaks"); err != nil {
		return prerrors.NewToolExecutionError(
			"gitleaks",
			err.Error(),
			"Failed to install gitleaks. You can install it manually from: https://github.com/gitleaks/gitleaks#installation",
		)
	}

	// Run gitleaks
	return c.runGitleaks(ctx, files)
}

// FilterFiles returns all files (gitleaks scans all file types for secrets)
func (c *GitleaksCheck) FilterFiles(files []string) []string {
	// Return all files - gitleaks should scan everything for secrets
	return files
}

// runGitleaks runs gitleaks on the repository
func (c *GitleaksCheck) runGitleaks(ctx context.Context, _ []string) error {
	repoRoot, err := c.sharedCtx.GetRepoRoot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find repository root: %w", err)
	}

	// Add timeout for gitleaks command
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Build gitleaks command arguments
	// Use "detect" mode with --no-git to scan files directly
	args := []string{"detect", "--no-git", "--source", repoRoot, "--verbose"}

	// Look for custom config file
	configPath := c.findGitleaksConfig(repoRoot)
	if configPath != "" {
		args = append(args, "--config", configPath)
	}

	cmd := exec.CommandContext(ctx, "gitleaks", args...) //nolint:gosec // Command arguments are validated
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stdout.String() + stderr.String()

		// Check if it's a context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolExecutionError(
				"gitleaks",
				output,
				fmt.Sprintf("Gitleaks timed out after %v. Consider increasing GO_PRE_COMMIT_GITLEAKS_TIMEOUT.", c.timeout),
			)
		}

		// Check if secrets were found (exit code 1)
		// Gitleaks returns exit code 1 when secrets are found
		if strings.Contains(output, "leaks found") || strings.Contains(output, "Finding:") {
			formattedOutput := c.formatGitleaksErrors(output)
			return &prerrors.CheckError{
				Err:        prerrors.ErrSecretsFound,
				Message:    formattedOutput,
				Suggestion: "Remove secrets from code or add exceptions to .gitleaks.toml allowlist",
				Command:    "gitleaks detect",
				Output:     formattedOutput,
			}
		}

		// Check for config file errors
		if strings.Contains(output, "config") && (strings.Contains(output, "error") || strings.Contains(output, "invalid")) {
			return prerrors.NewToolExecutionError(
				"gitleaks",
				output,
				"Fix gitleaks configuration issues. Check your .gitleaks.toml file syntax.",
			)
		}

		// Generic failure
		return prerrors.NewToolExecutionError(
			"gitleaks",
			output,
			"Run 'gitleaks detect --verbose' manually to see detailed error output.",
		)
	}

	return nil
}

// findGitleaksConfig searches for .gitleaks.toml in standard locations
// Priority: 1) root directory, 2) .github directory
func (c *GitleaksCheck) findGitleaksConfig(repoRoot string) string {
	// Check if user specified custom config path via environment
	if c.config != nil {
		if customPath := os.Getenv("GO_PRE_COMMIT_GITLEAKS_CONFIG"); customPath != "" {
			absPath := customPath
			if !filepath.IsAbs(customPath) {
				absPath = filepath.Join(repoRoot, customPath)
			}
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Try root directory first
	rootConfig := filepath.Join(repoRoot, ".gitleaks.toml")
	if _, err := os.Stat(rootConfig); err == nil {
		return rootConfig
	}

	// Try .github folder
	githubConfig := filepath.Join(repoRoot, ".github", ".gitleaks.toml")
	if _, err := os.Stat(githubConfig); err == nil {
		return githubConfig
	}

	// No custom config found - gitleaks will use its defaults
	return ""
}

// formatGitleaksErrors extracts and formats secret findings for clearer display
func (c *GitleaksCheck) formatGitleaksErrors(output string) string {
	var result strings.Builder
	lines := strings.Split(output, "\n")
	findingCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for finding markers
		// Gitleaks output includes lines like: "Finding: <description>"
		if strings.Contains(line, "Finding:") || strings.Contains(line, "Secret:") ||
			strings.Contains(line, "File:") || strings.Contains(line, "Line:") {
			if findingCount > 0 && strings.Contains(line, "Finding:") {
				result.WriteString("\n")
			}
			result.WriteString(line)
			result.WriteString("\n")
			if strings.Contains(line, "Finding:") {
				findingCount++
			}
		}
	}

	// If we found specific errors, return them with header
	if findingCount > 0 {
		header := fmt.Sprintf("Found %d secret(s):\n", findingCount)
		return header + result.String()
	}

	// Otherwise return the original output
	return output
}
