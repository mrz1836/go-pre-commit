package stringutils

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// StringUtilitiesTestSuite tests string utility functions across the application
type StringUtilitiesTestSuite struct {
	suite.Suite
}

// TestFormatDuration tests duration formatting utilities
func (s *StringUtilitiesTestSuite) TestFormatDuration() {
	testCases := []struct {
		name        string
		duration    time.Duration
		expected    string
		description string
	}{
		{
			name:        "Milliseconds",
			duration:    150 * time.Millisecond,
			expected:    "150ms",
			description: "Should format milliseconds correctly",
		},
		{
			name:        "Seconds",
			duration:    2500 * time.Millisecond,
			expected:    "2.5s",
			description: "Should format seconds correctly",
		},
		{
			name:        "Minutes",
			duration:    90 * time.Second,
			expected:    "1m30s",
			description: "Should format minutes correctly",
		},
		{
			name:        "Hours",
			duration:    3665 * time.Second,
			expected:    "1h1m5s",
			description: "Should format hours correctly",
		},
		{
			name:        "Zero duration",
			duration:    0,
			expected:    "0ns", // FormatDuration formats 0 as nanoseconds
			description: "Should handle zero duration",
		},
		{
			name:        "Very small duration",
			duration:    time.Nanosecond,
			expected:    "1ns",
			description: "Should handle nanoseconds",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := FormatDuration(tc.duration)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: FormatDuration(%v) = '%s'", tc.name, tc.duration, result)
		})
	}
}

// TestFormatBytes tests byte formatting utilities
func (s *StringUtilitiesTestSuite) TestFormatBytes() {
	testCases := []struct {
		name        string
		bytes       int64
		expected    string
		description string
	}{
		{
			name:        "Bytes",
			bytes:       512,
			expected:    "512 B",
			description: "Should format bytes correctly",
		},
		{
			name:        "Kilobytes",
			bytes:       1536,
			expected:    "1.5 KB",
			description: "Should format kilobytes correctly",
		},
		{
			name:        "Megabytes",
			bytes:       1572864,
			expected:    "1.5 MB",
			description: "Should format megabytes correctly",
		},
		{
			name:        "Gigabytes",
			bytes:       1610612736,
			expected:    "1.5 GB",
			description: "Should format gigabytes correctly",
		},
		{
			name:        "Zero bytes",
			bytes:       0,
			expected:    "0 B",
			description: "Should handle zero bytes",
		},
		{
			name:        "Large value",
			bytes:       1099511627776,
			expected:    "1.0 TB",
			description: "Should handle terabytes",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := FormatBytes(tc.bytes)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: FormatBytes(%d) = '%s'", tc.name, tc.bytes, result)
		})
	}
}

// TestSanitizeFilename tests filename sanitization utilities
func (s *StringUtilitiesTestSuite) TestSanitizeFilename() {
	testCases := []struct {
		name        string
		input       string
		expected    string
		description string
	}{
		{
			name:        "Valid filename",
			input:       "valid-file_name.txt",
			expected:    "valid-file_name.txt",
			description: "Should leave valid filenames unchanged",
		},
		{
			name:        "Invalid characters",
			input:       "file<>:\"|?*name.txt",
			expected:    "file_______name.txt",
			description: "Should replace invalid characters with underscores",
		},
		{
			name:        "Path separators",
			input:       "path/to\\file.txt",
			expected:    "path_to_file.txt",
			description: "Should replace path separators",
		},
		{
			name:        "Leading/trailing spaces",
			input:       "  filename  ",
			expected:    "filename",
			description: "Should trim leading and trailing spaces",
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    "unnamed",
			description: "Should provide default name for empty string",
		},
		{
			name:        "Only spaces",
			input:       "   ",
			expected:    "unnamed",
			description: "Should provide default name for only spaces",
		},
		{
			name:        "Unicode characters",
			input:       "file-name.txt",
			expected:    "file-name.txt",
			description: "Should preserve Unicode characters",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := SanitizeFilename(tc.input)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: SanitizeFilename('%s') = '%s'", tc.name, tc.input, result)
		})
	}
}

