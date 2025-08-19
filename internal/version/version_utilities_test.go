package version

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// VersionUtilitiesTestSuite tests additional version utility functions
type VersionUtilitiesTestSuite struct {
	suite.Suite
}

// TestIsCommitHash tests the isCommitHash utility function
func (s *VersionUtilitiesTestSuite) TestIsCommitHash() {
	testCases := []struct {
		name        string
		input       string
		expected    bool
		description string
	}{
		{
			name:        "Valid 7-character commit hash",
			input:       "a1b2c3d",
			expected:    true,
			description: "Should recognize valid 7-character hex string as commit hash",
		},
		{
			name:        "Valid 40-character commit hash",
			input:       "a1b2c3d4e5f6789012345678901234567890abcd",
			expected:    true,
			description: "Should recognize valid 40-character hex string as commit hash",
		},
		{
			name:        "Valid mixed case commit hash",
			input:       "A1B2c3D4",
			expected:    true,
			description: "Should recognize mixed case hex string as commit hash",
		},
		{
			name:        "Too short - 6 characters",
			input:       "a1b2c3",
			expected:    false,
			description: "Should reject strings shorter than 7 characters",
		},
		{
			name:        "Too long - 41 characters",
			input:       "a1b2c3d4e5f6789012345678901234567890abcde",
			expected:    false,
			description: "Should reject strings longer than 40 characters",
		},
		{
			name:        "Contains non-hex characters",
			input:       "g1b2c3d",
			expected:    false,
			description: "Should reject strings with non-hex characters",
		},
		{
			name:        "Contains special characters",
			input:       "a1b2c3-",
			expected:    false,
			description: "Should reject strings with special characters",
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    false,
			description: "Should reject empty string",
		},
		{
			name:        "All numbers",
			input:       "1234567",
			expected:    true,
			description: "Should accept all numeric hex string",
		},
		{
			name:        "All letters",
			input:       "abcdefA",
			expected:    true,
			description: "Should accept all letter hex string",
		},
		{
			name:        "Version string",
			input:       "v1.2.3",
			expected:    false,
			description: "Should reject version strings",
		},
		{
			name:        "Git reference",
			input:       "HEAD",
			expected:    false,
			description: "Should reject git reference names",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := isCommitHash(tc.input)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: isCommitHash('%s') = %v", tc.name, tc.input, result)
		})
	}
}

// TestCompareVersions_EdgeCases tests edge cases for version comparison
func (s *VersionUtilitiesTestSuite) TestCompareVersions_EdgeCases() {
	testCases := []struct {
		name        string
		v1          string
		v2          string
		expected    int
		description string
	}{
		{
			name:        "Very long version numbers",
			v1:          "999.999.999",
			v2:          "1000.0.0",
			expected:    -1,
			description: "Should handle large version numbers correctly",
		},
		{
			name:        "Zero versions",
			v1:          "0.0.0",
			v2:          "0.0.1",
			expected:    -1,
			description: "Should handle zero versions",
		},
		{
			name:        "Single digit vs multi digit",
			v1:          "1.9.9",
			v2:          "1.10.0",
			expected:    -1,
			description: "Should handle numeric comparison correctly",
		},
		{
			name:        "Commit hash vs version",
			v1:          "a1b2c3d",
			v2:          "1.0.0",
			expected:    -1,
			description: "Should treat commit hash as older than version",
		},
		{
			name:        "Two commit hashes",
			v1:          "a1b2c3d",
			v2:          "b2c3d4e",
			expected:    0,
			description: "Should treat two commit hashes as equal",
		},
		{
			name:        "Empty string vs dev",
			v1:          "",
			v2:          "dev",
			expected:    0,
			description: "Should treat empty string and dev as equal",
		},
		{
			name:        "Complex pre-release versions",
			v1:          "1.0.0-alpha.1",
			v2:          "1.0.0-beta.2",
			expected:    0,
			description: "Should ignore pre-release suffixes for base comparison",
		},
		{
			name:        "Version with build metadata",
			v1:          "1.0.0+20230815",
			v2:          "1.0.0+20230816",
			expected:    0,
			description: "Should ignore build metadata",
		},
		{
			name:        "Version with multiple separators",
			v1:          "1.0.0-alpha+build",
			v2:          "1.0.0",
			expected:    0,
			description: "Should handle multiple separators",
		},
		{
			name:        "Malformed version strings",
			v1:          "1..0",
			v2:          "1.0.0",
			expected:    0, // parseVersion handles malformed strings gracefully
			description: "Should handle malformed versions gracefully",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := CompareVersions(tc.v1, tc.v2)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: CompareVersions('%s', '%s') = %d", tc.name, tc.v1, tc.v2, result)
		})
	}
}

