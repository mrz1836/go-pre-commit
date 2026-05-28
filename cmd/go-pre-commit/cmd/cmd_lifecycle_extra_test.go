package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/output"
	"github.com/mrz1836/go-pre-commit/internal/runner"
	"github.com/mrz1836/go-pre-commit/internal/update"
)

// lintCheckName is a check name reused across the run-option tests.
const lintCheckName = "lint"

// chdirToRepo changes the working directory to repoPath for the duration of the
// test (restored automatically by t.Chdir).
func chdirToRepo(t *testing.T, repoPath string) {
	t.Helper()
	t.Chdir(repoPath)
}

// ---- install command ----

func TestInstallCmd_RunE_FlagParsing(t *testing.T) {
	repo := setupTempGitRepo(t, true, true)
	chdirToRepo(t, repo)

	app := NewCLIApp("test", "commit", "date")
	builder := NewCommandBuilder(app)
	cmd := builder.BuildInstallCmd()

	// Invoke RunE directly; flag defaults (force=false, hook-type=[pre-commit])
	// drive the flag-parsing branches.
	out := captureCmdOutput(t, func() {
		require.NoError(t, cmd.RunE(cmd, nil))
	})
	assert.Contains(t, out, "Successfully installed")
	assert.FileExists(t, filepath.Join(repo, ".git", "hooks", hookTypePreCommit))
}

func TestInstallCmd_VerboseAndExistingHook(t *testing.T) {
	t.Run("verbose output", func(t *testing.T) {
		repo := setupTempGitRepo(t, true, true)
		chdirToRepo(t, repo)

		app := NewCLIApp("test", "commit", "date")
		builder := NewCommandBuilder(app)
		builder.app.config.Verbose = true

		out := captureCmdOutput(t, func() {
			require.NoError(t, builder.runInstallWithConfig(
				InstallConfig{HookTypes: []string{hookTypePreCommit}}, nil, nil))
		})
		assert.Contains(t, out, "Repository root:")
		assert.Contains(t, out, "Installing pre-commit hook")
	})

	t.Run("existing hook without force warns and continues", func(t *testing.T) {
		// An existing (non-managed) hook without --force must not fail the install;
		// it warns and skips that hook. handleExistingHook wraps os.ErrExist, so
		// install.go detects it with errors.Is (os.IsExist would not unwrap %w).
		repo := setupTempGitRepo(t, true, true)
		chdirToRepo(t, repo)

		hookPath := filepath.Join(repo, ".git", "hooks", hookTypePreCommit)
		require.NoError(t, os.MkdirAll(filepath.Dir(hookPath), 0o750))
		require.NoError(t, os.WriteFile(hookPath, []byte("#!/bin/sh\necho existing\n"), 0o600))

		app := NewCLIApp("test", "commit", "date")
		builder := NewCommandBuilder(app)

		out := captureCmdOutput(t, func() {
			require.NoError(t, builder.runInstallWithConfig(
				InstallConfig{Force: false, HookTypes: []string{hookTypePreCommit}}, nil, nil))
		})
		assert.Contains(t, out, "Hook already exists")
		assert.Contains(t, out, "No hooks were installed")
	})
}

// ---- uninstall command ----

