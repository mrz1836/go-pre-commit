package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

func TestNewWhitespaceCheckCreation(t *testing.T) {
	check := NewWhitespaceCheck()

	require.NotNil(t, check)
	assert.Equal(t, "whitespace", check.Name())
	assert.Equal(t, "Fix trailing whitespace", check.Description())
	assert.Equal(t, 30*time.Second, check.timeout)
	assert.False(t, check.autoStage)
	assert.Nil(t, check.config)
}

func TestNewWhitespaceCheckWithTimeout(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "custom timeout 10 seconds",
			timeout:         10 * time.Second,
			expectedTimeout: 10 * time.Second,
		},
		{
			name:            "custom timeout 60 seconds",
			timeout:         60 * time.Second,
			expectedTimeout: 60 * time.Second,
		},
		{
			name:            "zero timeout",
			timeout:         0,
			expectedTimeout: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := NewWhitespaceCheckWithTimeout(tt.timeout)

			require.NotNil(t, check)
			assert.Equal(t, "whitespace", check.Name())
			assert.Equal(t, "Fix trailing whitespace", check.Description())
			assert.Equal(t, tt.expectedTimeout, check.timeout)
			assert.False(t, check.autoStage)
			assert.Nil(t, check.config)
		})
	}
}

func TestWhitespaceCheckMetadata(t *testing.T) {
	check := NewWhitespaceCheck()
	metadata := check.Metadata()

	require.NotNil(t, metadata)

	// Type assertion to CheckMetadata
	checkMeta, ok := metadata.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "whitespace", checkMeta.Name)
	assert.Equal(t, "Remove trailing whitespace from text files", checkMeta.Description)
	assert.Equal(t, "formatting", checkMeta.Category)
	assert.Equal(t, 30*time.Second, checkMeta.DefaultTimeout)
	assert.True(t, checkMeta.RequiresFiles)
	assert.Empty(t, checkMeta.Dependencies)
	assert.Equal(t, 1*time.Second, checkMeta.EstimatedDuration)

	// Check file patterns
	expectedPatterns := []string{"*.go", "*.md", "*.txt", "*.yml", "*.yaml", "*.json", "Makefile"}
	assert.ElementsMatch(t, expectedPatterns, checkMeta.FilePatterns)
}

func TestWhitespaceCheckRunTrailingSpaces(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		expectedFixed bool
		expectedError bool
		errorType     error
		expectedFinal string
	}{
		{
			name:          "no trailing whitespace",
			fileContent:   "hello world\n",
			expectedFixed: false,
			expectedError: false,
			expectedFinal: "hello world\n",
		},
		{
			name:          "trailing spaces single line",
			fileContent:   "hello world   \n",
			expectedFixed: true,
			expectedError: true,
			errorType:     prerrors.ErrWhitespaceIssues,
			expectedFinal: "hello world\n",
		},
		{
			name:          "trailing tabs single line",
			fileContent:   "hello world\t\t\n",
			expectedFixed: true,
			expectedError: true,
			errorType:     prerrors.ErrWhitespaceIssues,
			expectedFinal: "hello world\n",
		},
		{
			name:          "trailing mixed spaces and tabs",
			fileContent:   "hello world \t \t  \n",
			expectedFixed: true,
			expectedError: true,
			errorType:     prerrors.ErrWhitespaceIssues,
			expectedFinal: "hello world\n",
		},
		{
			name:          "multiline with trailing whitespace",
			fileContent:   "line 1  \nline 2\t\nline 3   \n",
			expectedFixed: true,
			expectedError: true,
			errorType:     prerrors.ErrWhitespaceIssues,
			expectedFinal: "line 1\nline 2\nline 3\n",
		},
		{
			name:          "multiline no trailing whitespace",
			fileContent:   "line 1\nline 2\nline 3\n",
			expectedFixed: false,
			expectedError: false,
			expectedFinal: "line 1\nline 2\nline 3\n",
		},
		{
			name:          "file without final newline has trailing spaces",
			fileContent:   "hello world   ",
			expectedFixed: true,
			expectedError: true,
			errorType:     prerrors.ErrWhitespaceIssues,
			expectedFinal: "hello world",
		},
		{
			name:          "file without final newline no trailing spaces",
			fileContent:   "hello world",
			expectedFixed: false,
			expectedError: false,
			expectedFinal: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			// Create test file
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			check := NewWhitespaceCheck()
			ctx := context.Background()

			// Run the check
			err = check.Run(ctx, []string{testFile})

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorType != nil {
					require.ErrorIs(t, err, tt.errorType)
				}
			} else {
				require.NoError(t, err)
			}

			// Verify file content after processing
			content, err := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
			require.NoError(t, err)
			assert.Equal(t, tt.expectedFinal, string(content))
		})
	}
}

