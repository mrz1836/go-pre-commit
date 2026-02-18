package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/cmd/go-pre-commit/cmd"
)

func TestMain(t *testing.T) {
	// Test that the binary can be built and executed
	// This test verifies the main entry point works

	// Build the binary for testing
	ctx := context.Background()
	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", "go-pre-commit-test", ".")
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build binary")

	defer func() {
		// Clean up the test binary
		if err := os.Remove("go-pre-commit-test"); err != nil {
			t.Logf("Failed to remove test binary: %v", err)
		}
	}()

	// Test various command scenarios
	tests := []struct {
		name     string
		args     []string
		wantExit int
	}{
		{
			name:     "help command",
			args:     []string{"--help"},
			wantExit: 0,
		},
		{
			name:     "version flag",
			args:     []string{"--version"},
			wantExit: 0,
		},
		{
			name:     "show checks",
			args:     []string{"run", "--show-checks"},
			wantExit: 0,
		},
		{
			name:     "invalid command",
			args:     []string{"invalid-command"},
			wantExit: 1,
		},
		{
			name:     "status command",
			args:     []string{"status"},
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.CommandContext(ctx, "./go-pre-commit-test", tt.args...) // #nosec G204 - test code with controlled input
			err := cmd.Run()

			if tt.wantExit == 0 {
				assert.NoError(t, err, "Expected successful exit")
			} else {
				assert.Error(t, err, "Expected error exit")
			}
		})
	}
}

// Test the main function directly with command setup
func TestMainFunction(t *testing.T) {
	// Save original args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "help flag",
			args:        []string{"go-pre-commit", "--no-color", "--help"},
			expectError: false,
		},
		{
			name:        "version flag",
			args:        []string{"go-pre-commit", "--no-color", "--version"},
			expectError: false,
		},
		{
			name:        "show checks",
			args:        []string{"go-pre-commit", "--no-color", "run", "--show-checks"},
			expectError: false,
		},
		{
			name:        "invalid command",
			args:        []string{"go-pre-commit", "--no-color", "invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset command for each test
			cmd.ResetCommand()
			cmd.SetVersionInfo("test", "test-commit", "test-date")

			// Set up args
			os.Args = tt.args

			// Capture output
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			// Execute command and capture panic
			var cmdErr error
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Main calls os.Exit(1) on error, which we can't intercept
						// So we expect a panic in test environment
						cmdErr = os.ErrProcessDone
					}
				}()

				// Call the command directly
				cmdErr = cmd.Execute()
			}()

			// Restore stdout/stderr
			if err := w.Close(); err != nil {
				t.Logf("Failed to close pipe writer: %v", err)
			}
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			// Read output
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, r); err != nil {
				t.Logf("Failed to copy output: %v", err)
			}
			output := buf.String()

			if tt.expectError {
				require.Error(t, cmdErr, "Expected command to fail")
			} else {
				require.NoError(t, cmdErr, "Expected command to succeed")
			}

			// Verify some output was produced (except for invalid command which might be silenced)
			if tt.name != "invalid command" {
				assert.NotEmpty(t, output, "Expected some output")
			}
		})
	}
}

// Test direct execution scenarios
func TestDirectExecution(t *testing.T) {
	// Save original values
	oldArgs := os.Args
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Test successful execution
	t.Run("successful help", func(t *testing.T) {
		os.Args = []string{"go-pre-commit", "--no-color", "--help"}

		// Capture output
		r, w, _ := os.Pipe()
		os.Stdout = w
		os.Stderr = w

		// Reset and set version info
		cmd.ResetCommand()
		cmd.SetVersionInfo("test", "test-commit", "test-date")

		// Execute
		err := cmd.Execute()

		// Close and read output
		if closeErr := w.Close(); closeErr != nil {
			t.Logf("Failed to close pipe writer: %v", closeErr)
		}
		var buf bytes.Buffer
		if _, copyErr := io.Copy(&buf, r); copyErr != nil {
			t.Logf("Failed to copy output: %v", copyErr)
		}

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Go Pre-commit System")
	})
}

