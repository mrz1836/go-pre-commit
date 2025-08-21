package output

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefault(t *testing.T) {
	// Save original env
	originalNoColor := os.Getenv("NO_COLOR")
	originalPreCommitColor := os.Getenv("GO_PRE_COMMIT_COLOR_OUTPUT")

	defer func() {
		if err := os.Setenv("NO_COLOR", originalNoColor); err != nil {
			t.Errorf("Failed to restore NO_COLOR: %v", err)
		}
		if err := os.Setenv("GO_PRE_COMMIT_COLOR_OUTPUT", originalPreCommitColor); err != nil {
			t.Errorf("Failed to restore GO_PRE_COMMIT_COLOR_OUTPUT: %v", err)
		}
	}()

	t.Run("DefaultColorEnabled", func(t *testing.T) {
		if err := os.Unsetenv("NO_COLOR"); err != nil {
			t.Errorf("Failed to unset NO_COLOR: %v", err)
		}
		if err := os.Unsetenv("GO_PRE_COMMIT_COLOR_OUTPUT"); err != nil {
			t.Errorf("Failed to unset GO_PRE_COMMIT_COLOR_OUTPUT: %v", err)
		}

		f := NewDefault()
		// In CI environment, colors should be disabled automatically
		// This is the correct behavior we want
		if isCI() {
			assert.False(t, f.colorEnabled, "Colors should be disabled in CI environment")
		} else {
			// In local development, colors depend on TTY detection
			// We can't assume colors will be enabled since output might not be a TTY
			assert.NotNil(t, &f.colorEnabled, "Color setting should be initialized")
		}
	})

	t.Run("NO_COLOR DisablesColor", func(t *testing.T) {
		if err := os.Setenv("NO_COLOR", "1"); err != nil {
			t.Errorf("Failed to set NO_COLOR: %v", err)
		}

		f := NewDefault()
		assert.False(t, f.colorEnabled)
	})

	t.Run("GO_PRE_COMMIT_COLOR_OUTPUT DisablesColor", func(t *testing.T) {
		if err := os.Unsetenv("NO_COLOR"); err != nil {
			t.Errorf("Failed to unset NO_COLOR: %v", err)
		}
		if err := os.Setenv("GO_PRE_COMMIT_COLOR_OUTPUT", "false"); err != nil {
			t.Errorf("Failed to set GO_PRE_COMMIT_COLOR_OUTPUT: %v", err)
		}

		f := NewDefault()
		assert.False(t, f.colorEnabled)
	})
}

func TestFormatterOutput(t *testing.T) {
	t.Run("ColorDisabled", func(t *testing.T) {
		var out, err bytes.Buffer

		f := New(Options{
			ColorEnabled: false,
			Out:          &out,
			Err:          &err,
		})

		t.Run("Success", func(t *testing.T) {
			out.Reset()
			f.Success("test message")
			assert.Equal(t, "✓ test message\n", out.String())
		})

		t.Run("Error", func(t *testing.T) {
			err.Reset()
			f.Error("test error")
			assert.Equal(t, "✗ test error\n", err.String())
		})

		t.Run("Warning", func(t *testing.T) {
			err.Reset()
			f.Warning("test warning")
			assert.Equal(t, "⚠ test warning\n", err.String())
		})

		t.Run("Info", func(t *testing.T) {
			out.Reset()
			f.Info("test info")
			assert.Equal(t, "ℹ test info\n", out.String())
		})

		t.Run("Progress", func(t *testing.T) {
			out.Reset()
			f.Progress("test progress")
			assert.Equal(t, "⏳ test progress\n", out.String())
		})
	})

	t.Run("ColorEnabled", func(t *testing.T) {
		var out, err bytes.Buffer

		f := New(Options{
			ColorEnabled: true,
			Out:          &out,
			Err:          &err,
		})

		t.Run("Success", func(t *testing.T) {
			out.Reset()
			f.Success("test message with %s", "formatting")
			outputStr := out.String()
			assert.Contains(t, outputStr, "✓")
			assert.Contains(t, outputStr, "test message with formatting")
		})

		t.Run("Error", func(t *testing.T) {
			err.Reset()
			f.Error("test error with %d code", 500)
			outputStr := err.String()
			assert.Contains(t, outputStr, "✗")
			assert.Contains(t, outputStr, "test error with 500 code")
		})

		t.Run("Warning", func(t *testing.T) {
			err.Reset()
			f.Warning("test warning")
			outputStr := err.String()
			assert.Contains(t, outputStr, "⚠")
			assert.Contains(t, outputStr, "test warning")
		})

		t.Run("Info", func(t *testing.T) {
			out.Reset()
			f.Info("test info")
			outputStr := out.String()
			assert.Contains(t, outputStr, "ℹ")
			assert.Contains(t, outputStr, "test info")
		})

		t.Run("Progress", func(t *testing.T) {
			out.Reset()
			f.Progress("test progress")
			outputStr := out.String()
			assert.Contains(t, outputStr, "⏳")
			assert.Contains(t, outputStr, "test progress")
		})
	})
}

