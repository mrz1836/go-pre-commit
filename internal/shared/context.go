// Package shared provides shared context and caching for pre-commit checks
package shared

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// BuildTargetInfo contains information about a build target
type BuildTargetInfo struct {
	Name        string
	Exists      bool
	Description string
	Error       error
	LastChecked time.Time
}

var (
	// ErrMagexNotFound indicates magex is not installed or not found
	ErrMagexNotFound = errors.New("magex not found or not installed")
	// ErrMagexTargetTimeout indicates timeout checking magex target
	ErrMagexTargetTimeout = errors.New("timeout checking magex target")
)

// Context provides cached repository information and build target availability
type Context struct {
	repoRoot          string
	buildTargets      map[string]*BuildTargetInfo
	buildTargetsMutex sync.RWMutex
	repoRootOnce      sync.Once
	repoRootErr       error
}

// NewContext creates a new shared context for checks
func NewContext() *Context {
	return &Context{
		buildTargets: make(map[string]*BuildTargetInfo),
	}
}

// GetRepoRoot returns the repository root, caching the result
func (sc *Context) GetRepoRoot(ctx context.Context) (string, error) {
	sc.repoRootOnce.Do(func() {
		// Add timeout for git command
		timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(timeoutCtx, "git", "rev-parse", "--show-toplevel")
		output, err := cmd.Output()
		if err != nil {
			sc.repoRootErr = err
			return
		}
		sc.repoRoot = strings.TrimSpace(string(output))
	})

	return sc.repoRoot, sc.repoRootErr
}

// HasMagexTarget checks if a magex target exists, with caching
func (sc *Context) HasMagexTarget(ctx context.Context, target string) bool {
	info := sc.GetBuildTargetInfo(ctx, target)
	return info.Exists
}

// GetBuildTargetInfo gets detailed information about a build target
func (sc *Context) GetBuildTargetInfo(ctx context.Context, target string) *BuildTargetInfo {
	// Check cache first with TTL (5 minutes)
	sc.buildTargetsMutex.RLock()
	if info, exists := sc.buildTargets[target]; exists {
		// Check if cache entry is still valid (5 minutes)
		if time.Since(info.LastChecked) < 5*time.Minute {
			sc.buildTargetsMutex.RUnlock()
			return info
		}
	}
	sc.buildTargetsMutex.RUnlock()

	// Create new target info
	info := &BuildTargetInfo{
		Name:        target,
		LastChecked: time.Now(),
	}

	// Get repository root
	repoRoot, err := sc.GetRepoRoot(ctx)
	if err != nil {
		info.Error = fmt.Errorf("failed to find repository root: %w", err)
		sc.cacheBuildTargetInfo(target, info)
		return info
	}

	// Add timeout for build command
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Check if magex target exists by listing available commands
	cmd := exec.CommandContext(ctx, "magex", "help")
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := stdout.String() + stderr.String()

	if err == nil {
		// Check if the target exists in the magex help output
		info.Exists = strings.Contains(output, target)
		if info.Exists {
			// Try to extract description from magex help output
			info.Description = sc.extractMagexTargetDescription(output, target)
		}
	} else {
		info.Exists = false
		// Provide helpful error information
		if strings.Contains(output, "command not found") {
			info.Error = ErrMagexNotFound
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			info.Error = fmt.Errorf("%w '%s'", ErrMagexTargetTimeout, target)
		} else {
			info.Error = fmt.Errorf("error checking magex target '%s': %w", target, err)
		}
	}

	// Cache the result
	sc.cacheBuildTargetInfo(target, info)
	return info
}

// cacheBuildTargetInfo safely caches build target information
func (sc *Context) cacheBuildTargetInfo(target string, info *BuildTargetInfo) {
	sc.buildTargetsMutex.Lock()
	sc.buildTargets[target] = info
	sc.buildTargetsMutex.Unlock()
}

