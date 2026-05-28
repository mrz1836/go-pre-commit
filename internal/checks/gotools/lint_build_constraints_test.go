package gotools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

// errStubLintFailed is a static sentinel for stubbed golangci-lint retry failures.
var errStubLintFailed = errors.New("stub golangci-lint failure")

// buildTagSubdir is the package subdirectory used by the build-constraints tests.
const buildTagSubdir = "pkg"

// writeGoFile writes content to repoRoot/<buildTagSubdir>/a.go, creating dirs.
func writeGoFile(t *testing.T, repoRoot, content string) {
	t.Helper()
	full := filepath.Join(repoRoot, buildTagSubdir)
	require.NoError(t, os.MkdirAll(full, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(full, "a.go"), []byte(content), 0o600))
}

// stubLintRetry replaces the golangci-lint retry seam for the duration of the
// test, returning the supplied output/error.
func stubLintRetry(t *testing.T, output string, err error) {
	t.Helper()
	orig := runGolangciLintRetry
	t.Cleanup(func() { runGolangciLintRetry = orig })
	runGolangciLintRetry = func(_ context.Context, _ string, _ ...string) (string, error) {
		return output, err
	}
}

func TestHandleBuildConstraintsError(t *testing.T) {
	const origOutput = "build constraints exclude all Go files"

	t.Run("no build tags returns tool execution error", func(t *testing.T) {
		repoRoot := t.TempDir()
		writeGoFile(t, repoRoot, "package pkg\n")

		c := NewLintCheck()
		err := c.handleBuildConstraintsError(context.Background(), repoRoot, buildTagSubdir, origOutput)
		require.Error(t, err)

		var checkErr *prerrors.CheckError
		require.ErrorAs(t, err, &checkErr)
		assert.Contains(t, checkErr.Suggestion, "build-tags")
	})

	t.Run("tags detected and retry succeeds", func(t *testing.T) {
		repoRoot := t.TempDir()
		writeGoFile(t, repoRoot, "//go:build integration\n\npackage pkg\n")
		stubLintRetry(t, "", nil)

		c := NewLintCheck()
		err := c.handleBuildConstraintsError(context.Background(), repoRoot, buildTagSubdir, origOutput)
		require.NoError(t, err)
	})

	t.Run("tags detected and retry reports linting issues", func(t *testing.T) {
		repoRoot := t.TempDir()
		writeGoFile(t, repoRoot, "//go:build integration\n\npackage pkg\n")
		stubLintRetry(t, "pkg/a.go:3:1: some lint issue", errStubLintFailed)

		c := NewLintCheck()
		err := c.handleBuildConstraintsError(context.Background(), repoRoot, buildTagSubdir, origOutput)
		require.Error(t, err)
		require.ErrorIs(t, err, prerrors.ErrLintingIssues)
	})

	t.Run("tags detected and retry fails without lint pattern", func(t *testing.T) {
		repoRoot := t.TempDir()
		writeGoFile(t, repoRoot, "// +build integration\n\npackage pkg\n")
		stubLintRetry(t, "some unrelated tool failure", errStubLintFailed)

		c := NewLintCheck()
		err := c.handleBuildConstraintsError(context.Background(), repoRoot, buildTagSubdir, origOutput)
		require.Error(t, err)

		var checkErr *prerrors.CheckError
		require.ErrorAs(t, err, &checkErr)
		assert.Contains(t, checkErr.Suggestion, "Detected build tags")
	})
}

func TestLintRun_EmptyAndEnvTags(t *testing.T) {
	t.Run("empty file list returns nil", func(t *testing.T) {
		c := NewLintCheck()
		require.NoError(t, c.Run(context.Background(), nil))
	})

	t.Run("build tags env var is parsed and trimmed", func(t *testing.T) {
		// Run outside any git repo so runDirectLint returns early at repo-root
		// resolution, after the env tags have been parsed.
		t.Chdir(t.TempDir())
		t.Setenv("GO_PRE_COMMIT_BUILD_TAGS", " integration , e2e ")

		c := NewLintCheck()
		err := c.Run(context.Background(), []string{"foo.go"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repository root")
		assert.Equal(t, []string{"integration", "e2e"}, c.buildTags)
	})
}