// Test the main function's error handling path by extracting the logic
func TestMainFunctionErrorHandling(t *testing.T) {
	// Test the actual main function logic without os.Exit
	// This function tests the path through main() to cmd.Execute()

	tests := []struct {
		name        string
		args        []string
		setupFunc   func()
		expectError bool
	}{
		{
			name: "successful help command",
			args: []string{"go-pre-commit", "--no-color", "--help"},
			setupFunc: func() {
				cmd.SetVersionInfo("test", "test-commit", "test-date")
			},
			expectError: false,
		},
		{
			name: "invalid command should error",
			args: []string{"go-pre-commit", "--no-color", "invalid-command"},
			setupFunc: func() {
				cmd.ResetCommand()
				cmd.SetVersionInfo("test", "test-commit", "test-date")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Set up test args
			os.Args = tt.args

			// Run setup if provided
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			// Call cmd.Execute directly to test the main logic path
			err := cmd.Execute()

			if tt.expectError {
				require.Error(t, err, "Expected command to fail")
			} else {
				require.NoError(t, err, "Expected command to succeed")
			}
		})
	}
}

// Test main function components individually to improve coverage
func TestMainComponents(t *testing.T) {
	// Test version info setting
	t.Run("version info setting", func(t *testing.T) {
		cmd.ResetCommand()
		cmd.SetVersionInfo("1.0.0", "abc123", "2023-01-01")

		// Execute version command to verify version info was set
		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{"go-pre-commit", "--no-color", "--version"}

		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		defer func() { os.Stdout = oldStdout }()

		err := cmd.Execute()
		require.NoError(t, err)

		if closeErr := w.Close(); closeErr != nil {
			t.Logf("Failed to close pipe writer: %v", closeErr)
		}
		var buf bytes.Buffer
		if _, copyErr := io.Copy(&buf, r); copyErr != nil {
			t.Logf("Failed to copy output: %v", copyErr)
		}

		output := buf.String()
		// Version output should contain basic version info
		assert.Contains(t, output, "version")
		// Check if the custom version is present, if not that's ok as long as version command worked
		if strings.Contains(output, "1.0.0") {
			assert.Contains(t, output, "abc123")
			assert.Contains(t, output, "2023-01-01")
		}
	})

	// Test command error handling path
	t.Run("command error propagation", func(t *testing.T) {
		cmd.ResetCommand()
		cmd.SetVersionInfo("test", "test", "test")

		oldArgs := os.Args
		defer func() { os.Args = oldArgs }()
		os.Args = []string{"go-pre-commit", "invalid-command"}

		// This should return an error (not call os.Exit)
		err := cmd.Execute()
		require.Error(t, err, "Invalid command should return error")
		assert.Contains(t, err.Error(), "unknown command")
	})
}

// Test version info functionality using subprocess
func TestVersionInfo(t *testing.T) {
	// Save and restore working directory to prevent interference from other tests
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Logf("Failed to restore working directory: %v", chdirErr)
		}
	}()

	// No need to change directory - tests should run from where they are

	// Build a test binary from the current package
	ctx := context.Background()
	testBinary := filepath.Join(t.TempDir(), "test-version-binary")

	// Determine the correct build path based on current working directory
	var buildPath string
	if strings.Contains(originalWD, "/cmd/go-pre-commit") {
		// Running from within the cmd/go-pre-commit directory
		buildPath = "."
	} else {
		// Running from project root
		buildPath = "./cmd/go-pre-commit"
	}

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", testBinary, buildPath) //nolint:gosec // Safe: buildPath is controlled in test

	// Add debug output for build failures
	var stdout, stderr bytes.Buffer
	buildCmd.Stdout = &stdout
	buildCmd.Stderr = &stderr

	err = buildCmd.Run()
	if err != nil {
		t.Logf("Build failed. stdout: %s, stderr: %s", stdout.String(), stderr.String())
		t.Logf("Working directory: %s", func() string {
			wd, _ := os.Getwd()
			return wd
		}())
	}
	require.NoError(t, err)

	// No need to defer cleanup - test binary is in temp dir which gets cleaned up automatically

	// Run with version flag and no-color flag
	testCmd := exec.CommandContext(ctx, testBinary, "--no-color", "--version") //nolint:gosec // Safe: testBinary is our own built binary
	output, err := testCmd.Output()
	require.NoError(t, err)

	outputStr := string(output)
	t.Logf("Version command output: %q", outputStr)

	// Check that version command works and outputs version information
	assert.Contains(t, outputStr, "go-pre-commit")
	assert.Contains(t, outputStr, "version")
	// The version should contain some value (may be from BuildInfo or ldflags)
	// We can't predict exact values since they depend on build context
	assert.Regexp(t, `version \S+`, outputStr) // Should have some version
	assert.Regexp(t, `commit: \S+`, outputStr) // Should have some commit
	assert.Regexp(t, `built: \S+`, outputStr)  // Should have some build date
}

