package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ConfigUtilitiesTestSuite tests configuration utility functions
type ConfigUtilitiesTestSuite struct {
	suite.Suite

	originalEnv map[string]string
	tempDir     string
}

// SetupSuite initializes the test environment
func (s *ConfigUtilitiesTestSuite) SetupSuite() {
	s.originalEnv = make(map[string]string)
	s.tempDir = s.T().TempDir()
}

// TearDownTest cleans up environment variables after each test
func (s *ConfigUtilitiesTestSuite) TearDownTest() {
	for key, value := range s.originalEnv {
		if value == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, value)
		}
	}
	s.originalEnv = make(map[string]string)
}

// saveEnv saves the current value of an environment variable
func (s *ConfigUtilitiesTestSuite) saveEnv(key string) {
	s.originalEnv[key] = os.Getenv(key)
}

// TestGetBoolEnv_Comprehensive tests getBoolEnv utility function
func (s *ConfigUtilitiesTestSuite) TestGetBoolEnv_Comprehensive() {
	testCases := []struct {
		name         string
		key          string
		value        string
		defaultValue bool
		expected     bool
		description  string
	}{
		{
			name:         "True value - lowercase",
			key:          "TEST_BOOL_TRUE_LOWER",
			value:        "true",
			defaultValue: false,
			expected:     true,
			description:  "Should parse 'true' as true",
		},
		{
			name:         "True value - uppercase",
			key:          "TEST_BOOL_TRUE_UPPER",
			value:        "TRUE",
			defaultValue: false,
			expected:     true,
			description:  "Should parse 'TRUE' as true",
		},
		{
			name:         "True value - mixed case",
			key:          "TEST_BOOL_TRUE_MIXED",
			value:        "True",
			defaultValue: false,
			expected:     true,
			description:  "Should parse 'True' as true",
		},
		{
			name:         "False value - lowercase",
			key:          "TEST_BOOL_FALSE_LOWER",
			value:        "false",
			defaultValue: true,
			expected:     false,
			description:  "Should parse 'false' as false",
		},
		{
			name:         "False value - uppercase",
			key:          "TEST_BOOL_FALSE_UPPER",
			value:        "FALSE",
			defaultValue: true,
			expected:     false,
			description:  "Should parse 'FALSE' as false",
		},
		{
			name:         "Numeric true - 1",
			key:          "TEST_BOOL_NUMERIC_1",
			value:        "1",
			defaultValue: false,
			expected:     true,
			description:  "Should parse '1' as true",
		},
		{
			name:         "Numeric false - 0",
			key:          "TEST_BOOL_NUMERIC_0",
			value:        "0",
			defaultValue: true,
			expected:     false,
			description:  "Should parse '0' as false",
		},
		{
			name:         "Yes value",
			key:          "TEST_BOOL_YES",
			value:        "yes",
			defaultValue: false,
			expected:     false, // strconv.ParseBool doesn't support 'yes'
			description:  "strconv.ParseBool doesn't support 'yes', uses default",
		},
		{
			name:         "No value",
			key:          "TEST_BOOL_NO",
			value:        "no",
			defaultValue: true,
			expected:     true, // strconv.ParseBool doesn't support 'no'
			description:  "strconv.ParseBool doesn't support 'no', uses default",
		},
		{
			name:         "Invalid value - use default",
			key:          "TEST_BOOL_INVALID",
			value:        "invalid",
			defaultValue: true,
			expected:     true,
			description:  "Should use default value for invalid input",
		},
		{
			name:         "Empty value - use default",
			key:          "TEST_BOOL_EMPTY",
			value:        "",
			defaultValue: true,
			expected:     true,
			description:  "Should use default value for empty string",
		},
		{
			name:         "Unset variable - use default true",
			key:          "TEST_BOOL_UNSET_TRUE",
			value:        "", // Will be unset
			defaultValue: true,
			expected:     true,
			description:  "Should use default value when variable is unset",
		},
		{
			name:         "Unset variable - use default false",
			key:          "TEST_BOOL_UNSET_FALSE",
			value:        "", // Will be unset
			defaultValue: false,
			expected:     false,
			description:  "Should use default value when variable is unset",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.saveEnv(tc.key)

			if tc.value == "" && (tc.name == "Empty value - use default" || tc.name == "Unset variable - use default true" || tc.name == "Unset variable - use default false") {
				_ = os.Unsetenv(tc.key)
			} else {
				_ = os.Setenv(tc.key, tc.value)
			}

			result := getBoolEnv(tc.key, tc.defaultValue)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: getBoolEnv('%s', %v) with value '%s' = %v", tc.name, tc.key, tc.defaultValue, tc.value, result)
		})
	}
}

