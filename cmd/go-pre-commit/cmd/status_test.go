package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCmd_CommandStructure(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	statusCmd := builder.BuildStatusCmd()

	// Test basic command properties
	assert.Equal(t, "status", statusCmd.Use)
	assert.Contains(t, statusCmd.Short, "Show installation status")

	// Test that the command has proper structure
	assert.NotNil(t, statusCmd.RunE, "RunE function should be set")
}

func TestStatusCmd_runStatus(t *testing.T) {
	tests := []struct {
		name        string
		setupRepo   func(t *testing.T) string
		setupHooks  func(t *testing.T, repoPath string) // Optional hook setup
		wantErr     bool
		errContains string
	}{
		{
			name: "successful status check with no hooks",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForStatus(t, true, true) // enabled=true, hasConfig=true
			},
			wantErr: false,
		},
		{
			name: "successful status check with installed hooks",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForStatus(t, true, true)
			},
			setupHooks: func(t *testing.T, repoPath string) {
				// Create a sample pre-commit hook installed by our system
				hooksDir := filepath.Join(repoPath, ".git", "hooks")
				hookPath := filepath.Join(hooksDir, "pre-commit")
				hookContent := `#!/bin/bash
# Go Pre-commit Hook
# This hook is managed by Go pre-commit system
echo "Pre-commit hook running"
`
				err := os.WriteFile(hookPath, []byte(hookContent), 0o755) // #nosec G306 - Test hook file with executable permissions
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name: "status check with conflicting hook",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForStatus(t, true, true)
			},
			setupHooks: func(t *testing.T, repoPath string) {
				// Create a conflicting pre-commit hook (not created by our system)
				hooksDir := filepath.Join(repoPath, ".git", "hooks")
				hookPath := filepath.Join(hooksDir, "pre-commit")
				hookContent := `#!/bin/bash
echo "Some other pre-commit hook"
`
				err := os.WriteFile(hookPath, []byte(hookContent), 0o755) // #nosec G306 - Test hook file with executable permissions
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name: "disabled pre-commit system status",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForStatus(t, false, true) // enabled=false
			},
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
				configPath := filepath.Join(githubDir, ".env.shared")
				configContent := "ENABLE_GO_PRE_COMMIT=true\n"
				err = os.WriteFile(configPath, []byte(configContent), 0o644) // #nosec G306 - Test config file
				require.NoError(t, err)
				return tempDir
			},
			wantErr:     true,
			errContains: "failed to find git repository",
		},
		{
			name: "missing config file should error",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForStatus(t, true, false) // enabled=true, hasConfig=false
			},
			wantErr:     true,
			errContains: "failed to load configuration",
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

			// Setup hooks if provided
			if tt.setupHooks != nil {
				tt.setupHooks(t, repoPath)
			}

			// Create CLI app and command builder
			app := NewCLIApp("test", "test-commit", "test-date")
			builder := NewCommandBuilder(app)

			// Run the function
			err = builder.runStatus(nil, nil)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Logf("Unexpected error: %v", err)
				}
				// Most status checks should not error - they should handle cases gracefully
				assert.NoError(t, err, "Status check should not error")
			}
		})
	}
}

// setupTempGitRepoForStatus creates a temporary git repository for testing status functionality
func setupTempGitRepoForStatus(t *testing.T, enabled, hasConfig bool) string {
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

	// Create .git/hooks directory
	hooksDir := filepath.Join(gitDir, "hooks")
	err = os.MkdirAll(hooksDir, 0o755) // #nosec G301 - Test git hooks directory
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

		configPath := filepath.Join(githubDir, ".env.shared")
		configContent := "# Test configuration\n"
		if enabled {
			configContent += "ENABLE_GO_PRE_COMMIT=true\n"
		} else {
			configContent += "ENABLE_GO_PRE_COMMIT=false\n"
		}

		// Add basic check configurations
		configContent += "GO_PRE_COMMIT_ENABLE_FMT=true\n"
		configContent += "GO_PRE_COMMIT_ENABLE_FUMPT=true\n"
		configContent += "GO_PRE_COMMIT_ENABLE_LINT=true\n"
		configContent += "GO_PRE_COMMIT_ENABLE_MOD_TIDY=true\n"
		configContent += "GO_PRE_COMMIT_ENABLE_WHITESPACE=true\n"
		configContent += "GO_PRE_COMMIT_ENABLE_EOF=true\n"
		configContent += "GO_PRE_COMMIT_TIMEOUT_SECONDS=300\n"
		configContent += "GO_PRE_COMMIT_PARALLEL_WORKERS=4\n"

		err = os.WriteFile(configPath, []byte(configContent), 0o644) // #nosec G306 - Test config file
		require.NoError(t, err)
	}

	return tempDir
}