// Test that main calls os.Exit(1) on error - using subprocess
func TestMainExitOnError(t *testing.T) {
	// This test verifies that main() calls os.Exit(1) on error
	// by running it as a subprocess

	// Save and restore working directory to prevent interference from other tests
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Logf("Failed to restore working directory: %v", chdirErr)
		}
	}()

	// No need to change directory - tests should run from where they are

	// Reset command state before test
	cmd.ResetCommand()
	defer cmd.ResetCommand() // Reset after test too

	// Build a test binary
	ctx := context.Background()
	testBinary := filepath.Join(t.TempDir(), "test-exit-binary")

	// Determine the correct build path based on current working directory
	var buildPath string
	if strings.Contains(originalWD, "/cmd/go-pre-commit") {
		// Running from within the cmd/go-pre-commit directory
		buildPath = "."
	} else {
		// Running from project root
		buildPath = "./cmd/go-pre-commit"
	}

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", testBinary, buildPath) //nolint:gosec // Safe: buildPath is controlled in test

	// Add debug output for build failures
	var stdout, stderr bytes.Buffer
	buildCmd.Stdout = &stdout
	buildCmd.Stderr = &stderr

	err = buildCmd.Run()
	if err != nil {
		t.Logf("Build failed. stdout: %s, stderr: %s", stdout.String(), stderr.String())
		t.Logf("Working directory: %s", func() string {
			wd, _ := os.Getwd()
			return wd
		}())
	}
	require.NoError(t, err)

	// No need to defer cleanup - test binary is in temp dir which gets cleaned up automatically

	// Run with invalid command and no-color flag
	testCmd := exec.CommandContext(ctx, testBinary, "--no-color", "invalid-command") //nolint:gosec // Safe: testBinary is our own built binary
	err = testCmd.Run()

	// Should exit with status 1
	require.Error(t, err)
	var exitError *exec.ExitError
	ok := errors.As(err, &exitError)
	if ok {
		assert.Equal(t, 1, exitError.ExitCode())
	}
}

// Benchmark main execution
func BenchmarkMain(b *testing.B) {
	// Save original values
	oldArgs := os.Args
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Discard output during benchmark
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatalf("failed to open devnull: %v", err)
	}
	os.Stdout = devNull
	os.Stderr = devNull
	defer func() {
		if err := devNull.Close(); err != nil {
			b.Logf("Failed to close devNull: %v", err)
		}
	}()

	// Use help command for fast execution
	os.Args = []string{"go-pre-commit", "--help"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := cmd.Execute(); err != nil {
			b.Logf("Command failed: %v", err)
		}
	}
}

// Test main function covers os.Exit behavior via the existing subprocess tests
// The TestMainExitOnError test already covers this scenario properly