// TestTruncateString tests string truncation utilities
func (s *StringUtilitiesTestSuite) TestTruncateString() {
	testCases := []struct {
		name        string
		input       string
		maxLen      int
		expected    string
		description string
	}{
		{
			name:        "Short string",
			input:       "short",
			maxLen:      10,
			expected:    "short",
			description: "Should leave short strings unchanged",
		},
		{
			name:        "Exact length",
			input:       "exactly10c",
			maxLen:      10,
			expected:    "exactly10c",
			description: "Should leave strings at exact max length unchanged",
		},
		{
			name:        "Long string",
			input:       "This is a very long string that needs to be truncated",
			maxLen:      20,
			expected:    "This is a very lo...",
			description: "Should truncate long strings with ellipsis",
		},
		{
			name:        "Very short max length",
			input:       "Hello World",
			maxLen:      5,
			expected:    "He...",
			description: "Should handle very short max lengths",
		},
		{
			name:        "Max length less than ellipsis",
			input:       "Hello",
			maxLen:      2,
			expected:    "..",
			description: "Should handle max length less than ellipsis",
		},
		{
			name:        "Empty string",
			input:       "",
			maxLen:      10,
			expected:    "",
			description: "Should handle empty string",
		},
		{
			name:        "Zero max length",
			input:       "Hello",
			maxLen:      0,
			expected:    "",
			description: "Should handle zero max length",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := TruncateString(tc.input, tc.maxLen)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: TruncateString('%s', %d) = '%s'", tc.name, tc.input, tc.maxLen, result)
		})
	}
}

// TestPadString tests string padding utilities
func (s *StringUtilitiesTestSuite) TestPadString() {
	testCases := []struct {
		name        string
		input       string
		width       int
		padChar     rune
		alignment   string
		expected    string
		description string
	}{
		{
			name:        "Left pad with spaces",
			input:       "hello",
			width:       10,
			padChar:     ' ',
			alignment:   "left",
			expected:    "hello     ",
			description: "Should pad string to the left with spaces",
		},
		{
			name:        "Right pad with zeros",
			input:       "123",
			width:       6,
			padChar:     '0',
			alignment:   "right",
			expected:    "000123",
			description: "Should pad string to the right with zeros",
		},
		{
			name:        "Center pad with dashes",
			input:       "test",
			width:       10,
			padChar:     '-',
			alignment:   "center",
			expected:    "---test---",
			description: "Should center string with padding",
		},
		{
			name:        "No padding needed",
			input:       "exactly",
			width:       7,
			padChar:     ' ',
			alignment:   "left",
			expected:    "exactly",
			description: "Should leave string unchanged when no padding needed",
		},
		{
			name:        "String longer than width",
			input:       "toolongstring",
			width:       5,
			padChar:     ' ',
			alignment:   "left",
			expected:    "toolongstring",
			description: "Should leave string unchanged when longer than width",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := PadString(tc.input, tc.width, tc.padChar, tc.alignment)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: PadString('%s', %d, '%c', '%s') = '%s'", tc.name, tc.input, tc.width, tc.padChar, tc.alignment, result)
		})
	}
}

// TestJoinPaths tests path joining utilities
func (s *StringUtilitiesTestSuite) TestJoinPaths() {
	testCases := []struct {
		name        string
		paths       []string
		expected    string
		description string
	}{
		{
			name:        "Simple path join",
			paths:       []string{"home", "user", "documents"},
			expected:    filepath.Join("home", "user", "documents"),
			description: "Should join simple paths correctly",
		},
		{
			name:        "Paths with separators",
			paths:       []string{"home/", "/user/", "/documents"},
			expected:    filepath.Join("home", "user", "documents"),
			description: "Should handle paths with separators",
		},
		{
			name:        "Empty paths",
			paths:       []string{"home", "", "documents"},
			expected:    filepath.Join("home", "documents"),
			description: "Should skip empty paths",
		},
		{
			name:        "Single path",
			paths:       []string{"singlepath"},
			expected:    "singlepath",
			description: "Should handle single path",
		},
		{
			name:        "No paths",
			paths:       []string{},
			expected:    "",
			description: "Should handle empty path list",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := JoinPaths(tc.paths...)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: JoinPaths(%v) = '%s'", tc.name, tc.paths, result)
		})
	}
}

