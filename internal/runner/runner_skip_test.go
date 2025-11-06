package runner

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-pre-commit/internal/config"
)

// SkipFunctionalityTestSuite tests the skip functionality of the Runner
type SkipFunctionalityTestSuite struct {
	suite.Suite

	runner      *Runner
	cfg         *config.Config
	originalEnv map[string]string
}

// SetupSuite initializes the test environment
func (s *SkipFunctionalityTestSuite) SetupSuite() {
	// Create a test configuration
	s.cfg = &config.Config{
		Enabled: true,
		Timeout: 120,
	}

	// Set up checks
	s.cfg.Checks.Fmt = true
	s.cfg.Checks.Fumpt = true
	s.cfg.Checks.Goimports = true
	s.cfg.Checks.Lint = true
	s.cfg.Checks.ModTidy = true
	s.cfg.Checks.Whitespace = true
	s.cfg.Checks.EOF = true
	s.cfg.Checks.AIDetection = false

	// Set up performance
	s.cfg.Performance.ParallelWorkers = 2

	// Create a runner instance
	s.runner = New(s.cfg, "/tmp")
}

// SetupTest saves and clears environment variables before each test
func (s *SkipFunctionalityTestSuite) SetupTest() {
	s.originalEnv = make(map[string]string)
	envVars := []string{"SKIP", "GO_PRE_COMMIT_SKIP"}

	for _, key := range envVars {
		s.originalEnv[key] = os.Getenv(key)
		_ = os.Unsetenv(key)
	}
}

// TearDownTest restores environment variables after each test
func (s *SkipFunctionalityTestSuite) TearDownTest() {
	for key, value := range s.originalEnv {
		if value != "" {
			_ = os.Setenv(key, value)
		} else {
			_ = os.Unsetenv(key)
		}
	}
}

// TestParseSkipValue tests the parseSkipValue method directly
func (s *SkipFunctionalityTestSuite) TestParseSkipValue() {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty String",
			input:    "",
			expected: nil,
		},
		{
			name:     "Single Check",
			input:    "fumpt",
			expected: []string{"fumpt"},
		},
		{
			name:     "Multiple Checks",
			input:    "fumpt,lint,whitespace",
			expected: []string{"fumpt", "lint", "whitespace"},
		},
		{
			name:     "Special Value All",
			input:    "all",
			expected: []string{"fmt", "fumpt", "gitleaks", "goimports", "lint", "mod-tidy", "whitespace", "eof", "ai_detection"},
		},
		{
			name:     "Special Value ALL (case insensitive)",
			input:    "ALL",
			expected: []string{"fmt", "fumpt", "gitleaks", "goimports", "lint", "mod-tidy", "whitespace", "eof", "ai_detection"},
		},
		{
			name:     "With Spaces",
			input:    "fumpt, lint, whitespace",
			expected: []string{"fumpt", "lint", "whitespace"},
		},
		{
			name:     "With Empty Entries",
			input:    "fumpt,,lint,",
			expected: []string{"fumpt", "lint"},
		},
		{
			name:     "Only Whitespace",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "Mixed Whitespace and Commas",
			input:    " , fumpt , , lint , ",
			expected: []string{"fumpt", "lint"},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := s.runner.parseSkipValue(tc.input)
			s.Equal(tc.expected, result, "parseSkipValue should return expected result for input: %s", tc.input)
		})
	}
}

// TestProcessSkipEnvironment tests environment variable processing
func (s *SkipFunctionalityTestSuite) TestProcessSkipEnvironment() {
	testCases := []struct {
		name        string
		envVars     map[string]string
		expected    []string
		description string
	}{
		{
			name:        "No Environment Variables",
			envVars:     map[string]string{},
			expected:    nil,
			description: "Should return nil when no skip environment variables are set",
		},
		{
			name: "SKIP Environment Variable",
			envVars: map[string]string{
				"SKIP": "fumpt,lint",
			},
			expected:    []string{"fumpt", "lint"},
			description: "Should parse SKIP environment variable",
		},
		{
			name: "GO_PRE_COMMIT_SKIP Environment Variable",
			envVars: map[string]string{
				"GO_PRE_COMMIT_SKIP": "whitespace,eof",
			},
			expected:    []string{"whitespace", "eof"},
			description: "Should parse GO_PRE_COMMIT_SKIP environment variable",
		},
		{
			name: "Both Variables Set - SKIP Takes Precedence",
			envVars: map[string]string{
				"SKIP":               "fumpt",
				"GO_PRE_COMMIT_SKIP": "lint",
			},
			expected:    []string{"fumpt"},
			description: "Should use SKIP when both are set (precedence order)",
		},
		{
			name: "Empty SKIP Variable Falls Back",
			envVars: map[string]string{
				"SKIP":               "",
				"GO_PRE_COMMIT_SKIP": "mod-tidy",
			},
			expected:    []string{"mod-tidy"},
			description: "Should fall back to GO_PRE_COMMIT_SKIP when SKIP is empty",
		},
		{
			name: "Whitespace Only SKIP Falls Back",
			envVars: map[string]string{
				"SKIP":               "   ",
				"GO_PRE_COMMIT_SKIP": "fmt",
			},
			expected:    []string{"fmt"},
			description: "Should fall back when SKIP contains only whitespace",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Clean up environment variables first
			_ = os.Unsetenv("SKIP")
			_ = os.Unsetenv("GO_PRE_COMMIT_SKIP")

			// Set environment variables
			for key, value := range tc.envVars {
				_ = os.Setenv(key, value)
			}

			result := s.runner.processSkipEnvironment()
			s.Equal(tc.expected, result, tc.description)

			// Clean up after test
			_ = os.Unsetenv("SKIP")
			_ = os.Unsetenv("GO_PRE_COMMIT_SKIP")
		})
	}
}

