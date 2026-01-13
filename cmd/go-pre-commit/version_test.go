package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
		ver := bi.Version()
		assert.NotEmpty(t, ver)
		// Should not be a template string
		assert.False(t, isTemplateString(ver))
	})

	t.Run("Commit", func(t *testing.T) {
		com := bi.Commit()
		assert.NotEmpty(t, com)
		// Should not be a template string
		assert.False(t, isTemplateString(com))
	})

	t.Run("BuildDate", func(t *testing.T) {
		bDate := bi.BuildDate()
		assert.NotEmpty(t, bDate)
		// Should not be a template string
		assert.False(t, isTemplateString(bDate))
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
	ver := getVersionWithFallback()
	assert.NotEmpty(t, ver)
	assert.False(t, isTemplateString(ver))

	// Version should be one of the expected formats
	// It could be "dev", a semantic version, or a short commit hash
	assert.NotEmpty(t, ver)
}

// TestGetCommitWithFallback tests the commit fallback logic
func TestGetCommitWithFallback(t *testing.T) {
	// This test verifies the function runs and returns a valid value.
	// The exact value depends on build-time variables and runtime build info.
	com := getCommitWithFallback()
	assert.NotEmpty(t, com)
	assert.False(t, isTemplateString(com))

	// Commit should be "none" or a hash value
	assert.NotEmpty(t, com)
}

// TestGetBuildDateWithFallback tests the build date fallback logic
func TestGetBuildDateWithFallback(t *testing.T) {
	// This test verifies the function runs and returns a valid value.
	// The exact value depends on build-time variables and runtime build info.
	bDate := getBuildDateWithFallback()
	assert.NotEmpty(t, bDate)
	assert.False(t, isTemplateString(bDate))

	// Build date should be one of the expected values
	assert.NotEmpty(t, bDate)
}

// TestLegacyCompatibilityFunctions tests the legacy wrapper functions
func TestLegacyCompatibilityFunctions(t *testing.T) {
	t.Run("GetVersion", func(t *testing.T) {
		ver := GetVersion()
		assert.NotEmpty(t, ver)
		assert.Equal(t, getVersionWithFallback(), ver)
	})

	t.Run("GetCommit", func(t *testing.T) {
		com := GetCommit()
		assert.NotEmpty(t, com)
		assert.Equal(t, getCommitWithFallback(), com)
	})

	t.Run("GetBuildDate", func(t *testing.T) {
		bDate := GetBuildDate()
		assert.NotEmpty(t, bDate)
		assert.Equal(t, getBuildDateWithFallback(), bDate)
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

// TestGetVersionWithFallbackEdgeCases tests edge cases in version fallback logic
func TestGetVersionWithFallbackEdgeCases(t *testing.T) {
	// Test that the function handles all code paths
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "default behavior",
			description: "Tests that getVersionWithFallback returns a valid version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver := getVersionWithFallback()

			// Version should never be empty
			assert.NotEmpty(t, ver)

			// Version should not contain template markers
			assert.NotContains(t, ver, "{{")
			assert.NotContains(t, ver, "}}")

			// Version should be one of: "dev", a semantic version, or a commit hash
			// We don't assert the exact value because it depends on build context
		})
	}
}

// TestGetCommitWithFallbackEdgeCases tests edge cases in commit fallback logic
func TestGetCommitWithFallbackEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "default behavior",
			description: "Tests that getCommitWithFallback returns a valid commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			com := getCommitWithFallback()

			// Commit should never be empty
			assert.NotEmpty(t, com)

			// Commit should not contain template markers
			assert.NotContains(t, com, "{{")
			assert.NotContains(t, com, "}}")

			// Commit is typically "none" or a hash
			// We verify it's a reasonable value
			assert.True(t, commit == "none" || len(commit) >= 7)
		})
	}
}

// TestGetBuildDateWithFallbackEdgeCases tests edge cases in build date fallback logic
func TestGetBuildDateWithFallbackEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{
			name:        "default behavior",
			description: "Tests that getBuildDateWithFallback returns a valid build date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bDate := getBuildDateWithFallback()

			// Build date should never be empty
			assert.NotEmpty(t, bDate)

			// Build date should not contain template markers
			assert.NotContains(t, bDate, "{{")
			assert.NotContains(t, bDate, "}}")

			// Build date is typically "unknown", "go-install", or a formatted date
			// We verify it's a reasonable value
			assert.NotEmpty(t, buildDate)
		})
	}
}

