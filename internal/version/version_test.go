package version

import (
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
			expected: -1, // Both dev versions
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
