package validation

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-pre-commit/internal/config"
	precommiterrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

// ConfigValidationTestSuite validates configuration loading under various scenarios
type ConfigValidationTestSuite struct {
	suite.Suite

	tempDir     string
	originalWD  string
	originalEnv map[string]string
}

// SetupSuite initializes the test environment
func (s *ConfigValidationTestSuite) SetupSuite() {
	// Robust working directory capture for CI environments
	s.originalWD = s.getSafeWorkingDirectory()

	// Create temporary directory structure
	s.tempDir = s.T().TempDir()

	// Store original environment variables that might affect tests
	s.originalEnv = make(map[string]string)
	envVarsToSave := []string{
		"ENABLE_GO_PRE_COMMIT", "GO_PRE_COMMIT_LOG_LEVEL",
		"GO_PRE_COMMIT_ENABLE_FUMPT", "GO_PRE_COMMIT_ENABLE_LINT",
		"GO_PRE_COMMIT_ENABLE_MOD_TIDY", "GO_PRE_COMMIT_ENABLE_WHITESPACE",
		"GO_PRE_COMMIT_ENABLE_EOF", "GO_PRE_COMMIT_TIMEOUT_SECONDS",
		"GO_PRE_COMMIT_MAX_FILE_SIZE_MB", "GO_PRE_COMMIT_MAX_FILES_OPEN",
		"GO_PRE_COMMIT_FUMPT_VERSION", "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION",
		"GO_PRE_COMMIT_PARALLEL_WORKERS", "GO_PRE_COMMIT_FAIL_FAST",
		"GO_PRE_COMMIT_FUMPT_TIMEOUT", "GO_PRE_COMMIT_LINT_TIMEOUT",
		"GO_PRE_COMMIT_MOD_TIDY_TIMEOUT", "GO_PRE_COMMIT_WHITESPACE_TIMEOUT",
		"GO_PRE_COMMIT_EOF_TIMEOUT", "GO_PRE_COMMIT_HOOKS_PATH",
		"GO_PRE_COMMIT_COLOR_OUTPUT", "NO_COLOR", "CI",
		// Additional CI environment variables
		"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "BUILDKITE",
		"CIRCLECI", "TRAVIS", "APPVEYOR", "AZURE_HTTP_USER_AGENT",
		"TEAMCITY_VERSION", "DRONE", "SEMAPHORE", "CODEBUILD_BUILD_ID",
		"GO_PRE_COMMIT_AUTO_ADJUST_CI_TIMEOUTS", "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT",
	}

	for _, envVar := range envVarsToSave {
		s.originalEnv[envVar] = os.Getenv(envVar)
	}

	// Set up required environment variables for tests
	s.Require().NoError(os.Setenv("GO_PRE_COMMIT_FUMPT_VERSION", "latest"))
	s.Require().NoError(os.Setenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION", "latest"))

	// Change to temp directory for tests
	s.Require().NoError(os.Chdir(s.tempDir))
}

// TearDownSuite cleans up the test environment
func (s *ConfigValidationTestSuite) TearDownSuite() {
	// Restore original working directory
	_ = os.Chdir(s.originalWD)

	// Restore original environment variables
	for envVar, originalValue := range s.originalEnv {
		if originalValue == "" {
			s.Require().NoError(os.Unsetenv(envVar))
		} else {
			s.Require().NoError(os.Setenv(envVar, originalValue))
		}
	}
}

// getSafeWorkingDirectory attempts to get current working directory with fallbacks for CI
func (s *ConfigValidationTestSuite) getSafeWorkingDirectory() string {
	// First attempt: standard os.Getwd()
	if wd, err := os.Getwd(); err == nil {
		// Verify the directory actually exists and is accessible
		if _, statErr := os.Stat(wd); statErr == nil {
			return wd
		}
	}

	// Fallback 1: Try to find git repository root
	if gitRoot, err := s.findGitRoot(); err == nil {
		// Verify git root exists and is accessible
		if _, statErr := os.Stat(gitRoot); statErr == nil {
			return gitRoot
		}
	}

	// Fallback 2: Use current user's home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		return homeDir
	}

	// Final fallback: Use temp directory
	return os.TempDir()
}