// extractTargetDescription tries to extract a description for a build target
func (sc *Context) extractTargetDescription(ctx context.Context, repoRoot, target string) string {
	// Check if the parent context is already canceled/timed out
	if ctx.Err() != nil {
		return ""
	}

	// If parent context has very little time left (less than 50ms), don't even try
	if parentDeadline, ok := ctx.Deadline(); ok {
		if time.Until(parentDeadline) < 50*time.Millisecond {
			return ""
		}
	}

	// Try to get help information
	// Use the shorter of the parent context deadline or 2 seconds
	deadline := 2 * time.Second
	if parentDeadline, ok := ctx.Deadline(); ok {
		if time.Until(parentDeadline) < deadline {
			deadline = time.Until(parentDeadline)
		}
	}
	helpCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	cmd := exec.CommandContext(helpCtx, "make", "help")
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err == nil && helpCtx.Err() == nil {
		// Successfully got help output, try to parse it
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, target) {
				// Extract description after target name
				parts := strings.SplitN(line, target, 2)
				if len(parts) > 1 {
					desc := strings.TrimSpace(parts[1])
					desc = strings.TrimLeft(desc, ":-")
					desc = strings.TrimSpace(desc)
					if desc != "" {
						return desc
					}
				}
			}
		}

		// Help succeeded but no description found for target
		// Only use fallbacks if help succeeded (meaning help target exists)
		commonTargets := map[string]string{
			"fumpt":     "Format Go code with gofumpt",
			"lint":      "Run golangci-lint on Go code",
			"mod-tidy":  "Tidy Go module dependencies",
			"test":      "Run tests",
			"build":     "Build the project",
			"clean":     "Clean build artifacts",
			"install":   "Install dependencies",
			"help":      "Show help information",
			"format":    "Format source code",
			"check":     "Run checks",
			"validate":  "Validate code",
			"generate":  "Generate code",
			"docs":      "Generate documentation",
			"coverage":  "Generate test coverage",
			"benchmark": "Run benchmarks",
		}

		if desc, exists := commonTargets[target]; exists {
			// Check if the original context has timed out before using fallback
			if ctx.Err() != nil {
				return ""
			}
			return desc
		}
	}

	// Help failed (no help target, timeout, or other error) - return empty
	return ""
}

// extractMagexTargetDescription extracts description for a magex target from help output
func (sc *Context) extractMagexTargetDescription(helpOutput, target string) string {
	lines := strings.Split(helpOutput, "\n")
	for _, line := range lines {
		if strings.Contains(line, target) {
			// Extract description from magex help output format
			parts := strings.Fields(line)
			if len(parts) > 1 && parts[0] == target {
				desc := strings.Join(parts[1:], " ")
				desc = strings.TrimSpace(desc)
				if desc != "" {
					return desc
				}
			}
		}
	}

	// Fallback descriptions for common magex targets
	commonTargets := map[string]string{
		"format":   "Format Go code with gofumpt",
		"lint":     "Run golangci-lint on Go code",
		"mod:tidy": "Tidy Go module dependencies",
		"test":     "Run tests",
		"build":    "Build the project",
		"clean":    "Clean build artifacts",
		"install":  "Install dependencies",
		"help":     "Show help information",
	}

	if desc, exists := commonTargets[target]; exists {
		return desc
	}

	return ""
}

// GetAvailableBuildTargets returns all available build targets
func (sc *Context) GetAvailableBuildTargets(ctx context.Context) ([]string, error) {
	repoRoot, err := sc.GetRepoRoot(ctx)
	if err != nil {
		// Check if it's a timeout vs not a git repo
		if errors.Is(ctx.Err(), context.DeadlineExceeded) ||
			strings.Contains(err.Error(), "context deadline exceeded") {
			// Timeout case - return fallback targets
			return []string{"help", "build", "test", "clean", "install"}, nil
		}
		// Not a git repo or other error - return the error
		return nil, fmt.Errorf("failed to find repository root: %w", err)
	}

	// Add timeout for build command
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to list all targets
	cmd := exec.CommandContext(ctx, "make", "-qp")
	cmd.Dir = repoRoot

	output, err := cmd.Output()
	if err != nil {
		// Fallback to common targets if make -qp fails
		return []string{"help", "build", "test", "clean", "install"}, nil
	}

	targets := sc.parseBuildTargets(string(output))
	return targets, nil
}

// parseBuildTargets extracts target names from make -qp output
func (sc *Context) parseBuildTargets(output string) []string {
	var targets []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines, comments, and variable assignments
		if line == "" || strings.HasPrefix(line, "#") || strings.Contains(line, "=") {
			continue
		}

		// Look for target definitions (lines ending with :)
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "\t") {
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				target := strings.TrimSpace(parts[0])

				// Skip internal/automatic targets
				if !strings.HasPrefix(target, ".") &&
					!strings.Contains(target, "/") &&
					!strings.Contains(target, "%") &&
					target != "" {
					targets = append(targets, target)
				}
			}
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, target := range targets {
		if !seen[target] {
			seen[target] = true
			unique = append(unique, target)
		}
	}

	return unique
}

// ExecuteBuildTarget executes a build target with proper timeout
func (sc *Context) ExecuteBuildTarget(ctx context.Context, target string, timeout time.Duration) error {
	repoRoot, err := sc.GetRepoRoot(ctx)
	if err != nil {
		return err
	}

	// Add timeout for build command
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "make", target)
	cmd.Dir = repoRoot
	return cmd.Run()
}
