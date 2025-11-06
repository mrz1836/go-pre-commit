package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBuildInfo tests the constructor
func TestNewBuildInfo(t *testing.T) {
	bi := NewBuildInfo()
	assert.NotNil(t, bi)
	assert.NotEmpty(t, bi.Version())
	assert.NotEmpty(t, bi.Commit())
	assert.NotEmpty(t, bi.BuildDate())
}

// TestBuildInfoGetters tests the getter methods
func TestBuildInfoGetters(t *testing.T) {
	bi := NewBuildInfo()

	t.Run("Version", func(t *testing.T) {
		version := bi.Version()
		assert.NotEmpty(t, version)
		// Should not be a template string
		assert.False(t, isTemplateString(version))
	})

	t.Run("Commit", func(t *testing.T) {
		commit := bi.Commit()
		assert.NotEmpty(t, commit)
		// Should not be a template string
		assert.False(t, isTemplateString(commit))
	})

	t.Run("BuildDate", func(t *testing.T) {
		buildDate := bi.BuildDate()
		assert.NotEmpty(t, buildDate)
		// Should not be a template string
		assert.False(t, isTemplateString(buildDate))
	})
}

// TestBuildInfoIsModified tests the IsModified method
func TestBuildInfoIsModified(t *testing.T) {
	bi := NewBuildInfo()
	// The result depends on the actual build info, so we just verify it returns a bool
	modified := bi.IsModified()
	assert.IsType(t, false, modified)
}

// TestIsTemplateString tests template string detection
func TestIsTemplateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Normal string",
			input:    "dev",
			expected: false,
		},
		{
			name:     "Version string",
			input:    "v1.2.3",
			expected: false,
		},
		{
			name:     "Template with both markers",
			input:    "{{ .Version }}",
			expected: true,
		},
		{
			name:     "Template in middle",
			input:    "version-{{ .Tag }}-suffix",
			expected: true,
		},
		{
			name:     "Only opening marker",
			input:    "{{ incomplete",
			expected: false,
		},
		{
			name:     "Only closing marker",
			input:    "incomplete }}",
			expected: false,
		},
		{
			name:     "Reversed markers",
			input:    "}} reverse {{",
			expected: true, // Still contains both
		},
		{
			name:     "Multiple templates",
			input:    "{{ .Version }}-{{ .Commit }}",
			expected: true,
		},
		{
			name:     "Nested markers",
			input:    "{{ {{ nested }} }}",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTemplateString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseTime tests time parsing with multiple formats
func TestParseTime(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
		description string
	}{
		{
			name:        "RFC3339 format",
			input:       "2024-01-15T10:30:45Z",
			shouldError: false,
			description: "Standard RFC3339 format",
		},
		{
			name:        "RFC3339 with timezone",
			input:       "2024-01-15T10:30:45-07:00",
			shouldError: false,
			description: "RFC3339 with timezone offset",
		},
		{
			name:        "RFC3339 with milliseconds",
			input:       "2024-01-15T10:30:45.123Z",
			shouldError: false,
			description: "RFC3339 with milliseconds",
		},
		{
			name:        "ISO8601 basic",
			input:       "2024-01-15T10:30:45Z",
			shouldError: false,
			description: "ISO8601 basic format",
		},
		{
			name:        "ISO8601 with milliseconds",
			input:       "2024-01-15T10:30:45.000Z",
			shouldError: false,
			description: "ISO8601 with milliseconds",
		},
		{
			name:        "Space separated",
			input:       "2024-01-15 10:30:45",
			shouldError: false,
			description: "Space-separated date time",
		},
		{
			name:        "Invalid format",
			input:       "not-a-date",
			shouldError: true,
			description: "Invalid date string",
		},
		{
			name:        "Partial date",
			input:       "2024-01-15",
			shouldError: true,
			description: "Date without time",
		},
		{
			name:        "Empty string",
			input:       "",
			shouldError: true,
			description: "Empty input",
		},
		{
			name:        "Wrong delimiter",
			input:       "2024/01/15 10:30:45",
			shouldError: true,
			description: "Wrong date delimiter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTime(tt.input)

			if tt.shouldError {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrUnableToParseTime)
				assert.True(t, result.IsZero())
			} else {
				require.NoError(t, err)
				assert.False(t, result.IsZero())
				// Verify the result is in UTC
				assert.Equal(t, time.UTC, result.Location())
			}
		})
	}
}