// findGitRoot attempts to find the git repository root
func (s *ConfigValidationTestSuite) findGitRoot() (string, error) {
	// Start from current executable's directory if possible
	if exePath, err := os.Executable(); err == nil {
		dir := filepath.Dir(exePath)
		for dir != filepath.Dir(dir) { // Stop at root
			if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
				return dir, nil
			}
			dir = filepath.Dir(dir)
		}
	}

	// Try common project paths relative to GOPATH or GOMOD
	if goPath := os.Getenv("GOPATH"); goPath != "" {
		projectPath := filepath.Join(goPath, "src", "github.com", "mrz1836", "go-pre-commit")
		if _, err := os.Stat(projectPath); err == nil {
			return projectPath, nil
		}
	}

	return "", os.ErrNotExist
}

// TearDownTest cleans up after each test
func (s *ConfigValidationTestSuite) TearDownTest() {
	// Clean up any environment variables set during the test
	envVarsToClean := []string{
		"ENABLE_GO_PRE_COMMIT", "GO_PRE_COMMIT_LOG_LEVEL",
		"GO_PRE_COMMIT_ENABLE_FUMPT", "GO_PRE_COMMIT_ENABLE_LINT",
		"GO_PRE_COMMIT_ENABLE_MOD_TIDY", "GO_PRE_COMMIT_ENABLE_WHITESPACE",
		"GO_PRE_COMMIT_ENABLE_EOF", "GO_PRE_COMMIT_TIMEOUT_SECONDS",
		"GO_PRE_COMMIT_MAX_FILE_SIZE_MB", "GO_PRE_COMMIT_MAX_FILES_OPEN",
		"GO_PRE_COMMIT_FUMPT_VERSION", "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION",
		"GO_PRE_COMMIT_PARALLEL_WORKERS", "GO_PRE_COMMIT_FAIL_FAST",
		"GO_PRE_COMMIT_FUMPT_TIMEOUT", "GO_PRE_COMMIT_LINT_TIMEOUT",
		"GO_PRE_COMMIT_MOD_TIDY_TIMEOUT", "GO_PRE_COMMIT_WHITESPACE_TIMEOUT",
		"GO_PRE_COMMIT_EOF_TIMEOUT", "GO_PRE_COMMIT_HOOKS_PATH",
		"GO_PRE_COMMIT_COLOR_OUTPUT", "NO_COLOR", "CI",
	}

	for _, envVar := range envVarsToClean {
		if originalValue, exists := s.originalEnv[envVar]; exists {
			if originalValue == "" {
				s.Require().NoError(os.Unsetenv(envVar))
			} else {
				s.Require().NoError(os.Setenv(envVar, originalValue))
			}
		} else {
			s.Require().NoError(os.Unsetenv(envVar))
		}
	}
}

// TestMissingConfigFile validates behavior when .env.base is missing
func (s *ConfigValidationTestSuite) TestMissingConfigFile() {
	// Ensure no .env.base file exists
	_, err := config.Load()

	// Should return a specific error about missing env file
	s.Require().Error(err)
	s.True(errors.Is(err, precommiterrors.ErrEnvFileNotFound) ||
		strings.Contains(err.Error(), ".env.base"))
}

