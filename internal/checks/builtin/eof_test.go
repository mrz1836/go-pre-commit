package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

func TestNewEOFCheckCreation(t *testing.T) {
	check := NewEOFCheck()

	require.NotNil(t, check)
	assert.Equal(t, "eof", check.Name())
	assert.Equal(t, "Ensure files end with newline", check.Description())
	assert.Equal(t, 30*time.Second, check.timeout)
}

func TestNewEOFCheckWithTimeout(t *testing.T) {
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
			check := NewEOFCheckWithTimeout(tt.timeout)

			require.NotNil(t, check)
			assert.Equal(t, "eof", check.Name())
			assert.Equal(t, "Ensure files end with newline", check.Description())
			assert.Equal(t, tt.expectedTimeout, check.timeout)
		})
	}
}

func TestEOFCheckMetadata(t *testing.T) {
	check := NewEOFCheck()
	metadata := check.Metadata()

	require.NotNil(t, metadata)

	// Type assertion to CheckMetadata
	checkMeta, ok := metadata.(CheckMetadata)
	require.True(t, ok, "Metadata should be of type CheckMetadata")

	assert.Equal(t, "eof", checkMeta.Name)
	assert.Equal(t, "Ensure text files end with a newline character", checkMeta.Description)
	assert.Equal(t, "formatting", checkMeta.Category)
	assert.Equal(t, 30*time.Second, checkMeta.DefaultTimeout)
	assert.True(t, checkMeta.RequiresFiles)
	assert.Empty(t, checkMeta.Dependencies)
	assert.Equal(t, 1*time.Second, checkMeta.EstimatedDuration)

	// Check file patterns
	expectedPatterns := []string{"*.go", "*.md", "*.txt", "*.yml", "*.yaml", "*.json", "Makefile"}
	assert.ElementsMatch(t, expectedPatterns, checkMeta.FilePatterns)
}

func TestEOFCheckRunValidFiles(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		expectedFixed bool
		expectedError bool
		errorType     error
		expectedFinal string
	}{
		{
			name:          "file already ends with newline",
			fileContent:   "hello world\n",
			expectedFixed: false,
			expectedError: false,
			expectedFinal: "hello world\n",
		},
		{
			name:          "file missing newline at end",
			fileContent:   "hello world",
			expectedFixed: true,
			expectedError: true,
			errorType:     prerrors.ErrEOFIssues,
			expectedFinal: "hello world\n",
		},
		{
			name:          "multiline file with newline",
			fileContent:   "line 1\nline 2\nline 3\n",
			expectedFixed: false,
			expectedError: false,
			expectedFinal: "line 1\nline 2\nline 3\n",
		},
		{
			name:          "multiline file missing newline",
			fileContent:   "line 1\nline 2\nline 3",
			expectedFixed: true,
			expectedError: true,
			errorType:     prerrors.ErrEOFIssues,
			expectedFinal: "line 1\nline 2\nline 3\n",
		},
		{
			name:          "single character file without newline",
			fileContent:   "a",
			expectedFixed: true,
			expectedError: true,
			errorType:     prerrors.ErrEOFIssues,
			expectedFinal: "a\n",
		},
		{
			name:          "single character file with newline",
			fileContent:   "a\n",
			expectedFixed: false,
			expectedError: false,
			expectedFinal: "a\n",
		},
		{
			name:          "file with only newline",
			fileContent:   "\n",
			expectedFixed: false,
			expectedError: false,
			expectedFinal: "\n",
		},
		{
			name:          "file with mixed line endings needs newline",
			fileContent:   "line 1\r\nline 2",
			expectedFixed: true,
			expectedError: true,
			errorType:     prerrors.ErrEOFIssues,
			expectedFinal: "line 1\r\nline 2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			// Create test file
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			check := NewEOFCheck()
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
			content, err := os.ReadFile(testFile) // #nosec G304 -- test file path is controlled
			require.NoError(t, err)
			assert.Equal(t, tt.expectedFinal, string(content))
		})
	}
}

func TestEOFCheckRunEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.txt")

	// Create empty file
	err := os.WriteFile(emptyFile, []byte(""), 0o600)
	require.NoError(t, err)

	check := NewEOFCheck()
	ctx := context.Background()

	// Empty files should be skipped without error
	err = check.Run(ctx, []string{emptyFile})
	require.NoError(t, err)

	// Verify file remains empty
	content, err := os.ReadFile(emptyFile) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Empty(t, content, "Empty file should remain empty")
}

func TestEOFCheckRunMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files with different states
	goodFile1 := filepath.Join(tmpDir, "good1.txt")
	err := os.WriteFile(goodFile1, []byte("good content\n"), 0o600)
	require.NoError(t, err)

	goodFile2 := filepath.Join(tmpDir, "good2.txt")
	err = os.WriteFile(goodFile2, []byte("more good\ncontent\n"), 0o600)
	require.NoError(t, err)

	badFile1 := filepath.Join(tmpDir, "bad1.txt")
	err = os.WriteFile(badFile1, []byte("bad content"), 0o600)
	require.NoError(t, err)

	badFile2 := filepath.Join(tmpDir, "bad2.txt")
	err = os.WriteFile(badFile2, []byte("more\nbad content"), 0o600)
	require.NoError(t, err)

	emptyFile := filepath.Join(tmpDir, "empty.txt")
	err = os.WriteFile(emptyFile, []byte(""), 0o600)
	require.NoError(t, err)

	check := NewEOFCheck()
	ctx := context.Background()

	// Test with all good files
	err = check.Run(ctx, []string{goodFile1, goodFile2, emptyFile})
	require.NoError(t, err)

	// Test with mixed files (should return error due to bad files)
	err = check.Run(ctx, []string{goodFile1, badFile1, badFile2, emptyFile})
	require.Error(t, err)
	require.ErrorIs(t, err, prerrors.ErrEOFIssues)

	// Verify bad files were fixed
	content1, err := os.ReadFile(badFile1) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "bad content\n", string(content1))

	content2, err := os.ReadFile(badFile2) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "more\nbad content\n", string(content2))

	// Verify good files unchanged
	goodContent1, err := os.ReadFile(goodFile1) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "good content\n", string(goodContent1))

	// Verify empty file unchanged
	emptyContent, err := os.ReadFile(emptyFile) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Empty(t, emptyContent)
}

func TestEOFCheckRunErrorCases(t *testing.T) {
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
				// Create file with content that needs newline
				err := os.WriteFile(testFile, []byte("content without newline"), 0o600)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := tt.setupFunc(t)

			check := NewEOFCheck()
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

func TestEOFCheckRunWithTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	err := os.WriteFile(testFile, []byte("content"), 0o600)
	require.NoError(t, err)

	// Test with very short timeout
	check := NewEOFCheckWithTimeout(1 * time.Nanosecond)
	ctx := context.Background()

	err = check.Run(ctx, []string{testFile})
	// The test might pass or fail due to timing, but should not panic
	// We're mainly testing that timeout context is handled properly
	if err != nil {
		// If timeout occurred, it should be a context error
		isTimeoutOrEOF := errors.Is(err, context.DeadlineExceeded) || errors.Is(err, prerrors.ErrEOFIssues)
		assert.True(t, isTimeoutOrEOF,
			"Error should be either timeout or EOF issues, got: %v", err)
	}
}

func TestEOFCheckRunWithCancelledContext(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	err := os.WriteFile(testFile, []byte("content"), 0o600)
	require.NoError(t, err)

	check := NewEOFCheck()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	err = check.Run(ctx, []string{testFile})
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestEOFCheckFilterFiles(t *testing.T) {
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
			},
			expectedFiles: []string{
				"main.go",
				"README.md",
				"config.yaml",
				"data.json",
				"notes.txt",
				"Makefile",
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
			},
			expectedFiles: []string{
				"Dockerfile",
				"LICENSE",
				"README",
				"Makefile",
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
			},
			expectedFiles: []string{
				"src/main.go",
				"docs/README.md",
				"config/app.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := NewEOFCheck()
			filtered := check.FilterFiles(tt.inputFiles)
			assert.ElementsMatch(t, tt.expectedFiles, filtered)
		})
	}
}

