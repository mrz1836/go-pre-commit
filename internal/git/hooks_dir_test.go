package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

var errMockGitFailed = errors.New("mock git failed")

// mockGit is a gitRunner that returns canned output and records its calls
type mockGit struct {
	output string
	err    error
	calls  []mockGitCall
}

type mockGitCall struct {
	dir  string
	args []string
}

func (m *mockGit) run(_ context.Context, dir string, args ...string) ([]byte, error) {
	m.calls = append(m.calls, mockGitCall{dir: dir, args: args})
	if m.err != nil {
		return nil, m.err
	}
	return []byte(m.output), nil
}

// revParseOutput builds the output of "git rev-parse --show-toplevel --git-path hooks"
func revParseOutput(topLevel, hooksDir string) string {
	return topLevel + "\n" + hooksDir + "\n"
}

// newMockInstaller returns an installer for repoRoot that uses the mock instead of git
func newMockInstaller(repoRoot string, mock *mockGit) *Installer {
	installer := NewInstaller(repoRoot, "")
	installer.runGit = mock.run
	return installer
}

func TestHooksDirFromGit(t *testing.T) {
	root := t.TempDir()
	commonHooks := filepath.Join(t.TempDir(), "main", ".git", "hooks")

	testCases := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "main checkout returns relative path",
			output:   revParseOutput(root, ".git/hooks"),
			expected: filepath.Join(root, ".git", "hooks"),
		},
		{
			name:     "linked worktree returns shared hooks directory",
			output:   revParseOutput(root, commonHooks),
			expected: commonHooks,
		},
		{
			name:     "relative core.hooksPath resolves from working tree root",
			output:   revParseOutput(root, ".husky"),
			expected: filepath.Join(root, ".husky"),
		},
		{
			name:     "windows line endings are trimmed",
			output:   root + "\r\n.git/hooks\r\n",
			expected: filepath.Join(root, ".git", "hooks"),
		},
		{
			name:     "unclean path is cleaned",
			output:   revParseOutput(root, "./.git/../.git/hooks/"),
			expected: filepath.Join(root, ".git", "hooks"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockGit{output: tc.output}

			dir, err := hooksDirFromGit(context.Background(), mock.run, root)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, dir)

			require.Len(t, mock.calls, 1)
			assert.Equal(t, root, mock.calls[0].dir)
			assert.Equal(t, []string{"rev-parse", "--show-toplevel", "--git-path", "hooks"}, mock.calls[0].args)
		})
	}
}

func TestHooksDirFromGit_Errors(t *testing.T) {
	root := t.TempDir()

	testCases := []struct {
		name        string
		mock        *mockGit
		expectedErr error
	}{
		{
			name:        "git fails",
			mock:        &mockGit{err: errMockGitFailed},
			expectedErr: errMockGitFailed,
		},
		{
			name:        "empty output",
			mock:        &mockGit{output: ""},
			expectedErr: prerrors.ErrHooksDirUnresolved,
		},
		{
			name:        "missing hooks line",
			mock:        &mockGit{output: root + "\n"},
			expectedErr: prerrors.ErrHooksDirUnresolved,
		},
		{
			name:        "too many lines",
			mock:        &mockGit{output: root + "\n.git/hooks\nextra\n"},
			expectedErr: prerrors.ErrHooksDirUnresolved,
		},
		{
			name:        "empty hooks line",
			mock:        &mockGit{output: root + "\n\n"},
			expectedErr: prerrors.ErrHooksDirUnresolved,
		},
		{
			name:        "git resolved an enclosing repository",
			mock:        &mockGit{output: revParseOutput(filepath.Dir(root), ".git/hooks")},
			expectedErr: prerrors.ErrHooksDirUnresolved,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := hooksDirFromGit(context.Background(), tc.mock.run, root)
			require.ErrorIs(t, err, tc.expectedErr)
			assert.Empty(t, dir)
		})
	}
}