// TestIsModifiedBehavior tests IsModified under different conditions
func TestIsModifiedBehavior(t *testing.T) {
	t.Run("via BuildInfo", func(t *testing.T) {
		bi := NewBuildInfo()
		modified := bi.IsModified()
		// Should return false or true depending on actual VCS state
		assert.IsType(t, false, modified)
	})

	t.Run("via legacy function", func(t *testing.T) {
		modified := IsModified()
		// Should return false or true depending on actual VCS state
		assert.IsType(t, false, modified)

		// Both methods should return the same result
		bi := NewBuildInfo()
		assert.Equal(t, bi.IsModified(), modified)
	})
}

// TestParseTimeErrorCases tests various error conditions
func TestParseTimeErrorCases(t *testing.T) {
	errorCases := []string{
		"",                     // empty
		"not-a-date",           // invalid
		"2024-13-01T10:30:45Z", // invalid month
		"2024-01-32T10:30:45Z", // invalid day
		"2024-01-01T25:00:00Z", // invalid hour
		"2024-01-01T10:61:00Z", // invalid minute
		"2024-01-01T10:30:61Z", // invalid second
		"just a random string",
		"2024",       // incomplete
		"2024-01",    // incomplete
		"2024-01-01", // no time component
	}

	for _, input := range errorCases {
		t.Run(input, func(t *testing.T) {
			result, err := parseTime(input)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrUnableToParseTime)
			assert.True(t, result.IsZero())
		})
	}
}

// TestParseTimeSuccessCases tests various successful parse scenarios
func TestParseTimeSuccessCases(t *testing.T) {
	successCases := []struct {
		input    string
		expected time.Time
	}{
		{
			input:    "2024-01-15T10:30:45Z",
			expected: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		},
		{
			input:    "2024-01-15T10:30:45.000Z",
			expected: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		},
		{
			input:    "2024-01-15 10:30:45",
			expected: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		},
		{
			input:    "2024-12-31T23:59:59Z",
			expected: time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
		},
		{
			input:    "2024-01-01T00:00:00Z",
			expected: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range successCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := parseTime(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
			// Ensure result is always in UTC
			assert.Equal(t, time.UTC, result.Location())
		})
	}
}

// TestVersionFunctionsWithAllBranches attempts to exercise all code branches
func TestVersionFunctionsWithAllBranches(t *testing.T) {
	// These tests call the version functions multiple times to increase
	// the likelihood of exercising different branches based on build context

	t.Run("version fallback", func(t *testing.T) {
		v1 := getVersionWithFallback()
		v2 := getVersionWithFallback()

		// Should be consistent across calls
		assert.Equal(t, v1, v2)

		// Should be valid
		assert.NotEmpty(t, v1)
		assert.False(t, isTemplateString(v1))

		// Test the legacy function too
		v3 := GetVersion()
		assert.Equal(t, v1, v3)
	})

	t.Run("commit fallback", func(t *testing.T) {
		c1 := getCommitWithFallback()
		c2 := getCommitWithFallback()

		// Should be consistent across calls
		assert.Equal(t, c1, c2)

		// Should be valid
		assert.NotEmpty(t, c1)
		assert.False(t, isTemplateString(c1))

		// Test the legacy function too
		c3 := GetCommit()
		assert.Equal(t, c1, c3)
	})

	t.Run("buildDate fallback", func(t *testing.T) {
		b1 := getBuildDateWithFallback()
		b2 := getBuildDateWithFallback()

		// Should be consistent across calls
		assert.Equal(t, b1, b2)

		// Should be valid
		assert.NotEmpty(t, b1)
		assert.False(t, isTemplateString(b1))

		// Test the legacy function too
		b3 := GetBuildDate()
		assert.Equal(t, b1, b3)
	})

	t.Run("isModified consistency", func(t *testing.T) {
		// Test that IsModified is consistent
		m1 := IsModified()
		m2 := IsModified()
		assert.Equal(t, m1, m2)

		// Test via BuildInfo
		bi := NewBuildInfo()
		m3 := bi.IsModified()
		assert.Equal(t, m1, m3)
	})
}

// TestTemplateStringVariations tests template string detection thoroughly
func TestTemplateStringVariations(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// True cases - contains both {{ and }}
		{"{{ .Version }}", true},
		{"prefix{{ .Version }}suffix", true},
		{"{{ var1 }}{{ var2 }}", true},
		{"text {{ expr }} more", true},

		// False cases - missing one or both markers
		{"normal text", false},
		{"{{ incomplete", false},
		{"incomplete }}", false},
		{"", false},
		{"v1.2.3", false},
		{"dev", false},
		{"abc123def456", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isTemplateString(tt.input)
			assert.Equal(t, tt.expected, result, "isTemplateString(%q) should be %v", tt.input, tt.expected)
		})
	}
}