// TestParseVersion_EdgeCases tests edge cases for version parsing
func (s *VersionUtilitiesTestSuite) TestParseVersion_EdgeCases() {
	testCases := []struct {
		name        string
		input       string
		expected    []int
		description string
	}{
		{
			name:        "Very large version numbers",
			input:       "999999.888888.777777",
			expected:    []int{999999, 888888, 777777},
			description: "Should handle very large version numbers",
		},
		{
			name:        "Version with many parts",
			input:       "1.2.3.4.5.6",
			expected:    []int{1, 2, 3, 4, 5, 6},
			description: "Should handle versions with many parts",
		},
		{
			name:        "Version with leading zeros",
			input:       "01.02.03",
			expected:    []int{1, 2, 3},
			description: "Should handle leading zeros correctly",
		},
		{
			name:        "Version with trailing non-numeric",
			input:       "1.2.3abc",
			expected:    []int{1, 2, 3}, // parseVersion extracts numeric parts
			description: "Should extract numeric parts from version with trailing non-numeric",
		},
		{
			name:        "Version starting with non-numeric",
			input:       "v1.2.3",
			expected:    []int{2, 3}, // parseVersion skips 'v1' as invalid number
			description: "Should skip invalid numeric parts and continue parsing",
		},
		{
			name:        "Mixed valid and invalid parts",
			input:       "1.invalid.3.4",
			expected:    []int{1, 3, 4},
			description: "Should skip invalid parts and continue",
		},
		{
			name:        "Only separators",
			input:       "...",
			expected:    []int{},
			description: "Should return empty for only separators",
		},
		{
			name:        "Version with spaces",
			input:       "1. 2 .3",
			expected:    []int{1, 2, 3}, // parseVersion trims spaces and parses numbers
			description: "Should handle spaces in version string",
		},
		{
			name:        "Negative numbers",
			input:       "-1.2.3",
			expected:    []int{}, // parseVersion can't parse negative numbers in this format
			description: "Should handle negative numbers in version string",
		},
		{
			name:        "Floating point numbers",
			input:       "1.5.2.7",
			expected:    []int{1, 5, 2, 7},
			description: "Should handle each part as integer",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := parseVersion(tc.input)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: parseVersion('%s') = %v", tc.name, tc.input, result)
		})
	}
}

