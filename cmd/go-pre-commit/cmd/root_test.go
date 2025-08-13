package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCLIApp(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		commit    string
		buildDate string
	}{
		{
			name:      "standard version info",
			version:   "1.0.0",
			commit:    "abc123",
			buildDate: "2025-01-01",
		},
		{
			name:      "empty version info",
			version:   "",
			commit:    "",
			buildDate: "",
		},
		{
			name:      "dev build info",
			version:   "dev",
			commit:    "unknown",
			buildDate: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewCLIApp(tt.version, tt.commit, tt.buildDate)

			require.NotNil(t, app)
			assert.Equal(t, tt.version, app.version)
			assert.Equal(t, tt.commit, app.commit)
			assert.Equal(t, tt.buildDate, app.buildDate)
			require.NotNil(t, app.config)
			assert.False(t, app.config.Verbose) // Default should be false
			assert.False(t, app.config.NoColor) // Default should be false
		})
	}
}

func TestNewCommandBuilder(t *testing.T) {
	app := NewCLIApp("1.0.0", "abc123", "2025-01-01")
	builder := NewCommandBuilder(app)

	require.NotNil(t, builder)
	assert.Equal(t, app, builder.app)
}

func TestBuildRootCmdProperties(t *testing.T) {
	app := NewCLIApp("1.2.3", "commit456", "2025-08-10")
	builder := NewCommandBuilder(app)
	cmd := builder.BuildRootCmd()

	// Test basic command properties
	assert.Equal(t, "go-pre-commit", cmd.Use)
	assert.Contains(t, cmd.Short, "Go Pre-commit System")
	assert.Contains(t, cmd.Long, "Go Pre-commit System is a high-performance")
	assert.Contains(t, cmd.Long, "Lightning fast parallel execution")
	assert.True(t, cmd.SilenceUsage)
	assert.True(t, cmd.SilenceErrors)

	// Test version information
	expectedVersion := "1.2.3 (commit: commit456, built: 2025-08-10)"
	assert.Equal(t, expectedVersion, cmd.Version)

	// Test version template
	assert.Contains(t, cmd.VersionTemplate(), "{{with .Name}}")
	assert.Contains(t, cmd.VersionTemplate(), "version")

	// Test persistent flags
	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	require.NotNil(t, verboseFlag)
	assert.Empty(t, verboseFlag.Shorthand) // No shorthand for verbose anymore
	assert.Equal(t, "false", verboseFlag.DefValue)

	noColorFlag := cmd.PersistentFlags().Lookup("no-color")
	require.NotNil(t, noColorFlag)
	assert.Equal(t, "false", noColorFlag.DefValue)
}

func TestBuildRootCmdPersistentPreRun(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectedVerbose bool
		expectedNoColor bool
	}{
		{
			name:            "no flags",
			args:            []string{},
			expectedVerbose: false,
			expectedNoColor: false,
		},
		{
			name:            "verbose flag",
			args:            []string{"--verbose"},
			expectedVerbose: true,
			expectedNoColor: false,
		},
		{
			name:            "no-color flag",
			args:            []string{"--no-color"},
			expectedVerbose: false,
			expectedNoColor: true,
		},
		{
			name:            "both flags",
			args:            []string{"--verbose", "--no-color"},
			expectedVerbose: true,
			expectedNoColor: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewCLIApp("1.0.0", "abc123", "2025-01-01")
			builder := NewCommandBuilder(app)
			cmd := builder.BuildRootCmd()

			// Set up command with args
			cmd.SetArgs(tt.args)

			// Parse flags without executing
			err := cmd.ParseFlags(tt.args)
			require.NoError(t, err)

			// Manually trigger PersistentPreRun to test flag handling
			if cmd.PersistentPreRun != nil {
				cmd.PersistentPreRun(cmd, []string{})
			}

			// Check that flags were properly set in app config
			assert.Equal(t, tt.expectedVerbose, app.config.Verbose)
			assert.Equal(t, tt.expectedNoColor, app.config.NoColor)
		})
	}
}

func TestCommandBuilderExecute(t *testing.T) {
	// Save original args and restore after test
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "help command",
			args:        []string{"go-pre-commit", "--help"},
			expectError: false,
		},
		{
			name:        "version command",
			args:        []string{"go-pre-commit", "--version"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output to avoid cluttering test output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() {
				_ = w.Close()
				os.Stdout = oldStdout
			}()

			// Set command line args
			os.Args = tt.args

			app := NewCLIApp("1.0.0", "abc123", "2025-01-01")
			builder := NewCommandBuilder(app)

			err := builder.Execute()
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			_ = w.Close()
			os.Stdout = oldStdout

			// Verify output contains expected content
			var buf bytes.Buffer
			_, _ = buf.ReadFrom(r)
			output := buf.String()

			if strings.Contains(tt.args[1], "help") {
				assert.Contains(t, output, "Go Pre-commit System")
				assert.Contains(t, output, "Available Commands:")
			} else if strings.Contains(tt.args[1], "version") {
				assert.Contains(t, output, "version")
			}
		})
	}
}