func TestUninstallCmd_RunE(t *testing.T) {
	t.Run("flag parsing with no installed hooks reports not found", func(t *testing.T) {
		repo := setupTempGitRepo(t, true, true)
		chdirToRepo(t, repo)

		app := NewCLIApp("test", "commit", "date")
		builder := NewCommandBuilder(app)
		cmd := builder.BuildUninstallCmd()

		out := captureCmdOutput(t, func() {
			require.NoError(t, cmd.RunE(cmd, nil))
		})
		assert.Contains(t, out, "not found or not managed")
	})

	t.Run("empty hook list reports none uninstalled", func(t *testing.T) {
		repo := setupTempGitRepo(t, true, true)
		chdirToRepo(t, repo)

		app := NewCLIApp("test", "commit", "date")
		builder := NewCommandBuilder(app)
		out := captureCmdOutput(t, func() {
			require.NoError(t, builder.runUninstallWithHooks([]string{}, nil, nil))
		})
		assert.Contains(t, out, "No hooks were uninstalled")
	})

	t.Run("verbose mixed uninstalled and not-found", func(t *testing.T) {
		repo := setupTempGitRepo(t, true, true)
		chdirToRepo(t, repo)

		// Install a pre-commit hook first so it can be uninstalled.
		app := NewCLIApp("test", "commit", "date")
		builder := NewCommandBuilder(app)
		require.NoError(t, builder.runInstallWithConfig(
			InstallConfig{HookTypes: []string{hookTypePreCommit}}, nil, nil))

		builder.app.config.Verbose = true
		out := captureCmdOutput(t, func() {
			require.NoError(t, builder.runUninstallWithHooks(
				[]string{hookTypePreCommit, hookTypePrePush}, nil, nil))
		})
		assert.Contains(t, out, "Uninstalling pre-commit hook")
		assert.Contains(t, out, "Successfully uninstalled hooks")
		assert.Contains(t, out, "not found or not managed")
	})

	t.Run("no git repository errors", func(t *testing.T) {
		chdirToRepo(t, t.TempDir())

		app := NewCLIApp("test", "commit", "date")
		builder := NewCommandBuilder(app)
		err := builder.runUninstallWithHooks([]string{hookTypePreCommit}, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find git repository")
	})
}

// ---- buildRunnerOptions / run helpers ----

func TestBuildRunnerOptions_CheckSelection(t *testing.T) {
	formatter := output.NewDefault()

	t.Run("positional arg selects single check", func(t *testing.T) {
		opts := buildRunnerOptions(RunConfig{}, []string{lintCheckName}, nil, formatter)
		assert.Equal(t, []string{lintCheckName}, opts.OnlyChecks)
		assert.Empty(t, opts.SkipChecks)
	})

	t.Run("only flag", func(t *testing.T) {
		opts := buildRunnerOptions(RunConfig{OnlyChecks: []string{"fmt", lintCheckName}}, nil, nil, formatter)
		assert.Equal(t, []string{"fmt", lintCheckName}, opts.OnlyChecks)
	})

	t.Run("skip flag", func(t *testing.T) {
		opts := buildRunnerOptions(RunConfig{SkipChecks: []string{"gitleaks"}}, nil, nil, formatter)
		assert.Equal(t, []string{"gitleaks"}, opts.SkipChecks)
		assert.Empty(t, opts.OnlyChecks)
	})

	t.Run("positional arg takes precedence over only", func(t *testing.T) {
		opts := buildRunnerOptions(RunConfig{OnlyChecks: []string{"fmt"}}, []string{lintCheckName}, nil, formatter)
		assert.Equal(t, []string{lintCheckName}, opts.OnlyChecks)
	})
}

func TestBuildRunnerOptions_ProgressCallback(t *testing.T) {
	t.Run("callback set when progress enabled", func(t *testing.T) {
		var opts runner.Options
		// Build the formatter and exercise the callback inside the capture so the
		// formatter binds to the redirected streams.
		out := captureCmdOutput(t, func() {
			formatter := output.NewDefault()
			opts = buildRunnerOptions(
				RunConfig{ShowProgress: true, Quiet: false, Parallel: 2, FailFast: true, GracefulDegradation: true, DebugTimeout: true},
				nil, []string{"a.go"}, formatter)
			require.NotNil(t, opts.ProgressCallback)

			// Exercise each status branch of the callback.
			opts.ProgressCallback(lintCheckName, "running", time.Second)
			opts.ProgressCallback(lintCheckName, "passed", time.Second)
			opts.ProgressCallback(lintCheckName, "failed", time.Second)
			opts.ProgressCallback(lintCheckName, "skipped", time.Second)
			opts.ProgressCallback(lintCheckName, "unknown-status", time.Second)
		})
		assert.Equal(t, 2, opts.Parallel)
		assert.True(t, opts.FailFast)
		assert.True(t, opts.GracefulDegradation)
		assert.True(t, opts.DebugTimeout)
		assert.Equal(t, []string{"a.go"}, opts.Files)
		assert.NotEmpty(t, out)
	})

	t.Run("callback nil when quiet", func(t *testing.T) {
		formatter := output.NewDefault()
		opts := buildRunnerOptions(RunConfig{ShowProgress: true, Quiet: true}, nil, nil, formatter)
		assert.Nil(t, opts.ProgressCallback)
	})

	t.Run("callback nil when progress disabled", func(t *testing.T) {
		formatter := output.NewDefault()
		opts := buildRunnerOptions(RunConfig{ShowProgress: false}, nil, nil, formatter)
		assert.Nil(t, opts.ProgressCallback)
	})
}

func TestIsProgressLine(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"Running lint check", true},
		{"Checking files", true},
		{"Analyzing code", true},
		{"42 files linted", true},
		{"some error: boom", false},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, isProgressLine(tt.line), tt.line)
	}
}

