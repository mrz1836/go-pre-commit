package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestLoad(t *testing.T) {
	// Clean environment to avoid interference from other tests
	envVarsToClean := []string{
		"GO_PRE_COMMIT_LOG_LEVEL", "ENABLE_GO_PRE_COMMIT",
		"GO_PRE_COMMIT_MAX_FILE_SIZE_MB", "GO_PRE_COMMIT_MAX_FILES_OPEN",
		"GO_PRE_COMMIT_TIMEOUT_SECONDS", "GO_PRE_COMMIT_ENABLE_FUMPT",
		"GO_PRE_COMMIT_ENABLE_LINT", "GO_PRE_COMMIT_ENABLE_MOD_TIDY",
		"GO_PRE_COMMIT_ENABLE_WHITESPACE", "GO_PRE_COMMIT_ENABLE_EOF",
		// CI-related environment variables
		"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "BUILDKITE",
		"CIRCLECI", "TRAVIS", "APPVEYOR", "AZURE_HTTP_USER_AGENT",
		"TEAMCITY_VERSION", "DRONE", "SEMAPHORE", "CODEBUILD_BUILD_ID",
		"GO_PRE_COMMIT_AUTO_ADJUST_CI_TIMEOUTS", "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT",
	}

	originalEnvs := make(map[string]string)
	for _, envVar := range envVarsToClean {
		originalEnvs[envVar] = os.Getenv(envVar)
		_ = os.Unsetenv(envVar)
	}

	defer func() {
		for envVar, value := range originalEnvs {
			if value != "" {
				_ = os.Setenv(envVar, value)
			} else {
				_ = os.Unsetenv(envVar)
			}
		}
	}()

	// Create isolated test directory with .env.base file
	tmpDir := t.TempDir()
	originalWD, err := os.Getwd()
	require.NoError(t, err)

	// Create .github/.env.base in test directory with test configuration
	githubDir := filepath.Join(tmpDir, ".github")
	require.NoError(t, os.MkdirAll(githubDir, 0o750))
	envFile := filepath.Join(githubDir, ".env.base")
	envContent := `# Test environment configuration
ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=debug
GO_PRE_COMMIT_MAX_FILE_SIZE_MB=10
GO_PRE_COMMIT_MAX_FILES_OPEN=100
GO_PRE_COMMIT_TIMEOUT_SECONDS=300
GO_PRE_COMMIT_ENABLE_FUMPT=true
GO_PRE_COMMIT_ENABLE_LINT=true
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
GO_PRE_COMMIT_ENABLE_EOF=true
`
	require.NoError(t, os.WriteFile(envFile, []byte(envContent), 0o600))

	// Change to test directory and restore after test
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(originalWD) }()

	// Test loading configuration
	cfg, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify some expected values
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, int64(10*1024*1024), cfg.MaxFileSize)
	assert.Equal(t, 100, cfg.MaxFilesOpen)
	assert.Equal(t, 300, cfg.Timeout)

	// Check that checks are enabled by default
	assert.True(t, cfg.Checks.Fumpt)
	assert.True(t, cfg.Checks.Lint)
	assert.True(t, cfg.Checks.ModTidy)
	assert.True(t, cfg.Checks.Whitespace)
	assert.True(t, cfg.Checks.EOF)
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"true value", "true", false, true},
		{"false value", "false", true, false},
		{"empty value", "", true, true},
		{"invalid value", "invalid", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("TEST_BOOL", tt.envValue)
			defer func() { _ = os.Unsetenv("TEST_BOOL") }()

			result := getBoolEnv("TEST_BOOL", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"valid int", "42", 0, 42},
		{"empty value", "", 10, 10},
		{"invalid value", "abc", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("TEST_INT", tt.envValue)
			defer func() { _ = os.Unsetenv("TEST_INT") }()

			result := getIntEnv("TEST_INT", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
	}{
		{"value set", "test", "default", "test"},
		{"empty value", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv("TEST_STRING", tt.envValue)
			defer func() { _ = os.Unsetenv("TEST_STRING") }()

			result := getStringEnv("TEST_STRING", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Comprehensive test suite for config functionality

type ConfigTestSuite struct {
	suite.Suite

	tempDir string
	oldDir  string
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "config_test_*")
	s.Require().NoError(err)

	s.oldDir, err = os.Getwd()
	s.Require().NoError(err)

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Clear environment variables to ensure clean test state
	s.clearEnvVars()
}

func (s *ConfigTestSuite) TearDownTest() {
	// Clear environment variables after test
	s.clearEnvVars()

	if s.oldDir != "" {
		err := os.Chdir(s.oldDir)
		s.Require().NoError(err)
	}
	if s.tempDir != "" {
		err := os.RemoveAll(s.tempDir)
		s.Require().NoError(err)
	}
}

func (s *ConfigTestSuite) clearEnvVars() {
	envVars := []string{
		"ENABLE_GO_PRE_COMMIT",
		"GO_PRE_COMMIT_LOG_LEVEL",
		"GO_PRE_COMMIT_MAX_FILE_SIZE_MB",
		"GO_PRE_COMMIT_MAX_FILES_OPEN",
		"GO_PRE_COMMIT_TIMEOUT_SECONDS",
		"GO_PRE_COMMIT_ENABLE_FUMPT",
		"GO_PRE_COMMIT_ENABLE_LINT",
		"GO_PRE_COMMIT_ENABLE_MOD_TIDY",
		"GO_PRE_COMMIT_ENABLE_WHITESPACE",
		"GO_PRE_COMMIT_ENABLE_EOF",
		"GO_PRE_COMMIT_FUMPT_VERSION",
		"GO_PRE_COMMIT_GOLANGCI_LINT_VERSION",
		"GO_PRE_COMMIT_PARALLEL_WORKERS",
		"GO_PRE_COMMIT_FAIL_FAST",
		"GO_PRE_COMMIT_FUMPT_TIMEOUT",
		"GO_PRE_COMMIT_LINT_TIMEOUT",
		"GO_PRE_COMMIT_MOD_TIDY_TIMEOUT",
		"GO_PRE_COMMIT_WHITESPACE_TIMEOUT",
		"GO_PRE_COMMIT_EOF_TIMEOUT",
		"GO_PRE_COMMIT_HOOKS_PATH",
		"GO_PRE_COMMIT_EXCLUDE_PATTERNS",
		"GO_PRE_COMMIT_COLOR_OUTPUT",
		// CI-related environment variables
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"JENKINS_URL",
		"BUILDKITE",
		"CIRCLECI",
		"TRAVIS",
		"APPVEYOR",
		"AZURE_HTTP_USER_AGENT",
		"TEAMCITY_VERSION",
		"DRONE",
		"SEMAPHORE",
		"CODEBUILD_BUILD_ID",
		"GO_PRE_COMMIT_AUTO_ADJUST_CI_TIMEOUTS",
		"GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT",
	}

	for _, envVar := range envVars {
		_ = os.Unsetenv(envVar)
	}
}

func (s *ConfigTestSuite) createEnvFile(content string) {
	githubDir := filepath.Join(s.tempDir, ".github")
	err := os.MkdirAll(githubDir, 0o750)
	s.Require().NoError(err)

	envFile := filepath.Join(githubDir, ".env.base")
	err = os.WriteFile(envFile, []byte(content), 0o600)
	s.Require().NoError(err)
}

// TestLoadWithCustomConfiguration tests loading with custom environment variables
func (s *ConfigTestSuite) TestLoadWithCustomConfiguration() {
	envContent := `# Custom configuration
ENABLE_GO_PRE_COMMIT=false
GO_PRE_COMMIT_LOG_LEVEL=debug
GO_PRE_COMMIT_MAX_FILE_SIZE_MB=5
GO_PRE_COMMIT_MAX_FILES_OPEN=50
GO_PRE_COMMIT_TIMEOUT_SECONDS=300
GO_PRE_COMMIT_ENABLE_FUMPT=false
GO_PRE_COMMIT_ENABLE_LINT=false
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true
GO_PRE_COMMIT_ENABLE_WHITESPACE=false
GO_PRE_COMMIT_ENABLE_EOF=true
GO_PRE_COMMIT_FUMPT_VERSION=v0.5.0
GO_PRE_COMMIT_GOLANGCI_LINT_VERSION=v1.54.0
GO_PRE_COMMIT_PARALLEL_WORKERS=4
GO_PRE_COMMIT_FAIL_FAST=true
GO_PRE_COMMIT_FUMPT_TIMEOUT=60
GO_PRE_COMMIT_LINT_TIMEOUT=90
GO_PRE_COMMIT_MOD_TIDY_TIMEOUT=45
GO_PRE_COMMIT_WHITESPACE_TIMEOUT=15
GO_PRE_COMMIT_EOF_TIMEOUT=10
GO_PRE_COMMIT_HOOKS_PATH=.git/custom-hooks
GO_PRE_COMMIT_EXCLUDE_PATTERNS=vendor/,dist/,build/
GO_PRE_COMMIT_COLOR_OUTPUT=false
`
	s.createEnvFile(envContent)

	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)

	// Core settings
	s.False(cfg.Enabled)
	s.Equal("debug", cfg.LogLevel)
	s.Equal(int64(5*1024*1024), cfg.MaxFileSize)
	s.Equal(50, cfg.MaxFilesOpen)
	s.Equal(300, cfg.Timeout)

	// Check configurations
	s.False(cfg.Checks.Fumpt)
	s.False(cfg.Checks.Lint)
	s.True(cfg.Checks.ModTidy)
	s.False(cfg.Checks.Whitespace)
	s.True(cfg.Checks.EOF)

	// Tool versions
	s.Equal("v0.5.0", cfg.ToolVersions.Fumpt)
	s.Equal("v1.54.0", cfg.ToolVersions.GolangciLint)

	// Performance settings
	s.Equal(4, cfg.Performance.ParallelWorkers)
	s.True(cfg.Performance.FailFast)

	// Check timeouts
	s.Equal(60, cfg.CheckTimeouts.Fumpt)
	s.Equal(90, cfg.CheckTimeouts.Lint)
	s.Equal(45, cfg.CheckTimeouts.ModTidy)
	s.Equal(15, cfg.CheckTimeouts.Whitespace)
	s.Equal(10, cfg.CheckTimeouts.EOF)

	// Git settings
	s.Equal(".git/custom-hooks", cfg.Git.HooksPath)
	s.Equal([]string{"vendor/", "dist/", "build/"}, cfg.Git.ExcludePatterns)

	// UI settings
	s.False(cfg.UI.ColorOutput)

	// Directory should be empty for PATH-based binary lookup approach
	// We no longer use directory-based approach, binary is found via PATH
	s.Empty(cfg.Directory)
}

// TestLoadWithMinimalConfiguration tests loading with minimal configuration
func (s *ConfigTestSuite) TestLoadWithMinimalConfiguration() {
	envContent := `ENABLE_GO_PRE_COMMIT=true
`
	s.createEnvFile(envContent)

	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)

	// Should use defaults for unspecified values
	s.True(cfg.Enabled)
	s.Equal("info", cfg.LogLevel)
	s.Equal(int64(10*1024*1024), cfg.MaxFileSize)
	s.Equal(100, cfg.MaxFilesOpen)
	s.Equal(300, cfg.Timeout)
	s.True(cfg.Checks.Fumpt)
	s.True(cfg.Checks.Lint)
	s.True(cfg.Checks.ModTidy)
	s.True(cfg.Checks.Whitespace)
	s.True(cfg.Checks.EOF)
}

// TestLoadWithEmptyExcludePatterns tests exclude patterns handling
func (s *ConfigTestSuite) TestLoadWithEmptyExcludePatterns() {
	envContent := `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_EXCLUDE_PATTERNS=
`
	s.createEnvFile(envContent)

	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)
	// When empty string is provided via environment variable,
	// getStringEnv returns the default value "vendor/,node_modules/,.git/"
	// So we expect the default patterns to be present
	s.Equal([]string{"vendor/", "node_modules/", ".git/"}, cfg.Git.ExcludePatterns)
}

// TestLoadWithSpacedExcludePatterns tests exclude patterns with spaces
func (s *ConfigTestSuite) TestLoadWithSpacedExcludePatterns() {
	envContent := `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_EXCLUDE_PATTERNS=vendor/ , node_modules/ , .git/
`
	s.createEnvFile(envContent)

	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)
	s.Equal([]string{"vendor/", "node_modules/", ".git/"}, cfg.Git.ExcludePatterns)
}

