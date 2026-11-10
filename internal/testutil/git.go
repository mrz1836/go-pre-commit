package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// IsolateGit keeps the developer's git setup out of a test that runs git:
// HOME points at an empty temporary directory, git reads no global or system
// configuration, and GIT_* variables inherited from an enclosing hook
// (GIT_DIR, GIT_INDEX_FILE, ...) are unset. Everything is restored when tb
// finishes. It uses Setenv, so the test cannot run in parallel.
func IsolateGit(tb testing.TB) {
	tb.Helper()
	for key, value := range environMap(os.Environ()) {
		if strings.HasPrefix(key, "GIT_") {
			tb.Setenv(key, value) // restores the variable after the test
			if err := os.Unsetenv(key); err != nil {
				tb.Fatalf("failed to unset %s: %v", key, err)
			}
		}
	}
	tb.Setenv("HOME", tb.TempDir())
	tb.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	tb.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	tb.Setenv("GIT_AUTHOR_NAME", "Test")
	tb.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	tb.Setenv("GIT_COMMITTER_NAME", "Test")
	tb.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

// NewGitRepo isolates git (see IsolateGit) and returns the root of a new,
// empty repository in a temporary directory, as git reports it. It skips the
// test when git is not installed.
func NewGitRepo(tb testing.TB) string {
	tb.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		tb.Skip("git not available")
	}
	IsolateGit(tb)

	dir := tb.TempDir()
	if output, err := runGit(tb, dir, "init", "-q"); err != nil {
		tb.Fatalf("git init: %v: %s", err, output)
	}
	root, err := runGit(tb, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		tb.Fatalf("git rev-parse: %v: %s", err, root)
	}
	return filepath.FromSlash(root)
}

// runGit runs git in dir and returns its trimmed combined output
func runGit(tb testing.TB, dir string, args ...string) (string, error) {
	tb.Helper()
	cmd := exec.CommandContext(tb.Context(), "git", args...) // #nosec G204 - git binary path is fixed, args are test-controlled
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// RunGit runs git with args in dir, failing the test on error, and returns
// its trimmed output
func RunGit(tb testing.TB, dir string, args ...string) string {
	tb.Helper()
	output, err := runGit(tb, dir, args...)
	if err != nil {
		tb.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return output
}
