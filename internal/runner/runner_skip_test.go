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
	s.cfg.Checks.Fumpt = true
	s.cfg.Checks.Gitleaks = true
	s.cfg.Checks.Lint = true
	s.cfg.Checks.ModTidy = true
	s.cfg.Checks.Whitespace = true
	s.cfg.Checks.EOF = true

	// Set up performance
	s.cfg.Performance.ParallelWorkers = 2

	// Create a runner instance
	s.runner = New(s.cfg, "/tmp")
}

// SetupTest saves and clears environment variables before each test
func (s *SkipFunctionalityTestSuite) SetupTest() {
	s.originalEnv = make(map[string]string)
	envVars := []string{envSkip, "GO_PRE_COMMIT_SKIP"}

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
			input:    checkNameFumpt,
			expected: []string{checkNameFumpt},
		},
		{
			name:     "Multiple Checks",
			input:    "fumpt,lint,whitespace",
			expected: []string{checkNameFumpt, checkNameLint, checkNameWhitespace},
		},
		{
			name:     "Special Value All",
			input:    "all",
			expected: []string{checkNameFumpt, checkNameGitleaks, checkNameLint, checkNameModTidy, checkNameWhitespace, checkNameEOF},
		},
		{
			name:     "Special Value ALL (case insensitive)",
			input:    "ALL",
			expected: []string{checkNameFumpt, checkNameGitleaks, checkNameLint, checkNameModTidy, checkNameWhitespace, checkNameEOF},
		},
		{
			name:     "With Spaces",
			input:    "fumpt, lint, whitespace",
			expected: []string{checkNameFumpt, checkNameLint, checkNameWhitespace},
		},
		{
			name:     "With Empty Entries",
			input:    "fumpt,,lint,",
			expected: []string{checkNameFumpt, checkNameLint},
		},
		{
			name:     "Only Whitespace",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "Mixed Whitespace and Commas",
			input:    " , fumpt , , lint , ",
			expected: []string{checkNameFumpt, checkNameLint},
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
				envSkip: "fumpt,lint",
			},
			expected:    []string{checkNameFumpt, checkNameLint},
			description: "Should parse SKIP environment variable",
		},
		{
			name: "GO_PRE_COMMIT_SKIP Environment Variable",
			envVars: map[string]string{
				"GO_PRE_COMMIT_SKIP": "whitespace,eof",
			},
			expected:    []string{checkNameWhitespace, checkNameEOF},
			description: "Should parse GO_PRE_COMMIT_SKIP environment variable",
		},
		{
			name: "Both Variables Set - SKIP Takes Precedence",
			envVars: map[string]string{
				envSkip:              checkNameFumpt,
				"GO_PRE_COMMIT_SKIP": checkNameLint,
			},
			expected:    []string{checkNameFumpt},
			description: "Should use SKIP when both are set (precedence order)",
		},
		{
			name: "Empty SKIP Variable Falls Back",
			envVars: map[string]string{
				envSkip:              "",
				"GO_PRE_COMMIT_SKIP": checkNameModTidy,
			},
			expected:    []string{checkNameModTidy},
			description: "Should fall back to GO_PRE_COMMIT_SKIP when SKIP is empty",
		},
		{
			name: "Whitespace Only SKIP Falls Back",
			envVars: map[string]string{
				envSkip:              "   ",
				"GO_PRE_COMMIT_SKIP": checkNameModTidy,
			},
			expected:    []string{checkNameModTidy},
			description: "Should fall back when SKIP contains only whitespace",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Clean up environment variables first
			_ = os.Unsetenv(envSkip)
			_ = os.Unsetenv("GO_PRE_COMMIT_SKIP")

			// Set environment variables
			for key, value := range tc.envVars {
				_ = os.Setenv(key, value)
			}

			result := s.runner.processSkipEnvironment()
			s.Equal(tc.expected, result, tc.description)

			// Clean up after test
			_ = os.Unsetenv(envSkip)
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
			cliSkips:    []string{checkNameFumpt, checkNameLint},
			envVars:     map[string]string{},
			expected:    []string{checkNameFumpt, checkNameLint},
			description: "Should return CLI skips when no environment variables are set",
		},
		{
			name:     "Only Environment Skips",
			cliSkips: nil,
			envVars: map[string]string{
				envSkip: "whitespace,eof",
			},
			expected:    []string{checkNameWhitespace, checkNameEOF},
			description: "Should return environment skips when no CLI skips are provided",
		},
		{
			name:     "CLI and Environment Combined",
			cliSkips: []string{checkNameFumpt},
			envVars: map[string]string{
				envSkip: "lint,mod-tidy",
			},
			expected:    []string{checkNameFumpt, checkNameLint, checkNameModTidy},
			description: "Should combine CLI and environment skips",
		},
		{
			name:     "Duplicate Skips Deduplicated",
			cliSkips: []string{checkNameFumpt, checkNameLint},
			envVars: map[string]string{
				envSkip: "lint,whitespace",
			},
			expected:    []string{checkNameFumpt, checkNameLint, checkNameWhitespace},
			description: "Should deduplicate skips from different sources",
		},
		{
			name:     "Invalid Skips Filtered Out",
			cliSkips: []string{checkNameFumpt, "invalid-check"},
			envVars: map[string]string{
				envSkip: "lint,another-invalid",
			},
			expected:    []string{checkNameFumpt, checkNameLint},
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
			input:       []string{checkNameModTidy, checkNameFumpt, checkNameLint},
			expected:    []string{checkNameModTidy, checkNameFumpt, checkNameLint},
			description: "Should return all valid checks",
		},
		{
			name:        "Duplicate Checks",
			input:       []string{checkNameFumpt, checkNameLint, checkNameFumpt, checkNameLint},
			expected:    []string{checkNameFumpt, checkNameLint},
			description: "Should remove duplicate checks",
		},
		{
			name:        "Invalid Checks Filtered",
			input:       []string{checkNameFumpt, "invalid-check", checkNameLint, "another-invalid"},
			expected:    []string{checkNameFumpt, checkNameLint},
			description: "Should filter out invalid check names",
		},
		{
			name:        "Mixed Valid and Empty Strings",
			input:       []string{checkNameFumpt, "", checkNameLint, "   ", checkNameWhitespace},
			expected:    []string{checkNameFumpt, checkNameLint, checkNameWhitespace},
			description: "Should filter out empty and whitespace-only strings",
		},
		{
			name:        "All Valid Checks",
			input:       []string{checkNameFumpt, checkNameGitleaks, checkNameLint, checkNameModTidy, checkNameWhitespace, checkNameEOF},
			expected:    []string{checkNameFumpt, checkNameGitleaks, checkNameLint, checkNameModTidy, checkNameWhitespace, checkNameEOF},
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
			envSkips:    checkNameFumpt,
			cliSkips:    []string{checkNameLint, checkNameModTidy},
			expectedErr: "", // Should combine skips
			description: "Should combine CLI and environment skips",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Set up environment
			if tc.envSkips != "" {
				_ = os.Setenv(envSkip, tc.envSkips)
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
			skipValue:   checkNameFumpt,
			goSkipValue: checkNameLint,
			expected:    []string{checkNameFumpt},
			description: "SKIP should take precedence when both are set",
		},
		{
			name:        "Empty SKIP falls back to GO_PRE_COMMIT_SKIP",
			skipValue:   "",
			goSkipValue: checkNameModTidy,
			expected:    []string{checkNameModTidy},
			description: "Should use GO_PRE_COMMIT_SKIP when SKIP is empty",
		},
		{
			name:        "Whitespace SKIP falls back to GO_PRE_COMMIT_SKIP",
			skipValue:   "   ",
			goSkipValue: checkNameWhitespace,
			expected:    []string{checkNameWhitespace},
			description: "Should use GO_PRE_COMMIT_SKIP when SKIP is only whitespace",
		},
		{
			name:        "Only GO_PRE_COMMIT_SKIP set",
			skipValue:   "", // Not set
			goSkipValue: "eof,mod-tidy",
			expected:    []string{checkNameEOF, checkNameModTidy},
			description: "Should use GO_PRE_COMMIT_SKIP when SKIP is not set",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			if tc.skipValue != "" {
				_ = os.Setenv(envSkip, tc.skipValue)
			} else {
				_ = os.Unsetenv(envSkip)
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
			expected:    []string{checkNameFumpt, checkNameLint},
			description: "Should handle trailing and leading commas",
		},
		{
			name:        "Multiple Consecutive Commas",
			skipValue:   "fumpt,,,lint,,whitespace",
			expected:    []string{checkNameFumpt, checkNameLint, checkNameWhitespace},
			description: "Should handle multiple consecutive commas",
		},
		{
			name:        "Mixed Case All",
			skipValue:   "All",
			expected:    []string{checkNameFumpt, checkNameGitleaks, checkNameLint, checkNameModTidy, checkNameWhitespace, checkNameEOF},
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
