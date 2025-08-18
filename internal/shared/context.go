// Package shared provides shared context and caching for pre-commit checks
package shared

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Context provides cached repository information
type Context struct {
	repoRoot     string
	repoRootOnce sync.Once
	repoRootErr  error
}

// NewContext creates a new shared context for checks
func NewContext() *Context {
	return &Context{}
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
