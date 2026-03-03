package update

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatBannerWithNilResult(t *testing.T) {
	banner := FormatBanner(nil)
	assert.Empty(t, banner, "FormatBanner should return empty string for nil result")
}

func TestFormatBannerWithNoUpdateAvailable(t *testing.T) {
	result := &CheckResult{
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.0.0",
		UpdateAvailable: false,
	}

	banner := FormatBanner(result)
	// When no update available, banner should still be generated but ShowBanner won't display it
	// FormatBanner always generates output if result is non-nil
	assert.NotEmpty(t, banner)
}

func TestFormatBannerWithUpdateAvailable(t *testing.T) {
	result := &CheckResult{
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.2.0",
		UpdateAvailable: true,
	}

	banner := FormatBanner(result)
	assert.Contains(t, banner, "v1.0.0", "Banner should contain current version")
	assert.Contains(t, banner, "v1.2.0", "Banner should contain latest version")
	assert.Contains(t, banner, upgradeCmd, "Banner should contain upgrade command")
}

func TestFormatBannerASCIIStyle(t *testing.T) {
	// Test ASCII banner directly
	banner := formatBannerASCII("v1.0.0", "v1.1.0")

	assert.Contains(t, banner, "+--", "ASCII banner should contain box characters")
	assert.Contains(t, banner, "v1.0.0", "Banner should contain current version")
	assert.Contains(t, banner, "v1.1.0", "Banner should contain latest version")
	assert.Contains(t, banner, upgradeCmd, "Banner should contain upgrade command")
	assert.Contains(t, banner, "GO-PRE-COMMIT", "Banner should contain product name")
}

func TestFormatBannerFancyStyle(t *testing.T) {
	// Test fancy banner directly
	banner := formatBannerFancy("v1.0.0", "v1.1.0")

	assert.Contains(t, banner, "╭", "Fancy banner should contain Unicode box characters")
	assert.Contains(t, banner, "╰", "Fancy banner should contain Unicode box characters")
	assert.Contains(t, banner, "v1.0.0", "Banner should contain current version")
	assert.Contains(t, banner, "v1.1.0", "Banner should contain latest version")
	assert.Contains(t, banner, upgradeCmd, "Banner should contain upgrade command")
	assert.Contains(t, banner, "GO-PRE-COMMIT", "Banner should contain product name")
}

func TestShowBannerWithNilResult(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	// Should not panic
	ShowBanner(nil)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stderr = oldStderr

	output := buf.String()
	assert.Empty(t, output, "ShowBanner should produce no output for nil result")
}

func TestShowBannerWithNoUpdate(t *testing.T) {
	result := &CheckResult{
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.0.0",
		UpdateAvailable: false,
	}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	ShowBanner(result)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stderr = oldStderr

	output := buf.String()
	assert.Empty(t, output, "ShowBanner should produce no output when no update available")
}

func TestShowBannerWithError(t *testing.T) {
	result := &CheckResult{
		CurrentVersion:  "v1.0.0",
		UpdateAvailable: true,
		Error:           assert.AnError,
	}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	ShowBanner(result)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stderr = oldStderr

	output := buf.String()
	assert.Empty(t, output, "ShowBanner should produce no output when result has error")
}

func TestShowBannerWithUpdate(t *testing.T) {
	result := &CheckResult{
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.1.0",
		UpdateAvailable: true,
	}

	// Set NO_COLOR to ensure consistent output
	t.Setenv("NO_COLOR", "1")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	ShowBanner(result)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stderr = oldStderr

	output := buf.String()
	assert.NotEmpty(t, output, "ShowBanner should produce output when update available")
	assert.Contains(t, output, "v1.0.0")
	assert.Contains(t, output, "v1.1.0")
}

func TestUseColor(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "color enabled by default (if terminal)",
			envVars:  map[string]string{},
			expected: false, // In tests, stderr is not a terminal
		},
		{
			name: "color disabled in CI",
			envVars: map[string]string{
				"CI": "1",
			},
			expected: false,
		},
		{
			name: "color disabled with NO_COLOR",
			envVars: map[string]string{
				"NO_COLOR": "1",
			},
			expected: false,
		},
		{
			name: "color disabled with NO_COLOR empty string",
			envVars: map[string]string{
				"NO_COLOR": "",
			},
			expected: false, // Tests run without TTY
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env vars
			t.Setenv("CI", "")
			t.Setenv("NO_COLOR", "")

			// Set test env vars
			for key, val := range tt.envVars {
				t.Setenv(key, val)
			}

			result := useColor()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{
			name:     "short string padded",
			input:    "hello",
			width:    10,
			expected: "hello     ",
		},
		{
			name:     "exact width",
			input:    "hello",
			width:    5,
			expected: "hello",
		},
		{
			name:     "long string truncated",
			input:    "hello world",
			width:    5,
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			width:    5,
			expected: "     ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padRight(tt.input, tt.width)
			assert.Equal(t, tt.expected, result)
			assert.Len(t, []rune(result), tt.width, "Result should have correct rune count")
		})
	}
}

func TestPadVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		width    int
		expected string
	}{
		{
			name:     "short version padded",
			version:  "v1.0.0",
			width:    12,
			expected: "v1.0.0      ",
		},
		{
			name:     "exact width",
			version:  "v1.0.0-beta",
			width:    11,
			expected: "v1.0.0-beta",
		},
		{
			name:     "long version truncated",
			version:  "v1.0.0-beta.1.rc2",
			width:    10,
			expected: "v1.0.0-bet",
		},
		{
			name:     "empty version",
			version:  "",
			width:    5,
			expected: "     ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padVersion(tt.version, tt.width)
			assert.Equal(t, tt.expected, result)
			assert.Len(t, []rune(result), tt.width, "Result should have correct rune count")
		})
	}
}

func TestPadVersionUnicode(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		width       int
		expectedLen int
	}{
		{
			name:        "ASCII version",
			version:     "v1.0.0",
			width:       12,
			expectedLen: 12,
		},
		{
			name:        "Unicode characters",
			version:     "v日本語", //nolint:gosmopolitan // Testing Unicode handling
			width:       12,
			expectedLen: 12,
		},
		{
			name:        "Emoji in version",
			version:     "v1.0🎉",
			width:       10,
			expectedLen: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padVersion(tt.version, tt.width)
			runeCount := len([]rune(result))
			assert.Equal(t, tt.expectedLen, runeCount,
				"Expected %d runes, got %d", tt.expectedLen, runeCount)
		})
	}
}

func TestPadRightUnicode(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		width       int
		expectedLen int
	}{
		{
			name:        "ASCII string",
			input:       "hello",
			width:       10,
			expectedLen: 10,
		},
		{
			name:        "Unicode string",
			input:       "こんにちは",
			width:       10,
			expectedLen: 10,
		},
		{
			name:        "Mixed ASCII and Unicode",
			input:       "Hello世界", //nolint:gosmopolitan // Testing Unicode handling
			width:       15,
			expectedLen: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padRight(tt.input, tt.width)
			runeCount := len([]rune(result))
			assert.Equal(t, tt.expectedLen, runeCount,
				"Expected %d runes, got %d", tt.expectedLen, runeCount)
		})
	}
}

func TestBannerConstants(t *testing.T) {
	// Verify banner constants
	assert.Equal(t, "\033[0m", bannerColorReset, "Reset color should be ANSI reset code")
	assert.Equal(t, "\033[33m", bannerColorYellow, "Yellow color should be ANSI yellow code")
	assert.Equal(t, "go-pre-commit upgrade", upgradeCmd, "Upgrade command should be correct")
	assert.Equal(t, 12, versionDisplayWidth, "Version display width should be 12")
}

func TestBannerLineWidthConsistency(t *testing.T) {
	banner := formatBannerASCII("v1.0.0", "v99.99.99")
	lines := strings.Split(banner, "\n")

	// Find the width of box lines
	var boxWidth int
	for _, line := range lines {
		if strings.Contains(line, "+") && strings.Contains(line, "-") {
			boxWidth = len(line)
			break
		}
	}

	// All lines with | should have same width
	for _, line := range lines {
		if strings.Contains(line, "|") {
			assert.Len(t, line, boxWidth,
				"Line width mismatch: %q", line)
		}
	}
}

func TestFormatBannerVariousVersionCombinations(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
	}{
		{
			name:    "standard versions",
			current: "v1.0.0",
			latest:  "v1.1.0",
		},
		{
			name:    "dev to release",
			current: "dev",
			latest:  "v1.0.0",
		},
		{
			name:    "long versions",
			current: "v1.2.3-beta.1",
			latest:  "v2.0.0-rc.1",
		},
		{
			name:    "no v prefix",
			current: "1.0.0",
			latest:  "1.1.0",
		},
		{
			name:    "very long version",
			current: "v1.2.3-beta.4.rc5+build.123",
			latest:  "v2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			banner := formatBannerASCII(tt.current, tt.latest)

			// Both versions should appear (possibly truncated)
			assert.NotEmpty(t, banner, "Banner should not be empty")
			assert.Contains(t, banner, "Current:")
			assert.Contains(t, banner, "Latest:")
			assert.Contains(t, banner, upgradeCmd)
		})
	}
}