func TestWhitespaceCheckRunLineEndingHandling(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		expectedFixed bool
		expectedFinal string
		description   string
	}{
		{
			name:          "CRLF line endings with trailing spaces",
			fileContent:   "line 1  \r\nline 2\t\r\nline 3\r\n",
			expectedFixed: true,
			expectedFinal: "line 1\nline 2\nline 3\n",
			description:   "Should normalize line endings to LF while removing trailing whitespace",
		},
		{
			name:          "CRLF line endings no trailing spaces",
			fileContent:   "line 1\r\nline 2\r\nline 3\r\n",
			expectedFixed: false,
			expectedFinal: "line 1\r\nline 2\r\nline 3\r\n",
			description:   "Should leave CRLF files unchanged when no trailing whitespace",
		},
		{
			name:          "mixed line endings with trailing spaces",
			fileContent:   "line 1  \nline 2\t\r\nline 3   \n",
			expectedFixed: true,
			expectedFinal: "line 1\nline 2\nline 3\n",
			description:   "Should normalize mixed line endings to LF while removing trailing whitespace",
		},
		{
			name:          "file ending with CRLF and trailing spaces",
			fileContent:   "content   \r\n",
			expectedFixed: true,
			expectedFinal: "content\n",
			description:   "Should normalize CRLF to LF while removing trailing whitespace",
		},
		{
			name:          "file ending with CRLF no trailing spaces",
			fileContent:   "content\r\n",
			expectedFixed: false,
			expectedFinal: "content\r\n",
			description:   "Should leave CRLF ending unchanged when no trailing whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			// Create test file
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			check := NewWhitespaceCheck()
			ctx := context.Background()

			// Run the check
			err = check.Run(ctx, []string{testFile})

			if tt.expectedFixed {
				require.Error(t, err)
				require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)
			} else {
				require.NoError(t, err)
			}

			// Verify file content after processing
			content, err := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
			require.NoError(t, err)
			assert.Equal(t, tt.expectedFinal, string(content), tt.description)
		})
	}
}

func TestWhitespaceCheckRunEmptyFiles(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		expectedFixed bool
		expectedFinal string
		description   string
	}{
		{
			name:          "completely empty file",
			fileContent:   "",
			expectedFixed: false,
			expectedFinal: "",
			description:   "Empty files should remain empty",
		},
		{
			name:          "file with single newline",
			fileContent:   "\n",
			expectedFixed: false,
			expectedFinal: "\n",
			description:   "Single newline should be preserved",
		},
		{
			name:          "file with only spaces",
			fileContent:   "   ",
			expectedFixed: true,
			expectedFinal: "",
			description:   "File with only spaces should become empty",
		},
		{
			name:          "file with only tabs",
			fileContent:   "\t\t\t",
			expectedFixed: true,
			expectedFinal: "",
			description:   "File with only tabs should become empty",
		},
		{
			name:          "file with only mixed whitespace",
			fileContent:   " \t  \t ",
			expectedFixed: true,
			expectedFinal: "\n",
			description:   "File with only mixed whitespace should preserve newline",
		},
		{
			name:          "file with spaces and newline",
			fileContent:   "   \n",
			expectedFixed: true,
			expectedFinal: "\n",
			description:   "File with trailing spaces before newline should preserve newline",
		},
		{
			name:          "file with tabs and newline",
			fileContent:   "\t\t\n",
			expectedFixed: true,
			expectedFinal: "\n",
			description:   "File with trailing tabs before newline should preserve newline",
		},
		{
			name:          "file with substantial whitespace content",
			fileContent:   "      \t\t   \t  ", // More than 5 chars
			expectedFixed: true,
			expectedFinal: "\n",
			description:   "Substantial whitespace content should preserve a newline",
		},
		{
			name:          "file with substantial whitespace and final newline",
			fileContent:   "      \t\t   \t  \n", // More than 5 chars plus newline
			expectedFixed: true,
			expectedFinal: "\n",
			description:   "Substantial whitespace with final newline should preserve single newline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			// Create test file
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			check := NewWhitespaceCheck()
			ctx := context.Background()

			// Run the check
			err = check.Run(ctx, []string{testFile})

			if tt.expectedFixed {
				require.Error(t, err)
				require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)
			} else {
				require.NoError(t, err)
			}

			// Verify file content after processing
			content, err := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
			require.NoError(t, err)
			assert.Equal(t, tt.expectedFinal, string(content), tt.description)
		})
	}
}

func TestWhitespaceCheckRunMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files with different states
	goodFile1 := filepath.Join(tmpDir, "good1.txt")
	err := os.WriteFile(goodFile1, []byte("good content\n"), 0o600)
	require.NoError(t, err)

	goodFile2 := filepath.Join(tmpDir, "good2.txt")
	err = os.WriteFile(goodFile2, []byte("more good\ncontent\n"), 0o600)
	require.NoError(t, err)

	badFile1 := filepath.Join(tmpDir, "bad1.txt")
	err = os.WriteFile(badFile1, []byte("bad content   \n"), 0o600)
	require.NoError(t, err)

	badFile2 := filepath.Join(tmpDir, "bad2.txt")
	err = os.WriteFile(badFile2, []byte("more\nbad content\t\t\n"), 0o600)
	require.NoError(t, err)

	emptyFile := filepath.Join(tmpDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte(""), 0o600)
	require.NoError(t, err)

	check := NewWhitespaceCheck()
	ctx := context.Background()

	// Test with all good files
	err = check.Run(ctx, []string{goodFile1, goodFile2, emptyFile})
	require.NoError(t, err)

	// Test with mixed files (should return error due to bad files)
	err = check.Run(ctx, []string{goodFile1, badFile1, badFile2, emptyFile})
	require.Error(t, err)
	require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)

	// Verify bad files were fixed
	content1, err := os.ReadFile(badFile1) //nolint:gosec // test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "bad content\n", string(content1))

	content2, err := os.ReadFile(badFile2) //nolint:gosec // test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "more\nbad content\n", string(content2))

	// Verify good files unchanged
	goodContent1, err := os.ReadFile(goodFile1) //nolint:gosec // test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "good content\n", string(goodContent1))

	// Verify empty file unchanged
	emptyContent, err := os.ReadFile(emptyFile) //nolint:gosec // test file path is controlled
	require.NoError(t, err)
	assert.Empty(t, emptyContent)
}

func TestWhitespaceCheckRunErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) []string
		expectedError bool
		errorContains string
	}{
		{
			name: "no files provided",
			setupFunc: func(_ *testing.T) []string {
				return []string{}
			},
			expectedError: false,
		},
		{
			name: "non-existent file",
			setupFunc: func(_ *testing.T) []string {
				return []string{"/nonexistent/file.txt"}
			},
			expectedError: true,
			errorContains: "failed to read file",
		},
		{
			name: "read-only file for write",
			setupFunc: func(t *testing.T) []string {
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "readonly.txt")
				// Create file with trailing whitespace that needs fixing
				err := os.WriteFile(testFile, []byte("content with spaces   \n"), 0o600)
				require.NoError(t, err)
				// Make file read-only
				err = os.Chmod(testFile, 0o400)
				require.NoError(t, err)

				// Cleanup function to make file writable again for cleanup
				t.Cleanup(func() {
					_ = os.Chmod(testFile, 0o600)
				})

				return []string{testFile}
			},
			expectedError: true,
			errorContains: "failed to write file",
		},
		{
			name: "directory instead of file",
			setupFunc: func(t *testing.T) []string {
				tmpDir := t.TempDir()
				subDir := filepath.Join(tmpDir, "subdir")
				err := os.Mkdir(subDir, 0o750)
				require.NoError(t, err)
				return []string{subDir}
			},
			expectedError: true,
			errorContains: "failed to read file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := tt.setupFunc(t)

			check := NewWhitespaceCheck()
			ctx := context.Background()

			err := check.Run(ctx, files)

			if tt.expectedError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWhitespaceCheckRunWithTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file with trailing whitespace
	err := os.WriteFile(testFile, []byte("content   \n"), 0o600)
	require.NoError(t, err)

	// Test with very short timeout
	check := NewWhitespaceCheckWithTimeout(1 * time.Nanosecond)
	ctx := context.Background()

	err = check.Run(ctx, []string{testFile})
	// The test might pass or fail due to timing, but should not panic
	// We're mainly testing that timeout context is handled properly
	if err != nil {
		// If timeout occurred, it should be a context error or whitespace issues
		isTimeoutOrWhitespace := errors.Is(err, context.DeadlineExceeded) || errors.Is(err, prerrors.ErrWhitespaceIssues)
		assert.True(t, isTimeoutOrWhitespace,
			"Error should be either timeout or whitespace issues, got: %v", err)
	}
}

