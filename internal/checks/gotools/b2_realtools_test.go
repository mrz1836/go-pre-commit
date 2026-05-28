package gotools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/shared"
)

func TestFumptRun_SuccessWithAutoStage(t *testing.T) {
	if _, err := exec.LookPath("gofumpt"); err != nil {
		t.Skip("gofumpt not installed")
	}

	dir := t.TempDir()
	initGitRepoAt(t, dir)
	cmd := exec.CommandContext(context.Background(), "go", "mod", "init", "example.com/fumpt")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	// Intentionally badly-formatted file (extra spaces gofumpt will fix).
	src := "package main\n\nfunc  main()  {\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600))
	gitAddCommit(t, dir, "init")
	t.Chdir(dir)

	cfg := &config.Config{}
	cfg.CheckBehaviors.FumptAutoStage = true
	cfg.CheckTimeouts.Fumpt = 30
	c := NewFumptCheckWithFullConfig(shared.NewContext(), cfg)

	require.NoError(t, c.Run(context.Background(), []string{"main.go"}))

	formatted, err := os.ReadFile(filepath.Join(dir, "main.go")) //nolint:gosec // test path
	require.NoError(t, err)
	assert.NotEqual(t, src, string(formatted), "gofumpt should have reformatted the file")

	// File should be staged (no unstaged modifications remaining for main.go).
	status := exec.CommandContext(context.Background(), "git", "status", "--porcelain", "main.go")
	status.Dir = dir
	statusOut, err := status.CombinedOutput()
	require.NoError(t, err)
	// A staged-only change is reported as "M " (index modified, worktree clean).
	if len(statusOut) > 0 {
		assert.NotEqual(t, byte(' '), statusOut[0], "main.go change should be staged in the index")
	}
}

func TestRunGitleaks_Modes(t *testing.T) {
	if _, err := exec.LookPath("gitleaks"); err != nil {
		t.Skip("gitleaks not installed")
	}

	t.Run("changed-files mode on clean repo passes", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepoAt(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello world\n"), 0o600))
		gitAddCommit(t, dir, "init")
		t.Chdir(dir)

		cfg := &config.Config{}
		cfg.Checks.GitleaksAllFiles = false
		cfg.CheckTimeouts.Gitleaks = 60
		c := NewGitleaksCheckWithFullConfig(shared.NewContext(), cfg)

		require.NoError(t, c.runGitleaks(context.Background(), []string{"a.txt"}))
	})

	t.Run("detects committed secret", func(t *testing.T) {
		dir := t.TempDir()
		initGitRepoAt(t, dir)
		// Fake GitHub PAT-shaped token (matches gitleaks' github-pat rule). This
		// is a test fixture only, not a real credential. The repo's own gitleaks
		// scan ignores it two ways: the .gitleaksignore fingerprint at the repo
		// root (for --no-git scans) and the inline allow directive below (for the
		// CI gitleaks-action's git-history scan, where fingerprints are
		// commit-specific).
		secret := "github_token = ghp_x9Kf3mQ7zP2wL8nR4tV6yB1cD5eG0hJ7kM2x\n" //nolint:gosec // gitleaks:allow fake test fixture, not a real credential
		require.NoError(t, os.WriteFile(filepath.Join(dir, "creds.txt"), []byte(secret), 0o600))
		gitAddCommit(t, dir, "add creds")
		t.Chdir(dir)

		cfg := &config.Config{}
		cfg.Checks.GitleaksAllFiles = true
		cfg.CheckTimeouts.Gitleaks = 60
		c := NewGitleaksCheckWithFullConfig(shared.NewContext(), cfg)

		err := c.runGitleaks(context.Background(), []string{"creds.txt"})
		require.Error(t, err)
		assert.ErrorIs(t, err, prerrors.ErrSecretsFound)
	})
}
