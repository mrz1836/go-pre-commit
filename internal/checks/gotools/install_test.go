package gotools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/tools"
)

// toolCheckRepo makes a git repository with a Go module and one Go file, and
// changes into it for the rest of the test.
func toolCheckRepo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	initGitRepoAt(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, fileGoMod), []byte(testGoModContent), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, testFileMainGo), []byte("package main\n\nfunc main() {}\n"), 0o600))
	t.Chdir(dir)
}

// toolChecks are the checks that run a tool which go-pre-commit installs.
func toolChecks() []struct {
	name    string
	tool    string
	run     func(ctx context.Context) error
	canSkip bool
} {
	return []struct {
		name    string
		tool    string
		run     func(ctx context.Context) error
		canSkip bool
	}{
		{
			name:    "lint",
			tool:    "golangci-lint",
			run:     func(ctx context.Context) error { return NewLintCheck().Run(ctx, []string{testFileMainGo}) },
			canSkip: true,
		},
		{
			name: "gitleaks",
			tool: "gitleaks",
			run:  func(ctx context.Context) error { return NewGitleaksCheck().Run(ctx, []string{testFileMainGo}) },
		},
		{
			name: "fumpt",
			tool: "gofumpt",
			run:  func(ctx context.Context) error { return NewFumptCheck().Run(ctx, []string{testFileMainGo}) },
		},
	}
}

func TestToolChecksAutoInstallDisabled(t *testing.T) {
	makeToolsUnavailable(t) // also sets GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false
	t.Setenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION", "v2.15.0")
	t.Setenv("GO_PRE_COMMIT_GITLEAKS_VERSION", "v8.30.1")
	t.Setenv("GO_PRE_COMMIT_FUMPT_VERSION", "v0.9.2")
	toolCheckRepo(t)

	wantCommands := map[string]string{
		"golangci-lint": "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0",
		"gitleaks":      "go install github.com/zricethezav/gitleaks/v8@v8.30.1",
		"gofumpt":       "go install mvdan.cc/gofumpt@v0.9.2",
	}

	for _, tc := range toolChecks() {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(t.Context())

			var checkErr *prerrors.CheckError
			require.ErrorAs(t, err, &checkErr)
			require.ErrorIs(t, err, tools.ErrAutoInstallDisabled)
			require.ErrorIs(t, err, prerrors.ErrToolNotFound)
			assert.Contains(t, checkErr.Message, tc.tool+" not found")
			assert.Contains(t, checkErr.Message, tools.EnvAutoInstallTools+"=false")
			assert.Equal(t, "Install "+tc.tool+" with: "+wantCommands[tc.tool], checkErr.Suggestion)
			assert.Contains(t, checkErr.Output, wantCommands[tc.tool])
			assert.Equal(t, tc.canSkip, checkErr.CanSkip)
		})
	}
}

func TestToolChecksReportFailedInstall(t *testing.T) {
	// An invalid version stops each release install at its first step, before
	// any download, and golangci-lint's go install fallback finds no go on
	// PATH, so the installs fail without reaching the network.
	makeToolsUnavailable(t)
	t.Setenv(tools.EnvAutoInstallTools, "true")
	t.Setenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION", "not-a-version")
	t.Setenv("GO_PRE_COMMIT_GITLEAKS_VERSION", "not-a-version")
	toolCheckRepo(t)

	for _, tc := range toolChecks()[:2] { // golangci-lint and gitleaks
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(t.Context())

			var checkErr *prerrors.CheckError
			require.ErrorAs(t, err, &checkErr)
			require.ErrorIs(t, err, tools.ErrInstallFailed)
			require.ErrorIs(t, err, tools.ErrInvalidVersion)
			assert.Equal(t, tc.tool+" not found, and the automatic install did not make it available", checkErr.Message)
			// The cause names the tool, the version, and the failed step.
			assert.Contains(t, checkErr.Output, tc.tool+" not-a-version: check version:")
			assert.Contains(t, checkErr.Suggestion, "go install ")
		})
	}
}