// TestLoadMissingEnvFile tests behavior when .env.base file is not found
func (s *ConfigTestSuite) TestLoadMissingEnvFile() {
	// Don't create .env.base file
	cfg, err := Load()
	s.Require().Error(err)
	s.Nil(cfg)
	s.Contains(err.Error(), "failed to find .env.base")
}

// TestLoadCorruptedEnvFile tests behavior with corrupted env file
func (s *ConfigTestSuite) TestLoadCorruptedEnvFile() {
	// Create a directory instead of a file to simulate corruption
	githubDir := filepath.Join(s.tempDir, ".github")
	err := os.MkdirAll(githubDir, 0o750)
	s.Require().NoError(err)

	envPath := filepath.Join(githubDir, ".env.base")
	err = os.Mkdir(envPath, 0o750) // Create directory instead of file
	s.Require().NoError(err)

	cfg, err := Load()
	s.Require().Error(err)
	s.Nil(cfg)
	s.Contains(err.Error(), "failed to load")
}

// TestFindEnvFileInParentDirectories tests finding env file in parent directories
func (s *ConfigTestSuite) TestFindEnvFileInParentDirectories() {
	// Create env file in parent directory
	envContent := `ENABLE_GO_PRE_COMMIT=true
`
	s.createEnvFile(envContent)

	// Create subdirectory and change to it
	subDir := filepath.Join(s.tempDir, "subdir", "deep")
	err := os.MkdirAll(subDir, 0o750)
	s.Require().NoError(err)

	err = os.Chdir(subDir)
	s.Require().NoError(err)

	// Should find env file in parent
	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)
	s.True(cfg.Enabled)
}