func TestHooksDirFromGit_TopLevelThroughSymlink(t *testing.T) {
	realRoot := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(realRoot, link); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	// git reports the resolved path while the caller holds the symlinked one
	mock := &mockGit{output: revParseOutput(realRoot, ".git/hooks")}

	dir, err := hooksDirFromGit(context.Background(), mock.run, link)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(link, ".git", "hooks"), dir)
}

func TestHooksDirFromFilesystem(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(t *testing.T) (repoRoot, expected string)
	}{
		{
			name: "git directory",
			setup: func(t *testing.T) (string, string) {
				root := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o750))
				return root, filepath.Join(root, ".git", "hooks")
			},
		},
		{
			name: "linked worktree with absolute gitdir",
			setup: func(t *testing.T) (string, string) {
				base := t.TempDir()
				commonDir := filepath.Join(base, "main", ".git")
				worktreeGitDir := writeLinkedWorktree(t, commonDir, "wt", "../..")
				root := filepath.Join(base, "wt")
				require.NoError(t, os.MkdirAll(root, 0o750))
				writeFile(t, filepath.Join(root, ".git"), "gitdir: "+worktreeGitDir+"\n")
				return root, filepath.Join(commonDir, "hooks")
			},
		},
		{
			name: "linked worktree with relative gitdir",
			setup: func(t *testing.T) (string, string) {
				base := t.TempDir()
				commonDir := filepath.Join(base, "main", ".git")
				writeLinkedWorktree(t, commonDir, "wt", "../..")
				root := filepath.Join(base, "wt")
				require.NoError(t, os.MkdirAll(root, 0o750))
				writeFile(t, filepath.Join(root, ".git"), "gitdir: ../main/.git/worktrees/wt\r\n")
				return root, filepath.Join(commonDir, "hooks")
			},
		},
		{
			name: "linked worktree with absolute commondir",
			setup: func(t *testing.T) (string, string) {
				base := t.TempDir()
				commonDir := filepath.Join(base, "main", ".git")
				worktreeGitDir := writeLinkedWorktree(t, commonDir, "wt", commonDir)
				root := filepath.Join(base, "wt")
				require.NoError(t, os.MkdirAll(root, 0o750))
				writeFile(t, filepath.Join(root, ".git"), "gitdir: "+worktreeGitDir+"\n")
				return root, filepath.Join(commonDir, "hooks")
			},
		},
		{
			name: "submodule without commondir uses its own git directory",
			setup: func(t *testing.T) (string, string) {
				base := t.TempDir()
				moduleGitDir := filepath.Join(base, ".git", "modules", "sub")
				require.NoError(t, os.MkdirAll(moduleGitDir, 0o750))
				root := filepath.Join(base, "sub")
				require.NoError(t, os.MkdirAll(root, 0o750))
				writeFile(t, filepath.Join(root, ".git"), "gitdir: ../.git/modules/sub\n")
				return root, filepath.Join(moduleGitDir, "hooks")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root, expected := tc.setup(t)

			dir, err := hooksDirFromFilesystem(root)
			require.NoError(t, err)
			assert.Equal(t, expected, dir)
		})
	}
}

func TestHooksDirFromFilesystem_Errors(t *testing.T) {
	testCases := []struct {
		name        string
		gitFile     *string
		expectedErr error
	}{
		{
			name:        "no .git entry",
			expectedErr: prerrors.ErrNotGitRepository,
		},
		{
			name:        "malformed .git file",
			gitFile:     stringPtr("not a gitdir pointer\n"),
			expectedErr: prerrors.ErrInvalidGitFile,
		},
		{
			name:        "empty gitdir",
			gitFile:     stringPtr("gitdir:   \n"),
			expectedErr: prerrors.ErrInvalidGitFile,
		},
		{
			name:        "empty .git file",
			gitFile:     stringPtr(""),
			expectedErr: prerrors.ErrInvalidGitFile,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.gitFile != nil {
				writeFile(t, filepath.Join(root, ".git"), *tc.gitFile)
			}

			dir, err := hooksDirFromFilesystem(root)
			require.ErrorIs(t, err, tc.expectedErr)
			assert.Empty(t, dir)
		})
	}
}