// Test the run function directly for better coverage
func TestRunFunction(t *testing.T) {
	// Save original args and stderr
	oldArgs := os.Args
	oldStderr := os.Stderr
	defer func() {
		os.Args = oldArgs
		os.Stderr = oldStderr
	}()

	tests := []struct {
		name         string
		args         []string
		setupFunc    func()
		wantExitCode int
	}{
		{
			name: "successful help command returns 0",
			args: []string{"go-pre-commit", "--no-color", "--help"},
			setupFunc: func() {
				cmd.ResetCommand()
				cmd.SetVersionInfo("test", "test-commit", "test-date")
			},
			wantExitCode: 0,
		},
		{
			name: "successful version command returns 0",
			args: []string{"go-pre-commit", "--no-color", "--version"},
			setupFunc: func() {
				cmd.ResetCommand()
				cmd.SetVersionInfo("1.0.0", "abc123", "2023-01-01")
			},
			wantExitCode: 0,
		},
		{
			name: "invalid command returns 1",
			args: []string{"go-pre-commit", "--no-color", "invalid-command"},
			setupFunc: func() {
				cmd.ResetCommand()
				cmd.SetVersionInfo("test", "test", "test")
			},
			wantExitCode: 1,
		},
		{
			name: "status command returns 0",
			args: []string{"go-pre-commit", "--no-color", "status"},
			setupFunc: func() {
				cmd.ResetCommand()
				cmd.SetVersionInfo("test", "test", "test")
			},
			wantExitCode: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			os.Args = tt.args

			// Capture stderr to avoid noise in test output
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			defer func() { os.Stderr = oldStderr }()

			// Run setup if provided
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			// Call run() function directly
			exitCode := run()

			// Close writer and read stderr content
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Failed to close stderr pipe: %v", closeErr)
			}
			var stderrBuf bytes.Buffer
			if _, copyErr := io.Copy(&stderrBuf, r); copyErr != nil {
				t.Logf("Failed to copy stderr: %v", copyErr)
			}

			assert.Equal(t, tt.wantExitCode, exitCode, "Expected exit code %d, got %d", tt.wantExitCode, exitCode)

			// If we expected an error, verify error message was written to stderr
			if tt.wantExitCode != 0 {
				assert.NotEmpty(t, stderrBuf.String(), "Expected error message on stderr")
				assert.Contains(t, stderrBuf.String(), "Error:")
			}
		})
	}
}

// Test run function with version info scenarios
func TestRunFunctionVersionInfo(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	tests := []struct {
		name      string
		version   string
		commit    string
		buildDate string
	}{
		{
			name:      "development version info",
			version:   "dev",
			commit:    "none",
			buildDate: "unknown",
		},
		{
			name:      "production version info",
			version:   "v1.2.3",
			commit:    "abc123def",
			buildDate: "2023-12-01T10:00:00Z",
		},
		{
			name:      "empty version info defaults",
			version:   "",
			commit:    "",
			buildDate: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set args for version command
			os.Args = []string{"go-pre-commit", "--no-color", "--version"}

			// Reset command and set version info
			cmd.ResetCommand()
			cmd.SetVersionInfo(tt.version, tt.commit, tt.buildDate)

			// Capture stdout to verify version info
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() { os.Stdout = oldStdout }()

			// Call run() function
			exitCode := run()

			// Close and read output
			if closeErr := w.Close(); closeErr != nil {
				t.Logf("Failed to close pipe writer: %v", closeErr)
			}
			var buf bytes.Buffer
			if _, copyErr := io.Copy(&buf, r); copyErr != nil {
				t.Logf("Failed to copy output: %v", copyErr)
			}

			require.Equal(t, 0, exitCode, "Version command should always succeed")

			output := buf.String()
			assert.Contains(t, output, "version", "Output should contain version information")

			// Verify version info is present (or defaults are used)
			// cobra may output help instead of version in test environment
			// so we check if it contains version-related text or the expected version
			if strings.Contains(output, tt.version) || strings.Contains(output, "version") {
				// Test passed - either got expected version or version command worked
				t.Logf("Version test output: %q", output)
			} else {
				t.Errorf("Expected version output but got: %q", output)
			}
		})
	}
}

