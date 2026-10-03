package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

// gitRunner runs git with args in dir and returns its standard output.
// It is a seam so tests can replace the git binary with a mock.
type gitRunner func(ctx context.Context, dir string, args ...string) ([]byte, error)

// execGitRunner runs the real git binary
func execGitRunner(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...) // #nosec G204 - git binary path is fixed, args are internal
	cmd.Dir = dir
	return cmd.Output()
}

// resolveHooksDir returns the directory git runs hooks from for the working
// tree at repoRoot. It asks git first, which covers linked worktrees (where
// .git is a file), submodules, and core.hooksPath. If git cannot answer for
// this exact working tree, it falls back to reading the .git entry directly.
func resolveHooksDir(ctx context.Context, run gitRunner, repoRoot string) (string, error) {
	if run != nil {
		if dir, err := hooksDirFromGit(ctx, run, repoRoot); err == nil {
			return dir, nil
		}
	}
	return hooksDirFromFilesystem(repoRoot)
}

// hooksDirFromGit resolves the hooks directory with git rev-parse
func hooksDirFromGit(ctx context.Context, run gitRunner, repoRoot string) (string, error) {
	output, err := run(ctx, repoRoot, "rev-parse", "--show-toplevel", "--git-path", "hooks")
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(output), "\r\n"), "\n")
	if len(lines) != 2 {
		return "", fmt.Errorf("%w: unexpected git rev-parse output %q", prerrors.ErrHooksDirUnresolved, output)
	}
	topLevel := strings.TrimSuffix(lines[0], "\r")
	hooksDir := strings.TrimSuffix(lines[1], "\r")
	if topLevel == "" || hooksDir == "" {
		return "", fmt.Errorf("%w: empty git rev-parse output", prerrors.ErrHooksDirUnresolved)
	}

	// git searches parent directories for a repository, so a repoRoot that is
	// not itself a valid working tree would resolve to an enclosing repository.
	// Only trust the answer when it is about repoRoot.
	if !samePath(topLevel, repoRoot) {
		return "", fmt.Errorf("%w: git resolved working tree %s, expected %s",
			prerrors.ErrHooksDirUnresolved, topLevel, repoRoot)
	}

	// Relative paths are relative to the command's working directory (repoRoot).
	// That also matches git's rule for a relative core.hooksPath, which is
	// resolved from the top of the working tree where hooks run.
	hooksDir = filepath.FromSlash(hooksDir)
	if !filepath.IsAbs(hooksDir) {
		hooksDir = filepath.Join(repoRoot, hooksDir)
	}
	return filepath.Clean(hooksDir), nil
}

// hooksDirFromFilesystem resolves the hooks directory by reading the .git
// entry at repoRoot. A .git directory holds hooks itself. A .git file (linked
// worktree or submodule) points at a git directory, whose optional commondir
// file points at the directory shared by all worktrees, where hooks live.
func hooksDirFromFilesystem(repoRoot string) (string, error) {
	dotGit := filepath.Join(repoRoot, ".git")
	info, err := os.Stat(dotGit)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", prerrors.ErrNotGitRepository, repoRoot)
		}
		return "", fmt.Errorf("failed to stat %s: %w", dotGit, err)
	}

	gitDir := dotGit
	if !info.IsDir() {
		if gitDir, err = readGitFile(dotGit); err != nil {
			return "", err
		}
	}

	commonDir := gitDir
	if content, readErr := os.ReadFile(filepath.Join(gitDir, "commondir")); readErr == nil { //nolint:gosec // Path is derived from the repository's git directory
		if dir := strings.TrimSpace(string(content)); dir != "" {
			commonDir = resolveRelative(gitDir, dir)
		}
	}

	return filepath.Join(commonDir, "hooks"), nil
}

// readGitFile parses a "gitdir: <path>" .git file and returns the git
// directory it points at, resolved relative to the file's directory.
func readGitFile(path string) (string, error) {
	content, err := os.ReadFile(path) //nolint:gosec // Path is the repository's .git file
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}

	const prefix = "gitdir:"
	line, _, _ := strings.Cut(string(content), "\n")
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, prefix) {
		return "", fmt.Errorf("%w: %s", prerrors.ErrInvalidGitFile, path)
	}

	gitDir := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if gitDir == "" {
		return "", fmt.Errorf("%w: %s", prerrors.ErrInvalidGitFile, path)
	}

	return resolveRelative(filepath.Dir(path), gitDir), nil
}

// resolveRelative joins path onto base unless path is already absolute
func resolveRelative(base, path string) string {
	path = filepath.FromSlash(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(base, path)
}

// samePath reports whether a and b name the same directory, resolving
// symlinks (e.g. /tmp -> /private/tmp on macOS) where possible.
func samePath(a, b string) bool {
	return canonicalPath(a) == canonicalPath(b)
}

// canonicalPath returns a cleaned, symlink-resolved form of path
func canonicalPath(path string) string {
	path = filepath.Clean(filepath.FromSlash(path))
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return path
}