func TestWhitespaceCheckRunWithCancelledContext(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file with trailing whitespace
	err := os.WriteFile(testFile, []byte("content   \n"), 0o600)
	require.NoError(t, err)

	check := NewWhitespaceCheck()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	err = check.Run(ctx, []string{testFile})
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestWhitespaceCheckFilterFiles(t *testing.T) {
	tests := []struct {
		name          string
		inputFiles    []string
		expectedFiles []string
	}{
		{
			name:          "empty input",
			inputFiles:    []string{},
			expectedFiles: []string{},
		},
		{
			name: "text files only",
			inputFiles: []string{
				"main.go",
				"README.md",
				"config.yaml",
				"data.json",
				"notes.txt",
				"Makefile",
				"script.sh",
				"style.css",
			},
			expectedFiles: []string{
				"main.go",
				"README.md",
				"config.yaml",
				"data.json",
				"notes.txt",
				"Makefile",
				"script.sh",
				"style.css",
			},
		},
		{
			name: "mixed text and binary files",
			inputFiles: []string{
				"main.go",
				"image.png",
				"document.pdf",
				"README.md",
				"binary.exe",
				"config.yml",
				"library.so",
				"archive.zip",
			},
			expectedFiles: []string{
				"main.go",
				"README.md",
				"config.yml",
			},
		},
		{
			name: "binary files only",
			inputFiles: []string{
				"image.png",
				"archive.zip",
				"binary.exe",
				"library.dll",
				"video.mp4",
			},
			expectedFiles: []string{},
		},
		{
			name: "files without extensions",
			inputFiles: []string{
				"Dockerfile",
				"LICENSE",
				"README",
				"randomfile",
				"Makefile",
				"Jenkinsfile",
				".gitignore",
				".editorconfig",
			},
			expectedFiles: []string{
				"Dockerfile",
				"LICENSE",
				"README",
				"Makefile",
				"Jenkinsfile",
				".gitignore",
				".editorconfig",
			},
		},
		{
			name: "paths with directories",
			inputFiles: []string{
				"src/main.go",
				"docs/README.md",
				"assets/image.png",
				"config/app.yaml",
				"build/output.bin",
				"scripts/deploy.sh",
			},
			expectedFiles: []string{
				"src/main.go",
				"docs/README.md",
				"config/app.yaml",
				"scripts/deploy.sh",
			},
		},
		{
			name: "programming language files",
			inputFiles: []string{
				"main.go",
				"script.py",
				"component.js",
				"style.css",
				"index.html",
				"main.rs",
				"App.java",
				"main.cpp",
				"header.h",
			},
			expectedFiles: []string{
				"main.go",
				"script.py",
				"component.js",
				"style.css",
				"index.html",
				"main.rs",
				"App.java",
				"main.cpp",
				"header.h",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := NewWhitespaceCheck()
			filtered := check.FilterFiles(tt.inputFiles)
			assert.ElementsMatch(t, tt.expectedFiles, filtered)
		})
	}
}

func TestWhitespaceCheckProcessFileDirect(t *testing.T) {
	tests := []struct {
		name             string
		fileContent      string
		expectedModified bool
		expectedError    bool
		expectedContent  string
	}{
		{
			name:             "file with no trailing whitespace",
			fileContent:      "hello world\n",
			expectedModified: false,
			expectedError:    false,
			expectedContent:  "hello world\n",
		},
		{
			name:             "file with trailing spaces",
			fileContent:      "hello world   \n",
			expectedModified: true,
			expectedError:    false,
			expectedContent:  "hello world\n",
		},
		{
			name:             "file with trailing tabs",
			fileContent:      "hello world\t\t\n",
			expectedModified: true,
			expectedError:    false,
			expectedContent:  "hello world\n",
		},
		{
			name:             "empty file",
			fileContent:      "",
			expectedModified: false,
			expectedError:    false,
			expectedContent:  "",
		},
		{
			name:             "multiline with trailing whitespace",
			fileContent:      "line1  \nline2\t\nline3   \n",
			expectedModified: true,
			expectedError:    false,
			expectedContent:  "line1\nline2\nline3\n",
		},
		{
			name:             "multiline no trailing whitespace",
			fileContent:      "line1\nline2\nline3\n",
			expectedModified: false,
			expectedError:    false,
			expectedContent:  "line1\nline2\nline3\n",
		},
		{
			name:             "file without final newline but with trailing spaces",
			fileContent:      "content   ",
			expectedModified: true,
			expectedError:    false,
			expectedContent:  "content",
		},
		{
			name:             "only whitespace content",
			fileContent:      "   \t  ",
			expectedModified: true,
			expectedError:    false,
			expectedContent:  "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			// Create test file
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			check := NewWhitespaceCheck()

			// Call processFile directly
			modified, err := check.processFile(testFile)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedModified, modified)

			// Verify file content
			content, err := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
			require.NoError(t, err)
			assert.Equal(t, tt.expectedContent, string(content))
		})
	}
}

func TestWhitespaceCheckBinaryFileHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that looks like binary (contains null bytes)
	binaryFile := filepath.Join(tmpDir, "binary.bin")
	binaryContent := []byte{0x00, 0xFF, 0xDE, 0xAD, 0x20, 0x20, 0x0A} // includes trailing spaces and newline
	err := os.WriteFile(binaryFile, binaryContent, 0o600)
	require.NoError(t, err)

	// Create a binary file without extension
	binaryFile2 := filepath.Join(tmpDir, "unknownbinary")
	binaryContent2 := []byte{0x89, 0x50, 0x4E, 0x47, 0x20, 0x20, 0x0A} // PNG header with trailing spaces
	err = os.WriteFile(binaryFile2, binaryContent2, 0o600)
	require.NoError(t, err)

	check := NewWhitespaceCheck()
	ctx := context.Background()

	// Binary files should be filtered out by extension
	filteredFiles := check.FilterFiles([]string{binaryFile})
	assert.Empty(t, filteredFiles, "Binary file with .bin extension should be filtered out")

	// Unknown binary file should be filtered out (isTextFile returns false for unknown files)
	filteredFiles2 := check.FilterFiles([]string{binaryFile2})
	assert.Empty(t, filteredFiles2, "File without known extension should be filtered out")

	// Running check directly on binary file should process it
	// (this tests the case where filtering is bypassed)
	err = check.Run(ctx, []string{binaryFile})
	// Binary content with trailing spaces should be modified
	require.Error(t, err, "Binary file processing should result in whitespace issues")
	require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)

	// Verify binary file content was modified (trailing spaces removed)
	content, err := os.ReadFile(binaryFile) //nolint:gosec // test file path is controlled
	require.NoError(t, err)
	// Should have trailing spaces removed but preserve binary content
	expected := []byte{0x00, 0xFF, 0xDE, 0xAD, 0x0A}
	assert.Equal(t, expected, content)
}