// TestCombineSkipSources tests the combination of CLI and environment skips
func (s *SkipFunctionalityTestSuite) TestCombineSkipSources() {
	testCases := []struct {
		name        string
		cliSkips    []string
		envVars     map[string]string
		expected    []string
		description string
	}{
		{
			name:        "No Skips",
			cliSkips:    nil,
			envVars:     map[string]string{},
			expected:    []string{},
			description: "Should return empty slice when no skips are specified",
		},
		{
			name:        "Only CLI Skips",
			cliSkips:    []string{"fumpt", "lint"},
			envVars:     map[string]string{},
			expected:    []string{"fumpt", "lint"},
			description: "Should return CLI skips when no environment variables are set",
		},
		{
			name:     "Only Environment Skips",
			cliSkips: nil,
			envVars: map[string]string{
				"SKIP": "whitespace,eof",
			},
			expected:    []string{"whitespace", "eof"},
			description: "Should return environment skips when no CLI skips are provided",
		},
		{
			name:     "CLI and Environment Combined",
			cliSkips: []string{"fumpt"},
			envVars: map[string]string{
				"SKIP": "lint,mod-tidy",
			},
			expected:    []string{"fumpt", "lint", "mod-tidy"},
			description: "Should combine CLI and environment skips",
		},
		{
			name:     "Duplicate Skips Deduplicated",
			cliSkips: []string{"fumpt", "lint"},
			envVars: map[string]string{
				"SKIP": "lint,whitespace",
			},
			expected:    []string{"fumpt", "lint", "whitespace"},
			description: "Should deduplicate skips from different sources",
		},
		{
			name:     "Invalid Skips Filtered Out",
			cliSkips: []string{"fumpt", "invalid-check"},
			envVars: map[string]string{
				"SKIP": "lint,another-invalid",
			},
			expected:    []string{"fumpt", "lint"},
			description: "Should filter out invalid check names",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Set environment variables
			for key, value := range tc.envVars {
				_ = os.Setenv(key, value)
			}

			result := s.runner.combineSkipSources(tc.cliSkips)
			s.ElementsMatch(tc.expected, result, tc.description)
		})
	}
}

// TestDeduplicateAndValidateSkips tests skip deduplication and validation
func (s *SkipFunctionalityTestSuite) TestDeduplicateAndValidateSkips() {
	testCases := []struct {
		name        string
		input       []string
		expected    []string
		description string
	}{
		{
			name:        "Empty Input",
			input:       []string{},
			expected:    []string{},
			description: "Should return empty slice for empty input",
		},
		{
			name:        "Valid Checks",
			input:       []string{"fmt", "fumpt", "lint"},
			expected:    []string{"fmt", "fumpt", "lint"},
			description: "Should return all valid checks",
		},
		{
			name:        "Duplicate Checks",
			input:       []string{"fumpt", "lint", "fumpt", "lint"},
			expected:    []string{"fumpt", "lint"},
			description: "Should remove duplicate checks",
		},
		{
			name:        "Invalid Checks Filtered",
			input:       []string{"fumpt", "invalid-check", "lint", "another-invalid"},
			expected:    []string{"fumpt", "lint"},
			description: "Should filter out invalid check names",
		},
		{
			name:        "Mixed Valid and Empty Strings",
			input:       []string{"fumpt", "", "lint", "   ", "whitespace"},
			expected:    []string{"fumpt", "lint", "whitespace"},
			description: "Should filter out empty and whitespace-only strings",
		},
		{
			name:        "All Valid Checks",
			input:       []string{"fmt", "fumpt", "gitleaks", "goimports", "lint", "mod-tidy", "whitespace", "eof", "ai_detection"},
			expected:    []string{"fmt", "fumpt", "gitleaks", "goimports", "lint", "mod-tidy", "whitespace", "eof", "ai_detection"},
			description: "Should accept all valid check names",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := s.runner.deduplicateAndValidateSkips(tc.input)
			s.ElementsMatch(tc.expected, result, tc.description)
		})
	}
}