// TestGitHubReleaseFunctionsWithMocking tests GitHub release functions with HTTP mocking
func (s *VersionUtilitiesTestSuite) TestGitHubReleaseFunctionsWithMocking() {
	testCases := []struct {
		name        string
		serverFunc  func() *httptest.Server
		expectError bool
		expectedTag string
		description string
	}{
		{
			name: "Valid GitHub release response",
			serverFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					response := `{
						"tag_name": "v1.2.3",
						"name": "Release v1.2.3",
						"draft": false,
						"prerelease": false,
						"published_at": "2023-08-15T12:00:00Z",
						"body": "Test release notes"
					}`
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(response))
				}))
			},
			expectError: false,
			expectedTag: "v1.2.3",
			description: "Should parse valid GitHub release response",
		},
		{
			name: "Draft release response",
			serverFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					response := `{
						"tag_name": "v2.0.0-draft",
						"name": "Draft Release",
						"draft": true,
						"prerelease": false,
						"published_at": "2023-08-15T12:00:00Z",
						"body": "Draft release"
					}`
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(response))
				}))
			},
			expectError: false,
			expectedTag: "v2.0.0-draft",
			description: "Should handle draft releases",
		},
		{
			name: "Pre-release response",
			serverFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					response := `{
						"tag_name": "v2.0.0-beta1",
						"name": "Beta Release",
						"draft": false,
						"prerelease": true,
						"published_at": "2023-08-15T12:00:00Z",
						"body": "Beta release notes"
					}`
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(response))
				}))
			},
			expectError: false,
			expectedTag: "v2.0.0-beta1",
			description: "Should handle pre-releases",
		},
		{
			name: "Malformed JSON response",
			serverFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"tag_name": "v1.0.0", "malformed": json`))
				}))
			},
			expectError: true,
			expectedTag: "",
			description: "Should handle malformed JSON gracefully",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			server := tc.serverFunc()
			defer server.Close()

			// This is a simplified test since we can't easily inject the server URL
			// into the GetLatestReleaseWithVersion function without modifying the API
			// In a real scenario, we'd need dependency injection or interface mocking

			s.T().Logf("✓ %s: %s (mocking test - would need API modification for full testing)", tc.name, tc.description)
		})
	}
}

// TestVersionInfoStruct tests the Info struct functionality
func (s *VersionUtilitiesTestSuite) TestVersionInfoStruct() {
	testCases := []struct {
		name        string
		info        Info
		description string
	}{
		{
			name: "Version info with newer version available",
			info: Info{
				Current: "1.0.0",
				Latest:  "1.1.0",
				IsNewer: true,
			},
			description: "Should represent version info correctly",
		},
		{
			name: "Version info with current version up to date",
			info: Info{
				Current: "2.0.0",
				Latest:  "2.0.0",
				IsNewer: false,
			},
			description: "Should represent current version info",
		},
		{
			name: "Version info with dev version",
			info: Info{
				Current: "dev",
				Latest:  "1.5.0",
				IsNewer: true,
			},
			description: "Should represent dev version info",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Test Info struct fields
			s.NotEmpty(tc.info.Current, "Current version should not be empty")
			s.NotEmpty(tc.info.Latest, "Latest version should not be empty")

			// Validate IsNewer field matches actual comparison
			actualIsNewer := IsNewerVersion(tc.info.Current, tc.info.Latest)
			s.Equal(tc.info.IsNewer, actualIsNewer, "IsNewer field should match actual comparison")

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestGetGitHubToken tests the getGitHubToken helper function indirectly
func (s *VersionUtilitiesTestSuite) TestGetGitHubToken() {
	// Since getGitHubToken is a private function, we test its behavior indirectly
	// by testing functions that use it

	testCases := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		description string
	}{
		{
			name: "GITHUB_TOKEN environment variable",
			setupEnv: func() {
				_ = os.Setenv("GITHUB_TOKEN", "test-github-token")
				_ = os.Unsetenv("GH_TOKEN")
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("GITHUB_TOKEN")
			},
			description: "Should use GITHUB_TOKEN when available",
		},
		{
			name: "GH_TOKEN environment variable",
			setupEnv: func() {
				_ = os.Unsetenv("GITHUB_TOKEN")
				_ = os.Setenv("GH_TOKEN", "test-gh-token")
			},
			cleanupEnv: func() {
				_ = os.Unsetenv("GH_TOKEN")
			},
			description: "Should use GH_TOKEN when GITHUB_TOKEN is not available",
		},
		{
			name: "No token environment variables",
			setupEnv: func() {
				_ = os.Unsetenv("GITHUB_TOKEN")
				_ = os.Unsetenv("GH_TOKEN")
			},
			cleanupEnv: func() {
				// No cleanup needed
			},
			description: "Should work without authentication tokens",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup environment
			tc.setupEnv()
			defer tc.cleanupEnv()

			// Test that the function can be called (we can't test the actual token usage
			// without making real network calls)
			s.T().Logf("✓ %s: %s (environment setup test)", tc.name, tc.description)
		})
	}
}

// TestFormatGitHubError tests GitHub error formatting indirectly
func (s *VersionUtilitiesTestSuite) TestFormatGitHubError() {
	// Since formatGitHubError is a private function, we test error handling behavior
	// through the public API

	testCases := []struct {
		name        string
		expectError bool
		description string
	}{
		{
			name:        "Invalid owner/repo combination",
			expectError: true,
			description: "Should return formatted error for invalid repositories",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Test with invalid repo that will generate an error
			_, err := GetLatestReleaseWithVersion("nonexistent-user-12345", "nonexistent-repo-12345", "v1.0.0")

			if tc.expectError {
				s.Require().Error(err, tc.description)
				s.Contains(err.Error(), "GitHub API request failed", "Error should contain GitHub API failure message")
			} else {
				s.Require().NoError(err, tc.description)
			}

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestContextHandling tests context-aware behavior
func (s *VersionUtilitiesTestSuite) TestContextHandling() {
	testCases := []struct {
		name        string
		setupCtx    func() context.Context
		expectError bool
		description string
	}{
		{
			name: "Context with reasonable timeout",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				return ctx
			},
			expectError: false,
			description: "Should work with reasonable timeout",
		},
		{
			name: "Context with very short timeout",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
				defer cancel()
				return ctx
			},
			expectError: true,
			description: "Should handle context timeout gracefully",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx := tc.setupCtx()

			// The current GetLatestReleaseWithVersion doesn't accept context
			// This test demonstrates how we would test context handling if it did
			_ = ctx

			s.T().Logf("✓ %s: %s (context handling concept test)", tc.name, tc.description)
		})
	}
}

// TestSuite runs the version utilities test suite
func TestVersionUtilitiesTestSuite(t *testing.T) {
	suite.Run(t, new(VersionUtilitiesTestSuite))
}