// TestInvalidConfigFile validates behavior with malformed config files
func (s *ConfigValidationTestSuite) TestInvalidConfigFile() {
	testCases := []struct {
		name        string
		content     string
		shouldError bool
		description string
	}{
		{
			name:        "Empty File",
			content:     "",
			shouldError: false, // Should use defaults
			description: "Empty configuration file should use defaults",
		},
		{
			name:        "Invalid Boolean",
			content:     "ENABLE_GO_PRE_COMMIT=maybe\n",
			shouldError: false, // Should use default for invalid boolean
			description: "Invalid boolean should fall back to default",
		},
		{
			name:        "Invalid Integer",
			content:     "GO_PRE_COMMIT_TIMEOUT_SECONDS=not-a-number\n",
			shouldError: false, // Should use default for invalid integer
			description: "Invalid integer should fall back to default",
		},
		{
			name: "Invalid Timeout Values",
			content: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_TIMEOUT_SECONDS=0
GO_PRE_COMMIT_FUMPT_TIMEOUT=-1
`,
			shouldError: true, // Validation should catch invalid timeouts
			description: "Zero or negative timeouts should cause validation error",
		},
		{
			name: "Invalid Log Level",
			content: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=invalid
`,
			shouldError: true, // Validation should catch invalid log level
			description: "Invalid log level should cause validation error",
		},
		{
			name: "Malformed Environment Variables",
			content: `ENABLE_GO_PRE_COMMIT=true
=invalid_line_no_key
KEY_WITHOUT_VALUE
VALID_KEY=valid_value
`,
			shouldError: true, // godotenv should fail on malformed syntax
			description: "Malformed environment variables should cause loading to fail",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Clean environment variables before each subtest
			envVarsToClean := []string{
				"ENABLE_GO_PRE_COMMIT", "GO_PRE_COMMIT_LOG_LEVEL",
				"GO_PRE_COMMIT_ENABLE_FUMPT", "GO_PRE_COMMIT_ENABLE_LINT",
				"GO_PRE_COMMIT_ENABLE_MOD_TIDY", "GO_PRE_COMMIT_ENABLE_WHITESPACE",
				"GO_PRE_COMMIT_ENABLE_EOF", "GO_PRE_COMMIT_TIMEOUT_SECONDS",
				"GO_PRE_COMMIT_MAX_FILE_SIZE_MB", "GO_PRE_COMMIT_MAX_FILES_OPEN",
				"GO_PRE_COMMIT_FUMPT_VERSION", "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION",
				"GO_PRE_COMMIT_PARALLEL_WORKERS", "GO_PRE_COMMIT_FAIL_FAST",
				"GO_PRE_COMMIT_FUMPT_TIMEOUT", "GO_PRE_COMMIT_LINT_TIMEOUT",
				"GO_PRE_COMMIT_MOD_TIDY_TIMEOUT", "GO_PRE_COMMIT_WHITESPACE_TIMEOUT",
				"GO_PRE_COMMIT_EOF_TIMEOUT", "GO_PRE_COMMIT_HOOKS_PATH",
				"GO_PRE_COMMIT_EXCLUDE_PATTERNS", "GO_PRE_COMMIT_WHITESPACE_AUTO_STAGE",
				"GO_PRE_COMMIT_EOF_AUTO_STAGE", "GO_PRE_COMMIT_COLOR_OUTPUT",
			}
			for _, envVar := range envVarsToClean {
				s.Require().NoError(os.Unsetenv(envVar))
			}

			// Create .github directory
			githubDir := filepath.Join(s.tempDir, ".github")
			s.Require().NoError(os.MkdirAll(githubDir, 0o750))

			// Create .env.base file with test content
			envFile := filepath.Join(githubDir, ".env.base")
			s.Require().NoError(os.WriteFile(envFile, []byte(tc.content), 0o600))

			// Try to load configuration
			cfg, err := config.Load()

			if tc.shouldError {
				s.Require().Error(err, tc.description)
			} else {
				s.Require().NoError(err, tc.description)
				s.NotNil(cfg, "Configuration should be loaded successfully")
			}

			// Clean up for next test
			s.Require().NoError(os.RemoveAll(githubDir))
		})
	}
}

