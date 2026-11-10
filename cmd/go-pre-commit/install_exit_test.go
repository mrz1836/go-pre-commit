package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/testutil"
)

// otherHook is a hook another tool put in place
const otherHook = "#!/bin/sh\necho 'another tool'\n"

// TestRunInstallExitStatus checks the exit status of "go-pre-commit install"
// when a hook is already in place. A script runs install and trusts its exit
// status, so it must be non-zero whenever a requested hook was not installed.
func TestRunInstallExitStatus(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		existing     map[string]string // hook type -> content already in the hooks directory
		wantExitCode int
		wantOurs     []string          // hook types that must hold go-pre-commit's hook
		wantKept     map[string]string // hook types that must keep their content
		wantOutput   []string
	}{
		{
			name:         "another tool's hook without force exits 1 and keeps the hook",
			args:         []string{"install"},
			existing:     map[string]string{"pre-commit": otherHook},
			wantExitCode: 1,
			wantKept:     map[string]string{"pre-commit": otherHook},
			wantOutput: []string{
				"Hook already exists: pre-commit (use --force to overwrite)",
				"No hooks were installed",
				"Error: hooks not installed: pre-commit (use --force to overwrite)",
			},
		},
		{
			name:         "another tool's hook among others exits 1 and installs the rest",
			args:         []string{"install", "--hook-type", "pre-commit", "--hook-type", "pre-push"},
			existing:     map[string]string{"pre-commit": otherHook},
			wantExitCode: 1,
			wantOurs:     []string{"pre-push"},
			wantKept:     map[string]string{"pre-commit": otherHook},
			wantOutput: []string{
				"Hook already exists: pre-commit (use --force to overwrite)",
				"Successfully installed hooks: [pre-push]",
				"Error: hooks not installed: pre-commit (use --force to overwrite)",
			},
		},
		{
			name:         "no hook types exits 1",
			args:         []string{"install", "--hook-type", ""},
			wantExitCode: 1,
			wantOutput:   []string{"Error: no hook types given"},
		},
		{
			name:         "another tool's hook with force exits 0",
			args:         []string{"install", "--force"},
			existing:     map[string]string{"pre-commit": otherHook},
			wantExitCode: 0,
			wantOurs:     []string{"pre-commit"},
			wantOutput:   []string{"Successfully installed hooks: [pre-commit]"},
		},
		{
			name:         "reinstall over go-pre-commit's own hook exits 0",
			args:         []string{"install"},
			existing:     map[string]string{"pre-commit": "#!/bin/bash\n# Go Pre-commit Hook\n"},
			wantExitCode: 0,
			wantOurs:     []string{"pre-commit"},
			wantOutput:   []string{"Successfully installed hooks: [pre-commit]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooksDir := newInstallRepo(t)
			for hookType, content := range tt.existing {
				writeHook(t, filepath.Join(hooksDir, hookType), content)
			}

			exitCode, output := runCLI(t, append([]string{flagNoColor}, tt.args...)...)

			assert.Equal(t, tt.wantExitCode, exitCode, output)
			for _, want := range tt.wantOutput {
				assert.Contains(t, output, want)
			}
			for _, hookType := range tt.wantOurs {
				assert.Contains(t, readHook(t, filepath.Join(hooksDir, hookType)), "Go Pre-commit Hook",
					"%s must hold go-pre-commit's hook", hookType)
			}
			for hookType, content := range tt.wantKept {
				assert.Equal(t, content, readHook(t, filepath.Join(hooksDir, hookType)),
					"%s must be left as it was", hookType)
			}
		})
	}
}

// newInstallRepo creates a git repository with go-pre-commit enabled, isolated
// from the developer's git setup and environment, makes it the working
// directory for the test, and returns its hooks directory
func newInstallRepo(t *testing.T) string {
	t.Helper()
	testutil.IsolateEnv(t)           // config.Load writes the configuration into the process environment
	t.Setenv("NO_UPDATE_CHECK", "1") // keep the update notice off the network
	repo := testutil.NewGitRepo(t)

	githubDir := filepath.Join(repo, ".github")
	require.NoError(t, os.MkdirAll(githubDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(githubDir, ".env.base"), []byte("ENABLE_GO_PRE_COMMIT=true\n"), 0o600))
	t.Chdir(repo)

	return filepath.Join(repo, ".git", "hooks")
}

// runCLI runs the command line with args and returns its exit status and
// everything it wrote to stdout and stderr
func runCLI(t *testing.T, args ...string) (int, string) {
	t.Helper()

	oldArgs, oldStdout, oldStderr := os.Args, os.Stdout, os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Args = append([]string{binaryCLIName}, args...)
	os.Stdout, os.Stderr = w, w
	defer func() {
		os.Args, os.Stdout, os.Stderr = oldArgs, oldStdout, oldStderr
	}()

	exitCode := run()

	require.NoError(t, w.Close())
	var output bytes.Buffer
	_, err = io.Copy(&output, r)
	require.NoError(t, err)
	return exitCode, output.String()
}

func writeHook(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755)) // #nosec G306 - a hook must be executable
}

func readHook(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path) //nolint:gosec // test file path is controlled
	require.NoError(t, err)
	return string(content)
}
