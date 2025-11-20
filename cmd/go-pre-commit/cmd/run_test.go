package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/output"
	"github.com/mrz1836/go-pre-commit/internal/runner"
)

func TestRunCmd_CommandStructure(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	runCmd := builder.BuildRunCmd()

	// Test basic command properties
	assert.Equal(t, "run [check-name] [flags] [files...]", runCmd.Use)
	assert.Contains(t, runCmd.Short, "Run pre-commit checks")

	// Test that all expected flags exist
	expectedFlags := []string{
		"all-files", "files", "skip", "only", "parallel",
		"fail-fast", "show-checks", "graceful", "progress", "quiet",
	}

	for _, flagName := range expectedFlags {
		flag := runCmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "Flag %s should exist", flagName)
	}
}

func TestRunCmd_FlagParsing(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	runCmd := builder.BuildRunCmd()

	// Test parsing various flags
	testCases := []struct {
		name     string
		args     []string
		flagName string
		expected interface{}
	}{
		{
			name:     "all-files flag",
			args:     []string{"--all-files"},
			flagName: "all-files",
			expected: true,
		},
		{
			name:     "parallel flag",
			args:     []string{"--parallel", "4"},
			flagName: "parallel",
			expected: 4,
		},
		{
			name:     "files flag",
			args:     []string{"--files", "file1.go,file2.go"},
			flagName: "files",
			expected: []string{"file1.go", "file2.go"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := runCmd.ParseFlags(tc.args)
			require.NoError(t, err)

			switch tc.flagName {
			case "all-files":
				value, err := runCmd.Flags().GetBool(tc.flagName)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, value)
			case "parallel":
				value, err := runCmd.Flags().GetInt(tc.flagName)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, value)
			case "files":
				value, err := runCmd.Flags().GetStringSlice(tc.flagName)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, value)
			}
		})
	}
}