func TestWhitespaceCheckSpecialCharacters(t *testing.T) {
	tests := []struct {
		name             string
		fileContent      string
		expectedModified bool
		expectedContent  string
		description      string
	}{
		{
			name:             "file with null bytes and trailing spaces",
			fileContent:      "hello\x00world   \n",
			expectedModified: true,
			expectedContent:  "hello\x00world\n",
			description:      "Should remove trailing spaces while preserving null bytes",
		},
		{
			name:             "file with unicode characters and trailing tabs",
			fileContent:      "hello 🌍 world\t\t\n",
			expectedModified: true,
			expectedContent:  "hello 🌍 world\n",
			description:      "Should remove trailing tabs while preserving unicode",
		},
		{
			name:             "file with control characters and trailing whitespace",
			fileContent:      "hello\tworld\r  \t \n",
			expectedModified: true,
			expectedContent:  "hello\tworld\r\n",
			description:      "Should remove trailing whitespace while preserving internal control chars",
		},
		{
			name:             "file ending with CRLF and trailing spaces",
			fileContent:      "hello world   \r\n",
			expectedModified: true,
			expectedContent:  "hello world\n",
			description:      "Should normalize CRLF and remove trailing spaces",
		},
		{
			name:             "file with form feed and trailing spaces",
			fileContent:      "page1\fpage2   \n",
			expectedModified: true,
			expectedContent:  "page1\fpage2\n",
			description:      "Should remove trailing spaces while preserving form feed",
		},
		{
			name:             "file with vertical tab and mixed trailing whitespace",
			fileContent:      "line1\vtab \t  \n",
			expectedModified: true,
			expectedContent:  "line1\vtab\n",
			description:      "Should remove all trailing spaces and tabs while preserving vertical tab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			// Create test file
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			check := NewWhitespaceCheck()
			ctx := context.Background()

			err = check.Run(ctx, []string{testFile})

			if tt.expectedModified {
				require.Error(t, err)
				require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)
			} else {
				require.NoError(t, err)
			}

			// Verify file content
			content, err := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
			require.NoError(t, err)
			assert.Equal(t, tt.expectedContent, string(content), tt.description)
		})
	}
}

func TestWhitespaceCheckLargeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// Create a large file with trailing whitespace on multiple lines
	var content strings.Builder
	for i := 0; i < 10000; i++ {
		if i%2 == 0 {
			content.WriteString(fmt.Sprintf("line %d   \n", i)) // trailing spaces
		} else {
			content.WriteString(fmt.Sprintf("line %d\n", i)) // no trailing spaces
		}
	}

	err := os.WriteFile(testFile, []byte(content.String()), 0o600)
	require.NoError(t, err)

	check := NewWhitespaceCheck()
	ctx := context.Background()

	err = check.Run(ctx, []string{testFile})
	require.Error(t, err)
	require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)

	// Verify trailing whitespace was removed
	processedContent, err := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
	require.NoError(t, err)

	// Check that no lines have trailing spaces
	lines := strings.Split(string(processedContent), "\n")
	for i, line := range lines {
		if line != "" { // Skip empty last line from split
			assert.Equal(t, strings.TrimRight(line, " \t"), line,
				"Line %d should not have trailing whitespace: %q", i, line)
		}
	}

	// Verify content length is smaller (spaces were removed)
	assert.Less(t, len(processedContent), len(content.String()))
}

func TestWhitespaceCheckConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple test files with trailing whitespace
	files := make([]string, 20)
	for i := 0; i < 20; i++ {
		files[i] = filepath.Join(tmpDir, fmt.Sprintf("test%d.txt", i))
		content := fmt.Sprintf("content %d   \n", i) // trailing spaces
		err := os.WriteFile(files[i], []byte(content), 0o600)
		require.NoError(t, err)
	}

	check := NewWhitespaceCheck()
	ctx := context.Background()

	// Run check (should fix all files)
	err := check.Run(ctx, files)
	require.Error(t, err)
	require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)

	// Verify all files were fixed
	for i, file := range files {
		content, err := os.ReadFile(file) //nolint:gosec // test file path is controlled
		require.NoError(t, err, "Failed to read file %d", i)
		expected := fmt.Sprintf("content %d\n", i)
		assert.Equal(t, expected, string(content), "File %d content mismatch", i)
	}
}

func TestWhitespaceCheckErrorAccumulation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid file with trailing whitespace
	validFile := filepath.Join(tmpDir, "valid.txt")
	err := os.WriteFile(validFile, []byte("valid content   \n"), 0o600)
	require.NoError(t, err)

	// Create files that will cause read errors
	nonExistentFile1 := filepath.Join(tmpDir, "nonexistent1.txt")
	nonExistentFile2 := filepath.Join(tmpDir, "nonexistent2.txt")

	files := []string{validFile, nonExistentFile1, nonExistentFile2}

	check := NewWhitespaceCheck()
	ctx := context.Background()

	err = check.Run(ctx, files)
	require.Error(t, err)

	// Error should contain information about all failed files
	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "nonexistent1.txt")
	assert.Contains(t, errorMsg, "nonexistent2.txt")
	assert.Contains(t, errorMsg, "failed to read file")

	// Valid file should still have been processed
	content, err := os.ReadFile(validFile) //nolint:gosec // test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "valid content\n", string(content))
}