// ---- status command ----

func TestStatusCmd_Verbose(t *testing.T) {
	repo := setupTempGitRepoForStatus(t, true, true)
	chdirToRepo(t, repo)

	// Install a hook so the "found hooks" + verbose detail branches run.
	app := NewCLIApp("test", "commit", "date")
	builder := NewCommandBuilder(app)
	require.NoError(t, builder.runInstallWithConfig(
		InstallConfig{HookTypes: []string{hookTypePreCommit}}, nil, nil))

	builder.app.config.Verbose = true
	out := captureCmdOutput(t, func() {
		require.NoError(t, builder.runStatus(nil, nil))
	})
	assert.Contains(t, out, "Repository root:")
	assert.Contains(t, out, "Go Pre-commit System Status")
	assert.Contains(t, out, "Checks enabled")
	assert.Contains(t, out, "Timeout:")
}

func TestStatusCmd_NoHooks(t *testing.T) {
	repo := setupTempGitRepoForStatus(t, true, true)
	chdirToRepo(t, repo)

	app := NewCLIApp("test", "commit", "date")
	builder := NewCommandBuilder(app)
	out := captureCmdOutput(t, func() {
		require.NoError(t, builder.runStatus(nil, nil))
	})
	assert.Contains(t, out, "No hooks are currently installed")
}

// ---- root: SetUpdateChan + PersistentPostRunE ----

func postRunE(t *testing.T, builder *CommandBuilder, cmdName string) error {
	t.Helper()
	root := builder.BuildRootCmd()
	require.NotNil(t, root.PersistentPostRunE)
	return root.PersistentPostRunE(&cobra.Command{Use: cmdName}, nil)
}

func TestSetUpdateChan(t *testing.T) {
	app := NewCLIApp("1.0.0", "c", "d")
	ch := make(chan *update.CheckResult, 1)
	app.SetUpdateChan(ch)
	assert.NotNil(t, app.updateChan)

	app.SetUpdateChan(nil)
	assert.Nil(t, app.updateChan)
}

func TestPersistentPostRunE(t *testing.T) {
	t.Run("upgrade command skips banner", func(t *testing.T) {
		app := NewCLIApp("1.0.0", "c", "d")
		ch := make(chan *update.CheckResult, 1)
		ch <- &update.CheckResult{UpdateAvailable: true, CurrentVersion: "1.0.0", LatestVersion: "2.0.0"}
		app.SetUpdateChan(ch)
		builder := NewCommandBuilder(app)

		require.NoError(t, postRunE(t, builder, "upgrade"))
		// Channel not drained because upgrade short-circuits before the select.
		assert.Len(t, ch, 1)
	})

	t.Run("nil channel skips", func(t *testing.T) {
		app := NewCLIApp("1.0.0", "c", "d")
		builder := NewCommandBuilder(app)
		require.NoError(t, postRunE(t, builder, "run"))
	})

	t.Run("update available shows banner", func(t *testing.T) {
		app := NewCLIApp("1.0.0", "c", "d")
		ch := make(chan *update.CheckResult, 1)
		ch <- &update.CheckResult{UpdateAvailable: true, CurrentVersion: "1.0.0", LatestVersion: "2.0.0"}
		app.SetUpdateChan(ch)
		builder := NewCommandBuilder(app)

		out := captureCmdOutput(t, func() {
			require.NoError(t, postRunE(t, builder, "run"))
		})
		assert.Contains(t, out, "2.0.0")
	})

	t.Run("nil result shows no banner", func(t *testing.T) {
		app := NewCLIApp("1.0.0", "c", "d")
		ch := make(chan *update.CheckResult, 1)
		ch <- nil
		app.SetUpdateChan(ch)
		builder := NewCommandBuilder(app)
		require.NoError(t, postRunE(t, builder, "run"))
	})

	t.Run("timeout when no result", func(t *testing.T) {
		app := NewCLIApp("1.0.0", "c", "d")
		ch := make(chan *update.CheckResult) // never sent
		app.SetUpdateChan(ch)
		builder := NewCommandBuilder(app)

		start := time.Now()
		require.NoError(t, postRunE(t, builder, "run"))
		assert.GreaterOrEqual(t, time.Since(start), 400*time.Millisecond)
	})
}