func TestResolveHooksDir(t *testing.T) {
	t.Run("prefers git", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o750))
		mock := &mockGit{output: revParseOutput(root, ".husky")}

		dir, err := resolveHooksDir(context.Background(), mock.run, root)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(root, ".husky"), dir)
	})

	t.Run("falls back to the filesystem when git fails", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o750))
		mock := &mockGit{err: errMockGitFailed}

		dir, err := resolveHooksDir(context.Background(), mock.run, root)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(root, ".git", "hooks"), dir)
	})

	t.Run("falls back to the filesystem when git resolves another repository", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o750))
		mock := &mockGit{output: revParseOutput(filepath.Dir(root), ".git/hooks")}

		dir, err := resolveHooksDir(context.Background(), mock.run, root)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(root, ".git", "hooks"), dir)
	})

	t.Run("nil runner uses the filesystem", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o750))

		dir, err := resolveHooksDir(context.Background(), nil, root)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(root, ".git", "hooks"), dir)
	})

	t.Run("not a repository", func(t *testing.T) {
		mock := &mockGit{err: errMockGitFailed}

		_, err := resolveHooksDir(context.Background(), mock.run, t.TempDir())
		require.ErrorIs(t, err, prerrors.ErrNotGitRepository)
	})
}

func TestInstaller_WorktreeWithMockGit(t *testing.T) {
	base := t.TempDir()
	commonHooks := filepath.Join(base, "main", ".git", "hooks")
	root := filepath.Join(base, "wt")
	require.NoError(t, os.MkdirAll(root, 0o750))
	// In a linked worktree .git is a file, so <root>/.git/hooks cannot be created
	writeFile(t, filepath.Join(root, ".git"), "gitdir: "+filepath.Join(base, "main", ".git", "worktrees", "wt")+"\n")

	installer := newMockInstaller(root, &mockGit{output: revParseOutput(root, commonHooks)})

	dir, err := installer.HooksDir()
	require.NoError(t, err)
	assert.Equal(t, commonHooks, dir)

	require.NoError(t, installer.InstallHook("pre-commit", false))

	hookPath := filepath.Join(commonHooks, "pre-commit")
	content, err := os.ReadFile(hookPath) //nolint:gosec // test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, installer.GenerateHookScript(), string(content))
	assert.True(t, installer.IsHookInstalled("pre-commit"))

	status, err := installer.GetInstallationStatus("pre-commit")
	require.NoError(t, err)
	assert.Equal(t, hookPath, status.HookPath)
	assert.True(t, status.Installed)
	assert.False(t, status.Outdated)

	removed, err := installer.UninstallHook("pre-commit")
	require.NoError(t, err)
	assert.True(t, removed)
	assert.NoFileExists(t, hookPath)
}

func TestInstaller_CoreHooksPathWithMockGit(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0o750))

	installer := newMockInstaller(root, &mockGit{output: revParseOutput(root, ".githooks")})
	require.NoError(t, installer.InstallHook("pre-commit", false))

	assert.FileExists(t, filepath.Join(root, ".githooks", "pre-commit"))
	assert.NoFileExists(t, filepath.Join(root, ".git", "hooks", "pre-commit"))
}

func TestInstaller_HooksDirUnresolved(t *testing.T) {
	// Neither git nor the filesystem can locate a repository
	installer := newMockInstaller(t.TempDir(), &mockGit{err: errMockGitFailed})

	err := installer.InstallHook("pre-commit", false)
	require.ErrorIs(t, err, prerrors.ErrNotGitRepository)

	removed, err := installer.UninstallHook("pre-commit")
	require.ErrorIs(t, err, prerrors.ErrNotGitRepository)
	assert.False(t, removed)

	assert.False(t, installer.IsHookInstalled("pre-commit"))

	status, err := installer.GetInstallationStatus("pre-commit")
	require.ErrorIs(t, err, prerrors.ErrNotGitRepository)
	assert.Equal(t, "pre-commit", status.HookType)
	assert.Empty(t, status.HookPath)
}