// TestEnvironmentVariablePrecedence validates environment variable precedence
func (s *ConfigValidationTestSuite) TestEnvironmentVariablePrecedence() {
	// Create base .env.base file
	githubDir := filepath.Join(s.tempDir, ".github")
	s.Require().NoError(os.MkdirAll(githubDir, 0o750))

	envFile := filepath.Join(githubDir, ".env.base")
	baseConfig := `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=info
GO_PRE_COMMIT_TIMEOUT_SECONDS=300
GO_PRE_COMMIT_PARALLEL_WORKERS=2
GO_PRE_COMMIT_ENABLE_FUMPT=true
GO_PRE_COMMIT_ENABLE_LINT=false
`
	s.Require().NoError(os.WriteFile(envFile, []byte(baseConfig), 0o600))

	testCases := []struct {
		name            string
		envOverrides    map[string]string
		expectedLog     string
		expectedWorkers int
		expectedFumpt   bool
		description     string
	}{
		{
			name: "Environment Variables Override File",
			envOverrides: map[string]string{
				"GO_PRE_COMMIT_LOG_LEVEL":        "debug",
				"GO_PRE_COMMIT_PARALLEL_WORKERS": "4",
				"GO_PRE_COMMIT_ENABLE_FUMPT":     "false",
			},
			expectedLog:     "debug",
			expectedWorkers: 4,
			expectedFumpt:   false,
			description:     "Environment variables should take precedence over file values",
		},
		{
			name: "Partial Override",
			envOverrides: map[string]string{
				"GO_PRE_COMMIT_LOG_LEVEL": "warn",
			},
			expectedLog:     "warn",
			expectedWorkers: 2,    // From file
			expectedFumpt:   true, // From file
			description:     "Only overridden values should change, others use file values",
		},
		{
			name:            "No Override",
			envOverrides:    map[string]string{},
			expectedLog:     "info",
			expectedWorkers: 2,
			expectedFumpt:   true,
			description:     "Without overrides, should use file values",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Set environment variable overrides
			for key, value := range tc.envOverrides {
				s.Require().NoError(os.Setenv(key, value))
			}

			// Load configuration
			cfg, err := config.Load()
			s.Require().NoError(err, tc.description)

			// Validate expected values
			s.Equal(tc.expectedLog, cfg.LogLevel, "Log level should match expected")
			s.Equal(tc.expectedWorkers, cfg.Performance.ParallelWorkers, "Workers should match expected")
			s.Equal(tc.expectedFumpt, cfg.Checks.Fumpt, "Fumpt setting should match expected")

			// Clean up environment variables
			for key := range tc.envOverrides {
				s.Require().NoError(os.Unsetenv(key))
			}
		})
	}
}

