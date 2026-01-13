package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildAndRunWithVCSInfo tests building and running the binary with VCS info
func TestBuildAndRunWithVCSInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Logf("Failed to restore working directory: %v", chdirErr)
		}
	}()

	var buildPath string
	if strings.Contains(originalWD, "/cmd/go-pre-commit") {
		buildPath = "."
	} else {
		buildPath = "./cmd/go-pre-commit"
	}

	ctx := context.Background()

	testCases := []struct {
		name      string
		buildArgs []string
		runArgs   []string
		checkFunc func(t *testing.T, output string, exitCode int)
	}{
		{
			name:      "build with default settings and check version",
			buildArgs: []string{},
			runArgs:   []string{"--no-color", "--version"},
			checkFunc: func(t *testing.T, output string, exitCode int) {
				assert.Equal(t, 0, exitCode)
				assert.Contains(t, output, "version")
			},
		},
		{
			name:      "build with default settings and check help",
			buildArgs: []string{},
			runArgs:   []string{"--no-color", "--help"},
			checkFunc: func(t *testing.T, output string, exitCode int) {
				assert.Equal(t, 0, exitCode)
				assert.Contains(t, output, "Pre-commit")
			},
		},
		{
			name:      "build with default settings and check status",
			buildArgs: []string{},
			runArgs:   []string{"--no-color", "status"},
			checkFunc: func(t *testing.T, output string, exitCode int) {
				assert.Equal(t, 0, exitCode)
				// Status command should work
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testBinary := filepath.Join(t.TempDir(), "test-binary")

			// Build command
			buildCmdArgs := make([]string, 0, 3+len(tc.buildArgs)+1)
			buildCmdArgs = append(buildCmdArgs, "build", "-o", testBinary)
			buildCmdArgs = append(buildCmdArgs, tc.buildArgs...)
			buildCmdArgs = append(buildCmdArgs, buildPath)

			buildCmd := exec.CommandContext(ctx, "go", buildCmdArgs...) //nolint:gosec // Safe: controlled test

			var buildStdout, buildStderr bytes.Buffer
			buildCmd.Stdout = &buildStdout
			buildCmd.Stderr = &buildStderr

			err := buildCmd.Run()
			if err != nil {
				t.Logf("Build failed. stdout: %s, stderr: %s", buildStdout.String(), buildStderr.String())
			}
			require.NoError(t, err)

			// Run command
			runCmd := exec.CommandContext(ctx, testBinary, tc.runArgs...) //nolint:gosec // Safe: our binary
			output, err := runCmd.CombinedOutput()

			exitCode := 0
			if err != nil {
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					exitCode = exitErr.ExitCode()
				}
			}

			tc.checkFunc(t, string(output), exitCode)
		})
	}
}

// TestBinaryWithDifferentLdflagsScenarios tests various ldflags scenarios
func TestBinaryWithDifferentLdflagsScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Logf("Failed to restore working directory: %v", chdirErr)
		}
	}()

	var buildPath string
	if strings.Contains(originalWD, "/cmd/go-pre-commit") {
		buildPath = "."
	} else {
		buildPath = "./cmd/go-pre-commit"
	}

	ctx := context.Background()

	testCases := []struct {
		name    string
		ldflags string
		check   func(t *testing.T, output string)
	}{
		{
			name:    "production release build",
			ldflags: `-X "main.version=v2.0.0" -X "main.commit=abc123456789" -X "main.buildDate=2024-01-15T10:30:00Z"`,
			check: func(t *testing.T, output string) {
				assert.Contains(t, output, "v2.0.0")
				assert.Contains(t, output, "abc123")
			},
		},
		{
			name:    "development build with template placeholders",
			ldflags: `-X "main.version={{ .Version }}" -X "main.commit={{ .Commit }}" -X "main.buildDate={{ .BuildDate }}"`,
			check: func(t *testing.T, output string) {
				// Should fall back to build info, not show templates
				assert.NotContains(t, output, "{{")
				assert.NotContains(t, output, "}}")
				assert.Contains(t, output, "version")
			},
		},
		{
			name:    "partial ldflags - only version",
			ldflags: `-X "main.version=v1.5.0"`,
			check: func(t *testing.T, output string) {
				assert.Contains(t, output, "v1.5.0")
			},
		},
		{
			name:    "empty ldflags values",
			ldflags: `-X "main.version=" -X "main.commit=" -X "main.buildDate="`,
			check: func(t *testing.T, output string) {
				// Should fall back to build info
				assert.Contains(t, output, "version")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testBinary := filepath.Join(t.TempDir(), "test-ldflags")

			buildCmd := exec.CommandContext(ctx, "go", "build", "-ldflags", tc.ldflags, "-o", testBinary, buildPath) //nolint:gosec // Safe: controlled test

			var buildStdout, buildStderr bytes.Buffer
			buildCmd.Stdout = &buildStdout
			buildCmd.Stderr = &buildStderr

			err := buildCmd.Run()
			if err != nil {
				t.Logf("Build failed. stdout: %s, stderr: %s", buildStdout.String(), buildStderr.String())
			}
			require.NoError(t, err)

			// Run with version flag
			runCmd := exec.CommandContext(ctx, testBinary, "--no-color", "--version") //nolint:gosec // Safe: our binary
			output, err := runCmd.Output()
			require.NoError(t, err)

			tc.check(t, string(output))
		})
	}
}

// TestBinaryExitCodes tests various exit code scenarios
func TestBinaryExitCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Logf("Failed to restore working directory: %v", chdirErr)
		}
	}()

	var buildPath string
	if strings.Contains(originalWD, "/cmd/go-pre-commit") {
		buildPath = "."
	} else {
		buildPath = "./cmd/go-pre-commit"
	}

	ctx := context.Background()
	testBinary := filepath.Join(t.TempDir(), "test-exit-codes")

	// Build once
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", testBinary, buildPath) //nolint:gosec // Safe: controlled test
	var buildStdout, buildStderr bytes.Buffer
	buildCmd.Stdout = &buildStdout
	buildCmd.Stderr = &buildStderr

	err = buildCmd.Run()
	if err != nil {
		t.Logf("Build failed. stdout: %s, stderr: %s", buildStdout.String(), buildStderr.String())
	}
	require.NoError(t, err)

	testCases := []struct {
		name             string
		args             []string
		expectedExitCode int
	}{
		{
			name:             "help should exit 0",
			args:             []string{"--no-color", "--help"},
			expectedExitCode: 0,
		},
		{
			name:             "version should exit 0",
			args:             []string{"--no-color", "--version"},
			expectedExitCode: 0,
		},
		{
			name:             "status should exit 0",
			args:             []string{"--no-color", "status"},
			expectedExitCode: 0,
		},
		{
			name:             "invalid command should exit 1",
			args:             []string{"--no-color", "this-command-does-not-exist"},
			expectedExitCode: 1,
		},
		{
			name:             "show checks should exit 0",
			args:             []string{"--no-color", "run", "--show-checks"},
			expectedExitCode: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runCmd := exec.CommandContext(ctx, testBinary, tc.args...) //nolint:gosec // Safe: our binary
			err := runCmd.Run()

			if tc.expectedExitCode == 0 {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					assert.Equal(t, tc.expectedExitCode, exitErr.ExitCode())
				}
			}
		})
	}
}