func TestGetInstallationStatus_OutdatedHook(t *testing.T) {
	root := t.TempDir()
	hooksDir := filepath.Join(root, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0o750))
	installer := newMockInstaller(root, &mockGit{output: revParseOutput(root, ".git/hooks")})

	// A hook from an earlier release, with the installing checkout's path baked in
	legacyHook := "#!/bin/bash\n# Go Pre-commit Hook\nREPO_ROOT=\"/old/checkout\"\nexec go-pre-commit run\n"
	writeFile(t, filepath.Join(hooksDir, "pre-commit"), legacyHook)
	require.NoError(t, os.Chmod(filepath.Join(hooksDir, "pre-commit"), 0o755)) //nolint:gosec // hook must be executable

	status, err := installer.GetInstallationStatus("pre-commit")
	require.NoError(t, err)
	assert.True(t, status.Installed)
	assert.True(t, status.IsOurHook)
	assert.True(t, status.Outdated)
	assert.Contains(t, status.Message, "outdated")

	// Reinstalling updates our own hook in place without --force
	require.NoError(t, installer.InstallHook("pre-commit", false))

	status, err = installer.GetInstallationStatus("pre-commit")
	require.NoError(t, err)
	assert.True(t, status.Installed)
	assert.False(t, status.Outdated)
	assert.Equal(t, "Go pre-commit hook installed and ready", status.Message)
}

func TestHookScript_ResolvesRootAtRuntime(t *testing.T) {
	installer := NewInstaller("/install/time/checkout", "")
	script := installer.GenerateHookScript()

	assert.Contains(t, script, `REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"`)
	assert.Contains(t, script, `CONFIG_FILE="$REPO_ROOT/.github/.env.base"`)
	assert.NotContains(t, script, "/install/time/checkout", "hook must not embed the installing checkout's path")
	assert.Equal(t, script, NewInstaller("/another/checkout", "").GenerateHookScript(),
		"every checkout and worktree must get the same hook")
}

// Integration tests with the real git binary

func TestInstaller_LinkedWorktree_Integration(t *testing.T) {
	mainRoot, worktreeRoot := newRepoWithWorktree(t)

	// Installing from a worktree used to fail with "mkdir <wt>/.git: not a directory"
	installer := NewInstaller(worktreeRoot, "")
	require.NoError(t, installer.InstallHook("pre-commit", false))

	sharedHook := filepath.Join(mainRoot, ".git", "hooks", "pre-commit")
	assert.FileExists(t, sharedHook)

	// The main checkout sees the same hook, and either tree can uninstall it
	assert.True(t, NewInstaller(mainRoot, "").IsHookInstalled("pre-commit"))
	status, err := installer.GetInstallationStatus("pre-commit")
	require.NoError(t, err)
	assert.True(t, samePath(sharedHook, status.HookPath), "status path %s, expected %s", status.HookPath, sharedHook)

	removed, err := NewInstaller(mainRoot, "").UninstallHook("pre-commit")
	require.NoError(t, err)
	assert.True(t, removed)
	assert.False(t, installer.IsHookInstalled("pre-commit"))
}

func TestInstaller_CoreHooksPath_Integration(t *testing.T) {
	mainRoot, worktreeRoot := newRepoWithWorktree(t)
	runGitCmd(t, mainRoot, "config", "core.hooksPath", ".githooks")

	require.NoError(t, NewInstaller(worktreeRoot, "").InstallHook("pre-commit", false))

	// A relative core.hooksPath is relative to the working tree running the hook
	assert.FileExists(t, filepath.Join(worktreeRoot, ".githooks", "pre-commit"))
	assert.NoFileExists(t, filepath.Join(mainRoot, ".git", "hooks", "pre-commit"))
}