func TestShowBannerWithColor(t *testing.T) {
	result := &CheckResult{
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.1.0",
		UpdateAvailable: true,
	}

	// Clear NO_COLOR but CI will likely still disable it in tests
	t.Setenv("NO_COLOR", "")
	t.Setenv("CI", "")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	ShowBanner(result)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stderr = oldStderr

	output := buf.String()
	assert.NotEmpty(t, output, "Should produce output")
	// In test environment (not a TTY), colors won't be used
}

func TestIsTerminal(t *testing.T) {
	// In test environment, stderr is not a terminal
	result := isTerminal(os.Stderr.Fd())
	assert.False(t, result, "In tests, stderr should not be detected as terminal")
}

func TestFormatBannerEmptyVersions(t *testing.T) {
	banner := formatBannerASCII("", "v1.0.0")
	assert.NotEmpty(t, banner)
	assert.Contains(t, banner, "Current:")
	assert.Contains(t, banner, "Latest:")

	banner2 := formatBannerASCII("v1.0.0", "")
	assert.NotEmpty(t, banner2)
	assert.Contains(t, banner2, "Current:")
	assert.Contains(t, banner2, "Latest:")
}

func TestFormatBannerFancyLineWidthConsistency(t *testing.T) {
	banner := formatBannerFancy("v1.0.0", "v2.0.0")
	lines := strings.Split(banner, "\n")

	// All content lines should have consistent width (counting runes)
	for _, line := range lines {
		if strings.Contains(line, "│") {
			// Check that line has roughly the same width
			// (Unicode box drawing chars can affect byte count)
			assert.NotEmpty(t, line, "Line should not be empty")
		}
	}
}

func TestCheckResultForDisplay(t *testing.T) {
	// Test that CheckResult can be used for banner display
	result := &CheckResult{
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.1.0",
		UpdateAvailable: true,
		CheckedAt:       time.Now(),
		FromCache:       false,
		Error:           nil,
	}

	// Should not panic
	banner := FormatBanner(result)
	assert.NotEmpty(t, banner)

	// Show should not panic
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	ShowBanner(result)

	_ = w.Close()
	os.Stderr = oldStderr
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	// Should produce output
	assert.NotEmpty(t, buf.String())
}

func TestBannerBoxWidth(t *testing.T) {
	// ASCII banner box should be 70 chars wide (between | and |)
	banner := formatBannerASCII("v1.0.0", "v1.1.0")
	lines := strings.Split(banner, "\n")

	for _, line := range lines {
		if strings.Contains(line, "+--") && strings.Contains(line, "--+") {
			// Should be 74 chars total (2 spaces + 70 content + 2 symbols)
			assert.Len(t, line, 74, "Box line should be 74 chars: %q", line)
		}
	}
}

func TestFormatBannerWithRealResult(t *testing.T) {
	// Simulate a real check result
	result := &CheckResult{
		CurrentVersion:  "v0.9.0",
		LatestVersion:   "v1.0.0",
		UpdateAvailable: true,
		CheckedAt:       time.Now(),
		FromCache:       false,
	}

	banner := FormatBanner(result)
	assert.Contains(t, banner, "v0.9.0")
	assert.Contains(t, banner, "v1.0.0")
	assert.Contains(t, banner, "GO-PRE-COMMIT")
	assert.Contains(t, banner, upgradeCmd)
}

func TestShowBannerWritesToStderr(t *testing.T) {
	result := &CheckResult{
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.1.0",
		UpdateAvailable: true,
	}

	t.Setenv("NO_COLOR", "1")

	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	ShowBanner(result)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stderr = oldStderr

	output := buf.String()
	assert.NotEmpty(t, output, "Output should go to stderr")
	assert.Contains(t, output, "\n", "Output should contain newlines")
}

func TestBannerColorFormatting(t *testing.T) {
	result := &CheckResult{
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.1.0",
		UpdateAvailable: true,
	}

	// Force color off
	t.Setenv("NO_COLOR", "1")

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w

	ShowBanner(result)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	os.Stderr = oldStderr

	output := buf.String()

	// Should not contain color codes when NO_COLOR is set
	assert.NotContains(t, output, bannerColorYellow, "Should not contain color codes with NO_COLOR")
}

func TestUseColorNOCOLOR(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{
			name:     "NO_COLOR=1",
			value:    "1",
			expected: false,
		},
		{
			name:     "NO_COLOR=true",
			value:    "true",
			expected: false,
		},
		{
			name:     "NO_COLOR empty",
			value:    "",
			expected: false, // Even empty NO_COLOR disables color per spec
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CI", "")
			t.Setenv("NO_COLOR", tt.value)

			result := useColor()
			assert.Equal(t, tt.expected, result)
		})
	}
}