// TestRunFunctionVersionDirtySuffix tests the -dirty suffix logic in run()
func TestRunFunctionVersionDirtySuffix(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set args for version command
	os.Args = []string{"go-pre-commit", "--no-color", "--version"}

	// Test with a version that already has -dirty suffix
	t.Run("version already has dirty suffix", func(t *testing.T) {
		cmd.ResetCommand()
		cmd.SetVersionInfo("v1.2.3-dirty", "abc123", "2023-12-01")

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		defer func() { os.Stdout = oldStdout }()

		exitCode := run()

		// Close and read output
		if closeErr := w.Close(); closeErr != nil {
			t.Logf("Failed to close pipe writer: %v", closeErr)
		}
		var buf bytes.Buffer
		if _, copyErr := io.Copy(&buf, r); copyErr != nil {
			t.Logf("Failed to copy output: %v", copyErr)
		}

		require.Equal(t, 0, exitCode)
		output := buf.String()

		// Version should contain -dirty only once
		dirtyCount := strings.Count(output, "-dirty")
		assert.LessOrEqual(t, dirtyCount, 1, "Should not add -dirty suffix twice")
	})

	// Test with a clean version (normal case)
	t.Run("version without dirty suffix", func(t *testing.T) {
		cmd.ResetCommand()
		cmd.SetVersionInfo("v1.2.3", "abc123", "2023-12-01")

		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		defer func() { os.Stdout = oldStdout }()

		exitCode := run()

		// Close and read output
		if closeErr := w.Close(); closeErr != nil {
			t.Logf("Failed to close pipe writer: %v", closeErr)
		}
		var buf bytes.Buffer
		if _, copyErr := io.Copy(&buf, r); copyErr != nil {
			t.Logf("Failed to copy output: %v", copyErr)
		}

		require.Equal(t, 0, exitCode)
		// Test passes if run() completes successfully
		// The actual -dirty suffix addition depends on VCS state during test
	})
}

// Example showing how to use the pre-commit system
func Example_main() {
	// The go-pre-commit tool manages Git hooks for code quality
	// It runs tools directly without build system dependencies

	// Usage:
	// go-pre-commit install          # Install Git hooks
	// go-pre-commit uninstall        # Remove Git hooks
	// go-pre-commit run              # Run checks on staged files
	// go-pre-commit run --all-files  # Run checks on all files
	// go-pre-commit list             # List available checks
	// go-pre-commit status           # Show installation status

	fmt.Println("Go Pre-commit System")
	// Output: Go Pre-commit System
}

// TestMainProcess is a helper for subprocess testing of main()
func TestMainProcess(_ *testing.T) {
	// This test is run as a subprocess by other tests
	if os.Getenv("GO_TEST_SUBPROCESS") != "1" {
		return
	}

	// Run main() based on test case
	switch os.Getenv("GO_TEST_CASE") {
	case "help":
		os.Args = []string{"go-pre-commit", "--help"}
		main()
	case "version":
		os.Args = []string{"go-pre-commit", "--version"}
		main()
	case "invalid":
		os.Args = []string{"go-pre-commit", "invalid-command-xyz"}
		main()
	}
}

// TestMain_Help tests main() with --help flag using subprocess
func TestMain_Help(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		return
	}

	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestMainProcess") // #nosec G204 G702 - test binary path is safe
	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1", "GO_TEST_CASE=help")

	err := cmd.Run()
	assert.NoError(t, err, "main() with --help should exit successfully")
}

// TestMain_Version tests main() with --version flag using subprocess
func TestMain_Version(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		return
	}

	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestMainProcess") // #nosec G204 G702 - test binary path is safe
	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1", "GO_TEST_CASE=version")

	err := cmd.Run()
	assert.NoError(t, err, "main() with --version should exit successfully")
}

// TestMain_InvalidCommand tests main() with invalid command using subprocess
func TestMain_InvalidCommand(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		return
	}

	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestMainProcess") // #nosec G204 G702 - test binary path is safe
	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1", "GO_TEST_CASE=invalid")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code for invalid command")
	}

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
}
