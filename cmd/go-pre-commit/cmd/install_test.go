package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/config"
)

func TestInstallCmd_ParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T, cmd *cobra.Command)
	}{
		{
			name: "force flag",
			args: []string{"--force"},
			validate: func(t *testing.T, cmd *cobra.Command) {
				force, err := cmd.Flags().GetBool("force")
				require.NoError(t, err)
				assert.True(t, force)
			},
		},
		{
			name: "force flag short",
			args: []string{"-f"},
			validate: func(t *testing.T, cmd *cobra.Command) {
				force, err := cmd.Flags().GetBool("force")
				require.NoError(t, err)
				assert.True(t, force)
			},
		},
		{
			name: "hook-type flag single",
			args: []string{"--hook-type", "pre-push"},
			validate: func(t *testing.T, cmd *cobra.Command) {
				hookTypes, err := cmd.Flags().GetStringSlice("hook-type")
				require.NoError(t, err)
				assert.Equal(t, []string{"pre-push"}, hookTypes)
			},
		},
		{
			name: "hook-type flag multiple",
			args: []string{"--hook-type", "pre-commit", "--hook-type", "pre-push"},
			validate: func(t *testing.T, cmd *cobra.Command) {
				hookTypes, err := cmd.Flags().GetStringSlice("hook-type")
				require.NoError(t, err)
				assert.Equal(t, []string{"pre-commit", "pre-push"}, hookTypes)
			},
		},
		{
			name: "default hook type",
			args: []string{},
			validate: func(t *testing.T, cmd *cobra.Command) {
				hookTypes, err := cmd.Flags().GetStringSlice("hook-type")
				require.NoError(t, err)
				assert.Equal(t, []string{"pre-commit"}, hookTypes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh CLI app and command builder for each test case
			app := NewCLIApp("test", "test-commit", "test-date")
			builder := NewCommandBuilder(app)
			installCmd := builder.BuildInstallCmd()

			// Parse the flags
			err := installCmd.ParseFlags(tt.args)
			require.NoError(t, err)

			// Validate
			tt.validate(t, installCmd)
		})
	}
}

func TestInstallCmd_CommandStructure(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	cmd := builder.BuildInstallCmd()

	// Verify command has correct structure
	assert.Equal(t, "install", cmd.Name())
	assert.Contains(t, cmd.Short, "Install")

	// Check flags exist
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)

	hookTypeFlag := cmd.Flags().Lookup("hook-type")
	assert.NotNil(t, hookTypeFlag)
}

func TestInstallCmd_runInstallWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupRepo   func(t *testing.T) string // Returns temp git repo path
		config      InstallConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "successful install with default config",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepo(t, true, true) // enabled=true, hasConfig=true
			},
			config: InstallConfig{
				Force:     false,
				HookTypes: []string{"pre-commit"},
			},
			wantErr: false,
		},
		{
			name: "successful install with force flag",
			setupRepo: func(t *testing.T) string {
				repoPath := setupTempGitRepo(t, true, true)
				// Create existing hook to test force overwrite
				hooksDir := filepath.Join(repoPath, ".git", "hooks")
				hookPath := filepath.Join(hooksDir, "pre-commit")
				err := os.WriteFile(hookPath, []byte("#!/bin/bash\necho 'existing hook'"), 0o755) // #nosec G306 - Test file with appropriate executable permissions
				require.NoError(t, err)
				return repoPath
			},
			config: InstallConfig{
				Force:     true,
				HookTypes: []string{"pre-commit"},
			},
			wantErr: false,
		},
		{
			name: "successful install multiple hook types",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepo(t, true, true)
			},
			config: InstallConfig{
				Force:     false,
				HookTypes: []string{"pre-commit", "pre-push"},
			},
			wantErr: false,
		},
		{
			name: "disabled pre-commit system should not error",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepo(t, false, true) // enabled=false
			},
			config: InstallConfig{
				Force:     false,
				HookTypes: []string{"pre-commit"},
			},
			wantErr: false, // Should not error, just print warning and return early
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
			config: InstallConfig{
				Force:     false,
				HookTypes: []string{"pre-commit"},
			},
			wantErr:     true,
			errContains: "failed to find git repository",
		},
		{
			name: "missing config file should error",
			setupRepo: func(t *testing.T) string {
				return setupTempGitRepo(t, true, false) // enabled=true, hasConfig=false
			},
			config: InstallConfig{
				Force:     false,
				HookTypes: []string{"pre-commit"},
			},
			wantErr:     true,
			errContains: "failed to load configuration",
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

			// Create CLI app and command builder
			app := NewCLIApp("test", "test-commit", "test-date")
			builder := NewCommandBuilder(app)

			// Run the function
			err = builder.runInstallWithConfig(tt.config, nil, nil)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)

				// For successful installations, verify hooks were actually created
				if tt.name != "disabled pre-commit system should not error" {
					for _, hookType := range tt.config.HookTypes {
						hookPath := filepath.Join(repoPath, ".git", "hooks", hookType)
						_, err := os.Stat(hookPath)
						assert.NoError(t, err, "Hook file should exist: %s", hookPath)
					}
				}
				// For disabled system test, see TestInstallCmd_DisabledSystemBehavior
				// for specific verification that no hooks are created
			}
		})
	}
}

