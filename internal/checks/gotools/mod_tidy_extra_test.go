package gotools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/shared"
)

// initGitRepoAt initializes a git repository in dir with a deterministic,
// signature-free configuration suitable for tests.
func initGitRepoAt(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // test git command with controlled args
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	run("init")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")
	run("config", "commit.gpgsign", "false")
}

// gitAddCommit stages and commits everything in dir.
func gitAddCommit(t *testing.T, dir, msg string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-m", msg}} {
		cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // test git command with controlled args
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
}

func TestCheckUncommittedChanges_Direct(t *testing.T) {
	c := NewModTidyCheckWithSharedContext(shared.NewContext())

	t.Run("clean committed go.mod returns nil", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepoAt(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/m\n\ngo 1.25\n"), 0o600))
		gitAddCommit(t, dir, "add go.mod")

		require.NoError(t, c.checkUncommittedChanges(context.Background(), dir, dir))
	})

	t.Run("untracked go.mod is reported", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepoAt(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/m\n\ngo 1.25\n"), 0o600))

		err := c.checkUncommittedChanges(context.Background(), dir, dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git status")
	})

	t.Run("modified go.mod is reported", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepoAt(t, dir)
		modPath := filepath.Join(dir, "go.mod")
		require.NoError(t, os.WriteFile(modPath, []byte("module example.com/m\n\ngo 1.25\n"), 0o600))
		gitAddCommit(t, dir, "add go.mod")
		require.NoError(t, os.WriteFile(modPath, []byte("module example.com/m\n\ngo 1.25\n// changed\n"), 0o600))

		err := c.checkUncommittedChanges(context.Background(), dir, dir)
		require.Error(t, err)
	})

	t.Run("non-git directory returns git status error", func(t *testing.T) {
		dir := t.TempDir() // not a git repo
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/m\n"), 0o600))

		err := c.checkUncommittedChanges(context.Background(), dir, dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check git status")
	})
}

func TestRunModTidyOnModule(t *testing.T) {
	c := NewModTidyCheckWithSharedContext(shared.NewContext())

	t.Run("already-tidy stdlib-only module returns nil", func(t *testing.T) {
		dir := t.TempDir()
		// Independent module with only a stdlib import; no network needed.
		cmd := exec.CommandContext(context.Background(), "go", "mod", "init", "example.com/tidy")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"),
			[]byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hi\") }\n"), 0o600))

		require.NoError(t, c.runModTidyOnModule(context.Background(), dir, dir))
	})

	t.Run("missing go.mod returns error", func(t *testing.T) {
		dir := t.TempDir() // no go.mod
		err := c.runModTidyOnModule(context.Background(), dir, dir)
		require.Error(t, err)
	})
}