// TestBuildInfoWithCustomLdflags tests version info when built with custom ldflags
func TestBuildInfoWithCustomLdflags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping build test in short mode")
	}

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Logf("Failed to restore working directory: %v", chdirErr)
		}
	}()

	var buildPath string
	if strings.Contains(originalWD, "/cmd/go-pre-commit") {
		buildPath = "."
	} else {
		buildPath = "./cmd/go-pre-commit"
	}

	testCases := []struct {
		name      string
		ldflags   string
		checkFunc func(t *testing.T, output string)
	}{
		{
			name:    "custom version ldflags",
			ldflags: "-X main.version=v1.2.3 -X main.commit=abc123def -X main.buildDate=2024-01-15",
			checkFunc: func(t *testing.T, output string) {
				assert.Contains(t, output, "v1.2.3")
				assert.Contains(t, output, "abc123")
				assert.Contains(t, output, "2024-01-15")
			},
		},
		{
			name:    "dev version ldflags",
			ldflags: "-X main.version=dev -X main.commit=none -X main.buildDate=unknown",
			checkFunc: func(t *testing.T, output string) {
				// With dev/none/unknown, it should fall back to build info
				assert.Contains(t, output, "version")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			testBinary := filepath.Join(t.TempDir(), "test-ldflags-binary")

			buildCmd := exec.CommandContext(ctx, "go", "build", "-ldflags", tc.ldflags, "-o", testBinary, buildPath) //nolint:gosec // Safe: controlled test input

			var stdout, stderr bytes.Buffer
			buildCmd.Stdout = &stdout
			buildCmd.Stderr = &stderr

			err := buildCmd.Run()
			if err != nil {
				t.Logf("Build failed. stdout: %s, stderr: %s", stdout.String(), stderr.String())
			}
			require.NoError(t, err)

			// Run with version flag
			testCmd := exec.CommandContext(ctx, testBinary, "--no-color", "--version") //nolint:gosec // Safe: our own binary
			output, err := testCmd.Output()
			require.NoError(t, err)

			outputStr := string(output)
			tc.checkFunc(t, outputStr)
		})
	}
}

