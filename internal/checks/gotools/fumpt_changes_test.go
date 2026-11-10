package gotools

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/shared"
	"github.com/mrz1836/go-pre-commit/internal/testutil"
	"github.com/mrz1836/go-pre-commit/internal/tools"
)

// installFakeGofumpt puts script on an isolated PATH as gofumpt, next to git,
// so the check finds it and can never install the real tool
func installFakeGofumpt(t *testing.T, script string) {
	t.Helper()
	testutil.IsolatedPath(t, map[string]string{"gofumpt": script})
	tools.CleanCache()
	t.Cleanup(tools.CleanCache)
}

// TestFumptCheckReportsChangedFiles tests that fumpt reports the files gofumpt
// rewrote, in the relative form they were given, and still passes
func TestFumptCheckReportsChangedFiles(t *testing.T) {
	repo := testutil.NewGitRepo(t)
	t.Chdir(repo)
	installFakeGofumpt(t, testutil.FakeGofumpt)

	require.NoError(t, os.WriteFile("clean.go", []byte("package main\n"), 0o600))
	require.NoError(t, os.MkdirAll("pkg", 0o750))
	require.NoError(t, os.WriteFile(filepath.Join("pkg", "messy.go"), []byte("package main // messy\n"), 0o600))

	check := NewFumptCheckWithConfig(shared.NewContext(), 30*time.Second)
	changed, err := check.RunWithChanges(t.Context(), []string{"clean.go", "pkg/messy.go"})

	require.NoError(t, err)
	assert.Equal(t, []string{"pkg/messy.go"}, changed)

	content, err := os.ReadFile(filepath.Join(repo, "pkg", "messy.go")) // #nosec G304 - the test's own temporary file
	require.NoError(t, err)
	assert.Equal(t, "package main\n", string(content), "fumpt still writes the formatted file")

	changed, err = check.RunWithChanges(t.Context(), []string{"clean.go", "pkg/messy.go"})
	require.NoError(t, err)
	assert.Equal(t, []string{}, changed, "nothing left to change")
}

// TestFumptCheckRunStillFormats tests that Run keeps formatting files in place
func TestFumptCheckRunStillFormats(t *testing.T) {
	repo := testutil.NewGitRepo(t)
	t.Chdir(repo)
	installFakeGofumpt(t, testutil.FakeGofumpt)
	require.NoError(t, os.WriteFile("messy.go", []byte("package main // messy\n"), 0o600))

	require.NoError(t, NewFumptCheckWithConfig(shared.NewContext(), 30*time.Second).Run(t.Context(), []string{"messy.go"}))

	content, err := os.ReadFile(filepath.Join(repo, "messy.go")) // #nosec G304 - the test's own temporary file
	require.NoError(t, err)
	assert.Equal(t, "package main\n", string(content))
}

// TestFumptCheckReportsNoChangesOnFailure tests that a gofumpt failure fails
// the check with a suggestion and reports no changed files
func TestFumptCheckReportsNoChangesOnFailure(t *testing.T) {
	repo := testutil.NewGitRepo(t)
	t.Chdir(repo)
	installFakeGofumpt(t, testutil.FakeGofumptSyntaxError)
	require.NoError(t, os.WriteFile("broken.go", []byte("package\n"), 0o600))

	changed, err := NewFumptCheckWithConfig(shared.NewContext(), 30*time.Second).RunWithChanges(t.Context(), []string{"broken.go"})

	var checkErr *prerrors.CheckError
	require.ErrorAs(t, err, &checkErr)
	assert.Contains(t, checkErr.Output, "syntax error")
	assert.Contains(t, checkErr.Suggestion, "syntax errors")
	assert.Equal(t, []string{}, changed)
}
