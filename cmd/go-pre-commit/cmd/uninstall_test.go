package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUninstallCmd_ParseFlags(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	uninstallCmd := builder.BuildUninstallCmd()

	// Parse the flags
	err := uninstallCmd.ParseFlags([]string{"--hook-type", "pre-push"})
	require.NoError(t, err)

	// Validate flags were parsed correctly
	hookTypes, err := uninstallCmd.Flags().GetStringSlice("hook-type")
	require.NoError(t, err)
	assert.Equal(t, []string{"pre-push"}, hookTypes)
}

func TestUninstallCmd_CommandStructure(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	cmd := builder.BuildUninstallCmd()

	// Verify command has correct structure
	assert.Equal(t, "uninstall", cmd.Name())
	assert.Contains(t, cmd.Short, "Uninstall")

	// Check flags exist
	hookTypeFlag := cmd.Flags().Lookup("hook-type")
	assert.NotNil(t, hookTypeFlag)
}

func TestUninstallCmd_runUninstallWithHooks(t *testing.T) {
	tests := []struct {
		name        string
		setupRepo   func(t *testing.T) string
		setupHooks  func(t *testing.T, repoPath string) // Optional hook setup
		hookTypes   []string
		wantErr     bool
		errContains string
	}{
		{
			name: "successful uninstall of existing hook",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForUninstall(t)
			},
			setupHooks: func(t *testing.T, repoPath string) {
				// Create a pre-commit hook installed by our system
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
			hookTypes: []string{"pre-commit"},
			wantErr:   false,
		},
		{
			name: "successful uninstall of multiple hooks",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForUninstall(t)
			},
			setupHooks: func(t *testing.T, repoPath string) {
				// Create multiple hooks installed by our system
				hooksDir := filepath.Join(repoPath, ".git", "hooks")

				// Pre-commit hook
				hookContent := `#!/bin/bash
# Go Pre-commit Hook
# This hook is managed by Go pre-commit system
echo "Pre-commit hook running"
`
				err := os.WriteFile(filepath.Join(hooksDir, "pre-commit"), []byte(hookContent), 0o755) // #nosec G306 - Test hook file with executable permissions
				require.NoError(t, err)

				// Pre-push hook
				err = os.WriteFile(filepath.Join(hooksDir, "pre-push"), []byte(hookContent), 0o755) // #nosec G306 - Test hook file with executable permissions
				require.NoError(t, err)
			},
			hookTypes: []string{"pre-commit", "pre-push"},
			wantErr:   false,
		},
		{
			name: "uninstall non-existent hook",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForUninstall(t)
			},
			hookTypes: []string{"pre-commit"},
			wantErr:   false, // Should not error, just report not found
		},
		{
			name: "attempt to uninstall non-managed hook",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForUninstall(t)
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
			hookTypes: []string{"pre-commit"},
			wantErr:   false, // Should not error, just report not managed by us
		},
		{
			name: "no git repository should error",
			setupRepo: func(t *testing.T) string {
				// Create temp dir without .git
				tempDir := t.TempDir()
				return tempDir
			},
			hookTypes:   []string{"pre-commit"},
			wantErr:     true,
			errContains: "failed to find git repository",
		},
		{
			name: "uninstall with empty hook types",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepoForUninstall(t)
			},
			hookTypes: []string{},
			wantErr:   false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			err = builder.runUninstallWithHooks(tt.hookTypes, nil, nil)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err, "Uninstall should not error")

				// Verify hooks were actually removed for successful uninstalls
				if tt.setupHooks != nil && len(tt.hookTypes) > 0 {
					for _, hookType := range tt.hookTypes {
						hookPath := filepath.Join(repoPath, ".git", "hooks", hookType)

						// Check if the hook file exists
						if _, err := os.Stat(hookPath); err == nil {
							// If file exists, check if it's our hook
							content, err := os.ReadFile(hookPath) // #nosec G304 - Path is safely constructed for test
							if err == nil {
								hookContent := string(content)
								if strings.Contains(hookContent, "Go Pre-commit Hook") {
									t.Errorf("Our hook should have been removed: %s", hookPath)
								}
								// If it's not our hook, that's fine - we don't remove non-managed hooks
							}
						}
					}
				}
			}
		})
	}
}

// setupTempGitRepoForUninstall creates a temporary git repository for testing uninstall functionality
func setupTempGitRepoForUninstall(t *testing.T) string {
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

	return tempDir
}