// TestHookScript_RunsInCommittingWorktree reproduces issue #184 end to end:
// install from the main checkout, then commit a new file from a linked worktree.
func TestHookScript_RunsInCommittingWorktree(t *testing.T) {
	requireBash(t)
	mainRoot, worktreeRoot := newRepoWithWorktree(t)
	require.NoError(t, NewInstaller(mainRoot, "").InstallHook("pre-commit", false))
	record := installFakeGoPreCommit(t)

	commitNewFile(t, worktreeRoot, "new.go")
	assert.Equal(t, canonicalPath(worktreeRoot), readTrimmed(t, record), "hook must run in the committing worktree")

	commitNewFile(t, mainRoot, "other.go")
	assert.Equal(t, canonicalPath(mainRoot), readTrimmed(t, record), "hook must still run in the main checkout")
}

// TestHookScript_SkipsUnconfiguredRepository covers a hooks directory shared by
// several repositories (core.hooksPath outside the repository): the hook must
// not run go-pre-commit, or block commits, in a repository it is not set up for.
func TestHookScript_SkipsUnconfiguredRepository(t *testing.T) {
	requireBash(t)
	isolateGitEnv(t)
	base := t.TempDir()
	sharedHooks := filepath.Join(base, "shared-hooks")
	configured := newRepo(t, filepath.Join(base, "configured"), true)
	unconfigured := newRepo(t, filepath.Join(base, "unconfigured"), false)
	for _, root := range []string{configured, unconfigured} {
		runGitCmd(t, root, "config", "core.hooksPath", sharedHooks)
	}

	require.NoError(t, NewInstaller(configured, "").InstallHook("pre-commit", false))
	require.FileExists(t, filepath.Join(sharedHooks, "pre-commit"))
	record := installFakeGoPreCommit(t)

	output := commitNewFile(t, unconfigured, "new.go")
	assert.NoFileExists(t, record, "go-pre-commit must not run in an unconfigured repository")
	assert.Contains(t, output, "not configured for this repository")

	commitNewFile(t, configured, "new.go")
	assert.Equal(t, canonicalPath(configured), readTrimmed(t, record), "hook must run in the configured repository")
}

// TestHookScript_FindsModularConfigInParentDirectory checks that the hook
// finds configuration the way go-pre-commit does: modular .github/env/*.env
// files, searched for from the repository root up through its parents.
func TestHookScript_FindsModularConfigInParentDirectory(t *testing.T) {
	requireBash(t)
	isolateGitEnv(t)
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, ".github", "env"), 0o750))
	writeFile(t, filepath.Join(base, ".github", "env", "10-pre-commit.env"), "ENABLE_GO_PRE_COMMIT=true\n")
	root := newRepo(t, filepath.Join(base, "nested", "repo"), false)

	require.NoError(t, NewInstaller(root, "").InstallHook("pre-commit", false))
	record := installFakeGoPreCommit(t)

	commitNewFile(t, root, "new.go")
	assert.Equal(t, canonicalPath(root), readTrimmed(t, record))
}

func TestHookScript_ValidBashSyntax(t *testing.T) {
	requireBash(t)
	script := filepath.Join(t.TempDir(), "pre-commit")
	writeFile(t, script, NewInstaller("", "").GenerateHookScript())

	output, err := exec.CommandContext(context.Background(), "bash", "-n", script).CombinedOutput() //nolint:gosec // test-controlled path
	require.NoError(t, err, "hook script has a bash syntax error: %s", output)
}

func requireBash(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("hook script tests require a Unix shell")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
}

