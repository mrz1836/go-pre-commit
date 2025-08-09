package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-pre-commit/internal/config"
)

// FuzzParseFileList tests the parseFileList function with various git output formats
func FuzzParseFileList(f *testing.F) {
	// Seed corpus with various git command outputs
	f.Add([]byte("file1.go\nfile2.txt\n"))
	f.Add([]byte(""))
	f.Add([]byte("\n\n\n"))
	f.Add([]byte("single-file.go"))
	f.Add([]byte("path/with/spaces in name.go\n"))
	f.Add([]byte("unicode/🚀file.go\nother.txt"))
	f.Add([]byte("file\x00with\x00nulls.go\n"))
	f.Add([]byte("very/deep/nested/path/file.go\nshallow.txt\n"))
	f.Add([]byte("\t\tfile-with-tabs.go\n  file-with-spaces.txt  \n"))

	f.Fuzz(func(t *testing.T, gitOutput []byte) {
		// Function should never panic regardless of input
		result := parseFileList(gitOutput)

		// Result should always be a slice (possibly empty)
		if result == nil {
			t.Error("parseFileList returned nil instead of empty slice")
		}

		// All returned files should be valid strings
		for i, file := range result {
			if file == "" {
				t.Errorf("parseFileList returned empty string at index %d", i)
			}

			// Files with null bytes might be valid in some contexts, so just note them
			// but don't fail the test - the function handles them as it sees fit
		}
	})
}

// FuzzFileClassifier tests file classification with various file paths
func FuzzFileClassifier(f *testing.F) {
	// Seed corpus with various file paths and names
	f.Add("normal.go")
	f.Add("file with spaces.txt")
	f.Add("unicode🚀file.go")
	f.Add("path/to/deep/file.go")
	f.Add(".hidden")
	f.Add("Makefile")
	f.Add("file.nonexistent_extension")
	f.Add("")
	f.Add("../../../etc/passwd")
	f.Add("file\x00with\x00nulls")

	f.Fuzz(func(t *testing.T, fileName string) {
		// Skip extremely long names to avoid resource exhaustion
		if len(fileName) > 1000 {
			t.Skip("Skipping very long filename")
		}

		// Skip empty or invalid filenames
		if fileName == "" || strings.Contains(fileName, "\x00") {
			t.Skip("Skipping invalid filename")
		}

		// Create temporary file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, filepath.Base(fileName))

		// Create file with some content
		testContent := []byte("package main\n\nfunc main() {}\n")
		err := os.WriteFile(testFile, testContent, 0o600)
		if err != nil {
			t.Skip("Failed to create test file")
		}

		// Create file classifier
		cfg := &config.Config{}
		classifier := NewFileClassifier(cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Classification should never panic
		infos, err := classifier.ClassifyFiles(ctx, []string{testFile})
		// Should handle the file gracefully
		if err != nil {
			// Error is acceptable, but check it's reasonable
			if strings.Contains(err.Error(), "panic") {
				t.Errorf("Classification panicked: %v", err)
			}
		}

		// If successful, verify structure
		if len(infos) > 0 {
			info := infos[0]
			if info.Path == "" {
				t.Error("FileInfo returned empty path")
			}
			if info.Size < 0 {
				t.Error("FileInfo returned negative size")
			}
		}
	})
}

// FuzzRepositoryOperations tests repository operations with various inputs
func FuzzRepositoryOperations(f *testing.F) {
	// Seed corpus with various path inputs
	f.Add("/tmp/test-repo")
	f.Add(".")
	f.Add("..")
	f.Add("/nonexistent/path")
	f.Add("")
	f.Add("path with spaces")
	f.Add("unicode🚀path")
	f.Add("very/deep/nested/repo/path")

	f.Fuzz(func(t *testing.T, repoPath string) {
		// Skip very long paths
		if len(repoPath) > 500 {
			t.Skip("Skipping very long path")
		}

		// Repository creation should never panic
		repo := NewRepository(repoPath)
		if repo == nil {
			t.Error("NewRepository returned nil")
		}

		// Operations may fail but shouldn't panic
		// These operations will likely fail for invalid repos, but shouldn't crash
		_, _ = repo.GetStagedFiles()
		_, _ = repo.GetAllFiles()
	})
}

// FuzzIsTextFile tests text file detection with various file contents
func FuzzIsTextFile(f *testing.F) {
	// Seed with various file content types
	f.Add([]byte("plain text content\n"))
	f.Add([]byte(""))
	f.Add([]byte("unicode content: 🚀🎯⚡\n"))
	f.Add([]byte{0x00, 0x01, 0x02, 0xFF}) // Binary data
	f.Add([]byte("mixed\x00content\n"))
	f.Add([]byte("very long text: " + string(make([]byte, 10000))))
	f.Add([]byte("\x89PNG\r\n\x1a\n")) // PNG header
	f.Add([]byte("control\r\n\t\vchars"))

	f.Fuzz(func(t *testing.T, content []byte) {
		// Skip extremely large files to avoid resource issues
		if len(content) > 100000 {
			t.Skip("Skipping very large content")
		}

		// Create temporary file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.dat")

		err := os.WriteFile(testFile, content, 0o600)
		if err != nil {
			t.Skip("Failed to create test file")
		}

		// Test with file classifier
		cfg := &config.Config{}
		classifier := NewFileClassifier(cfg)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Classification should handle any content without panic
		infos, err := classifier.ClassifyFiles(ctx, []string{testFile})

		// Check for reasonable behavior
		if err == nil && len(infos) > 0 {
			info := infos[0]

			// Binary and text should be mutually exclusive
			if info.IsBinary && info.IsText {
				t.Error("File classified as both binary and text")
			}

			// Empty files have special handling - no specific validation needed
		}
	})
}

// FuzzGetFileExtension tests extension extraction with malformed filenames
func FuzzGetFileExtension(f *testing.F) {
	// Seed with various filename patterns
	f.Add("file.go")
	f.Add("file.tar.gz")
	f.Add("file.")
	f.Add(".hidden")
	f.Add("noextension")
	f.Add("")
	f.Add("file..double.dot")
	f.Add("file with spaces.txt")
	f.Add("unicode🚀.file")
	f.Add("path/to/file.ext")

	f.Fuzz(func(t *testing.T, filename string) {
		// Skip very long names
		if len(filename) > 1000 {
			t.Skip("Skipping very long filename")
		}

		// Extract extension using filepath package (safe function)
		ext := filepath.Ext(filename)

		// Skip edge cases with unrealistically long extensions
		// This includes filenames like ".00000000..." or "file.00000000..." which are valid but unusual
		if len(ext) > 100 {
			t.Skip("Skipping filename with unrealistically long extension")
		}

		// Extension should start with dot if not empty
		if ext != "" && !strings.HasPrefix(ext, ".") {
			t.Errorf("Extension doesn't start with dot: %q from %q", ext, filename)
		}
	})
}