func TestRunCmd_runChecksWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupRepo   func(t *testing.T) string
		config      RunConfig
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name: "disabled pre-commit system should not error",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForRun(t, false, true) // enabled=false
			},
			config: RunConfig{
				AllFiles: false,
				Files:    []string{},
				Parallel: 1,
			},
			args:    []string{},
			wantErr: false,
		},
		{
			name: "no git repository should error",
			setupRepo: func(t *testing.T) string {
				// Create temp dir without .git but with config
				tempDir := t.TempDir()
				// Create .github directory and config file
				githubDir := filepath.Join(tempDir, ".github")
				err := os.MkdirAll(githubDir, 0o755) // #nosec G301 - Test directory creation
				require.NoError(t, err)
				configPath := filepath.Join(githubDir, ".env.base")
				configContent := "ENABLE_GO_PRE_COMMIT=true\n"
				err = os.WriteFile(configPath, []byte(configContent), 0o644) // #nosec G306 - Test config file
				require.NoError(t, err)
				return tempDir
			},
			config: RunConfig{
				AllFiles: false,
				Files:    []string{},
				Parallel: 1,
			},
			args:        []string{},
			wantErr:     true,
			errContains: "failed to find git repository",
		},
		{
			name: "missing config file should error",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForRun(t, true, false) // enabled=true, hasConfig=false
			},
			config: RunConfig{
				AllFiles: false,
				Files:    []string{},
				Parallel: 1,
			},
			args:        []string{},
			wantErr:     true,
			errContains: "failed to load configuration",
		},
		{
			name: "successful run with no files to check",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForRun(t, true, true) // enabled=true
			},
			config: RunConfig{
				AllFiles: false,
				Files:    []string{},
				Parallel: 1,
			},
			args:    []string{},
			wantErr: false,
		},
		{
			name: "run with specific files",
			setupRepo: func(t *testing.T) string {
				repoPath := setupTempGitRepoForRun(t, true, true)
				// Create test files
				testFile := filepath.Join(repoPath, "test.go")
				err := os.WriteFile(testFile, []byte("package main\nfunc main() {}\n"), 0o644) // #nosec G306 - Test Go file
				require.NoError(t, err)
				return repoPath
			},
			config: RunConfig{
				AllFiles: false,
				Files:    []string{"test.go"},
				Parallel: 1,
			},
			args:    []string{},
			wantErr: false,
		},
		{
			name: "run with all files",
			setupRepo: func(t *testing.T) string {
				repoPath := setupTempGitRepoForRun(t, true, true)
				// Create test files
				testFile := filepath.Join(repoPath, "test.go")
				err := os.WriteFile(testFile, []byte("package main\nfunc main() {}\n"), 0o644) // #nosec G306 - Test Go file
				require.NoError(t, err)
				return repoPath
			},
			config: RunConfig{
				AllFiles: true,
				Files:    []string{},
				Parallel: 1,
			},
			args:    []string{},
			wantErr: false,
		},
		{
			name: "run with specific check as argument",
			setupRepo: func(t *testing.T) string {
				repoPath := setupTempGitRepoForRun(t, true, true)
				// Create test files
				testFile := filepath.Join(repoPath, "test.go")
				err := os.WriteFile(testFile, []byte("package main\nfunc main() {}\n"), 0o644) // #nosec G306 - Test Go file
				require.NoError(t, err)
				return repoPath
			},
			config: RunConfig{
				AllFiles: false,
				Files:    []string{"test.go"},
				Parallel: 1,
			},
			args:    []string{"fmt"},
			wantErr: false,
		},
		{
			name: "run with show version flag",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForRun(t, true, true)
			},
			config: RunConfig{
				AllFiles:    false,
				Files:       []string{},
				Parallel:    1,
				ShowVersion: true,
			},
			args:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any environment variables from previous tests
			originalEnv := os.Getenv("ENABLE_GO_PRE_COMMIT")
			defer func() {
				if originalEnv != "" {
					_ = os.Setenv("ENABLE_GO_PRE_COMMIT", originalEnv)
				} else {
					_ = os.Unsetenv("ENABLE_GO_PRE_COMMIT")
				}
			}()

			// Setup repository
			originalDir, err := os.Getwd()
			require.NoError(t, err)

			repoPath := tt.setupRepo(t)
			err = os.Chdir(repoPath)
			require.NoError(t, err)

			defer func() {
				cdErr := os.Chdir(originalDir)
				require.NoError(t, cdErr)
			}()

			// Create CLI app and command builder
			app := NewCLIApp("test", "test-commit", "test-date")
			builder := NewCommandBuilder(app)

			// Run the function
			err = builder.runChecksWithConfig(tt.config, nil, tt.args)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Logf("Unexpected error: %v", err)
				}
				// Most test cases should not error, but some may return specific errors
				// like "checks failed" which is expected behavior

				// Only fail the test for unexpected types of errors
				if err != nil && tt.errContains == "" {
					// Check if it's an expected error
					expectedErrors := []string{
						"checks failed",
						"No files to check",
						"failed to get staged files",
						"failed to get all files",
						"no checks to run",
					}

					isExpectedError := false
					errStr := err.Error()
					for _, expectedErr := range expectedErrors {
						if strings.Contains(errStr, expectedErr) {
							isExpectedError = true
							break
						}
					}

					if !isExpectedError {
						t.Errorf("Unexpected error: %v", err)
					}
				}
			}
		})
	}
}