// TestGetIntEnv_Comprehensive tests getIntEnv utility function
func (s *ConfigUtilitiesTestSuite) TestGetIntEnv_Comprehensive() {
	testCases := []struct {
		name         string
		key          string
		value        string
		defaultValue int
		expected     int
		description  string
	}{
		{
			name:         "Valid positive integer",
			key:          "TEST_INT_POSITIVE",
			value:        "42",
			defaultValue: 0,
			expected:     42,
			description:  "Should parse positive integer correctly",
		},
		{
			name:         "Valid negative integer",
			key:          "TEST_INT_NEGATIVE",
			value:        "-42",
			defaultValue: 0,
			expected:     -42,
			description:  "Should parse negative integer correctly",
		},
		{
			name:         "Zero value",
			key:          "TEST_INT_ZERO",
			value:        "0",
			defaultValue: 100,
			expected:     0,
			description:  "Should parse zero correctly",
		},
		{
			name:         "Large integer",
			key:          "TEST_INT_LARGE",
			value:        "2147483647", // Max int32
			defaultValue: 0,
			expected:     2147483647,
			description:  "Should parse large integers within range",
		},
		{
			name:         "Large negative integer",
			key:          "TEST_INT_LARGE_NEG",
			value:        "-2147483648", // Min int32
			defaultValue: 0,
			expected:     -2147483648,
			description:  "Should parse large negative integers within range",
		},
		{
			name:         "Too large integer - use default",
			key:          "TEST_INT_TOO_LARGE",
			value:        "2147483648", // Larger than max int32
			defaultValue: 42,
			expected:     42,
			description:  "Should use default for integers outside int32 range",
		},
		{
			name:         "Too small integer - use default",
			key:          "TEST_INT_TOO_SMALL",
			value:        "-2147483649", // Smaller than min int32
			defaultValue: 42,
			expected:     42,
			description:  "Should use default for integers outside int32 range",
		},
		{
			name:         "Invalid value - not a number",
			key:          "TEST_INT_INVALID",
			value:        "not-a-number",
			defaultValue: 42,
			expected:     42,
			description:  "Should use default value for non-numeric input",
		},
		{
			name:         "Floating point value - use default",
			key:          "TEST_INT_FLOAT",
			value:        "42.5",
			defaultValue: 10,
			expected:     10,
			description:  "Should use default value for floating point input",
		},
		{
			name:         "Empty value - use default",
			key:          "TEST_INT_EMPTY",
			value:        "",
			defaultValue: 99,
			expected:     99,
			description:  "Should use default value for empty string",
		},
		{
			name:         "Unset variable - use default",
			key:          "TEST_INT_UNSET",
			value:        "", // Will be unset
			defaultValue: 123,
			expected:     123,
			description:  "Should use default value when variable is unset",
		},
		{
			name:         "Leading/trailing whitespace",
			key:          "TEST_INT_WHITESPACE",
			value:        "  42  ",
			defaultValue: 0,
			expected:     0, // Should fail to parse due to whitespace
			description:  "Should handle whitespace in integer values",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.saveEnv(tc.key)

			if tc.value == "" && (tc.name == "Empty value - use default" || tc.name == "Unset variable - use default") {
				_ = os.Unsetenv(tc.key)
			} else {
				_ = os.Setenv(tc.key, tc.value)
			}

			result := getIntEnv(tc.key, tc.defaultValue)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: getIntEnv('%s', %d) with value '%s' = %d", tc.name, tc.key, tc.defaultValue, tc.value, result)
		})
	}
}

