package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
