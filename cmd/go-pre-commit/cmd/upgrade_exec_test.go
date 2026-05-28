package cmd

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withFakeBinary creates an executable shell script named binName whose body is
// the supplied script, and sets PATH to contain only that directory so the
// command under test resolves to the fake. Skips on Windows where shell scripts
// are not directly executable.
func withFakeBinary(t *testing.T, binName, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake shell-script binary not supported on windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, binName)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o700)) //nolint:gosec // fake binary must be executable
	t.Setenv("PATH", dir)
}

// withEmptyPath points PATH at an empty directory so tool lookups fail.
func withEmptyPath(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
}

func TestGetGoPath_Branches(t *testing.T) {
	t.Run("returns gopath/bin from go env", func(t *testing.T) {
		withFakeBinary(t, "go", `echo /custom/gopath`)
		got, err := GetGoPath()
		require.NoError(t, err)
		assert.Equal(t, "/custom/gopath/bin", got)
	})

	t.Run("falls back to home when gopath empty", func(t *testing.T) {
		withFakeBinary(t, "go", `echo ""`)
		home := t.TempDir()
		t.Setenv("HOME", home)
		got, err := GetGoPath()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, "go", "bin"), filepath.Clean(got))
	})

	t.Run("errors when go is unavailable", func(t *testing.T) {
		withEmptyPath(t)
		_, err := GetGoPath()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get GOPATH")
	})
}

func TestGetBinaryLocation_Branches(t *testing.T) {
	t.Run("finds binary on path", func(t *testing.T) {
		name := "go-pre-commit"
		if runtime.GOOS == "windows" {
			name = "go-pre-commit.exe"
		}
		withFakeBinary(t, name, `exit 0`)
		got, err := GetBinaryLocation()
		require.NoError(t, err)
		assert.Contains(t, got, "go-pre-commit")
	})

	t.Run("errors when binary missing", func(t *testing.T) {
		withEmptyPath(t)
		_, err := GetBinaryLocation()
		require.Error(t, err)
	})
}

func TestDefaultGoInstall(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		withFakeBinary(t, "go", `exit 0`)
		require.NoError(t, defaultGoInstall(context.Background(), "example.com/pkg@latest"))
	})

	t.Run("propagates failure", func(t *testing.T) {
		withFakeBinary(t, "go", `exit 1`)
		require.Error(t, defaultGoInstall(context.Background(), "example.com/pkg@latest"))
	})
}

func TestReinstallHooks(t *testing.T) {
	t.Run("no hooks installed", func(t *testing.T) {
		isolateInTempGitRepo(t)
		app := NewCLIApp("1.0.0", "c", "d")
		builder := NewCommandBuilder(app)

		out := captureCmdOutput(t, func() {
			require.NoError(t, builder.reinstallHooks())
		})
		assert.Contains(t, out, "No hooks were installed")
	})

	t.Run("reinstalls existing hook", func(t *testing.T) {
		isolateInTempGitRepo(t)
		// Install a hook so reinstall has something to do. config.Load needs a
		// config file; create a minimal one in the temp repo.
		ghDir := filepath.Join(".github")
		require.NoError(t, os.MkdirAll(ghDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(ghDir, ".env.base"), []byte("ENABLE_GO_PRE_COMMIT=true\n"), 0o600))

		app := NewCLIApp("1.0.0", "c", "d")
		builder := NewCommandBuilder(app)
		require.NoError(t, builder.runInstallWithConfig(
			InstallConfig{HookTypes: []string{hookTypePreCommit}}, nil, nil))

		out := captureCmdOutput(t, func() {
			require.NoError(t, builder.reinstallHooks())
		})
		assert.Contains(t, out, "Reinstalled 1 hook(s)")
	})

	t.Run("errors outside git repo", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Chdir(t.TempDir())
		app := NewCLIApp("1.0.0", "c", "d")
		builder := NewCommandBuilder(app)

		err := builder.reinstallHooks()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find git repository")
	})
}

func TestCheckHookCompatibility(t *testing.T) {
	t.Run("silent outside git repo", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Chdir(t.TempDir())
		app := NewCLIApp("1.0.0", "c", "d")
		builder := NewCommandBuilder(app)

		out := captureCmdOutput(t, func() {
			require.NotPanics(t, builder.checkHookCompatibility)
		})
		assert.Empty(t, out)
	})

	t.Run("verbose message when hooks present", func(t *testing.T) {
		isolateInTempGitRepo(t)
		ghDir := filepath.Join(".github")
		require.NoError(t, os.MkdirAll(ghDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(ghDir, ".env.base"), []byte("ENABLE_GO_PRE_COMMIT=true\n"), 0o600))

		app := NewCLIApp("1.0.0", "c", "d")
		builder := NewCommandBuilder(app)
		require.NoError(t, builder.runInstallWithConfig(
			InstallConfig{HookTypes: []string{hookTypePreCommit}}, nil, nil))

		builder.app.config.Verbose = true
		out := captureCmdOutput(t, func() {
			builder.checkHookCompatibility()
		})
		assert.Contains(t, out, "Existing hooks detected")
	})

	t.Run("quiet when hooks present but not verbose", func(t *testing.T) {
		isolateInTempGitRepo(t)
		ghDir := filepath.Join(".github")
		require.NoError(t, os.MkdirAll(ghDir, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(ghDir, ".env.base"), []byte("ENABLE_GO_PRE_COMMIT=true\n"), 0o600))

		app := NewCLIApp("1.0.0", "c", "d")
		builder := NewCommandBuilder(app)
		require.NoError(t, builder.runInstallWithConfig(
			InstallConfig{HookTypes: []string{hookTypePreCommit}}, nil, nil))

		out := captureCmdOutput(t, func() {
			builder.checkHookCompatibility()
		})
		assert.Empty(t, out)
	})
}