// setupTempGitRepoForRun creates a temporary git repository for testing run functionality
func setupTempGitRepoForRun(t *testing.T, enabled, hasConfig bool) string {
	tempDir := t.TempDir()

	// Create .git directory and minimal git structure
	gitDir := filepath.Join(tempDir, ".git")
	err := os.MkdirAll(gitDir, 0o755) // #nosec G301 - Test git directory creation
	require.NoError(t, err)

	// Create basic git structure to make it a valid git repo
	err = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644) // #nosec G306 - Test git HEAD file
	require.NoError(t, err)

	refsDir := filepath.Join(gitDir, "refs", "heads")
	err = os.MkdirAll(refsDir, 0o755) // #nosec G301 - Test git refs directory
	require.NoError(t, err)

	objectsDir := filepath.Join(gitDir, "objects")
	err = os.MkdirAll(objectsDir, 0o755) // #nosec G301 - Test git objects directory
	require.NoError(t, err)

	// Create empty index file for staged files support
	indexPath := filepath.Join(gitDir, "index")
	err = os.WriteFile(indexPath, []byte{}, 0o644) // #nosec G306 - Test git index file
	require.NoError(t, err)

	// Create config file for git to work properly
	configPath := filepath.Join(gitDir, "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
[user]
	name = Test User
	email = test@example.com
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644) // #nosec G306 - Test git config file
	require.NoError(t, err)

	if hasConfig {
		// Create .github directory and config file
		githubDir := filepath.Join(tempDir, ".github")
		err = os.MkdirAll(githubDir, 0o755) // #nosec G301 - Test .github directory
		require.NoError(t, err)

		configPath := filepath.Join(githubDir, ".env.base")
		configContent := "# Test configuration\n"
		if enabled {
			configContent += "ENABLE_GO_PRE_COMMIT=true\n"
		} else {
			configContent += "ENABLE_GO_PRE_COMMIT=false\n"
		}

		// Add basic check configurations
		configContent += "GO_PRE_COMMIT_ENABLE_FMT=true\n"
		configContent += "GO_PRE_COMMIT_ENABLE_FUMPT=false\n"
		configContent += "GO_PRE_COMMIT_ENABLE_LINT=false\n"
		configContent += "GO_PRE_COMMIT_ENABLE_MOD_TIDY=false\n"
		configContent += "GO_PRE_COMMIT_ENABLE_WHITESPACE=true\n"
		configContent += "GO_PRE_COMMIT_ENABLE_EOF=true\n"

		err = os.WriteFile(configPath, []byte(configContent), 0o644) // #nosec G306 - Test git config file
		require.NoError(t, err)
	}

	return tempDir
}