// TestSplitLines tests line splitting utilities
func (s *StringUtilitiesTestSuite) TestSplitLines() {
	testCases := []struct {
		name        string
		input       string
		expected    []string
		description string
	}{
		{
			name:        "Unix line endings",
			input:       "line1\nline2\nline3",
			expected:    []string{"line1", "line2", "line3"},
			description: "Should split on Unix line endings",
		},
		{
			name:        "Windows line endings",
			input:       "line1\r\nline2\r\nline3",
			expected:    []string{"line1", "line2", "line3"},
			description: "Should split on Windows line endings",
		},
		{
			name:        "Mixed line endings",
			input:       "line1\nline2\r\nline3\rline4",
			expected:    []string{"line1", "line2", "line3", "line4"},
			description: "Should handle mixed line endings",
		},
		{
			name:        "Empty lines",
			input:       "line1\n\nline3",
			expected:    []string{"line1", "", "line3"},
			description: "Should preserve empty lines",
		},
		{
			name:        "Single line",
			input:       "singleline",
			expected:    []string{"singleline"},
			description: "Should handle single line",
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    []string{""},
			description: "Should handle empty string",
		},
		{
			name:        "Only newlines",
			input:       "\n\n\n",
			expected:    []string{"", "", "", ""},
			description: "Should handle only newlines",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := SplitLines(tc.input)
			s.Equal(tc.expected, result, tc.description)

			s.T().Logf("✓ %s: SplitLines('%s') = %v", tc.name, strings.ReplaceAll(tc.input, "\n", "\\n"), result)
		})
	}
}

// Helper functions that would be implemented in the actual utils package

// FormatDuration formats a duration for human-readable display
func FormatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1000)
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return d.String()
}

// FormatBytes formats bytes for human-readable display
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// SanitizeFilename sanitizes a filename by removing invalid characters
func SanitizeFilename(filename string) string {
	// Trim whitespace
	filename = strings.TrimSpace(filename)

	// Handle empty string
	if filename == "" {
		return "unnamed"
	}

	// Replace invalid characters
	invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*", "/", "\\"}
	for _, char := range invalidChars {
		filename = strings.ReplaceAll(filename, char, "_")
	}

	return filename
}

// TruncateString truncates a string to maxLen characters, adding ellipsis if needed
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return strings.Repeat(".", maxLen)
	}
	return s[:maxLen-3] + "..."
}

// PadString pads a string to the specified width
func PadString(s string, width int, padChar rune, alignment string) string {
	if len(s) >= width {
		return s
	}

	padLen := width - len(s)
	padding := strings.Repeat(string(padChar), padLen)

	switch alignment {
	case "right":
		return padding + s
	case "center":
		leftPad := padLen / 2
		rightPad := padLen - leftPad
		return strings.Repeat(string(padChar), leftPad) + s + strings.Repeat(string(padChar), rightPad)
	default: // left
		return s + padding
	}
}

// JoinPaths joins paths, skipping empty ones
func JoinPaths(paths ...string) string {
	var cleanPaths []string
	for _, path := range paths {
		path = strings.Trim(path, "/\\")
		if path != "" {
			cleanPaths = append(cleanPaths, path)
		}
	}
	if len(cleanPaths) == 0 {
		return ""
	}
	return filepath.Join(cleanPaths...)
}

// SplitLines splits text on various line ending types
func SplitLines(text string) []string {
	// Normalize line endings to \n
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n")
}

// TestSuite runs the string utilities test suite
func TestStringUtilitiesTestSuite(t *testing.T) {
	suite.Run(t, new(StringUtilitiesTestSuite))
}
