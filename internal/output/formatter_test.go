package output

import (
	"bytes"
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
		assert.True(t, f.colorEnabled)
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
			expectedSuggestion: "Install golangci-lint or ensure it's in your PATH. Run 'make install-lint' if available.",
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
			expectedSuggestion: "Fix the issues shown above. Run 'make lint' or 'golangci-lint run' to see full details.",
		},
		{
			name:               "UnknownError",
			output:             "some unknown error occurred",
			expectedMessage:    "Linting failed with unknown error",
			expectedSuggestion: "Run 'make lint' manually to see detailed output.",
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
			expectedSuggestion: "Install gofumpt with 'go install mvdan.cc/gofumpt@latest' or run 'make install-fumpt' if available.",
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
			name:               "TargetNotFound",
			command:            "make test",
			output:             "No rule to make target 'test'",
			expectedMessage:    "Make target 'test' not found",
			expectedSuggestion: "Check your Makefile for the 'test' target or run 'make help' to see available targets.",
		},
		{
			name:               "PermissionDenied",
			command:            "make build",
			output:             "Permission denied: cannot create output file",
			expectedMessage:    "Permission denied",
			expectedSuggestion: "Check file permissions and ensure you have write access to the project directory.",
		},
		{
			name:               "GenericError",
			command:            "make deploy",
			output:             "deployment failed with unknown error",
			expectedMessage:    "Make command 'make deploy' failed",
			expectedSuggestion: "Run 'make deploy' manually to see detailed error output.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, suggestion := f.parseGenericMakeError(tc.command, tc.output)
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
			command:            "make lint",
			output:             "golangci-lint: no such file or directory",
			expectedMessage:    "golangci-lint binary not found",
			expectedSuggestion: "Install golangci-lint or ensure it's in your PATH. Run 'make install-lint' if available.",
		},
		{
			name:               "FumptCommand",
			command:            "make fumpt",
			output:             "gofumpt: no such file or directory",
			expectedMessage:    "gofumpt binary not found",
			expectedSuggestion: "Install gofumpt with 'go install mvdan.cc/gofumpt@latest' or run 'make install-fumpt' if available.",
		},
		{
			name:               "ModTidyCommand",
			command:            "make mod-tidy",
			output:             "no go.mod file found",
			expectedMessage:    "No go.mod file found",
			expectedSuggestion: "Initialize a Go module with 'go mod init <module-name>'.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message, suggestion := f.ParseMakeError(tc.command, tc.output)
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

// TestParseMakeErrorWhitespaceHandling tests whitespace handling in ParseMakeError
func TestParseMakeErrorWhitespaceHandling(t *testing.T) {
	f := NewDefault()

	// Test with leading/trailing whitespace
	message, suggestion := f.ParseMakeError("make lint", "  \n\tgolangci-lint: no such file or directory\n\t  ")
	assert.Equal(t, "golangci-lint binary not found", message)
	assert.Contains(t, suggestion, "Install golangci-lint")

	// Test with empty output (just whitespace)
	message, suggestion = f.ParseMakeError("make unknown-target", "   \n\t   ")
	assert.Equal(t, "Make command 'make unknown-target' failed", message)
	assert.Contains(t, suggestion, "Run 'make unknown-target' manually")
}