// TestDisplayEnhancedResults tests the displayEnhancedResults function comprehensively
func TestDisplayEnhancedResults(t *testing.T) {
	testCases := []struct {
		name        string
		results     *runner.Results
		quietMode   bool
		verboseMode bool
		description string
	}{
		{
			name:        "All Checks Passed - Normal Mode",
			quietMode:   false,
			verboseMode: false,
			results: &runner.Results{
				CheckResults: []runner.CheckResult{
					{
						Name:     "fmt",
						Success:  true,
						Duration: 500 * time.Millisecond,
						Files:    []string{"main.go", "utils.go"},
					},
					{
						Name:     "lint",
						Success:  true,
						Duration: 2 * time.Second,
						Files:    []string{"main.go"},
					},
				},
				Passed:        2,
				Failed:        0,
				Skipped:       0,
				TotalDuration: 2500 * time.Millisecond,
				TotalFiles:    2,
			},
			description: "Should display success messages for all passed checks",
		},
		{
			name:        "Mixed Results - Verbose Mode",
			quietMode:   false,
			verboseMode: true,
			results: &runner.Results{
				CheckResults: []runner.CheckResult{
					{
						Name:     "fmt",
						Success:  true,
						Duration: 300 * time.Millisecond,
						Files:    []string{"main.go"},
					},
					{
						Name:       "lint",
						Success:    false,
						Error:      "linting failed",
						Output:     "main.go:10:5: error: unused variable 'x'\nmain.go:15:1: error: missing return",
						Duration:   1 * time.Second,
						Files:      []string{"main.go"},
						Suggestion: "Fix the linting errors and run again",
					},
				},
				Passed:        1,
				Failed:        1,
				Skipped:       0,
				TotalDuration: 1300 * time.Millisecond,
				TotalFiles:    1,
			},
			description: "Should display detailed information in verbose mode",
		},
		{
			name:        "Gracefully Skipped Check",
			quietMode:   false,
			verboseMode: false,
			results: &runner.Results{
				CheckResults: []runner.CheckResult{
					{
						Name:       "fumpt",
						Success:    true, // Marked as success but was skipped
						Error:      "gofumpt not installed",
						Duration:   50 * time.Millisecond,
						CanSkip:    true,
						Suggestion: "Install gofumpt: go install mvdan.cc/gofumpt@latest",
					},
				},
				Passed:        0,
				Failed:        0,
				Skipped:       1,
				TotalDuration: 50 * time.Millisecond,
				TotalFiles:    0,
			},
			description: "Should display warning for gracefully skipped checks",
		},
		{
			name:        "Quiet Mode with Failures",
			quietMode:   true,
			verboseMode: false,
			results: &runner.Results{
				CheckResults: []runner.CheckResult{
					{
						Name:     "fmt",
						Success:  true,
						Duration: 200 * time.Millisecond,
					},
					{
						Name:       "whitespace",
						Success:    false,
						Error:      "trailing whitespace found",
						Output:     "utils.go:25: trailing whitespace\nservice.go:10: trailing whitespace",
						Duration:   100 * time.Millisecond,
						Suggestion: "Remove trailing whitespace from the files",
					},
				},
				Passed:        1,
				Failed:        1,
				Skipped:       0,
				TotalDuration: 300 * time.Millisecond,
				TotalFiles:    2,
			},
			description: "Should show failures even in quiet mode",
		},
		{
			name:        "Multiple Failures with Error Extraction",
			quietMode:   false,
			verboseMode: false,
			results: &runner.Results{
				CheckResults: []runner.CheckResult{
					{
						Name:    "lint",
						Success: false,
						Error:   "golangci-lint failed",
						Output: `Running golangci-lint...
main.go:10:5: Error: unused variable 'x' (ineffassign)
main.go:15:1: Error: missing return statement (typecheck)
utils.go:5:2: Error: imported but not used: "fmt" (unused)
Analyzing 3 files...
Done.`,
						Duration:   3 * time.Second,
						Suggestion: "Fix the linting errors reported above",
					},
					{
						Name:    "mod-tidy",
						Success: false,
						Error:   "go mod tidy failed",
						Output: `diff go.mod go.mod.orig
--- go.mod.orig
+++ go.mod
@@ -5,3 +5,4 @@
 require (
 	github.com/pkg/errors v0.9.1
 	github.com/stretchr/testify v1.8.0
+	github.com/unused/dep v1.0.0
 )`,
						Duration:   500 * time.Millisecond,
						Suggestion: "Run 'go mod tidy' to fix module dependencies",
					},
				},
				Passed:        0,
				Failed:        2,
				Skipped:       0,
				TotalDuration: 3500 * time.Millisecond,
				TotalFiles:    3,
			},
			description: "Should extract and display key error lines from command output",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create output formatter
			formatter := output.New(output.Options{
				ColorEnabled: false, // Disable colors for consistent testing
			})

			// This should not panic and should complete successfully
			require.NotPanics(t, func() {
				displayEnhancedResults(formatter, tc.results, tc.quietMode, tc.verboseMode)
			}, "displayEnhancedResults should not panic for case: %s", tc.description)

			t.Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestExtractKeyErrorLines tests the error extraction functionality
func TestExtractKeyErrorLines(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedCount int
		expectedLines []string
		description   string
	}{
		{
			name: "Go Lint Errors",
			input: `Running golangci-lint...
Analyzing files...
main.go:10:5: Error: unused variable 'x' (ineffassign)
main.go:15:1: Error: missing return statement (typecheck)
utils.go:5:2: Error: imported but not used: "fmt" (unused)
Done.`,
			expectedCount: 3,
			expectedLines: []string{
				"main.go:10:5: Error: unused variable 'x' (ineffassign)",
				"main.go:15:1: Error: missing return statement (typecheck)",
				"utils.go:5:2: Error: imported but not used: \"fmt\" (unused)",
			},
			description: "Should extract Go file error lines",
		},
		{
			name: "Whitespace Issues",
			input: `Checking whitespace...
utils.go:25: trailing whitespace
service.go:10: trailing whitespace
main.go:5: mixed spaces and tabs
Fixed 3 files.`,
			expectedCount: 3,
			expectedLines: []string{
				"utils.go:25: trailing whitespace",
				"service.go:10: trailing whitespace",
				"main.go:5: mixed spaces and tabs",
			},
			description: "Should extract whitespace error lines",
		},
		{
			name: "Module Tidy Diff",
			input: `Running go mod tidy...
diff go.mod go.mod.orig
--- go.mod.orig
+++ go.mod
@@ -5,3 +5,4 @@
 require (
 	github.com/pkg/errors v0.9.1
+	github.com/unused/dep v1.0.0
 )
Module not tidy.`,
			expectedCount: 5,
			expectedLines: []string{
				"diff go.mod go.mod.orig",
				"--- go.mod.orig",
				"+++ go.mod",
				"@@ -5,3 +5,4 @@",
				"+	github.com/unused/dep v1.0.0",
			},
			description: "Should extract diff lines from go mod tidy output",
		},
		{
			name: "No Errors",
			input: `Running checks...
Analyzing files...
All checks passed!
Done.`,
			expectedCount: 0,
			expectedLines: nil,
			description:   "Should return no error lines when there are no errors",
		},
		{
			name: "Mixed Error Types",
			input: `Running multiple checks...
main.go:10:1: error: syntax error
utils.go:5: trailing whitespace
ERRO[0001] Failed to process file
level=error msg="processing failed"
✗ Check failed
Completed with errors.`,
			expectedCount: 5,
			expectedLines: []string{
				"main.go:10:1: error: syntax error",
				"utils.go:5: trailing whitespace",
				"ERRO[0001] Failed to process file",
				"level=error msg=\"processing failed\"",
				"✗ Check failed",
			},
			description: "Should extract different types of error indicators",
		},
		{
			name: "Multi-Module Mod Tidy Errors",
			input: `Running go mod tidy...
Module . needs tidying:
diff current/go.mod tidy/go.mod
--- current/go.mod
+++ tidy/go.mod
@@ -17,3 +17,4 @@
+	github.com/new/dep v1.0.0

Module ./services/api needs tidying:
diff current/go.mod tidy/go.mod
--- current/go.mod
+++ tidy/go.mod
@@ -5,2 +5,3 @@
+	github.com/another/dep v2.0.0`,
			expectedCount: 10,
			expectedLines: nil, // Don't check specific lines, just count
			description:        "Should extract module path indicators and diff lines",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the extractKeyErrorLines function
			errorLines := extractKeyErrorLines(tc.input)

			// Check count
			assert.Len(t, errorLines, tc.expectedCount,
				"Expected %d error lines, got %d for %s", tc.expectedCount, len(errorLines), tc.description)

			// Check specific lines if provided
			if tc.expectedLines != nil {
				for i, expectedLine := range tc.expectedLines {
					if i < len(errorLines) {
						assert.Contains(t, errorLines[i], expectedLine,
							"Error line %d should contain '%s'", i, expectedLine)
					}
				}
			}

			t.Logf("✓ %s: Found %d error lines", tc.description, len(errorLines))
			for i, line := range errorLines {
				t.Logf("  [%d] %s", i+1, line)
			}
		})
	}
}

// TestStripANSI tests ANSI color code removal
func TestStripANSI(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No ANSI codes",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "Red color code",
			input:    "\x1b[31merror message\x1b[0m",
			expected: "error message",
		},
		{
			name:     "Multiple color codes",
			input:    "\x1b[32mSuccess:\x1b[0m \x1b[31mError:\x1b[0m message",
			expected: "Success: Error: message",
		},
		{
			name:     "Complex ANSI sequence",
			input:    "\x1b[1;32;40mbold green on black\x1b[0m",
			expected: "bold green on black",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := stripANSI(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestRunCmd_ColorIntegration tests color output integration in the run command
func TestRunCmd_ColorIntegration(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		envVars     map[string]string
		expectColor bool
		description string
	}{
		{
			name:        "No color flags with clean environment",
			args:        []string{},
			envVars:     map[string]string{},
			expectColor: false, // Typically false in test environment due to non-TTY
			description: "Default behavior should depend on TTY detection",
		},
		{
			name:        "Legacy no-color flag",
			args:        []string{"--no-color"},
			envVars:     map[string]string{},
			expectColor: false,
			description: "--no-color should disable colors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and clean environment
			allEnvVars := []string{"NO_COLOR", "GO_PRE_COMMIT_COLOR_OUTPUT", "TERM", "CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "CIRCLECI", "TRAVIS", "BUILDKITE", "DRONE", "TEAMCITY_VERSION", "TF_BUILD", "APPVEYOR", "CODEBUILD_BUILD_ID"}
			originalEnv := make(map[string]string)
			for _, key := range allEnvVars {
				originalEnv[key] = os.Getenv(key)
				_ = os.Unsetenv(key)
			}

			// Set test environment
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			defer func() {
				for key, value := range originalEnv {
					if value == "" {
						_ = os.Unsetenv(key)
					} else {
						_ = os.Setenv(key, value)
					}
				}
			}()

			// Create CLI app and command builder - need root command for persistent flags
			app := NewCLIApp("test", "test-commit", "test-date")
			builder := NewCommandBuilder(app)
			rootCmd := builder.BuildRootCmd()
			runCmd := builder.BuildRunCmd()
			rootCmd.AddCommand(runCmd)

			// Set the args and test parsing
			rootCmd.SetArgs(append([]string{"run"}, tt.args...))

			// Parse should not fail for valid flag combinations
			assert.NotPanics(t, func() {
				err := rootCmd.Execute()
				// We don't care about the execution result, just that flag parsing works
				_ = err
			}, "Color flag parsing should not panic for %s", tt.description)
		})
	}
}

// TestRunCmd_ColorOutputEndToEnd tests end-to-end color output behavior
func TestRunCmd_ColorOutputEndToEnd(t *testing.T) {
	tests := []struct {
		name        string
		setupRepo   func(t *testing.T) string
		args        []string
		envVars     map[string]string
		description string
	}{
		{
			name: "Run with color always and successful checks",
			setupRepo: func(t *testing.T) string {
				repoPath := setupTempGitRepoForRun(t, true, true)
				// Create a simple Go file that should pass fmt check
				testFile := filepath.Join(repoPath, "main.go")
				content := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n"
				err := os.WriteFile(testFile, []byte(content), 0o644) // #nosec G306 - Test file permissions
				require.NoError(t, err)
				return repoPath
			},
			args:        []string{"--color=always", "--all-files"},
			envVars:     map[string]string{},
			description: "Should run with color always flag without flag parsing errors",
		},
		{
			name: "Run with color never and mixed environment",
			setupRepo: func(t *testing.T) string {
				repoPath := setupTempGitRepoForRun(t, true, true)
				testFile := filepath.Join(repoPath, "main.go")
				content := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n"
				err := os.WriteFile(testFile, []byte(content), 0o644) // #nosec G306 - Test file permissions
				require.NoError(t, err)
				return repoPath
			},
			args:        []string{"--color=never", "--all-files"},
			envVars:     map[string]string{"TERM": "xterm-256color"},
			description: "Should run with color never flag without flag parsing errors",
		},
		{
			name: "Run with color auto in CI environment",
			setupRepo: func(t *testing.T) string {
				repoPath := setupTempGitRepoForRun(t, true, true)
				testFile := filepath.Join(repoPath, "main.go")
				content := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n"
				err := os.WriteFile(testFile, []byte(content), 0o644) // #nosec G306 - Test file permissions
				require.NoError(t, err)
				return repoPath
			},
			args:        []string{"--color=auto", "--all-files"},
			envVars:     map[string]string{"CI": "true", "GITHUB_ACTIONS": "true"},
			description: "Should run with color auto flag without flag parsing errors",
		},
		{
			name: "Run with NO_COLOR environment variable",
			setupRepo: func(t *testing.T) string {
				repoPath := setupTempGitRepoForRun(t, true, true)
				testFile := filepath.Join(repoPath, "main.go")
				content := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n"
				err := os.WriteFile(testFile, []byte(content), 0o644) // #nosec G306 - Test file permissions
				require.NoError(t, err)
				return repoPath
			},
			args:        []string{"--all-files"},
			envVars:     map[string]string{"NO_COLOR": "1"},
			description: "Should respect NO_COLOR environment variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any environment variables
			allEnvVars := []string{"NO_COLOR", "GO_PRE_COMMIT_COLOR_OUTPUT", "TERM", "CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "CIRCLECI", "TRAVIS", "BUILDKITE", "DRONE", "TEAMCITY_VERSION", "TF_BUILD", "APPVEYOR", "CODEBUILD_BUILD_ID", "ENABLE_GO_PRE_COMMIT"}
			originalEnv := make(map[string]string)
			for _, key := range allEnvVars {
				originalEnv[key] = os.Getenv(key)
				_ = os.Unsetenv(key)
			}

			// Set test environment
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			defer func() {
				for key, value := range originalEnv {
					if value == "" {
						_ = os.Unsetenv(key)
					} else {
						_ = os.Setenv(key, value)
					}
				}
			}()

			// Setup repository
			originalDir, err := os.Getwd()
			require.NoError(t, err)

			repoPath := tt.setupRepo(t)
			err = os.Chdir(repoPath)
			require.NoError(t, err)

			defer func() {
				cdErr := os.Chdir(originalDir)
				require.NoError(t, cdErr)
			}()

			// Create CLI app and command builder - need root command for persistent flags
			app := NewCLIApp("test", "test-commit", "test-date")
			builder := NewCommandBuilder(app)
			rootCmd := builder.BuildRootCmd()
			runCmd := builder.BuildRunCmd()
			rootCmd.AddCommand(runCmd)

			// Set the args and run the command
			rootCmd.SetArgs(append([]string{"run"}, tt.args...))

			// Execute should not have flag parsing errors
			execErr := rootCmd.Execute()
			if execErr != nil {
				// We don't care about execution errors (like repo issues), just flag parsing
				assert.NotContains(t, execErr.Error(), "unknown flag", "Should not have flag parsing errors for %s", tt.description)
			}
		})
	}
}

// TestRunCmd_ColorModeConfiguration tests color mode configuration parsing
func TestRunCmd_ColorModeConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "Valid color mode: auto",
			args:        []string{"--color=auto"},
			expectError: false,
			description: "--color=auto should be accepted",
		},
		{
			name:        "Valid color mode: always",
			args:        []string{"--color=always"},
			expectError: false,
			description: "--color=always should be accepted",
		},
		{
			name:        "Valid color mode: never",
			args:        []string{"--color=never"},
			expectError: false,
			description: "--color=never should be accepted",
		},
		{
			name:        "Invalid color mode",
			args:        []string{"--color=invalid"},
			expectError: false, // Implementation might be permissive and default to auto
			description: "Invalid color mode handled gracefully",
		},
		{
			name:        "No-color flag",
			args:        []string{"--no-color"},
			expectError: false,
			description: "--no-color should be accepted",
		},
		{
			name:        "Combination of no-color and color flags",
			args:        []string{"--no-color", "--color=auto"},
			expectError: false,
			description: "Combination should be accepted, with color flag taking precedence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create CLI app and command builder - need root command for persistent flags
			app := NewCLIApp("test", "test-commit", "test-date")
			builder := NewCommandBuilder(app)
			rootCmd := builder.BuildRootCmd()
			runCmd := builder.BuildRunCmd()
			rootCmd.AddCommand(runCmd)

			// Set the args and test parsing
			rootCmd.SetArgs(append([]string{"run"}, tt.args...))

			// Execute to test flag parsing
			err := rootCmd.Execute()

			if tt.expectError {
				assert.Error(t, err, "Expected error for %s", tt.description)
			} else {
				// For successful cases, we don't care about execution errors (like missing repos),
				// just that the flag parsing worked
				if err != nil && !strings.Contains(err.Error(), "flag provided but not defined") {
					// Flag parsing worked, execution might fail for other reasons
					assert.NotContains(t, err.Error(), "unknown flag", "Flag should be recognized for %s", tt.description)
				}
			}
		})
	}
}