// TestSkipIntegrationWithRunner tests skip functionality integrated with runner execution
func (s *SkipFunctionalityTestSuite) TestSkipIntegrationWithRunner() {
	testCases := []struct {
		name        string
		envSkips    string
		cliSkips    []string
		expectedErr string
		description string
	}{
		{
			name:        "Skip All Checks",
			envSkips:    "all",
			cliSkips:    nil,
			expectedErr: "no checks to run", // Should fail with this error
			description: "Should handle skipping all checks",
		},
		{
			name:        "Skip Some Checks",
			envSkips:    "fumpt,lint",
			cliSkips:    nil,
			expectedErr: "", // May succeed depending on which checks are enabled
			description: "Should allow skipping some checks while others run",
		},
		{
			name:        "CLI Override Environment",
			envSkips:    "fumpt",
			cliSkips:    []string{"lint", "mod-tidy"},
			expectedErr: "", // Should combine skips
			description: "Should combine CLI and environment skips",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Set up environment
			if tc.envSkips != "" {
				_ = os.Setenv("SKIP", tc.envSkips)
			}

			// Create runner options
			opts := Options{
				Files:      []string{}, // Empty files should complete quickly
				SkipChecks: tc.cliSkips,
				Parallel:   1,
			}

			// Run with short timeout
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			_, err := s.runner.Run(ctx, opts)

			if tc.expectedErr != "" {
				s.Require().Error(err, tc.description)
				s.Contains(err.Error(), tc.expectedErr)
			} else {
				// May succeed or fail depending on available files/tools
				// We mainly test that it doesn't panic and handles skips
				s.T().Logf("Result for %s: %v", tc.name, err)
			}
		})
	}
}

// TestSkipEnvironmentVariablePrecedence tests the precedence order of skip environment variables
func (s *SkipFunctionalityTestSuite) TestSkipEnvironmentVariablePrecedence() {
	testCases := []struct {
		name        string
		skipValue   string
		goSkipValue string
		expected    []string
		description string
	}{
		{
			name:        "SKIP takes precedence over GO_PRE_COMMIT_SKIP",
			skipValue:   "fumpt",
			goSkipValue: "lint",
			expected:    []string{"fumpt"},
			description: "SKIP should take precedence when both are set",
		},
		{
			name:        "Empty SKIP falls back to GO_PRE_COMMIT_SKIP",
			skipValue:   "",
			goSkipValue: "mod-tidy",
			expected:    []string{"mod-tidy"},
			description: "Should use GO_PRE_COMMIT_SKIP when SKIP is empty",
		},
		{
			name:        "Whitespace SKIP falls back to GO_PRE_COMMIT_SKIP",
			skipValue:   "   ",
			goSkipValue: "whitespace",
			expected:    []string{"whitespace"},
			description: "Should use GO_PRE_COMMIT_SKIP when SKIP is only whitespace",
		},
		{
			name:        "Only GO_PRE_COMMIT_SKIP set",
			skipValue:   "", // Not set
			goSkipValue: "eof,fmt",
			expected:    []string{"eof", "fmt"},
			description: "Should use GO_PRE_COMMIT_SKIP when SKIP is not set",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.skipValue != "" {
				_ = os.Setenv("SKIP", tc.skipValue)
			} else {
				_ = os.Unsetenv("SKIP")
			}

			if tc.goSkipValue != "" {
				_ = os.Setenv("GO_PRE_COMMIT_SKIP", tc.goSkipValue)
			} else {
				_ = os.Unsetenv("GO_PRE_COMMIT_SKIP")
			}

			result := s.runner.processSkipEnvironment()
			s.Equal(tc.expected, result, tc.description)
		})
	}
}

// TestSkipEdgeCases tests edge cases in skip functionality
func (s *SkipFunctionalityTestSuite) TestSkipEdgeCases() {
	testCases := []struct {
		name        string
		skipValue   string
		expected    []string
		description string
	}{
		{
			name:        "Only Commas",
			skipValue:   ",,,",
			expected:    nil,
			description: "Should handle string with only commas",
		},
		{
			name:        "Trailing and Leading Commas",
			skipValue:   ",fumpt,lint,",
			expected:    []string{"fumpt", "lint"},
			description: "Should handle trailing and leading commas",
		},
		{
			name:        "Multiple Consecutive Commas",
			skipValue:   "fumpt,,,lint,,whitespace",
			expected:    []string{"fumpt", "lint", "whitespace"},
			description: "Should handle multiple consecutive commas",
		},
		{
			name:        "Mixed Case All",
			skipValue:   "All",
			expected:    []string{"fmt", "fumpt", "gitleaks", "goimports", "lint", "mod-tidy", "whitespace", "eof", "ai_detection"},
			description: "Should handle mixed case 'all' keyword",
		},
		{
			name:        "Special Characters in Check Names",
			skipValue:   "fumpt@version,lint#special,valid-check",
			expected:    []string{}, // All should be filtered as invalid
			description: "Should filter out check names with special characters",
		},
		{
			name:        "Very Long Check Name",
			skipValue:   "this-is-a-very-long-check-name-that-should-not-exist-in-any-reasonable-system",
			expected:    []string{}, // Should be filtered as invalid
			description: "Should filter out very long check names",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := s.runner.parseSkipValue(tc.skipValue)
			s.Equal(tc.expected, result, tc.description)
		})
	}
}

// TestSuite runs the skip functionality test suite
func TestSkipFunctionalityTestSuite(t *testing.T) {
	suite.Run(t, new(SkipFunctionalityTestSuite))
}