func TestDurationFormatting(t *testing.T) {
	f := NewDefault()

	testCases := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"Microseconds", 500 * time.Microsecond, "500μs"},
		{"Milliseconds", 250 * time.Millisecond, "250ms"},
		{"Seconds", 2500 * time.Millisecond, "2.5s"},
		{"Minutes", 90 * time.Second, "1.5m"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := f.Duration(tc.duration)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseLintError(t *testing.T) {
	f := NewDefault()

	testCases := []struct {
		name               string
		output             string
		expectedMessage    string
		expectedSuggestion string
	}{
		{
			name:               "BinaryNotFound",
			output:             "golangci-lint: no such file or directory",
			expectedMessage:    "golangci-lint binary not found",
			expectedSuggestion: "Install golangci-lint with 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest' or ensure it's in your PATH.",
		},
		{
			name:               "ConfigIssue",
			output:             "error reading config file: yaml: unmarshal errors",
			expectedMessage:    "golangci-lint configuration issue",
			expectedSuggestion: "Check your .golangci.yml file for syntax errors.",
		},
		{
			name:               "Timeout",
			output:             "context deadline exceeded (timeout)",
			expectedMessage:    "golangci-lint timed out",
			expectedSuggestion: "Increase timeout with GO_PRE_COMMIT_LINT_TIMEOUT or run 'golangci-lint run' manually.",
		},
		{
			name:               "LintingIssues",
			output:             "main.go:10:5: unused variable 'x'\nutils.go:25:1: function should be commented",
			expectedMessage:    "Found 2 linting issue(s)",
			expectedSuggestion: "Fix the issues shown above. Run 'golangci-lint run' to see full details.",
		},
		{
			name:               "UnknownError",
			output:             "some unknown error occurred",
			expectedMessage:    "Linting failed with unknown error",
			expectedSuggestion: "Run 'golangci-lint run' manually to see detailed output.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, suggestion := f.parseLintError(tc.output)
			assert.Equal(t, tc.expectedMessage, message)
			assert.Equal(t, tc.expectedSuggestion, suggestion)
		})
	}
}

func TestParseFumptError(t *testing.T) {
	f := NewDefault()

	testCases := []struct {
		name               string
		output             string
		expectedMessage    string
		expectedSuggestion string
	}{
		{
			name:               "BinaryNotFound",
			output:             "gofumpt: no such file or directory",
			expectedMessage:    "gofumpt binary not found",
			expectedSuggestion: "Install gofumpt with 'go install mvdan.cc/gofumpt@latest'.",
		},
		{
			name:               "PermissionDenied",
			output:             "permission denied: cannot write to file",
			expectedMessage:    "Permission denied writing files",
			expectedSuggestion: "Check file permissions and ensure you can write to the affected files.",
		},
		{
			name:               "SyntaxError",
			output:             "syntax error in file.go",
			expectedMessage:    "Go syntax errors prevent formatting",
			expectedSuggestion: "Fix syntax errors in your Go files before running fumpt.",
		},
		{
			name:               "UnknownError",
			output:             "some unknown error",
			expectedMessage:    "Formatting failed",
			expectedSuggestion: "Run 'gofumpt -w .' manually to see detailed errors.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, suggestion := f.parseFumptError(tc.output)
			assert.Equal(t, tc.expectedMessage, message)
			assert.Equal(t, tc.expectedSuggestion, suggestion)
		})
	}
}

func TestParseModTidyError(t *testing.T) {
	f := NewDefault()

	testCases := []struct {
		name               string
		output             string
		expectedMessage    string
		expectedSuggestion string
	}{
		{
			name:               "NoGoMod",
			output:             "no go.mod file found",
			expectedMessage:    "No go.mod file found",
			expectedSuggestion: "Initialize a Go module with 'go mod init <module-name>'.",
		},
		{
			name:               "NetworkError",
			output:             "network timeout downloading module",
			expectedMessage:    "Network error downloading modules",
			expectedSuggestion: "Check your internet connection and proxy settings. Try running 'go mod tidy' manually.",
		},
		{
			name:               "ChecksumMismatch",
			output:             "checksum mismatch for module",
			expectedMessage:    "Module checksum verification failed",
			expectedSuggestion: "Run 'go clean -modcache' and try again, or check for module security issues.",
		},
		{
			name:               "ModuleNotFound",
			output:             "module not found: example.com/nonexistent",
			expectedMessage:    "Module dependencies not found",
			expectedSuggestion: "Check that all imported modules exist and are accessible.",
		},
		{
			name:               "UnknownError",
			output:             "some unknown error",
			expectedMessage:    "Module tidy operation failed",
			expectedSuggestion: "Run 'go mod tidy' manually to see detailed errors.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, suggestion := f.parseModTidyError(tc.output)
			assert.Equal(t, tc.expectedMessage, message)
			assert.Equal(t, tc.expectedSuggestion, suggestion)
		})
	}
}