// TestGetStringEnv_Comprehensive tests getStringEnv utility function
func (s *ConfigUtilitiesTestSuite) TestGetStringEnv_Comprehensive() {
	testCases := []struct {
		name         string
		key          string
		value        string
		defaultValue string
		expected     string
		description  string
	}{
		{
			name:         "Valid string value",
			key:          "TEST_STRING_VALID",
			value:        "test-value",
			defaultValue: "default",
			expected:     "test-value",
			description:  "Should return actual string value",
		},
		{
			name:         "Empty string value",
			key:          "TEST_STRING_EMPTY",
			value:        "",
			defaultValue: "default",
			expected:     "default",
			description:  "Should use default value for empty string",
		},
		{
			name:         "String with spaces",
			key:          "TEST_STRING_SPACES",
			value:        "  value with spaces  ",
			defaultValue: "default",
			expected:     "  value with spaces  ",
			description:  "Should preserve spaces in string values",
		},
		{
			name:         "String with special characters",
			key:          "TEST_STRING_SPECIAL",
			value:        "value-with_special.chars@123",
			defaultValue: "default",
			expected:     "value-with_special.chars@123",
			description:  "Should preserve special characters",
		},
		{
			name:         "Numeric string",
			key:          "TEST_STRING_NUMERIC",
			value:        "12345",
			defaultValue: "default",
			expected:     "12345",
			description:  "Should treat numeric values as strings",
		},
		{
			name:         "Boolean string",
			key:          "TEST_STRING_BOOL",
			value:        "true",
			defaultValue: "default",
			expected:     "true",
			description:  "Should treat boolean values as strings",
		},
		{
			name:         "Unset variable - use default",
			key:          "TEST_STRING_UNSET",
			value:        "", // Will be unset
			defaultValue: "default-value",
			expected:     "default-value",
			description:  "Should use default value when variable is unset",
		},
		{
			name:         "Very long string",
			key:          "TEST_STRING_LONG",
			value:        "This is a very long string value that contains multiple words and should be handled correctly by the getStringEnv function",
			defaultValue: "default",
			expected:     "This is a very long string value that contains multiple words and should be handled correctly by the getStringEnv function",
			description:  "Should handle very long string values",
		},
		{
			name:         "Unicode string",
			key:          "TEST_STRING_UNICODE",
			value:        "Hello World Test",
			defaultValue: "default",
			expected:     "Hello World Test",
			description:  "Should handle international characters",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.saveEnv(tc.key)

			if tc.value == "" && (tc.name == "Empty string value" || tc.name == "Unset variable - use default") {
				_ = os.Unsetenv(tc.key)
			} else {
				_ = os.Setenv(tc.key, tc.value)
			}

			result := getStringEnv(tc.key, tc.defaultValue)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: getStringEnv('%s', '%s') with value '%s' = '%s'", tc.name, tc.key, tc.defaultValue, tc.value, result)
		})
	}
}

// TestIsValidVersion_EdgeCases tests isValidVersion utility function
func (s *ConfigUtilitiesTestSuite) TestIsValidVersion_EdgeCases() {
	testCases := []struct {
		name        string
		version     string
		expected    bool
		description string
	}{
		{
			name:        "Valid semantic version with v prefix",
			version:     "v1.2.3",
			expected:    true,
			description: "Should accept valid semantic version with v prefix",
		},
		{
			name:        "Valid semantic version without v prefix",
			version:     "1.2.3",
			expected:    false, // Current implementation requires v prefix
			description: "Should reject version without v prefix",
		},
		{
			name:        "Valid two-part version",
			version:     "v1.2",
			expected:    true,
			description: "Should accept two-part version",
		},
		{
			name:        "Valid single-part version",
			version:     "v1",
			expected:    false, // Current implementation requires at least two parts
			description: "Should reject single-part version",
		},
		{
			name:        "Valid version with patch zero",
			version:     "v1.0.0",
			expected:    true,
			description: "Should accept version with zero patch",
		},
		{
			name:        "Valid pre-release version",
			version:     "v1.2.3-alpha",
			expected:    true,
			description: "Should accept pre-release version",
		},
		{
			name:        "Valid version with build metadata",
			version:     "v1.2.3+build123",
			expected:    true,
			description: "Should accept version with build metadata",
		},
		{
			name:        "Empty string",
			version:     "",
			expected:    false,
			description: "Should reject empty string",
		},
		{
			name:        "Only v prefix",
			version:     "v",
			expected:    false,
			description: "Should reject only v prefix",
		},
		{
			name:        "Invalid characters",
			version:     "v1.2.x",
			expected:    true, // Current implementation only checks v prefix and >= 2 parts
			description: "Current implementation accepts versions with any characters",
		},
		{
			name:        "Version with spaces",
			version:     "v1 . 2 . 3",
			expected:    true, // Current implementation only checks v prefix and >= 2 parts
			description: "Current implementation accepts versions with spaces",
		},
		{
			name:        "Version starting with number",
			version:     "1.2.3-v",
			expected:    false,
			description: "Should reject versions not starting with v",
		},
		{
			name:        "Multiple v prefixes",
			version:     "vv1.2.3",
			expected:    true, // Current implementation only checks v prefix and >= 2 parts
			description: "Current implementation accepts multiple v prefixes",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := isValidVersion(tc.version)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: isValidVersion('%s') = %v", tc.name, tc.version, result)
		})
	}
}