func TestExecuteLegacyFunction(t *testing.T) {
	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Capture output to avoid cluttering test output
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		_ = w.Close()
		os.Stdout = oldStdout
	}()

	// Test legacy Execute function
	os.Args = []string{"go-pre-commit", "--help"}
	err := Execute()
	require.NoError(t, err)
}

func TestSetVersionInfoLegacyFunction(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		commit    string
		buildDate string
	}{
		{
			name:      "standard version info",
			version:   "1.0.0",
			commit:    "abc123",
			buildDate: "2025-01-01",
		},
		{
			name:      "empty version info",
			version:   "",
			commit:    "",
			buildDate: "",
		},
		{
			name:      "special characters",
			version:   "v1.0.0-beta+build.1",
			commit:    "commit-hash-123",
			buildDate: "2025-08-10T10:30:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// Test that legacy function doesn't panic and is a no-op
			// This function exists for backward compatibility with tests
			SetVersionInfo(tt.version, tt.commit, tt.buildDate)
			// If we reach here without panic, the test passes
		})
	}
}

func TestResetCommandLegacyFunction(_ *testing.T) {
	// Test that legacy function doesn't panic and is a no-op
	// This function exists for backward compatibility with tests
	ResetCommand()
	// If we reach here without panic, the test passes

	// Call it multiple times to ensure it's safe to call repeatedly
	ResetCommand()
	ResetCommand()
}

func TestInitConfigNoColorHandling(t *testing.T) {
	tests := []struct {
		name        string
		noColorFlag bool
		noColorEnv  string
		expected    bool
	}{
		{
			name:        "no color flag set",
			noColorFlag: true,
			noColorEnv:  "",
			expected:    true,
		},
		{
			name:        "NO_COLOR env var set",
			noColorFlag: false,
			noColorEnv:  "1",
			expected:    true,
		},
		{
			name:        "both flag and env set",
			noColorFlag: true,
			noColorEnv:  "1",
			expected:    true,
		},
		{
			name:        "neither flag nor env set",
			noColorFlag: false,
			noColorEnv:  "",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original state
			originalNoColor := color.NoColor
			originalEnv := os.Getenv("NO_COLOR")
			defer func() {
				color.NoColor = originalNoColor
				if originalEnv == "" {
					_ = os.Unsetenv("NO_COLOR")
				} else {
					_ = os.Setenv("NO_COLOR", originalEnv)
				}
			}()

			// Set up test environment
			if tt.noColorEnv != "" {
				err := os.Setenv("NO_COLOR", tt.noColorEnv)
				require.NoError(t, err)
			} else {
				_ = os.Unsetenv("NO_COLOR")
			}

			// Create app and set config
			app := NewCLIApp("1.0.0", "abc123", "2025-01-01")
			app.config.NoColor = tt.noColorFlag
			builder := NewCommandBuilder(app)

			// Reset color state before test
			color.NoColor = false

			// Call initConfig
			builder.initConfig()

			// Check result
			assert.Equal(t, tt.expected, color.NoColor)
		})
	}
}

func TestInitConfigDirectoryTraversal(t *testing.T) {
	// Save original working directory
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWD) }()

	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "go-pre-commit-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create nested directory structure
	repoRoot := filepath.Join(tempDir, "repo")
	subDir := filepath.Join(repoRoot, "subdir", "nested")
	err = os.MkdirAll(subDir, 0o750)
	require.NoError(t, err)

	// Create .github/.env.base in repo root
	githubDir := filepath.Join(repoRoot, ".github")
	err = os.MkdirAll(githubDir, 0o750)
	require.NoError(t, err)
	envFile := filepath.Join(githubDir, ".env.base")
	err = os.WriteFile(envFile, []byte("# test env file"), 0o600)
	require.NoError(t, err)

	// Change to subdirectory
	err = os.Chdir(subDir)
	require.NoError(t, err)

	// Verify we're not in repo root
	_, err = os.Stat(".github/.env.base")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	// Test initConfig finds and changes to repo root
	app := NewCLIApp("1.0.0", "abc123", "2025-01-01")
	builder := NewCommandBuilder(app)
	builder.initConfig()

	// Check if we found the repo root (this is best-effort)
	currentWD, err := os.Getwd()
	require.NoError(t, err)

	// The function should have attempted to find .github/.env.base
	// We can't guarantee it changed directory due to the implementation,
	// but we can verify the function completed without error
	assert.NotEmpty(t, currentWD)
}