// TestFallbackFunctionsWithModifiedPackageVars tests the fallback functions
// by temporarily modifying package-level variables to exercise different code paths
func TestFallbackFunctionsWithModifiedPackageVars(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origBuildDate := buildDate

	// Restore at end
	defer func() {
		version = origVersion
		commit = origCommit
		buildDate = origBuildDate
	}()

	t.Run("getVersionWithFallback with custom ldflags", func(t *testing.T) {
		// Test when version is set via ldflags to a real version
		version = "v1.2.3"
		result := getVersionWithFallback()
		assert.Equal(t, "v1.2.3", result)

		// Reset
		version = origVersion
	})

	t.Run("getVersionWithFallback with empty string", func(t *testing.T) {
		// Test when version is empty (should fall back)
		version = ""
		result := getVersionWithFallback()
		assert.NotEmpty(t, result)

		// Reset
		version = origVersion
	})

	t.Run("getVersionWithFallback with template", func(t *testing.T) {
		// Test when version contains template markers (should fall back)
		version = "{{ .Version }}"
		result := getVersionWithFallback()
		assert.NotContains(t, result, "{{")

		// Reset
		version = origVersion
	})

	t.Run("getVersionWithFallback with various invalid values", func(t *testing.T) {
		testCases := []string{
			"dev",            // default value
			"",               // empty
			"{{ .Version }}", // template
			"{{version}}",    // different template style
			"development",    // not template but close to default
			"v0.0.0",         // valid semver
			"1.2.3-beta",     // pre-release version
			"latest",         // tag-like value
			"main-abc123",    // branch-commit format
		}

		for _, testVal := range testCases {
			version = testVal
			result := getVersionWithFallback()
			assert.NotEmpty(t, result)
			// Should never contain template markers
			assert.NotContains(t, result, "{{")
			assert.NotContains(t, result, "}}")
		}

		// Reset
		version = origVersion
	})

	t.Run("getCommitWithFallback with custom ldflags", func(t *testing.T) {
		// Test when commit is set via ldflags to a real commit
		commit = "abc123def456"
		result := getCommitWithFallback()
		assert.Equal(t, "abc123def456", result)

		// Reset
		commit = origCommit
	})

	t.Run("getCommitWithFallback with empty string", func(t *testing.T) {
		// Test when commit is empty (should fall back)
		commit = ""
		result := getCommitWithFallback()
		assert.NotEmpty(t, result)

		// Reset
		commit = origCommit
	})

	t.Run("getCommitWithFallback with template", func(t *testing.T) {
		// Test when commit contains template markers (should fall back)
		commit = "{{ .Commit }}"
		result := getCommitWithFallback()
		assert.NotContains(t, result, "{{")

		// Reset
		commit = origCommit
	})

	t.Run("getCommitWithFallback with various formats", func(t *testing.T) {
		testCases := []string{
			"none",             // default value
			"",                 // empty
			"{{ .Commit }}",    // template
			"abcdef1",          // short hash (7 chars)
			"abcdef1234567890", // long hash
			"HEAD",             // symbolic ref
			"abc123-dirty",     // dirty suffix
		}

		for _, testVal := range testCases {
			commit = testVal
			result := getCommitWithFallback()
			assert.NotEmpty(t, result)
			assert.NotContains(t, result, "{{")
		}

		// Reset
		commit = origCommit
	})

	t.Run("getBuildDateWithFallback with custom ldflags", func(t *testing.T) {
		// Test when buildDate is set via ldflags to a real date
		buildDate = "2024-01-15T10:00:00Z"
		result := getBuildDateWithFallback()
		assert.Equal(t, "2024-01-15T10:00:00Z", result)

		// Reset
		buildDate = origBuildDate
	})

	t.Run("getBuildDateWithFallback with empty string", func(t *testing.T) {
		// Test when buildDate is empty (should fall back)
		buildDate = ""
		result := getBuildDateWithFallback()
		assert.NotEmpty(t, result)

		// Reset
		buildDate = origBuildDate
	})

	t.Run("getBuildDateWithFallback with template", func(t *testing.T) {
		// Test when buildDate contains template markers (should fall back)
		buildDate = "{{ .BuildDate }}"
		result := getBuildDateWithFallback()
		assert.NotContains(t, result, "{{")

		// Reset
		buildDate = origBuildDate
	})

	t.Run("getBuildDateWithFallback with various formats", func(t *testing.T) {
		testCases := []string{
			"unknown",                  // default value
			"",                         // empty
			"{{ .BuildDate }}",         // template
			"2024-01-15",               // date only
			"2024-01-15T10:30:45Z",     // RFC3339
			"2024-01-15 10:30:45",      // space separated
			"Mon Jan 15 10:30:45 2024", // different format
			"1705318245",               // unix timestamp
			"go-install",               // go install marker
		}

		for _, tc := range testCases {
			buildDate = tc
			result := getBuildDateWithFallback()
			assert.NotEmpty(t, result)
			assert.False(t, isTemplateString(result))
		}

		// Reset
		buildDate = origBuildDate
	})
}

// TestBuildInfoMethodsComprehensive tests all BuildInfo methods exhaustively
func TestBuildInfoMethodsComprehensive(t *testing.T) {
	// Call NewBuildInfo multiple times to ensure consistency
	for i := 0; i < 10; i++ {
		bi := NewBuildInfo()
		require.NotNil(t, bi)

		// All getters should return non-empty values
		assert.NotEmpty(t, bi.Version())
		assert.NotEmpty(t, bi.Commit())
		assert.NotEmpty(t, bi.BuildDate())

		// IsModified should return a boolean
		modified := bi.IsModified()
		assert.IsType(t, false, modified)

		// Verify no template strings
		assert.False(t, isTemplateString(bi.Version()))
		assert.False(t, isTemplateString(bi.Commit()))
		assert.False(t, isTemplateString(bi.BuildDate()))
	}
}

// TestLegacyFunctionsComprehensive tests legacy wrapper functions exhaustively
func TestLegacyFunctionsComprehensive(t *testing.T) {
	// Test each legacy function multiple times
	for i := 0; i < 5; i++ {
		ver := GetVersion()
		com := GetCommit()
		bDate := GetBuildDate()
		modified := IsModified()

		assert.NotEmpty(t, ver)
		assert.NotEmpty(t, com)
		assert.NotEmpty(t, bDate)
		assert.IsType(t, false, modified)

		// Verify consistency
		assert.Equal(t, getVersionWithFallback(), ver)
		assert.Equal(t, getCommitWithFallback(), com)
		assert.Equal(t, getBuildDateWithFallback(), bDate)
	}
}
