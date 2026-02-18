package version

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "equal versions",
			v1:       "1.0.0",
			v2:       "1.0.0",
			expected: 0,
		},
		{
			name:     "equal versions with v prefix",
			v1:       "v1.0.0",
			v2:       "v1.0.0",
			expected: 0,
		},
		{
			name:     "mixed v prefix",
			v1:       "v1.0.0",
			v2:       "1.0.0",
			expected: 0,
		},
		{
			name:     "major version greater",
			v1:       "2.0.0",
			v2:       "1.0.0",
			expected: 1,
		},
		{
			name:     "major version less",
			v1:       "1.0.0",
			v2:       "2.0.0",
			expected: -1,
		},
		{
			name:     "minor version greater",
			v1:       "1.2.0",
			v2:       "1.1.0",
			expected: 1,
		},
		{
			name:     "minor version less",
			v1:       "1.1.0",
			v2:       "1.2.0",
			expected: -1,
		},
		{
			name:     "patch version greater",
			v1:       "1.0.2",
			v2:       "1.0.1",
			expected: 1,
		},
		{
			name:     "patch version less",
			v1:       "1.0.1",
			v2:       "1.0.2",
			expected: -1,
		},
		{
			name:     "dev version always older",
			v1:       "dev",
			v2:       "1.0.0",
			expected: -1,
		},
		{
			name:     "dev version compared to dev",
			v1:       "dev",
			v2:       "dev",
			expected: 0, // Both dev versions are equal
		},
		{
			name:     "version with suffix",
			v1:       "1.0.0-dirty",
			v2:       "1.0.0",
			expected: 0,
		},
		{
			name:     "version with rc suffix",
			v1:       "1.0.0-rc1",
			v2:       "1.0.0",
			expected: 0,
		},
		{
			name:     "different length versions",
			v1:       "1.0",
			v2:       "1.0.0",
			expected: 0,
		},
		{
			name:     "different length versions 2",
			v1:       "1.0.0",
			v2:       "1.0",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result, "CompareVersions(%s, %s)", tt.v1, tt.v2)
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		expected       bool
	}{
		{
			name:           "newer version available",
			currentVersion: "1.0.0",
			latestVersion:  "1.1.0",
			expected:       true,
		},
		{
			name:           "same version",
			currentVersion: "1.0.0",
			latestVersion:  "1.0.0",
			expected:       false,
		},
		{
			name:           "older version",
			currentVersion: "2.0.0",
			latestVersion:  "1.0.0",
			expected:       false,
		},
		{
			name:           "dev version",
			currentVersion: "dev",
			latestVersion:  "1.0.0",
			expected:       true,
		},
		{
			name:           "patch version newer",
			currentVersion: "1.0.1",
			latestVersion:  "1.0.2",
			expected:       true,
		},
		{
			name:           "with v prefix",
			currentVersion: "v1.0.0",
			latestVersion:  "v1.1.0",
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNewerVersion(tt.currentVersion, tt.latestVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "1.0.0",
			expected: "1.0.0",
		},
		{
			name:     "with v prefix",
			input:    "v1.0.0",
			expected: "1.0.0",
		},
		{
			name:     "with whitespace",
			input:    "  1.0.0  ",
			expected: "1.0.0",
		},
		{
			name:     "with suffix",
			input:    "1.0.0-dirty",
			expected: "1.0.0",
		},
		{
			name:     "with rc suffix",
			input:    "1.0.0-rc1",
			expected: "1.0.0",
		},
		{
			name:     "with git hash",
			input:    "1.0.0-g1234567",
			expected: "1.0.0",
		},
		{
			name:     "complex version",
			input:    "v1.2.3-rc1+build123",
			expected: "1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int
	}{
		{
			name:     "standard version",
			input:    "1.2.3",
			expected: []int{1, 2, 3},
		},
		{
			name:     "two part version",
			input:    "1.2",
			expected: []int{1, 2},
		},
		{
			name:     "single number",
			input:    "1",
			expected: []int{1},
		},
		{
			name:     "with suffix",
			input:    "1.2.3-alpha",
			expected: []int{1, 2, 3},
		},
		{
			name:     "with plus suffix",
			input:    "1.2.3+build",
			expected: []int{1, 2, 3},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []int{},
		},
		{
			name:     "non-numeric parts",
			input:    "a.b.c",
			expected: []int{},
		},
		{
			name:     "mixed numeric and non-numeric",
			input:    "1.a.3",
			expected: []int{1, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetLatestRelease(t *testing.T) {
	// This is an integration test that requires network access
	// Skip in CI or when offline
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	release, err := GetLatestRelease("mrz1836", "go-pre-commit")
	// We don't fail if network is unavailable
	if err != nil {
		t.Logf("Could not fetch release (may be offline): %v", err)
		return
	}

	require.NotNil(t, release)
	assert.NotEmpty(t, release.TagName, "Release should have a tag name")
	assert.NotEmpty(t, release.PublishedAt, "Release should have a published date")
}

func TestGetGitHubToken(t *testing.T) {
	tests := []struct {
		name        string
		githubToken string
		ghToken     string
		expectedHas bool
	}{
		{
			name:        "no token set",
			githubToken: "",
			ghToken:     "",
			expectedHas: false,
		},
		{ // #nosec G101 - test data with example token values
			name:        "GITHUB_TOKEN set",
			githubToken: "test-github-token",
			ghToken:     "",
			expectedHas: true,
		},
		{
			name:        "GH_TOKEN set",
			githubToken: "",
			ghToken:     "test-gh-token",
			expectedHas: true,
		},
		{ // #nosec G101 - test data with example token values
			name:        "both tokens set - GITHUB_TOKEN takes precedence",
			githubToken: "test-github-token",
			ghToken:     "test-gh-token",
			expectedHas: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origGitHubToken := os.Getenv("GITHUB_TOKEN")
			origGHToken := os.Getenv("GH_TOKEN")
			defer func() {
				// Restore original values
				_ = os.Setenv("GITHUB_TOKEN", origGitHubToken)
				_ = os.Setenv("GH_TOKEN", origGHToken)
			}()

			// Set test values
			if tt.githubToken != "" {
				_ = os.Setenv("GITHUB_TOKEN", tt.githubToken)
			} else {
				_ = os.Unsetenv("GITHUB_TOKEN")
			}
			if tt.ghToken != "" {
				_ = os.Setenv("GH_TOKEN", tt.ghToken)
			} else {
				_ = os.Unsetenv("GH_TOKEN")
			}

			token := getGitHubToken()
			if tt.expectedHas {
				assert.NotEmpty(t, token)
			} else {
				assert.Empty(t, token)
			}
		})
	}
}

func TestCompareVersions_CommitHashCases(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "commit hash vs version",
			v1:       "abc1234",
			v2:       "1.0.0",
			expected: -1,
		},
		{
			name:     "version vs commit hash",
			v1:       "1.0.0",
			v2:       "abc1234",
			expected: 1,
		},
		{
			name:     "both commit hashes",
			v1:       "abc1234",
			v2:       "def5678",
			expected: 0,
		},
		{
			name:     "empty version vs version",
			v1:       "",
			v2:       "1.0.0",
			expected: -1,
		},
		{
			name:     "version vs empty",
			v1:       "1.0.0",
			v2:       "",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