// TestConfigurationDefaults validates default values are properly set
func (s *ConfigValidationTestSuite) TestConfigurationDefaults() {
	// Clean environment variables to ensure defaults are tested
	envVarsToClean := []string{
		"ENABLE_GO_PRE_COMMIT", "GO_PRE_COMMIT_LOG_LEVEL",
		"GO_PRE_COMMIT_ENABLE_FUMPT", "GO_PRE_COMMIT_ENABLE_LINT",
		"GO_PRE_COMMIT_ENABLE_MOD_TIDY", "GO_PRE_COMMIT_ENABLE_WHITESPACE",
		"GO_PRE_COMMIT_ENABLE_EOF", "GO_PRE_COMMIT_TIMEOUT_SECONDS",
		"GO_PRE_COMMIT_MAX_FILE_SIZE_MB", "GO_PRE_COMMIT_MAX_FILES_OPEN",
		"GO_PRE_COMMIT_FUMPT_VERSION", "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION",
		"GO_PRE_COMMIT_PARALLEL_WORKERS", "GO_PRE_COMMIT_FAIL_FAST",
		"GO_PRE_COMMIT_FUMPT_TIMEOUT", "GO_PRE_COMMIT_LINT_TIMEOUT",
		"GO_PRE_COMMIT_MOD_TIDY_TIMEOUT", "GO_PRE_COMMIT_WHITESPACE_TIMEOUT",
		"GO_PRE_COMMIT_EOF_TIMEOUT", "GO_PRE_COMMIT_HOOKS_PATH",
		"GO_PRE_COMMIT_COLOR_OUTPUT", "NO_COLOR", "CI",
		// Additional CI environment variables to ensure clean state
		"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "BUILDKITE",
		"CIRCLECI", "TRAVIS", "APPVEYOR", "AZURE_HTTP_USER_AGENT",
		"TEAMCITY_VERSION", "DRONE", "SEMAPHORE", "CODEBUILD_BUILD_ID",
		"GO_PRE_COMMIT_AUTO_ADJUST_CI_TIMEOUTS", "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT",
	}

	// Store and clear environment variables
	originalEnvValues := make(map[string]string)
	for _, envVar := range envVarsToClean {
		originalEnvValues[envVar] = os.Getenv(envVar)
		s.Require().NoError(os.Unsetenv(envVar))
	}
	defer func() {
		// Restore environment variables
		for envVar, originalValue := range originalEnvValues {
			if originalValue == "" {
				s.Require().NoError(os.Unsetenv(envVar))
			} else {
				s.Require().NoError(os.Setenv(envVar, originalValue))
			}
		}
	}()

	// Create minimal .env.base file with only required setting
	githubDir := filepath.Join(s.tempDir, ".github")
	s.Require().NoError(os.MkdirAll(githubDir, 0o750))

	envFile := filepath.Join(githubDir, ".env.base")
	minimalConfig := `ENABLE_GO_PRE_COMMIT=true
`
	s.Require().NoError(os.WriteFile(envFile, []byte(minimalConfig), 0o600))

	// Save current directory and change to a completely isolated directory
	currentDir, err := os.Getwd()
	s.Require().NoError(err)

	// Create a completely isolated directory that has no parent .env.base files
	isolatedDir := filepath.Join(os.TempDir(), "config-test-isolated")
	s.Require().NoError(os.MkdirAll(isolatedDir, 0o750))
	defer func() { _ = os.RemoveAll(isolatedDir) }()

	// Copy our test config to the isolated directory
	isolatedGithubDir := filepath.Join(isolatedDir, ".github")
	s.Require().NoError(os.MkdirAll(isolatedGithubDir, 0o750))
	isolatedEnvFile := filepath.Join(isolatedGithubDir, ".env.base")
	s.Require().NoError(os.WriteFile(isolatedEnvFile, []byte(minimalConfig), 0o600))

	s.Require().NoError(os.Chdir(isolatedDir))
	defer func() { _ = os.Chdir(currentDir) }()

	cfg, err := config.Load()
	s.Require().NoError(err, "Minimal configuration should load successfully")

	// Validate all defaults (based on actual code defaults in config.go)
	expectedDefaults := map[string]interface{}{
		"LogLevel":                    "info",
		"MaxFileSize":                 int64(10 * 1024 * 1024), // 10MB
		"MaxFilesOpen":                100,
		"Timeout":                     300, // Actual default from config.go line 96
		"Checks.Fumpt":                true,
		"Checks.Lint":                 true,
		"Checks.ModTidy":              true,
		"Checks.Whitespace":           true,
		"Checks.EOF":                  true,
		"ToolVersions.Fumpt":          "latest",
		"ToolVersions.GolangciLint":   "latest",
		"Performance.ParallelWorkers": 0, // Actual default from config.go line 114 (0 = auto)
		"Performance.FailFast":        false,
		"CheckTimeouts.Fumpt":         30,
		"CheckTimeouts.Lint":          60,
		"CheckTimeouts.ModTidy":       30,
		"CheckTimeouts.Whitespace":    30,
		"CheckTimeouts.EOF":           30,
		"Git.HooksPath":               ".git/hooks",
		"UI.ColorOutput":              true,
	}

	// Use reflection-like validation for nested structures
	s.Equal(expectedDefaults["LogLevel"], cfg.LogLevel)
	s.Equal(expectedDefaults["MaxFileSize"], cfg.MaxFileSize)
	s.Equal(expectedDefaults["MaxFilesOpen"], cfg.MaxFilesOpen)
	s.Equal(expectedDefaults["Timeout"], cfg.Timeout)

	s.Equal(expectedDefaults["Checks.Fumpt"], cfg.Checks.Fumpt)
	s.Equal(expectedDefaults["Checks.Lint"], cfg.Checks.Lint)
	s.Equal(expectedDefaults["Checks.ModTidy"], cfg.Checks.ModTidy)
	s.Equal(expectedDefaults["Checks.Whitespace"], cfg.Checks.Whitespace)
	s.Equal(expectedDefaults["Checks.EOF"], cfg.Checks.EOF)

	s.Equal(expectedDefaults["ToolVersions.Fumpt"], cfg.ToolVersions.Fumpt)
	s.Equal(expectedDefaults["ToolVersions.GolangciLint"], cfg.ToolVersions.GolangciLint)

	s.Equal(expectedDefaults["Performance.ParallelWorkers"], cfg.Performance.ParallelWorkers)
	s.Equal(expectedDefaults["Performance.FailFast"], cfg.Performance.FailFast)

	s.Equal(expectedDefaults["CheckTimeouts.Fumpt"], cfg.CheckTimeouts.Fumpt)
	s.Equal(expectedDefaults["CheckTimeouts.Lint"], cfg.CheckTimeouts.Lint)
	s.Equal(expectedDefaults["CheckTimeouts.ModTidy"], cfg.CheckTimeouts.ModTidy)
	s.Equal(expectedDefaults["CheckTimeouts.Whitespace"], cfg.CheckTimeouts.Whitespace)
	s.Equal(expectedDefaults["CheckTimeouts.EOF"], cfg.CheckTimeouts.EOF)

	s.Equal(expectedDefaults["Git.HooksPath"], cfg.Git.HooksPath)
	s.Equal(expectedDefaults["UI.ColorOutput"], cfg.UI.ColorOutput)

	// Validate exclude patterns default
	expectedExcludes := []string{"vendor/", "node_modules/", ".git/"}
	s.Equal(expectedExcludes, cfg.Git.ExcludePatterns)
}