func TestEOFCheckProcessFileDirect(t *testing.T) {
	tests := []struct {
		name             string
		fileContent      string
		expectedModified bool
		expectedError    bool
		expectedContent  string
	}{
		{
			name:             "file with newline",
			fileContent:      "hello\n",
			expectedModified: false,
			expectedError:    false,
			expectedContent:  "hello\n",
		},
		{
			name:             "file without newline",
			fileContent:      "hello",
			expectedModified: true,
			expectedError:    false,
			expectedContent:  "hello\n",
		},
		{
			name:             "empty file",
			fileContent:      "",
			expectedModified: false,
			expectedError:    false,
			expectedContent:  "",
		},
		{
			name:             "multiline with newline",
			fileContent:      "line1\nline2\n",
			expectedModified: false,
			expectedError:    false,
			expectedContent:  "line1\nline2\n",
		},
		{
			name:             "multiline without newline",
			fileContent:      "line1\nline2",
			expectedModified: true,
			expectedError:    false,
			expectedContent:  "line1\nline2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			// Create test file
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			check := NewEOFCheck()

			// Call processFile directly
			modified, err := check.processFile(testFile)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.expectedModified, modified)

			// Verify file content
			content, err := os.ReadFile(testFile) // #nosec G304 -- test file path is controlled
			require.NoError(t, err)
			assert.Equal(t, tt.expectedContent, string(content))
		})
	}
}

func TestEOFCheckBinaryFileHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a binary file
	binaryFile := filepath.Join(tmpDir, "binary.bin")
	binaryContent := []byte{0x00, 0xFF, 0xDE, 0xAD, 0xBE, 0xEF}
	err := os.WriteFile(binaryFile, binaryContent, 0o600)
	require.NoError(t, err)

	check := NewEOFCheck()
	ctx := context.Background()

	// Binary files should be filtered out
	filteredFiles := check.FilterFiles([]string{binaryFile})
	assert.Empty(t, filteredFiles, "Binary file should be filtered out")

	// Running check on filtered files should succeed
	err = check.Run(ctx, filteredFiles)
	require.NoError(t, err)

	// Running check directly on binary file should process it
	// (this tests the case where filtering is bypassed)
	err = check.Run(ctx, []string{binaryFile})
	require.Error(t, err, "Binary file processing should result in EOF issues")
	require.ErrorIs(t, err, prerrors.ErrEOFIssues)

	// Verify binary file content was modified (newline added)
	content, err := os.ReadFile(binaryFile) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	expected := append(binaryContent, '\n')
	assert.Equal(t, expected, content)
}

func TestEOFCheckSpecialCharacters(t *testing.T) {
	tests := []struct {
		name             string
		fileContent      string
		expectedModified bool
		expectedContent  string
	}{
		{
			name:             "file with null bytes",
			fileContent:      "hello\x00world",
			expectedModified: true,
			expectedContent:  "hello\x00world\n",
		},
		{
			name:             "file with unicode characters",
			fileContent:      "hello 🌍 world",
			expectedModified: true,
			expectedContent:  "hello 🌍 world\n",
		},
		{
			name:             "file with control characters",
			fileContent:      "hello\tworld\r",
			expectedModified: true,
			expectedContent:  "hello\tworld\r\n",
		},
		{
			name:             "file ending with carriage return and newline",
			fileContent:      "hello world\r\n",
			expectedModified: false,
			expectedContent:  "hello world\r\n",
		},
		{
			name:             "file ending with only carriage return",
			fileContent:      "hello world\r",
			expectedModified: true,
			expectedContent:  "hello world\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")

			// Create test file
			err := os.WriteFile(testFile, []byte(tt.fileContent), 0o600)
			require.NoError(t, err)

			check := NewEOFCheck()
			ctx := context.Background()

			err = check.Run(ctx, []string{testFile})

			if tt.expectedModified {
				require.Error(t, err)
				require.ErrorIs(t, err, prerrors.ErrEOFIssues)
			} else {
				require.NoError(t, err)
			}

			// Verify file content
			content, err := os.ReadFile(testFile) // #nosec G304 -- test file path is controlled
			require.NoError(t, err)
			assert.Equal(t, tt.expectedContent, string(content))
		})
	}
}

func TestEOFCheckLargeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// Create a large file without newline
	largecontent := make([]byte, 1024*1024) // 1MB
	for i := range largecontent {
		largecontent[i] = 'a'
	}

	err := os.WriteFile(testFile, largecontent, 0o600)
	require.NoError(t, err)

	check := NewEOFCheck()
	ctx := context.Background()

	err = check.Run(ctx, []string{testFile})
	require.Error(t, err)
	require.ErrorIs(t, err, prerrors.ErrEOFIssues)

	// Verify newline was added
	content, err := os.ReadFile(testFile) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Len(t, content, len(largecontent)+1)
	assert.Equal(t, byte('\n'), content[len(content)-1])
}

func TestEOFCheckConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple test files
	files := make([]string, 10)
	for i := 0; i < 10; i++ {
		files[i] = filepath.Join(tmpDir, fmt.Sprintf("test%d.txt", i))
		err := os.WriteFile(files[i], []byte(fmt.Sprintf("content %d", i)), 0o600)
		require.NoError(t, err)
	}

	check := NewEOFCheck()
	ctx := context.Background()

	// Run check (should fix all files)
	err := check.Run(ctx, files)
	require.Error(t, err)
	require.ErrorIs(t, err, prerrors.ErrEOFIssues)

	// Verify all files were fixed
	for i, file := range files {
		content, err := os.ReadFile(file) // #nosec G304 -- test file path is controlled
		require.NoError(t, err, "Failed to read file %d", i)
		expected := fmt.Sprintf("content %d\n", i)
		assert.Equal(t, expected, string(content), "File %d content mismatch", i)
	}
}

func TestEOFCheckErrorAccumulation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a valid file
	validFile := filepath.Join(tmpDir, "valid.txt")
	err := os.WriteFile(validFile, []byte("valid content"), 0o600)
	require.NoError(t, err)

	// Create files that will cause read errors
	nonExistentFile1 := filepath.Join(tmpDir, "nonexistent1.txt")
	nonExistentFile2 := filepath.Join(tmpDir, "nonexistent2.txt")

	files := []string{validFile, nonExistentFile1, nonExistentFile2}

	check := NewEOFCheck()
	ctx := context.Background()

	err = check.Run(ctx, files)
	require.Error(t, err)

	// Error should contain information about all failed files
	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "nonexistent1.txt")
	assert.Contains(t, errorMsg, "nonexistent2.txt")
	assert.Contains(t, errorMsg, "failed to read file")

	// Valid file should still have been processed
	content, err := os.ReadFile(validFile) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "valid content\n", string(content))
}

func TestEOFCheckMixedErrorsAndFixes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that needs fixing
	fixableFile := filepath.Join(tmpDir, "fixable.txt")
	err := os.WriteFile(fixableFile, []byte("needs fixing"), 0o600)
	require.NoError(t, err)

	// Create a file that will cause read error
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	files := []string{fixableFile, nonExistentFile}

	check := NewEOFCheck()
	ctx := context.Background()

	err = check.Run(ctx, files)
	require.Error(t, err)

	// Should contain both the read error and indicate EOF issues were found
	errorMsg := err.Error()
	assert.Contains(t, errorMsg, "EOF issues found")
	assert.Contains(t, errorMsg, "failed to read file")
	assert.Contains(t, errorMsg, "nonexistent.txt")

	// Fixable file should have been fixed
	content, err := os.ReadFile(fixableFile) // #nosec G304 -- test file path is controlled
	require.NoError(t, err)
	assert.Equal(t, "needs fixing\n", string(content))
}