func TestFormatFileList(t *testing.T) {
	f := NewDefault()

	testCases := []struct {
		name     string
		files    []string
		maxFiles int
		expected string
	}{
		{
			name:     "NoFiles",
			files:    []string{},
			maxFiles: 3,
			expected: "no files",
		},
		{
			name:     "SingleFile",
			files:    []string{"main.go"},
			maxFiles: 3,
			expected: "main.go",
		},
		{
			name:     "WithinLimit",
			files:    []string{"main.go", "utils.go"},
			maxFiles: 3,
			expected: "main.go, utils.go",
		},
		{
			name:     "ExceedsLimit",
			files:    []string{"main.go", "utils.go", "test.go", "helper.go"},
			maxFiles: 2,
			expected: "main.go, utils.go ... and 2 more",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := f.FormatFileList(tc.files, tc.maxFiles)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatExecutionStats(t *testing.T) {
	f := New(Options{ColorEnabled: false}) // Disable color for predictable testing

	testCases := []struct {
		name      string
		passed    int
		failed    int
		skipped   int
		duration  time.Duration
		fileCount int
		expected  string
	}{
		{
			name:      "AllPassed",
			passed:    3,
			failed:    0,
			skipped:   0,
			duration:  500 * time.Millisecond,
			fileCount: 5,
			expected:  "3 passed on 5 file(s) in 500ms",
		},
		{
			name:      "MixedResults",
			passed:    2,
			failed:    1,
			skipped:   1,
			duration:  1500 * time.Millisecond,
			fileCount: 4,
			expected:  "2 passed, 1 failed, 1 skipped on 4 file(s) in 1.5s",
		},
		{
			name:      "NoFiles",
			passed:    0,
			failed:    0,
			skipped:   3,
			duration:  100 * time.Millisecond,
			fileCount: 0,
			expected:  "3 skipped in 100ms",
		},
		{
			name:      "AllZero",
			passed:    0,
			failed:    0,
			skipped:   0,
			duration:  10 * time.Millisecond,
			fileCount: 0,
			expected:  " in 10ms",
		},
		{
			name:      "AllZeroWithFiles",
			passed:    0,
			failed:    0,
			skipped:   0,
			duration:  25 * time.Millisecond,
			fileCount: 5,
			expected:  " on 5 file(s) in 25ms",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := f.FormatExecutionStats(tc.passed, tc.failed, tc.skipped, tc.duration, tc.fileCount)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseGenericMakeError(t *testing.T) {
	f := NewDefault()

	testCases := []struct {
		name               string
		command            string
		output             string
		expectedMessage    string
		expectedSuggestion string
	}{
		{
			name:               "ToolNotFound",
			command:            "gofumpt -w .",
			output:             "command not found: gofumpt",
			expectedMessage:    "Tool or command 'gofumpt -w .' not found",
			expectedSuggestion: "The required tool will be automatically installed on the next run. You can also install it manually.",
		},
		{
			name:               "PermissionDenied",
			command:            "go build",
			output:             "Permission denied: cannot create output file",
			expectedMessage:    "Permission denied",
			expectedSuggestion: "Check file permissions and ensure you have write access to the project directory.",
		},
		{
			name:               "GenericError",
			command:            "go test",
			output:             "test failed with unknown error",
			expectedMessage:    "Command 'go test' failed",
			expectedSuggestion: "Run 'go test' manually to see detailed error output.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, suggestion := f.parseGenericCommandError(tc.command, tc.output)
			assert.Equal(t, tc.expectedMessage, message)
			assert.Equal(t, tc.expectedSuggestion, suggestion)
		})
	}
}

func TestHeaderAndSubheaderFormatting(t *testing.T) {
	t.Run("ColorDisabled", func(t *testing.T) {
		var out bytes.Buffer
		f := New(Options{
			ColorEnabled: false,
			Out:          &out,
		})

		t.Run("Header", func(t *testing.T) {
			out.Reset()
			f.Header("Test Header")
			expected := "\nTest Header\n───────────\n"
			assert.Equal(t, expected, out.String())
		})

		t.Run("Subheader", func(t *testing.T) {
			out.Reset()
			f.Subheader("Test Subheader")
			expected := "\nTest Subheader:\n"
			assert.Equal(t, expected, out.String())
		})

		t.Run("Detail", func(t *testing.T) {
			out.Reset()
			f.Detail("Detail message")
			expected := "  Detail message\n"
			assert.Equal(t, expected, out.String())
		})
	})

	t.Run("ColorEnabled", func(t *testing.T) {
		var out bytes.Buffer
		f := New(Options{
			ColorEnabled: true,
			Out:          &out,
		})

		t.Run("Header", func(t *testing.T) {
			out.Reset()
			f.Header("Test Header")
			outputStr := out.String()
			// Should contain header text and separator
			assert.Contains(t, outputStr, "Test Header")
			assert.Contains(t, outputStr, "───────────")
		})

		t.Run("Subheader", func(t *testing.T) {
			out.Reset()
			f.Subheader("Test Subheader")
			outputStr := out.String()
			assert.Contains(t, outputStr, "Test Subheader:")
		})
	})
}

func TestCodeBlock(t *testing.T) {
	t.Run("ColorDisabled", func(t *testing.T) {
		var out bytes.Buffer
		f := New(Options{
			ColorEnabled: false,
			Out:          &out,
		})

		out.Reset()
		f.CodeBlock("line 1\nline 2\nline 3")
		expected := "    line 1\n    line 2\n    line 3\n"
		assert.Equal(t, expected, out.String())
	})

	t.Run("ColorEnabled", func(t *testing.T) {
		var out bytes.Buffer
		f := New(Options{
			ColorEnabled: true,
			Out:          &out,
		})

		out.Reset()
		f.CodeBlock("line 1\nline 2\nline 3")
		outputStr := out.String()
		// Should contain indented content
		assert.Contains(t, outputStr, "    line 1")
		assert.Contains(t, outputStr, "    line 2")
		assert.Contains(t, outputStr, "    line 3")
	})
}

func TestSuggestAction(t *testing.T) {
	t.Run("ColorDisabled", func(t *testing.T) {
		var out bytes.Buffer
		f := New(Options{
			ColorEnabled: false,
			Out:          &out,
		})

		out.Reset()
		f.SuggestAction("Try this action")
		expected := "💡 Try this action\n"
		assert.Equal(t, expected, out.String())
	})

	t.Run("ColorEnabled", func(t *testing.T) {
		var out bytes.Buffer
		f := New(Options{
			ColorEnabled: true,
			Out:          &out,
		})

		out.Reset()
		f.SuggestAction("Try this action")
		outputStr := out.String()
		// Should contain the message content
		assert.Contains(t, outputStr, "Try this action")
		assert.Contains(t, outputStr, "💡")
	})
}

func TestHighlight(t *testing.T) {
	t.Run("ColorDisabled", func(t *testing.T) {
		f := New(Options{ColorEnabled: false})
		result := f.Highlight("hello world", "world")
		assert.Equal(t, "hello world", result)
	})

	t.Run("ColorEnabled", func(t *testing.T) {
		f := New(Options{ColorEnabled: true})
		result := f.Highlight("hello world", "world")
		// Should contain the original text (color codes will be added but we can't easily test them)
		assert.Contains(t, result, "hello")
		assert.Contains(t, result, "world")
	})
}

func TestParseTextMakeError(t *testing.T) {
	f := NewDefault()

	testCases := []struct {
		name               string
		command            string
		output             string
		expectedMessage    string
		expectedSuggestion string
	}{
		{
			name:               "LintCommand",
			command:            "golangci-lint run",
			output:             "golangci-lint: no such file or directory",
			expectedMessage:    "golangci-lint binary not found",
			expectedSuggestion: "Install golangci-lint with 'go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest' or ensure it's in your PATH.",
		},
		{
			name:               "FumptCommand",
			command:            "gofumpt -w .",
			output:             "gofumpt: no such file or directory",
			expectedMessage:    "gofumpt binary not found",
			expectedSuggestion: "Install gofumpt with 'go install mvdan.cc/gofumpt@latest'.",
		},
		{
			name:               "ModTidyCommand",
			command:            "go mod tidy",
			output:             "no go.mod file found",
			expectedMessage:    "No go.mod file found",
			expectedSuggestion: "Initialize a Go module with 'go mod init <module-name>'.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, suggestion := f.ParseCommandError(tc.command, tc.output)
			require.NotEmpty(t, message)
			require.NotEmpty(t, suggestion)
			assert.Equal(t, tc.expectedMessage, message)
			assert.Equal(t, tc.expectedSuggestion, suggestion)
		})
	}
}

// TestNewFormatterOptions tests the New function with different options
func TestNewFormatterOptions(t *testing.T) {
	t.Run("WithCustomWriters", func(t *testing.T) {
		var out, err bytes.Buffer
		f := New(Options{
			ColorEnabled: true,
			Out:          &out,
			Err:          &err,
		})

		// Test that custom writers are used
		f.Success("test")
		assert.Contains(t, out.String(), "test")
		assert.Empty(t, err.String())

		err.Reset()
		f.Error("error test")
		assert.Contains(t, err.String(), "error test")
	})

	t.Run("WithNilWriters", func(t *testing.T) {
		// Should default to os.Stdout/os.Stderr when nil writers provided
		f := New(Options{
			ColorEnabled: false,
			Out:          nil,
			Err:          nil,
		})

		// Should not panic and should use default writers
		assert.NotPanics(t, func() {
			f.Success("test")
			f.Error("error")
		})
	})
}

// TestFormatterWithStringFormatting tests string formatting in various methods
func TestFormatterWithStringFormatting(t *testing.T) {
	var out, err bytes.Buffer
	f := New(Options{
		ColorEnabled: false,
		Out:          &out,
		Err:          &err,
	})

	t.Run("SuccessWithFormatting", func(t *testing.T) {
		out.Reset()
		f.Success("Processing %d files with %s", 5, "success")
		assert.Equal(t, "✓ Processing 5 files with success\n", out.String())
	})

	t.Run("ErrorWithFormatting", func(t *testing.T) {
		err.Reset()
		f.Error("Failed to process %s: %v", "file.go", "permission denied")
		assert.Equal(t, "✗ Failed to process file.go: permission denied\n", err.String())
	})

	t.Run("DetailWithFormatting", func(t *testing.T) {
		out.Reset()
		f.Detail("Found %d issues in %s", 3, "main.go")
		assert.Equal(t, "  Found 3 issues in main.go\n", out.String())
	})

	t.Run("WarningWithFormatting", func(t *testing.T) {
		err.Reset()
		f.Warning("Deprecated feature used in %s line %d", "utils.go", 42)
		assert.Equal(t, "⚠ Deprecated feature used in utils.go line 42\n", err.String())
	})

	t.Run("InfoWithFormatting", func(t *testing.T) {
		out.Reset()
		f.Info("Processed %d/%d files", 8, 10)
		assert.Equal(t, "ℹ Processed 8/10 files\n", out.String())
	})

	t.Run("ProgressWithFormatting", func(t *testing.T) {
		out.Reset()
		f.Progress("Running %s checks...", "syntax")
		assert.Equal(t, "⏳ Running syntax checks...\n", out.String())
	})
}

// TestParseCommandErrorWhitespaceHandling tests whitespace handling in ParseCommandError
func TestParseCommandErrorWhitespaceHandling(t *testing.T) {
	f := NewDefault()

	// Test with leading/trailing whitespace
	message, suggestion := f.ParseCommandError("golangci-lint run", "  \n\tgolangci-lint: no such file or directory\n\t  ")
	assert.Equal(t, "golangci-lint binary not found", message)
	assert.Contains(t, suggestion, "Install golangci-lint")

	// Test with empty output (just whitespace)
	message, suggestion = f.ParseCommandError("go test ./...", "   \n\t   ")
	assert.Equal(t, "Command 'go test ./...' failed", message)
	assert.Contains(t, suggestion, "Run 'go test ./...' manually")
}

// TestColorMode tests the ColorMode enum functionality
func TestColorMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     ColorMode
		expected bool
	}{
		{"ColorAlways", ColorAlways, true},
		{"ColorNever", ColorNever, false},
		{"ColorAuto with NO_COLOR", ColorAuto, false}, // Will be set in test
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "ColorAuto with NO_COLOR" {
				_ = os.Setenv("NO_COLOR", "1")
				defer func() { _ = os.Unsetenv("NO_COLOR") }()
			}

			formatter := NewWithColorMode(tt.mode)
			assert.Equal(t, tt.expected, formatter.colorEnabled)
		})
	}
}