// TestConfigurationValidation validates the validation logic
func (s *ConfigValidationTestSuite) TestConfigurationValidation() {
	testCases := []struct {
		name          string
		config        string
		shouldError   bool
		expectedError string
		description   string
	}{
		{
			name: "Valid Configuration",
			config: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=debug
GO_PRE_COMMIT_TIMEOUT_SECONDS=60
GO_PRE_COMMIT_FUMPT_TIMEOUT=30
GO_PRE_COMMIT_LINT_TIMEOUT=120
GO_PRE_COMMIT_MOD_TIDY_TIMEOUT=30
GO_PRE_COMMIT_WHITESPACE_TIMEOUT=15
GO_PRE_COMMIT_EOF_TIMEOUT=15
GO_PRE_COMMIT_MAX_FILE_SIZE_MB=50
GO_PRE_COMMIT_MAX_FILES_OPEN=200
GO_PRE_COMMIT_PARALLEL_WORKERS=4
`,
			shouldError: false,
			description: "Valid configuration should pass validation",
		},
		{
			name: "Zero Timeout",
			config: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_TIMEOUT_SECONDS=0
`,
			shouldError:   true,
			expectedError: "TIMEOUT_SECONDS must be greater than 0",
			description:   "Zero timeout should fail validation",
		},
		{
			name: "Negative File Size",
			config: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_MAX_FILE_SIZE_MB=-1
`,
			shouldError:   true,
			expectedError: "MAX_FILE_SIZE_MB must be greater than 0",
			description:   "Negative file size should fail validation",
		},
		{
			name: "Invalid Log Level",
			config: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=invalid_level
`,
			shouldError:   true,
			expectedError: "LOG_LEVEL must be one of",
			description:   "Invalid log level should fail validation",
		},
		{
			name: "Invalid Tool Version",
			config: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_FUMPT_VERSION=invalid-version
`,
			shouldError:   true,
			expectedError: "FUMPT_VERSION must be 'latest' or a valid version",
			description:   "Invalid tool version should fail validation",
		},
		{
			name: "Negative Parallel Workers",
			config: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_PARALLEL_WORKERS=-1
`,
			shouldError:   true,
			expectedError: "PARALLEL_WORKERS must be 0 (auto) or positive",
			description:   "Negative parallel workers should fail validation",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Clean environment variables before each subtest
			envVarsToClean := []string{
				"ENABLE_GO_PRE_COMMIT", "GO_PRE_COMMIT_LOG_LEVEL",
				"GO_PRE_COMMIT_ENABLE_FUMPT", "GO_PRE_COMMIT_ENABLE_LINT",
				"GO_PRE_COMMIT_ENABLE_MOD_TIDY", "GO_PRE_COMMIT_ENABLE_WHITESPACE",
				"GO_PRE_COMMIT_ENABLE_EOF", "GO_PRE_COMMIT_TIMEOUT_SECONDS",
				"GO_PRE_COMMIT_MAX_FILE_SIZE_MB", "GO_PRE_COMMIT_MAX_FILES_OPEN",
				"GO_PRE_COMMIT_FUMPT_VERSION", "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION",
				"GO_PRE_COMMIT_PARALLEL_WORKERS", "GO_PRE_COMMIT_FAIL_FAST",
				"GO_PRE_COMMIT_FUMPT_TIMEOUT", "GO_PRE_COMMIT_LINT_TIMEOUT",
				"GO_PRE_COMMIT_MOD_TIDY_TIMEOUT", "GO_PRE_COMMIT_WHITESPACE_TIMEOUT",
				"GO_PRE_COMMIT_EOF_TIMEOUT", "GO_PRE_COMMIT_HOOKS_PATH",
				"GO_PRE_COMMIT_COLOR_OUTPUT", "NO_COLOR", "CI",
			}

			for _, envVar := range envVarsToClean {
				s.Require().NoError(os.Unsetenv(envVar))
			}

			// Create .github directory
			githubDir := filepath.Join(s.tempDir, ".github")
			s.Require().NoError(os.MkdirAll(githubDir, 0o750))

			// Create .env.base file with test content
			envFile := filepath.Join(githubDir, ".env.base")
			s.Require().NoError(os.WriteFile(envFile, []byte(tc.config), 0o600))

			// Try to load configuration
			cfg, err := config.Load()

			if tc.shouldError {
				s.Require().Error(err, tc.description)
				if tc.expectedError != "" && err != nil {
					s.Contains(err.Error(), tc.expectedError,
						"Error should contain expected message")
				}
				s.Nil(cfg, "Configuration should be nil on validation error")
			} else {
				s.Require().NoError(err, tc.description)
				s.NotNil(cfg, "Configuration should be loaded successfully")
			}

			// Clean up for next test
			s.Require().NoError(os.RemoveAll(githubDir))
		})
	}
}