func TestWhitespaceCheckMixedErrorsAndFixes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that needs fixing
	fixableFile := filepath.Join(tmpDir, "fixable.txt")
	err := os.WriteFile(fixableFile, []byte("needs fixing   \n"), 0o600)
	require.NoError(t, err)

	// Create a file that will cause read error
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	files := []string{fixableFile, nonExistentFile}

	check := NewWhitespaceCheck()
	ctx := context.Background()

	err = check.Run(ctx, files)
	require.Error(t, err)

	// Should contain both the read error and indicate whitespace issues were found
	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "whitespace issues found")
	assert.Contains(t, errorMsg, "failed to read file")
	assert.Contains(t, errorMsg, "nonexistent.txt")

	// Fixable file should have been fixed
	content, err := os.ReadFile(fixableFile) //nolint:gosec // test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "needs fixing\n", string(content))
}

func TestWhitespaceCheckIsTextFileFunction(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		// Programming languages
		{"Go file", "main.go", true},
		{"Python file", "script.py", true},
		{"JavaScript file", "app.js", true},
		{"TypeScript file", "component.ts", true},
		{"Java file", "Main.java", true},
		{"C++ file", "main.cpp", true},
		{"C header", "header.h", true},
		{"Rust file", "main.rs", true},

		// Markup and data
		{"HTML file", "index.html", true},
		{"CSS file", "style.css", true},
		{"JSON file", "config.json", true},
		{"YAML file", "config.yaml", true},
		{"YML file", "docker-compose.yml", true},
		{"XML file", "config.xml", true},
		{"TOML file", "Cargo.toml", true},

		// Documentation
		{"Markdown file", "README.md", true},
		{"Text file", "notes.txt", true},

		// Configuration
		{"Shell script", "deploy.sh", true},
		{"Bash script", "install.bash", true},
		{"Env file", "config.env", true},
		{"Ini file", "settings.ini", true},

		// Files without extensions
		{"Makefile", "Makefile", true},
		{"Dockerfile", "Dockerfile", true},
		{"Jenkinsfile", "Jenkinsfile", true},
		{"License file", "LICENSE", true},
		{"README without extension", "README", true},
		{"gitignore", ".gitignore", true},
		{"editorconfig", ".editorconfig", true},

		// Binary files
		{"PNG image", "logo.png", false},
		{"JPG image", "photo.jpg", false},
		{"GIF image", "animation.gif", false},
		{"PDF document", "manual.pdf", false},
		{"Zip archive", "release.zip", false},
		{"Executable", "program.exe", false},
		{"Library", "library.so", false},
		{"DLL library", "library.dll", false},
		{"Object file", "main.o", false},
		{"Video file", "demo.mp4", false},
		{"Audio file", "music.mp3", false},

		// Unknown extensions
		{"Unknown extension", "file.unknown", false},
		{"Random file", "randomfile", false},

		// Edge cases
		{"Empty filename", "", false},
		{"Just extension", ".txt", true},
		{"Multiple extensions", "archive.tar.gz", false}, // Only checks final extension
		{"Case sensitivity", "README.MD", true},          // Extension should be lowercased
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTextFile(tt.filename)
			assert.Equal(t, tt.expected, result, "isTextFile(%q) = %v, want %v", tt.filename, result, tt.expected)
		})
	}
}