// TestFindEnvFileInCurrentDirectory tests finding env file in current directory
func (s *ConfigTestSuite) TestFindEnvFileInCurrentDirectory() {
	envContent := `ENABLE_GO_PRE_COMMIT=true
`
	s.createEnvFile(envContent)

	// Should find env file in current directory
	envPath, err := findBaseEnvFile()
	s.Require().NoError(err)
	s.Equal(".github/.env.base", envPath)
}

// TestConfigStructInitialization tests that all config fields are properly initialized
func (s *ConfigTestSuite) TestConfigStructInitialization() {
	envContent := `ENABLE_GO_PRE_COMMIT=true
`
	s.createEnvFile(envContent)

	cfg, err := Load()
	s.Require().NoError(err)
	s.NotNil(cfg)

	// Verify all major struct fields are initialized
	// Directory is intentionally empty for PATH-based approach
	s.Empty(cfg.Directory)
	s.NotEmpty(cfg.LogLevel)
	s.Positive(cfg.MaxFileSize)
	s.Positive(cfg.MaxFilesOpen)
	s.Positive(cfg.Timeout)
	s.NotEmpty(cfg.ToolVersions.Fumpt)
	s.NotEmpty(cfg.ToolVersions.GolangciLint)
	s.GreaterOrEqual(cfg.Performance.ParallelWorkers, 0)
	s.Positive(cfg.CheckTimeouts.Fumpt)
	s.Positive(cfg.CheckTimeouts.Lint)
	s.Positive(cfg.CheckTimeouts.ModTidy)
	s.Positive(cfg.CheckTimeouts.Whitespace)
	s.Positive(cfg.CheckTimeouts.EOF)
	s.NotEmpty(cfg.Git.HooksPath)
}