// TestPartialConfiguration validates behavior with partial configurations
func (s *ConfigValidationTestSuite) TestPartialConfiguration() {
	testCases := []struct {
		name        string
		config      string
		description string
	}{
		{
			name: "Only Checks Enabled",
			config: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_ENABLE_FUMPT=true
GO_PRE_COMMIT_ENABLE_LINT=false
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true
GO_PRE_COMMIT_ENABLE_WHITESPACE=false
GO_PRE_COMMIT_ENABLE_EOF=true
`,
			description: "Configuration with only check settings should work",
		},
		{
			name: "Only Performance Settings",
			config: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_PARALLEL_WORKERS=1
GO_PRE_COMMIT_FAIL_FAST=true
`,
			description: "Configuration with only performance settings should work",
		},
		{
			name: "Only Timeout Settings",
			config: `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_FUMPT_TIMEOUT=45
GO_PRE_COMMIT_LINT_TIMEOUT=90
`,
			description: "Configuration with only timeout settings should work",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create .github directory
			githubDir := filepath.Join(s.tempDir, ".github")
			s.Require().NoError(os.MkdirAll(githubDir, 0o750))

			// Create .env.base file with test content
			envFile := filepath.Join(githubDir, ".env.base")
			s.Require().NoError(os.WriteFile(envFile, []byte(tc.config), 0o600))

			// Load configuration
			cfg, err := config.Load()
			s.Require().NoError(err, tc.description)
			s.NotNil(cfg, "Configuration should be loaded successfully")
			if cfg != nil {
				s.True(cfg.Enabled, "System should be enabled")
			}

			// Clean up for next test
			s.Require().NoError(os.RemoveAll(githubDir))
		})
	}
}