func TestWhitespaceCheckComplexWhitespacePatterns(t *testing.T) {
	tests := []struct {
		name            string
		fileContent     string
		expectedFixed   bool
		expectedContent string
		description     string
	}{
		{
			name:            "mixed spaces and tabs at end of lines",
			fileContent:     "line1 \t \nline2\t \t\nline3  \t  \n",
			expectedFixed:   true,
			expectedContent: "line1\nline2\nline3\n",
			description:     "Should remove all combinations of trailing spaces and tabs",
		},
		{
			name:            "indented lines with trailing whitespace",
			fileContent:     "\tindented line  \n    another indent\t\n        deep indent   \n",
			expectedFixed:   true,
			expectedContent: "\tindented line\n    another indent\n        deep indent\n",
			description:     "Should preserve leading whitespace while removing trailing whitespace",
		},
		{
			name:            "empty lines with different whitespace",
			fileContent:     "line1\n   \nline2\n\t\t\nline3\n \t \n",
			expectedFixed:   true,
			expectedContent: "line1\n\nline2\n\nline3\n\n",
			description:     "Should remove whitespace from empty lines",
		},
		{
			name:            "file with only empty whitespace lines",
			fileContent:     "   \n\t\t\n \t \n",
			expectedFixed:   true,
			expectedContent: "\n",
			description:     "Should remove all whitespace and preserve single newline",
		},
		{
			name:            "trailing whitespace with different newline styles",
			fileContent:     "unix line  \nwindows line\t\r\nmac line   \r",
			expectedFixed:   true,
			expectedContent: "unix line\nwindows line\nmac line",
			description:     "Should normalize line endings to LF while removing trailing whitespace",
		},
		{
			name:            "very long line with trailing whitespace",
			fileContent:     strings.Repeat("a", 1000) + "   \n",
			expectedFixed:   true,
			expectedContent: strings.Repeat("a", 1000) + "\n",
			description:     "Should handle very long lines correctly",
		},
		{
			name:            "unicode content with trailing whitespace",
			fileContent:     "Hello 世界 🌍  \nПривет мир\t\n你好世界   \n", //nolint:gosmopolitan // testing unicode handling
			expectedFixed:   true,
			expectedContent: "Hello 世界 🌍\nПривет мир\n你好世界\n", //nolint:gosmopolitan // testing unicode handling
			description:     "Should handle unicode content correctly while removing trailing whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			// Create test file
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			check := NewWhitespaceCheck()
			ctx := context.Background()

			err = check.Run(ctx, []string{testFile})

			if tt.expectedFixed {
				require.Error(t, err)
				require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)
			} else {
				require.NoError(t, err)
			}

			// Verify file content
			content, err := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
			require.NoError(t, err)
			assert.Equal(t, tt.expectedContent, string(content), tt.description)
		})
	}
}

func TestWhitespaceCheckEdgeCaseFileEndings(t *testing.T) {
	tests := []struct {
		name            string
		fileContent     string
		expectedFixed   bool
		expectedContent string
		description     string
	}{
		{
			name:            "file ending with multiple newlines and whitespace",
			fileContent:     "content\n\n  \n\n",
			expectedFixed:   true,
			expectedContent: "content\n\n\n\n",
			description:     "Should preserve multiple newlines while removing whitespace",
		},
		{
			name:            "file with trailing whitespace on last line no newline",
			fileContent:     "line1\nline2\nlast line   ",
			expectedFixed:   true,
			expectedContent: "line1\nline2\nlast line",
			description:     "Should remove trailing whitespace from last line even without final newline",
		},
		{
			name:            "file with only newlines and whitespace",
			fileContent:     "\n  \n\t\n   \n",
			expectedFixed:   true,
			expectedContent: "\n",
			description:     "Should remove all whitespace and preserve single newline",
		},
		{
			name:            "single character with trailing whitespace",
			fileContent:     "a   ",
			expectedFixed:   true,
			expectedContent: "a",
			description:     "Should handle single character content with trailing whitespace",
		},
		{
			name:            "single character with trailing whitespace and newline",
			fileContent:     "a   \n",
			expectedFixed:   true,
			expectedContent: "a\n",
			description:     "Should handle single character content with trailing whitespace and newline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			// Create test file
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			check := NewWhitespaceCheck()
			ctx := context.Background()

			err = check.Run(ctx, []string{testFile})

			if tt.expectedFixed {
				require.Error(t, err)
				require.ErrorIs(t, err, prerrors.ErrWhitespaceIssues)
			} else {
				require.NoError(t, err)
			}

			// Verify file content
			content, err := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
			require.NoError(t, err)
			assert.Equal(t, tt.expectedContent, string(content), tt.description)
		})
	}
}