// setupTempGitRepo creates a temporary git repository for testing
func setupTempGitRepo(t *testing.T, enabled, hasConfig bool) string {
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

		err = os.WriteFile(configPath, []byte(configContent), 0o644) // #nosec G306 - Test config file
		require.NoError(t, err)
	}

	return tempDir
}

func TestInstallCmd_ConfigurationLoading(t *testing.T) {
	// Test to debug configuration loading
	// Clean up any environment variables from previous tests
	originalEnv := os.Getenv("ENABLE_GO_PRE_COMMIT")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("ENABLE_GO_PRE_COMMIT", originalEnv)
		} else {
			_ = os.Unsetenv("ENABLE_GO_PRE_COMMIT")
		}
	}()

	// Clear environment variable BEFORE setting up the test to ensure clean state
	_ = os.Unsetenv("ENABLE_GO_PRE_COMMIT")

	originalDir, err := os.Getwd()
	require.NoError(t, err)

	repoPath := setupTempGitRepo(t, false, true) // enabled=false
	err = os.Chdir(repoPath)
	require.NoError(t, err)

	defer func() {
		cdErr := os.Chdir(originalDir)
		require.NoError(t, cdErr)
	}()

	// Load configuration and check its state
	cfg, err := config.Load()
	require.NoError(t, err)

	t.Logf("Configuration Enabled: %v", cfg.Enabled)
	assert.False(t, cfg.Enabled, "Configuration should be disabled")
}

func TestInstallCmd_DisabledSystemBehavior(t *testing.T) {
	// Specific test for disabled system behavior
	// Clean up any environment variables from previous tests
	originalEnv := os.Getenv("ENABLE_GO_PRE_COMMIT")
	defer func() {
		if originalEnv != "" {
			_ = os.Setenv("ENABLE_GO_PRE_COMMIT", originalEnv)
		} else {
			_ = os.Unsetenv("ENABLE_GO_PRE_COMMIT")
		}
	}()

	// Clear environment variable BEFORE setting up the test to ensure clean state
	_ = os.Unsetenv("ENABLE_GO_PRE_COMMIT")

	originalDir, err := os.Getwd()
	require.NoError(t, err)

	repoPath := setupTempGitRepo(t, false, true) // enabled=false
	err = os.Chdir(repoPath)
	require.NoError(t, err)

	defer func() {
		cdErr := os.Chdir(originalDir)
		require.NoError(t, cdErr)
	}()

	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)

	config := InstallConfig{
		Force:     false,
		HookTypes: []string{"pre-commit"},
	}

	// Run the function
	err = builder.runInstallWithConfig(config, nil, nil)
	require.NoError(t, err)

	// Verify no hooks were created
	hookPath := filepath.Join(repoPath, ".git", "hooks", "pre-commit")
	_, err = os.Stat(hookPath)
	assert.True(t, os.IsNotExist(err), "Hook file should not exist when disabled: %s", hookPath)
}
