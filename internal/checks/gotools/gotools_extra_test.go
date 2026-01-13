package gotools

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/shared"
)

func TestFumptCheck_Run_ContextCancelled_Extra(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	check := NewFumptCheck()
	err := check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
}

func TestLintCheck_Run_ContextCancelled_Extra(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	check := NewLintCheck()
	err := check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
}

func TestModTidyCheck_Run_ContextCancelled_Extra(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	check := NewModTidyCheck()
	err := check.Run(ctx, []string{"go.mod"})
	require.Error(t, err)
}

func TestFumptCheck_Run_NoFiles_Extra(t *testing.T) {
	ctx := context.Background()
	check := NewFumptCheck()
	err := check.Run(ctx, []string{})
	assert.NoError(t, err)
}

func TestLintCheck_Run_NoFiles_Extra(t *testing.T) {
	ctx := context.Background()
	check := NewLintCheck()
	err := check.Run(ctx, []string{})
	assert.NoError(t, err)
}

func TestModTidyCheck_Run_NoFiles_Extra(t *testing.T) {
	ctx := context.Background()
	check := NewModTidyCheck()
	err := check.Run(ctx, []string{})
	assert.NoError(t, err)
}

func TestNewFumptCheckWithConfig_Extra(t *testing.T) {
	sharedCtx := shared.NewContext()
	timeout := 10 * time.Second
	check := NewFumptCheckWithConfig(sharedCtx, timeout)
	assert.Equal(t, timeout, check.timeout)
	assert.Equal(t, sharedCtx, check.sharedCtx)
}

func TestGitleaksCheck_GetGitCommitRange_Extra(t *testing.T) {
	// This function requires a real git repository
	// Test with current directory which should be a git repo
	ctx := context.Background()
	check := NewGitleaksCheck()

	// Get the repository root
	repoRoot := "../../.." // Go up to the project root

	// Call the function
	commitRange, err := check.getGitCommitRange(ctx, repoRoot)

	// The function should either succeed or return a specific error
	if err != nil {
		// Check for expected error types
		assert.Contains(t, err.Error(), "git", "Error should be git-related")
	} else {
		// If successful, should return a valid commit range format
		assert.Contains(t, commitRange, "--no-merges")
		assert.Contains(t, commitRange, "--first-parent")
	}
}

func TestGitleaksCheck_GetGitCommitRange_NotARepo_Extra(t *testing.T) {
	ctx := context.Background()
	check := NewGitleaksCheck()

	// Test with a directory that's not a git repository
	_, err := check.getGitCommitRange(ctx, "/tmp")

	// Should return an error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestGitleaksCheck_GetGitCommitRange_ContextCanceled_Extra(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	check := NewGitleaksCheck()

	_, err := check.getGitCommitRange(ctx, ".")

	// Should return an error due to canceled context
	require.Error(t, err)
}