// installFakeGoPreCommit puts a stand-in go-pre-commit first on PATH and
// returns the file it records its working directory in. Like the real checks,
// it fails if a staged file is missing from the tree it runs in.
func installFakeGoPreCommit(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	record := filepath.Join(t.TempDir(), "ran-in")
	fake := "#!/bin/bash\n" +
		"pwd -P > \"$FAKE_RECORD\"\n" +
		"for f in $(git diff --cached --name-only --diff-filter=ACMR); do\n" +
		"  [[ -f \"$f\" ]] || { echo \"lstat $(pwd)/$f: no such file or directory\" >&2; exit 1; }\n" +
		"done\n"
	writeFile(t, filepath.Join(binDir, "go-pre-commit"), fake)
	require.NoError(t, os.Chmod(filepath.Join(binDir, "go-pre-commit"), 0o755)) //nolint:gosec // fake binary must be executable
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_RECORD", record)

	// The hook stays quiet in CI; clear the markers so its messages can be checked
	for _, key := range []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"} {
		t.Setenv(key, "")
	}
	return record
}

// commitNewFile commits a new Go file in root and returns git's output
func commitNewFile(t *testing.T, root, name string) string {
	t.Helper()
	writeFile(t, filepath.Join(root, name), "package x\n\nfunc F() {}\n")
	runGitCmd(t, root, "add", name)
	return runGitCmd(t, root, "commit", "-m", "add "+name)
}

// newRepoWithWorktree creates a repository configured for go-pre-commit, with
// one commit and a linked worktree, returning both working tree roots as git
// reports them
func newRepoWithWorktree(t *testing.T) (mainRoot, worktreeRoot string) {
	t.Helper()
	isolateGitEnv(t)

	base := t.TempDir()
	mainRoot = newRepo(t, filepath.Join(base, "main"), true)
	runGitCmd(t, mainRoot, "worktree", "add", "-q", filepath.Join(base, "wt"))
	worktreeRoot = runGitCmd(t, filepath.Join(base, "wt"), "rev-parse", "--show-toplevel")
	return mainRoot, filepath.FromSlash(worktreeRoot)
}

// newRepo creates a repository at dir with one commit, optionally committing
// a legacy .github/.env.base that enables go-pre-commit, and returns its root
// as git reports it. Call isolateGitEnv first.
func newRepo(t *testing.T, dir string, withConfig bool) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	require.NoError(t, os.MkdirAll(dir, 0o750))
	runGitCmd(t, dir, "init", "-q", "-b", "main")
	if withConfig {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".github"), 0o750))
		writeFile(t, filepath.Join(dir, ".github", ".env.base"), "ENABLE_GO_PRE_COMMIT=true\n")
		runGitCmd(t, dir, "add", ".github/.env.base")
	}
	runGitCmd(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	return filepath.FromSlash(runGitCmd(t, dir, "rev-parse", "--show-toplevel"))
}

// isolateGitEnv keeps the developer's git config and any GIT_* variables from
// an enclosing hook (GIT_DIR, GIT_INDEX_FILE, ...) out of the test
func isolateGitEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		key, value, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(key, "GIT_") {
			t.Setenv(key, value) // restores the variable after the test
			require.NoError(t, os.Unsetenv(key))
		}
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

// runGitCmd runs git in dir, failing the test on error, and returns trimmed stdout
func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...) // #nosec G204 - git binary path is fixed, args are test-controlled
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s: %s", strings.Join(args, " "), output)
	return strings.TrimSpace(string(output))
}

// writeLinkedWorktree creates the git directory of a linked worktree inside
// commonDir and returns its path
func writeLinkedWorktree(t *testing.T, commonDir, name, commonDirRef string) string {
	t.Helper()
	worktreeGitDir := filepath.Join(commonDir, "worktrees", name)
	require.NoError(t, os.MkdirAll(worktreeGitDir, 0o750))
	writeFile(t, filepath.Join(worktreeGitDir, "commondir"), commonDirRef+"\n")
	return worktreeGitDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func readTrimmed(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path) //nolint:gosec // test file path is controlled
	require.NoError(t, err)
	return strings.TrimSpace(string(content))
}

func stringPtr(s string) *string {
	return &s
}