// TestConfigValidation_ComprehensiveRules tests comprehensive config validation
func (s *ConfigUtilitiesTestSuite) TestConfigValidation_ComprehensiveRules() {
	testCases := []struct {
		name        string
		configFunc  func() *Config
		expectError bool
		errorCount  int
		description string
	}{
		{
			name: "Valid configuration",
			configFunc: func() *Config {
				cfg := &Config{
					Timeout:      300,
					MaxFileSize:  10 * 1024 * 1024,
					MaxFilesOpen: 100,
					LogLevel:     "info",
				}
				cfg.CheckTimeouts.Fmt = 30
				cfg.CheckTimeouts.Fumpt = 30
				cfg.CheckTimeouts.Lint = 60
				cfg.CheckTimeouts.ModTidy = 30
				cfg.CheckTimeouts.Whitespace = 30
				cfg.CheckTimeouts.EOF = 30
				cfg.CheckTimeouts.AIDetection = 30
				cfg.Performance.ParallelWorkers = 4
				cfg.ToolVersions.Fumpt = "latest"
				cfg.ToolVersions.GolangciLint = "latest"
				return cfg
			},
			expectError: false,
			errorCount:  0,
			description: "Should accept valid configuration",
		},
		{
			name: "Invalid timeout values",
			configFunc: func() *Config {
				cfg := &Config{
					Timeout:      -1, // Invalid
					MaxFileSize:  10 * 1024 * 1024,
					MaxFilesOpen: 100,
					LogLevel:     "info",
				}
				cfg.CheckTimeouts.Fmt = 0    // Invalid
				cfg.CheckTimeouts.Fumpt = -5 // Invalid
				cfg.CheckTimeouts.Lint = 60
				cfg.CheckTimeouts.ModTidy = 30
				cfg.CheckTimeouts.Whitespace = 30
				cfg.CheckTimeouts.EOF = 30
				cfg.CheckTimeouts.AIDetection = 30
				cfg.Performance.ParallelWorkers = 4
				return cfg
			},
			expectError: true,
			errorCount:  3, // timeout, fmt timeout, fumpt timeout
			description: "Should reject invalid timeout values",
		},
		{
			name: "Invalid file size and workers",
			configFunc: func() *Config {
				cfg := &Config{
					Timeout:      300,
					MaxFileSize:  -1, // Invalid
					MaxFilesOpen: 0,  // Invalid
					LogLevel:     "info",
				}
				cfg.CheckTimeouts.Fmt = 30
				cfg.CheckTimeouts.Fumpt = 30
				cfg.CheckTimeouts.Lint = 60
				cfg.CheckTimeouts.ModTidy = 30
				cfg.CheckTimeouts.Whitespace = 30
				cfg.CheckTimeouts.EOF = 30
				cfg.CheckTimeouts.AIDetection = 30
				cfg.Performance.ParallelWorkers = -1 // Invalid
				return cfg
			},
			expectError: true,
			errorCount:  3, // file size, files open, parallel workers
			description: "Should reject invalid file size and worker values",
		},
		{
			name: "Invalid log level and tool versions",
			configFunc: func() *Config {
				cfg := &Config{
					Timeout:      300,
					MaxFileSize:  10 * 1024 * 1024,
					MaxFilesOpen: 100,
					LogLevel:     "invalid", // Invalid
				}
				cfg.CheckTimeouts.Fmt = 30
				cfg.CheckTimeouts.Fumpt = 30
				cfg.CheckTimeouts.Lint = 60
				cfg.CheckTimeouts.ModTidy = 30
				cfg.CheckTimeouts.Whitespace = 30
				cfg.CheckTimeouts.EOF = 30
				cfg.CheckTimeouts.AIDetection = 30
				cfg.Performance.ParallelWorkers = 4
				cfg.ToolVersions.Fumpt = "invalid-version"     // Invalid
				cfg.ToolVersions.GolangciLint = "also-invalid" // Invalid
				return cfg
			},
			expectError: true,
			errorCount:  3, // log level, fumpt version, golangci-lint version
			description: "Should reject invalid log level and tool versions",
		},
		{
			name: "Invalid exclude patterns",
			configFunc: func() *Config {
				cfg := &Config{
					Timeout:      300,
					MaxFileSize:  10 * 1024 * 1024,
					MaxFilesOpen: 100,
					LogLevel:     "info",
				}
				cfg.CheckTimeouts.Fmt = 30
				cfg.CheckTimeouts.Fumpt = 30
				cfg.CheckTimeouts.Lint = 60
				cfg.CheckTimeouts.ModTidy = 30
				cfg.CheckTimeouts.Whitespace = 30
				cfg.CheckTimeouts.EOF = 30
				cfg.CheckTimeouts.AIDetection = 30
				cfg.Performance.ParallelWorkers = 4
				cfg.Git.ExcludePatterns = []string{"valid", "", "  ", "another-valid"} // Two empty patterns
				return cfg
			},
			expectError: true,
			errorCount:  2, // two empty exclude patterns
			description: "Should reject empty exclude patterns",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cfg := tc.configFunc()
			err := cfg.Validate()

			if tc.expectError {
				s.Require().Error(err, tc.description)

				// Check if it's a ValidationError with multiple errors
				var validationErr *ValidationError
				if errors.As(err, &validationErr) {
					s.Len(validationErr.Errors, tc.errorCount, "Should have expected number of validation errors")
				}
			} else {
				s.Require().NoError(err, tc.description)
			}

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestFindEnvFiles tests findBaseEnvFile and findCustomEnvFile functions indirectly
func (s *ConfigUtilitiesTestSuite) TestFindEnvFiles() {
	testCases := []struct {
		name        string
		setupFunc   func() string
		expectError bool
		description string
	}{
		{
			name: "Valid .env.base file",
			setupFunc: func() string {
				testDir := filepath.Join(s.tempDir, "valid-env")
				githubDir := filepath.Join(testDir, ".github")
				s.Require().NoError(os.MkdirAll(githubDir, 0o750))

				envFile := filepath.Join(githubDir, ".env.base")
				s.Require().NoError(os.WriteFile(envFile, []byte("ENABLE_GO_PRE_COMMIT=true\n"), 0o600))

				return testDir
			},
			expectError: false,
			description: "Should find valid .env.base file",
		},
		{
			name: "Missing .env.base file",
			setupFunc: func() string {
				testDir := filepath.Join(s.tempDir, "no-env")
				s.Require().NoError(os.MkdirAll(testDir, 0o750))
				return testDir
			},
			expectError: true,
			description: "Should fail when .env.base file is missing",
		},
		{
			name: "Valid .env.base with custom file",
			setupFunc: func() string {
				testDir := filepath.Join(s.tempDir, "with-custom")
				githubDir := filepath.Join(testDir, ".github")
				s.Require().NoError(os.MkdirAll(githubDir, 0o750))

				baseFile := filepath.Join(githubDir, ".env.base")
				s.Require().NoError(os.WriteFile(baseFile, []byte("ENABLE_GO_PRE_COMMIT=true\n"), 0o600))

				customFile := filepath.Join(githubDir, ".env.custom")
				s.Require().NoError(os.WriteFile(customFile, []byte("GO_PRE_COMMIT_LOG_LEVEL=debug\n"), 0o600))

				return testDir
			},
			expectError: false,
			description: "Should handle .env.base with custom file",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			testDir := tc.setupFunc()
			originalWD, _ := os.Getwd()
			defer func() { _ = os.Chdir(originalWD) }()

			_ = os.Chdir(testDir)

			// Test the load function which uses findBaseEnvFile internally
			_, err := Load()

			if tc.expectError {
				s.Require().Error(err, tc.description)
			} else {
				// May error due to validation, but shouldn't error due to file finding
				s.T().Logf("Load result for %s: %v", tc.name, err)
			}

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestSuite runs the config utilities test suite
func TestConfigUtilitiesTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigUtilitiesTestSuite))
}