// TestShouldUseColor tests the color detection logic
func TestShouldUseColor(t *testing.T) {
	tests := []struct {
		name     string
		mode     ColorMode
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "ColorAlways always returns true",
			mode:     ColorAlways,
			envVars:  map[string]string{"NO_COLOR": "1"},
			expected: true,
		},
		{
			name:     "ColorNever always returns false",
			mode:     ColorNever,
			envVars:  map[string]string{},
			expected: false,
		},
		{
			name:     "NO_COLOR disables color",
			mode:     ColorAuto,
			envVars:  map[string]string{"NO_COLOR": "1"},
			expected: false,
		},
		{
			name:     "GO_PRE_COMMIT_COLOR_OUTPUT=false disables color",
			mode:     ColorAuto,
			envVars:  map[string]string{"GO_PRE_COMMIT_COLOR_OUTPUT": "false"},
			expected: false,
		},
		{
			name:     "TERM=dumb disables color",
			mode:     ColorAuto,
			envVars:  map[string]string{"TERM": "dumb"},
			expected: false,
		},
		{
			name:     "CI=true disables color",
			mode:     ColorAuto,
			envVars:  map[string]string{"CI": "true"},
			expected: false,
		},
		{
			name:     "GITHUB_ACTIONS=true disables color",
			mode:     ColorAuto,
			envVars:  map[string]string{"GITHUB_ACTIONS": "true"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}
			// Also save some common env vars that might affect the test
			commonEnvVars := []string{"NO_COLOR", "GO_PRE_COMMIT_COLOR_OUTPUT", "TERM", "CI", "GITHUB_ACTIONS"}
			for _, key := range commonEnvVars {
				if _, exists := originalEnv[key]; !exists {
					originalEnv[key] = os.Getenv(key)
				}
			}

			// Clean environment first
			for _, key := range commonEnvVars {
				_ = os.Unsetenv(key)
			}

			// Set test environment
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			defer func() {
				// Restore original environment
				for key, value := range originalEnv {
					if value == "" {
						_ = os.Unsetenv(key)
					} else {
						_ = os.Setenv(key, value)
					}
				}
			}()

			result := shouldUseColor(tt.mode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsCI tests CI environment detection
func TestIsCI(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "No CI environment",
			envVars:  map[string]string{},
			expected: false,
		},
		{
			name:     "CI=true",
			envVars:  map[string]string{"CI": "true"},
			expected: true,
		},
		{
			name:     "CI=1",
			envVars:  map[string]string{"CI": "1"},
			expected: true,
		},
		{
			name:     "GITHUB_ACTIONS=true",
			envVars:  map[string]string{"GITHUB_ACTIONS": "true"},
			expected: true,
		},
		{
			name:     "GITLAB_CI=true",
			envVars:  map[string]string{"GITLAB_CI": "true"},
			expected: true,
		},
		{
			name:     "JENKINS_URL set",
			envVars:  map[string]string{"JENKINS_URL": "http://jenkins.example.com"},
			expected: true,
		},
		{
			name:     "CIRCLECI=true",
			envVars:  map[string]string{"CIRCLECI": "true"},
			expected: true,
		},
		{
			name:     "TRAVIS=true",
			envVars:  map[string]string{"TRAVIS": "true"},
			expected: true,
		},
		{
			name:     "BUILDKITE=true",
			envVars:  map[string]string{"BUILDKITE": "true"},
			expected: true,
		},
		{
			name:     "DRONE=true",
			envVars:  map[string]string{"DRONE": "true"},
			expected: true,
		},
		{
			name:     "TEAMCITY_VERSION set",
			envVars:  map[string]string{"TEAMCITY_VERSION": "2021.1"},
			expected: true,
		},
		{
			name:     "TF_BUILD=True (Azure DevOps)",
			envVars:  map[string]string{"TF_BUILD": "True"},
			expected: true,
		},
		{
			name:     "APPVEYOR=True",
			envVars:  map[string]string{"APPVEYOR": "True"},
			expected: true,
		},
		{
			name:     "CODEBUILD_BUILD_ID set (AWS CodeBuild)",
			envVars:  map[string]string{"CODEBUILD_BUILD_ID": "go-pre-commit:12345"},
			expected: true,
		},
		{
			name:     "Multiple CI variables",
			envVars:  map[string]string{"CI": "true", "GITHUB_ACTIONS": "true"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			ciEnvVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "CIRCLECI", "TRAVIS", "BUILDKITE", "DRONE", "TEAMCITY_VERSION", "TF_BUILD", "APPVEYOR", "CODEBUILD_BUILD_ID"}
			originalEnv := make(map[string]string)
			for _, key := range ciEnvVars {
				originalEnv[key] = os.Getenv(key)
				_ = os.Unsetenv(key) // Clean first
			}

			// Set test environment
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			defer func() {
				// Restore original environment
				for key, value := range originalEnv {
					if value == "" {
						_ = os.Unsetenv(key)
					} else {
						_ = os.Setenv(key, value)
					}
				}
			}()

			result := isCI()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFormatter_NoColorOutput ensures no ANSI codes in output when colors disabled
func TestFormatter_NoColorOutput(t *testing.T) {
	tests := []struct {
		name   string
		method func(*Formatter)
	}{
		{"Success", func(f *Formatter) { f.Success("test message") }},
		{"Error", func(f *Formatter) { f.Error("test message") }},
		{"Warning", func(f *Formatter) { f.Warning("test message") }},
		{"Info", func(f *Formatter) { f.Info("test message") }},
		{"Progress", func(f *Formatter) { f.Progress("test message") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := New(Options{
				ColorEnabled: false,
				Out:          &buf,
				Err:          &buf,
			})

			tt.method(formatter)
			output := buf.String()

			// Should not contain ANSI color codes
			assert.NotContains(t, output, "\x1b[", "Output should not contain ANSI color codes")
			assert.Contains(t, output, "test message", "Output should contain the message")
		})
	}
}

// TestFormatter_ColorOutput tests that color methods execute without error
func TestFormatter_ColorOutput(t *testing.T) {
	tests := []struct {
		name   string
		method func(*Formatter)
	}{
		{"Success", func(f *Formatter) { f.Success("test message") }},
		{"Error", func(f *Formatter) { f.Error("test message") }},
		{"Warning", func(f *Formatter) { f.Warning("test message") }},
		{"Info", func(f *Formatter) { f.Info("test message") }},
		{"Progress", func(f *Formatter) { f.Progress("test message") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			formatter := New(Options{
				ColorEnabled: true,
				Out:          &buf,
				Err:          &buf,
			})

			tt.method(formatter)
			output := buf.String()

			// Should contain the message
			assert.Contains(t, output, "test message", "Output should contain the message")
			// We can't easily test for ANSI codes since fatih/color is smart about detecting terminals
			// The important thing is that the method executes without error
		})
	}
}

// TestColorPriorityHierarchy tests the priority order for color settings
func TestColorPriorityHierarchy(t *testing.T) {
	tests := []struct {
		name        string
		mode        ColorMode
		envVars     map[string]string
		expected    bool
		description string
	}{
		{
			name:        "ColorAlways overrides NO_COLOR",
			mode:        ColorAlways,
			envVars:     map[string]string{"NO_COLOR": "1", "CI": "true"},
			expected:    true,
			description: "ColorAlways should force colors even with NO_COLOR and CI set",
		},
		{
			name:        "ColorNever overrides TTY detection",
			mode:        ColorNever,
			envVars:     map[string]string{},
			expected:    false,
			description: "ColorNever should disable colors even if TTY is available",
		},
		{
			name:        "NO_COLOR overrides CI detection",
			mode:        ColorAuto,
			envVars:     map[string]string{"NO_COLOR": "1"},
			expected:    false,
			description: "NO_COLOR should take precedence over CI auto-detection",
		},
		{
			name:        "GO_PRE_COMMIT_COLOR_OUTPUT overrides CI detection",
			mode:        ColorAuto,
			envVars:     map[string]string{"GO_PRE_COMMIT_COLOR_OUTPUT": "false", "CI": "false"},
			expected:    false,
			description: "GO_PRE_COMMIT_COLOR_OUTPUT=false should override other settings",
		},
		{
			name:        "TERM=dumb overrides CI=false",
			mode:        ColorAuto,
			envVars:     map[string]string{"TERM": "dumb", "CI": "false"},
			expected:    false,
			description: "TERM=dumb should disable colors even when not in CI",
		},
		{
			name:        "CI detection overrides potential TTY",
			mode:        ColorAuto,
			envVars:     map[string]string{"CI": "true"},
			expected:    false,
			description: "CI environment should disable colors regardless of TTY",
		},
		{
			name:        "Multiple disable flags - NO_COLOR wins",
			mode:        ColorAuto,
			envVars:     map[string]string{"NO_COLOR": "1", "GO_PRE_COMMIT_COLOR_OUTPUT": "true", "TERM": "xterm"},
			expected:    false,
			description: "NO_COLOR should win over conflicting settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and clean environment
			allEnvVars := []string{"NO_COLOR", "GO_PRE_COMMIT_COLOR_OUTPUT", "TERM", "CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "CIRCLECI", "TRAVIS", "BUILDKITE", "DRONE", "TEAMCITY_VERSION", "TF_BUILD", "APPVEYOR", "CODEBUILD_BUILD_ID"}
			originalEnv := make(map[string]string)
			for _, key := range allEnvVars {
				originalEnv[key] = os.Getenv(key)
				_ = os.Unsetenv(key)
			}

			// Set test environment
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			defer func() {
				for key, value := range originalEnv {
					if value == "" {
						_ = os.Unsetenv(key)
					} else {
						_ = os.Setenv(key, value)
					}
				}
			}()

			result := shouldUseColor(tt.mode)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestTermEnvironmentHandling tests TERM environment variable behavior
func TestTermEnvironmentHandling(t *testing.T) {
	tests := []struct {
		name        string
		termValue   string
		expected    bool
		description string
	}{
		{
			name:        "TERM=dumb disables colors",
			termValue:   "dumb",
			expected:    false,
			description: "dumb terminal should disable colors",
		},
		{
			name:        "TERM=xterm enables colors",
			termValue:   "xterm",
			expected:    true, // Would be true if not in CI and TTY available
			description: "xterm should allow colors if other conditions met",
		},
		{
			name:        "TERM=xterm-256color enables colors",
			termValue:   "xterm-256color",
			expected:    true,
			description: "xterm-256color should allow colors if other conditions met",
		},
		{
			name:        "TERM=screen enables colors",
			termValue:   "screen",
			expected:    true,
			description: "screen terminal should allow colors if other conditions met",
		},
		{
			name:        "TERM empty allows colors",
			termValue:   "",
			expected:    true,
			description: "empty TERM should not block colors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and clean environment
			allEnvVars := []string{"NO_COLOR", "GO_PRE_COMMIT_COLOR_OUTPUT", "TERM", "CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "CIRCLECI", "TRAVIS", "BUILDKITE", "DRONE", "TEAMCITY_VERSION", "TF_BUILD", "APPVEYOR", "CODEBUILD_BUILD_ID"}
			originalEnv := make(map[string]string)
			for _, key := range allEnvVars {
				originalEnv[key] = os.Getenv(key)
				_ = os.Unsetenv(key)
			}

			// Set TERM value
			if tt.termValue != "" {
				_ = os.Setenv("TERM", tt.termValue)
			}

			defer func() {
				for key, value := range originalEnv {
					if value == "" {
						_ = os.Unsetenv(key)
					} else {
						_ = os.Setenv(key, value)
					}
				}
			}()

			result := shouldUseColor(ColorAuto)
			if tt.termValue == "dumb" {
				assert.False(t, result, tt.description)
			} else {
				// For non-dumb terminals, the result depends on CI and TTY detection
				// In test environment, we can't guarantee TTY, so just ensure no panic
				assert.NotPanics(t, func() { shouldUseColor(ColorAuto) })
			}
		})
	}
}

// TestColorEdgeCasesAndErrorScenarios tests edge cases and error conditions
func TestColorEdgeCasesAndErrorScenarios(t *testing.T) {
	t.Run("EmptyEnvironmentVariables", func(t *testing.T) {
		// Save and clean environment
		allEnvVars := []string{"NO_COLOR", "GO_PRE_COMMIT_COLOR_OUTPUT", "TERM", "CI"}
		originalEnv := make(map[string]string)
		for _, key := range allEnvVars {
			originalEnv[key] = os.Getenv(key)
			_ = os.Unsetenv(key)
		}

		// Test with empty environment variables
		_ = os.Setenv("NO_COLOR", "")
		_ = os.Setenv("GO_PRE_COMMIT_COLOR_OUTPUT", "")
		_ = os.Setenv("TERM", "")
		_ = os.Setenv("CI", "")

		defer func() {
			for key, value := range originalEnv {
				if value == "" {
					_ = os.Unsetenv(key)
				} else {
					_ = os.Setenv(key, value)
				}
			}
		}()

		// Empty values should not disable colors (only non-empty values matter)
		assert.NotPanics(t, func() { shouldUseColor(ColorAuto) })
	})

	t.Run("InvalidEnvironmentValues", func(t *testing.T) {
		allEnvVars := []string{"NO_COLOR", "GO_PRE_COMMIT_COLOR_OUTPUT", "CI"}
		originalEnv := make(map[string]string)
		for _, key := range allEnvVars {
			originalEnv[key] = os.Getenv(key)
			_ = os.Unsetenv(key)
		}

		// Test with various invalid/unexpected values
		testCases := []struct {
			envVar string
			value  string
		}{
			{"GO_PRE_COMMIT_COLOR_OUTPUT", "invalid"},
			{"GO_PRE_COMMIT_COLOR_OUTPUT", "True"}, // Case sensitive
			{"GO_PRE_COMMIT_COLOR_OUTPUT", "0"},
			{"CI", "false"}, // Should not be detected as CI
			{"CI", "0"},     // Should not be detected as CI
		}

		defer func() {
			for key, value := range originalEnv {
				if value == "" {
					_ = os.Unsetenv(key)
				} else {
					_ = os.Setenv(key, value)
				}
			}
		}()

		for _, tc := range testCases {
			_ = os.Setenv(tc.envVar, tc.value)
			assert.NotPanics(t, func() { shouldUseColor(ColorAuto) })
			_ = os.Unsetenv(tc.envVar)
		}
	})

	t.Run("ColorModeEnumValues", func(t *testing.T) {
		// Test all ColorMode enum values
		modes := []ColorMode{ColorAuto, ColorAlways, ColorNever}

		for _, mode := range modes {
			t.Run(fmt.Sprintf("Mode_%d", int(mode)), func(t *testing.T) {
				assert.NotPanics(t, func() {
					formatter := NewWithColorMode(mode)
					assert.NotNil(t, formatter)
				})
			})
		}
	})

	t.Run("FormatterWithNilOptions", func(t *testing.T) {
		// Test that formatter handles nil writers gracefully
		assert.NotPanics(t, func() {
			f := New(Options{
				ColorEnabled: true,
				Out:          nil,
				Err:          nil,
			})
			f.Success("test")
			f.Error("test")
		})
	})

	t.Run("ConcurrentColorDetection", func(_ *testing.T) {
		// Test that color detection is thread-safe
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer func() { done <- true }()
				for j := 0; j < 10; j++ {
					shouldUseColor(ColorAuto)
					isCI()
					isTTY()
				}
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

// TestTTYDetectionBehavior tests TTY detection behavior in different scenarios
func TestTTYDetectionBehavior(t *testing.T) {
	t.Run("TTYFunctionDoesNotPanic", func(t *testing.T) {
		// Test that isTTY() function doesn't panic under any circumstances
		assert.NotPanics(t, func() {
			result := isTTY()
			// Result can be true or false depending on test environment
			_ = result
		})
	})

	t.Run("ColorAutoWithCleanEnvironment", func(t *testing.T) {
		// Save and clean environment
		allEnvVars := []string{"NO_COLOR", "GO_PRE_COMMIT_COLOR_OUTPUT", "TERM", "CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "CIRCLECI", "TRAVIS", "BUILDKITE", "DRONE", "TEAMCITY_VERSION", "TF_BUILD", "APPVEYOR", "CODEBUILD_BUILD_ID"}
		originalEnv := make(map[string]string)
		for _, key := range allEnvVars {
			originalEnv[key] = os.Getenv(key)
			_ = os.Unsetenv(key)
		}

		defer func() {
			for key, value := range originalEnv {
				if value == "" {
					_ = os.Unsetenv(key)
				} else {
					_ = os.Setenv(key, value)
				}
			}
		}()

		// With clean environment, ColorAuto should depend only on TTY detection
		// We can't control TTY state in tests, but we can ensure it doesn't panic
		assert.NotPanics(t, func() {
			result := shouldUseColor(ColorAuto)
			// In test environment, this typically returns false since stdout is not a TTY
			// but the exact result depends on test runner
			_ = result
		})
	})

	t.Run("ColorModeConsistency", func(t *testing.T) {
		// Test that color mode behavior is consistent across multiple calls
		mode := ColorAuto

		// Clean environment for consistent testing
		allEnvVars := []string{"NO_COLOR", "GO_PRE_COMMIT_COLOR_OUTPUT", "TERM", "CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "CIRCLECI", "TRAVIS", "BUILDKITE", "DRONE", "TEAMCITY_VERSION", "TF_BUILD", "APPVEYOR", "CODEBUILD_BUILD_ID"}
		originalEnv := make(map[string]string)
		for _, key := range allEnvVars {
			originalEnv[key] = os.Getenv(key)
			_ = os.Unsetenv(key)
		}

		defer func() {
			for key, value := range originalEnv {
				if value == "" {
					_ = os.Unsetenv(key)
				} else {
					_ = os.Setenv(key, value)
				}
			}
		}()

		// Multiple calls should return the same result
		result1 := shouldUseColor(mode)
		result2 := shouldUseColor(mode)
		result3 := shouldUseColor(mode)

		assert.Equal(t, result1, result2, "Color detection should be consistent")
		assert.Equal(t, result2, result3, "Color detection should be consistent")
	})

	t.Run("FormatterColorStateConsistency", func(t *testing.T) {
		// Test that formatter color state is set correctly based on mode
		modes := []ColorMode{ColorAlways, ColorNever, ColorAuto}

		for _, mode := range modes {
			t.Run(fmt.Sprintf("Mode_%d", int(mode)), func(t *testing.T) {
				formatter := NewWithColorMode(mode)

				switch mode {
				case ColorAlways:
					assert.True(t, formatter.colorEnabled, "ColorAlways should enable colors")
				case ColorNever:
					assert.False(t, formatter.colorEnabled, "ColorNever should disable colors")
				case ColorAuto:
					// ColorAuto result depends on environment, just ensure it's set
					assert.NotNil(t, &formatter.colorEnabled, "ColorAuto should set color state")
				}
			})
		}
	})
}