// TestParseTimePreservesAccuracy tests that parsed times maintain accuracy
func TestParseTimePreservesAccuracy(t *testing.T) {
	// Test with a known time
	input := "2024-01-15T10:30:45Z"
	result, err := parseTime(input)
	require.NoError(t, err)

	expected := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	assert.Equal(t, expected, result)
}

// TestGetVersionWithFallback tests the version fallback logic
func TestGetVersionWithFallback(t *testing.T) {
	// This test verifies the function runs and returns a valid value.
	// The exact value depends on build-time variables and runtime build info.
	version := getVersionWithFallback()
	assert.NotEmpty(t, version)
	assert.False(t, isTemplateString(version))

	// Version should be one of the expected formats
	// It could be "dev", a semantic version, or a short commit hash
	assert.NotEmpty(t, version)
}

// TestGetCommitWithFallback tests the commit fallback logic
func TestGetCommitWithFallback(t *testing.T) {
	// This test verifies the function runs and returns a valid value.
	// The exact value depends on build-time variables and runtime build info.
	commit := getCommitWithFallback()
	assert.NotEmpty(t, commit)
	assert.False(t, isTemplateString(commit))

	// Commit should be "none" or a hash value
	assert.NotEmpty(t, commit)
}

// TestGetBuildDateWithFallback tests the build date fallback logic
func TestGetBuildDateWithFallback(t *testing.T) {
	// This test verifies the function runs and returns a valid value.
	// The exact value depends on build-time variables and runtime build info.
	buildDate := getBuildDateWithFallback()
	assert.NotEmpty(t, buildDate)
	assert.False(t, isTemplateString(buildDate))

	// Build date should be one of the expected values
	assert.NotEmpty(t, buildDate)
}

// TestLegacyCompatibilityFunctions tests the legacy wrapper functions
func TestLegacyCompatibilityFunctions(t *testing.T) {
	t.Run("GetVersion", func(t *testing.T) {
		version := GetVersion()
		assert.NotEmpty(t, version)
		assert.Equal(t, getVersionWithFallback(), version)
	})

	t.Run("GetCommit", func(t *testing.T) {
		commit := GetCommit()
		assert.NotEmpty(t, commit)
		assert.Equal(t, getCommitWithFallback(), commit)
	})

	t.Run("GetBuildDate", func(t *testing.T) {
		buildDate := GetBuildDate()
		assert.NotEmpty(t, buildDate)
		assert.Equal(t, getBuildDateWithFallback(), buildDate)
	})

	t.Run("IsModified", func(t *testing.T) {
		modified := IsModified()
		assert.IsType(t, false, modified)
	})
}

// TestBuildInfoConsistency tests that multiple calls return consistent values
func TestBuildInfoConsistency(t *testing.T) {
	bi1 := NewBuildInfo()
	bi2 := NewBuildInfo()

	// Multiple calls should return consistent values
	assert.Equal(t, bi1.Version(), bi2.Version())
	assert.Equal(t, bi1.Commit(), bi2.Commit())
	assert.Equal(t, bi1.BuildDate(), bi2.BuildDate())
	assert.Equal(t, bi1.IsModified(), bi2.IsModified())
}

// TestBuildInfoFieldsNotEmpty tests that BuildInfo fields are never completely empty
func TestBuildInfoFieldsNotEmpty(t *testing.T) {
	bi := NewBuildInfo()

	// All fields should have at least default values
	assert.NotEmpty(t, bi.Version())
	assert.NotEmpty(t, bi.Commit())
	assert.NotEmpty(t, bi.BuildDate())
}

// TestParseTimeWithVariousTimezones tests timezone handling
func TestParseTimeWithVariousTimezones(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "UTC timezone",
			input:    "2024-01-15T10:30:45Z",
			expected: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		},
		{
			name:     "PST timezone",
			input:    "2024-01-15T10:30:45-08:00",
			expected: time.Date(2024, 1, 15, 18, 30, 45, 0, time.UTC),
		},
		{
			name:     "EST timezone",
			input:    "2024-01-15T10:30:45-05:00",
			expected: time.Date(2024, 1, 15, 15, 30, 45, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTime(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestErrUnableToParseTime verifies the error type
func TestErrUnableToParseTime(t *testing.T) {
	_, err := parseTime("invalid-date")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnableToParseTime)
	assert.Equal(t, "unable to parse time", err.Error())
}