// TestConfigurationDirectoryDetection validates directory detection logic
func (s *ConfigValidationTestSuite) TestConfigurationDirectoryDetection() {
	// Clean environment variables to ensure test isolation
	envVarsToClean := []string{
		"ENABLE_GO_PRE_COMMIT", "GO_PRE_COMMIT_LOG_LEVEL",
		"GO_PRE_COMMIT_ENABLE_FUMPT", "GO_PRE_COMMIT_ENABLE_LINT",
		"GO_PRE_COMMIT_ENABLE_MOD_TIDY", "GO_PRE_COMMIT_ENABLE_WHITESPACE",
		"GO_PRE_COMMIT_ENABLE_EOF", "GO_PRE_COMMIT_TIMEOUT_SECONDS",
		"GO_PRE_COMMIT_MAX_FILE_SIZE_MB", "GO_PRE_COMMIT_MAX_FILES_OPEN",
		"GO_PRE_COMMIT_FUMPT_VERSION", "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION",
		"GO_PRE_COMMIT_PARALLEL_WORKERS", "GO_PRE_COMMIT_FAIL_FAST",
		"GO_PRE_COMMIT_FUMPT_TIMEOUT", "GO_PRE_COMMIT_LINT_TIMEOUT",
		"GO_PRE_COMMIT_MOD_TIDY_TIMEOUT", "GO_PRE_COMMIT_WHITESPACE_TIMEOUT",
		"GO_PRE_COMMIT_EOF_TIMEOUT", "GO_PRE_COMMIT_HOOKS_PATH",
		"GO_PRE_COMMIT_COLOR_OUTPUT", "NO_COLOR", "CI",
	}

	// Store and clear environment variables
	originalEnvValues := make(map[string]string)
	for _, envVar := range envVarsToClean {
		originalEnvValues[envVar] = os.Getenv(envVar)
		s.Require().NoError(os.Unsetenv(envVar))
	}
	defer func() {
		// Restore environment variables
		for envVar, originalValue := range originalEnvValues {
			if originalValue == "" {
				s.Require().NoError(os.Unsetenv(envVar))
			} else {
				s.Require().NoError(os.Setenv(envVar, originalValue))
			}
		}
	}()

	// Save current directory
	currentDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() { _ = os.Chdir(currentDir) }()

	// Create a completely isolated directory structure in temp
	isolatedDir := filepath.Join(os.TempDir(), "config-test-directory-detection")
	s.Require().NoError(os.MkdirAll(isolatedDir, 0o750))
	defer func() { _ = os.RemoveAll(isolatedDir) }()

	// Create nested directory structure
	nestedDir := filepath.Join(isolatedDir, "deep", "nested", "directory")
	s.Require().NoError(os.MkdirAll(nestedDir, 0o750))

	// Create .github/.env.base at the isolated root
	githubDir := filepath.Join(isolatedDir, ".github")
	s.Require().NoError(os.MkdirAll(githubDir, 0o750))

	envFile := filepath.Join(githubDir, ".env.base")
	testConfig := `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=debug
`
	s.Require().NoError(os.WriteFile(envFile, []byte(testConfig), 0o600))

	// Change to nested directory
	s.Require().NoError(os.Chdir(nestedDir))

	// Configuration loading should find the .env.base file by walking up
	cfg, err := config.Load()
	s.Require().NoError(err, "Should find .env.base file by walking up directories")
	s.NotNil(cfg, "Configuration should be loaded")
	s.Equal("debug", cfg.LogLevel, "Should load configured log level")
}

// TestConfigurationHelp validates the configuration help functionality
func (s *ConfigValidationTestSuite) TestConfigurationHelp() {
	help := config.GetConfigHelp()

	// Validate that help contains key information
	s.Contains(help, "ENABLE_GO_PRE_COMMIT", "Help should document main enable flag")
	s.Contains(help, "GO_PRE_COMMIT_LOG_LEVEL", "Help should document log level")
	s.Contains(help, "Example .github/.env.base", "Help should include example")
	s.Contains(help, "Core Settings", "Help should have sections")
	s.Contains(help, "Check Configuration", "Help should document checks")
	s.Contains(help, "Performance Settings", "Help should document performance")

	// Should be non-empty and reasonably long
	s.Greater(len(help), 1000, "Help should be comprehensive")
}

// TestSuite runs the configuration validation test suite
func TestConfigValidationTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigValidationTestSuite))
}