// Unit tests for edge cases and error conditions

func TestGetBoolEnvEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"1 as true", "1", false, true},
		{"0 as false", "0", true, false},
		{"TRUE uppercase", "TRUE", false, true},
		{"FALSE uppercase", "FALSE", true, false},
		{"yes value", "yes", false, false}, // Should use default for invalid
		{"no value", "no", true, true},     // Should use default for invalid
		{"random string", "random", false, false},
		{"whitespace value", " true ", false, false}, // Whitespace should fail parsing
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.Setenv("TEST_BOOL_EDGE", tt.envValue))
			defer func() {
				if err := os.Unsetenv("TEST_BOOL_EDGE"); err != nil {
					t.Logf("Failed to unset TEST_BOOL_EDGE: %v", err)
				}
			}()

			result := getBoolEnv("TEST_BOOL_EDGE", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntEnvEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue int
		expected     int
	}{
		{"negative int", "-42", 0, -42},
		{"zero", "0", 10, 0},
		{"large number", "999999", 0, 999999},
		{"float value", "42.5", 5, 5}, // Should use default for invalid
		{"whitespace", " 42 ", 5, 5},  // Should use default for invalid
		{"hex value", "0x42", 5, 5},   // Should use default for invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.Setenv("TEST_INT_EDGE", tt.envValue))
			defer func() {
				if err := os.Unsetenv("TEST_INT_EDGE"); err != nil {
					t.Logf("Failed to unset TEST_INT_EDGE: %v", err)
				}
			}()

			result := getIntEnv("TEST_INT_EDGE", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStringEnvEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
	}{
		{"whitespace value", "  spaces  ", "default", "  spaces  "},
		{"special characters", "!@#$%^&*()", "default", "!@#$%^&*()"},
		{"unicode", "テスト", "default", "テスト"},
		{"newlines", "line1\nline2", "default", "line1\nline2"},
		{"empty string", "", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.Setenv("TEST_STRING_EDGE", tt.envValue))
			defer func() {
				if err := os.Unsetenv("TEST_STRING_EDGE"); err != nil {
					t.Logf("Failed to unset TEST_STRING_EDGE: %v", err)
				}
			}()

			result := getStringEnv("TEST_STRING_EDGE", tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFindBaseEnvFileErrors tests error conditions in findBaseEnvFile
func TestFindBaseEnvFileErrors(t *testing.T) {
	// Test when we can't get current working directory
	// This is hard to test directly, but we can test the search logic

	// Create temp directory structure without .github/.env.base
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			t.Logf("Failed to restore directory: %v", chErr)
		}
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Should return error when no .env.base found
	_, err = findBaseEnvFile()
	assert.Error(t, err)
}

// TestLoadIntegrationWithRealProject tests loading in a real project structure
func TestLoadIntegrationWithRealProject(t *testing.T) {
	// Create a realistic project structure
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			t.Logf("Failed to restore directory: %v", chErr)
		}
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create project structure
	projectDirs := []string{
		"cmd/myapp",
		"internal/pkg",
		"pkg/api",
		".github",
	}

	for _, dir := range projectDirs {
		err = os.MkdirAll(dir, 0o750)
		require.NoError(t, err)
	}

	// Create .env.base file
	envContent := `# Project configuration
ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=debug
GO_PRE_COMMIT_ENABLE_FUMPT=true
GO_PRE_COMMIT_ENABLE_LINT=true
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
GO_PRE_COMMIT_ENABLE_EOF=true
`
	envFile := filepath.Join(tmpDir, ".github", ".env.base")
	err = os.WriteFile(envFile, []byte(envContent), 0o600)
	require.NoError(t, err)

	// Test loading from various subdirectories
	subDirs := []string{
		".",
		"cmd/myapp",
		"internal/pkg",
		"pkg/api",
	}

	for _, subDir := range subDirs {
		t.Run("from "+subDir, func(t *testing.T) {
			err = os.Chdir(filepath.Join(tmpDir, subDir))
			require.NoError(t, err)

			cfg, err := Load()
			require.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.True(t, cfg.Enabled)
			assert.Equal(t, "debug", cfg.LogLevel)

			// Directory should be empty for PATH-based binary lookup approach
			// We no longer use directory-based approach, binary is found via PATH
			assert.Empty(t, cfg.Directory)
		})
	}
}