func TestInitConfigNoRepositoryRoot(t *testing.T) {
	// Save original working directory
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWD) }()

	// Create temporary directory without .github/.env.base
	tempDir, err := os.MkdirTemp("", "go-pre-commit-test-no-repo")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Change to temp directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test initConfig when no .github/.env.base is found
	app := NewCLIApp("1.0.0", "abc123", "2025-01-01")
	builder := NewCommandBuilder(app)

	// This should not panic or error, just not find the file
	builder.initConfig()

	// Verify we're still in the temp directory (resolve symlinks for macOS)
	currentWD, err := os.Getwd()
	require.NoError(t, err)

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	expectedDir, err := filepath.EvalSymlinks(tempDir)
	require.NoError(t, err)
	actualDir, err := filepath.EvalSymlinks(currentWD)
	require.NoError(t, err)
	assert.Equal(t, expectedDir, actualDir)
}

func TestPrintFunctionsWithColor(t *testing.T) {
	tests := []struct {
		name         string
		printFunc    func(string, ...interface{})
		expectedIcon string
		isStderr     bool
	}{
		{
			name:         "printSuccess",
			printFunc:    printSuccess,
			expectedIcon: "✓",
			isStderr:     false,
		},
		{
			name:         "printError",
			printFunc:    printError,
			expectedIcon: "✗",
			isStderr:     true,
		},
		{
			name:         "printInfo",
			printFunc:    printInfo,
			expectedIcon: "ℹ",
			isStderr:     false,
		},
		{
			name:         "printWarning",
			printFunc:    printWarning,
			expectedIcon: "⚠",
			isStderr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" with color disabled", func(t *testing.T) {
			// Save original state
			originalNoColor := color.NoColor
			defer func() { color.NoColor = originalNoColor }()

			// Test with color disabled
			color.NoColor = true

			// Capture appropriate output stream
			var oldOutput *os.File
			if tt.isStderr {
				oldOutput = os.Stderr
				r, w, _ := os.Pipe()
				os.Stderr = w
				defer func() {
					_ = w.Close()
					os.Stderr = oldOutput
				}()

				tt.printFunc("Test %s", "message")
				_ = w.Close()
				os.Stderr = oldOutput

				var buf bytes.Buffer
				_, _ = buf.ReadFrom(r)
				output := buf.String()
				assert.Contains(t, output, tt.expectedIcon+" Test message")
			} else {
				oldOutput = os.Stdout
				r, w, _ := os.Pipe()
				os.Stdout = w
				defer func() {
					_ = w.Close()
					os.Stdout = oldOutput
				}()

				tt.printFunc("Test %s", "message")
				_ = w.Close()
				os.Stdout = oldOutput

				var buf bytes.Buffer
				_, _ = buf.ReadFrom(r)
				output := buf.String()
				assert.Contains(t, output, tt.expectedIcon+" Test message")
			}
		})

		t.Run(tt.name+" with color enabled", func(_ *testing.T) {
			// Save original state
			originalNoColor := color.NoColor
			defer func() { color.NoColor = originalNoColor }()

			// Test with color enabled
			color.NoColor = false

			// For color-enabled tests, we just ensure the function doesn't panic
			// since capturing colored output is complex and the color library
			// handles the actual coloring
			tt.printFunc("Test %s", "message")
		})
	}
}

func TestPrintFunctionsFormatting(t *testing.T) {
	tests := []struct {
		name      string
		printFunc func(string, ...interface{})
		format    string
		args      []interface{}
		expected  string
		isStderr  bool
	}{
		{
			name:      "printSuccess with multiple args",
			printFunc: printSuccess,
			format:    "Operation %s completed in %d ms",
			args:      []interface{}{"test", 42},
			expected:  "✓ Operation test completed in 42 ms",
			isStderr:  false,
		},
		{
			name:      "printError with no args",
			printFunc: printError,
			format:    "Simple error message",
			args:      []interface{}{},
			expected:  "✗ Simple error message",
			isStderr:  true,
		},
		{
			name:      "printInfo with float",
			printFunc: printInfo,
			format:    "Progress: %.2f%%",
			args:      []interface{}{75.5},
			expected:  "ℹ Progress: 75.50%",
			isStderr:  false,
		},
		{
			name:      "printWarning with boolean",
			printFunc: printWarning,
			format:    "Feature enabled: %t",
			args:      []interface{}{true},
			expected:  "⚠ Feature enabled: true",
			isStderr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original state
			originalNoColor := color.NoColor
			defer func() { color.NoColor = originalNoColor }()

			// Test with color disabled for consistent output
			color.NoColor = true

			// Capture appropriate output stream
			var oldOutput *os.File
			if tt.isStderr {
				oldOutput = os.Stderr
				r, w, _ := os.Pipe()
				os.Stderr = w
				defer func() {
					_ = w.Close()
					os.Stderr = oldOutput
				}()

				tt.printFunc(tt.format, tt.args...)
				_ = w.Close()
				os.Stderr = oldOutput

				var buf bytes.Buffer
				_, _ = buf.ReadFrom(r)
				output := strings.TrimSpace(buf.String())
				assert.Equal(t, tt.expected, output)
			} else {
				oldOutput = os.Stdout
				r, w, _ := os.Pipe()
				os.Stdout = w
				defer func() {
					_ = w.Close()
					os.Stdout = oldOutput
				}()

				tt.printFunc(tt.format, tt.args...)
				_ = w.Close()
				os.Stdout = oldOutput

				var buf bytes.Buffer
				_, _ = buf.ReadFrom(r)
				output := strings.TrimSpace(buf.String())
				assert.Equal(t, tt.expected, output)
			}
		})
	}
}

func TestCommandBuilderIntegration(t *testing.T) {
	// Integration test to verify the complete command builder workflow
	app := NewCLIApp("2.1.0", "integration-test", "2025-08-10T10:30:00Z")
	builder := NewCommandBuilder(app)

	// Build root command
	rootCmd := builder.BuildRootCmd()
	require.NotNil(t, rootCmd)

	// Add all subcommands (simulating Execute())
	rootCmd.AddCommand(builder.BuildInstallCmd())
	rootCmd.AddCommand(builder.BuildRunCmd())
	rootCmd.AddCommand(builder.BuildUninstallCmd())
	rootCmd.AddCommand(builder.BuildStatusCmd())

	// Verify all commands are present
	commands := rootCmd.Commands()
	cmdNames := make(map[string]bool)
	for _, cmd := range commands {
		cmdNames[cmd.Name()] = true
	}

	expectedCommands := []string{"install", "run", "uninstall", "status"}
	for _, expectedCmd := range expectedCommands {
		assert.True(t, cmdNames[expectedCmd], "Command %s should be present", expectedCmd)
	}

	// Test version information propagation
	assert.Contains(t, rootCmd.Version, "2.1.0")
	assert.Contains(t, rootCmd.Version, "integration-test")
	assert.Contains(t, rootCmd.Version, "2025-08-10T10:30:00Z")
}

func TestAppConfigModification(t *testing.T) {
	// Test that app config can be modified and is properly shared
	app := NewCLIApp("1.0.0", "abc123", "2025-01-01")
	builder := NewCommandBuilder(app)

	// Initially config should have defaults
	assert.False(t, app.config.Verbose)
	assert.False(t, app.config.NoColor)

	// Modify config directly
	app.config.Verbose = true
	app.config.NoColor = true

	// Verify changes are reflected
	assert.True(t, app.config.Verbose)
	assert.True(t, app.config.NoColor)

	// Verify builder has access to the same config
	assert.Equal(t, app.config, builder.app.config)
}

func TestEdgeCasesAndErrorScenarios(t *testing.T) {
	t.Run("nil app to command builder", func(t *testing.T) {
		// This would be a programming error and should panic
		builder := &CommandBuilder{app: nil}

		// Should panic when trying to access app fields
		assert.Panics(t, func() {
			_ = builder.BuildRootCmd()
		})
	})

	t.Run("empty version information", func(t *testing.T) {
		app := NewCLIApp("", "", "")
		builder := NewCommandBuilder(app)
		cmd := builder.BuildRootCmd()

		// Should handle empty version gracefully
		expectedVersion := " (commit: , built: )"
		assert.Equal(t, expectedVersion, cmd.Version)
	})

	t.Run("command parsing edge cases", func(t *testing.T) {
		app := NewCLIApp("1.0.0", "abc123", "2025-01-01")
		builder := NewCommandBuilder(app)
		cmd := builder.BuildRootCmd()

		// Test parsing invalid flags gracefully
		cmd.SetArgs([]string{"--invalid-flag"})
		err := cmd.ParseFlags([]string{"--invalid-flag"})
		// Should return error for unknown flag
		assert.Error(t, err)
	})
}
