package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// FuzzWhitespaceCheck tests the whitespace check with various file contents
func FuzzWhitespaceCheck(f *testing.F) {
	// Seed corpus with various whitespace scenarios
	f.Add("normal line\n")
	f.Add("trailing spaces   \n")
	f.Add("tabs\t\t\n")
	f.Add("mixed   \t  \n")
	f.Add("no newline at end")
	f.Add("multiple\nlines\nwith spaces  \n")
	f.Add("")
	f.Add("\n\n\n")
	f.Add("unicode content: 🚀   \n")
	f.Add("null\x00bytes\x00here  \n")

	f.Fuzz(func(t *testing.T, content string) {
		// Create temporary file with fuzzed content
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		err := os.WriteFile(testFile, []byte(content), 0o600)
		if err != nil {
			t.Skip("Failed to create test file")
		}

		// Test whitespace check - should never panic
		check := NewWhitespaceCheckWithTimeout(5 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Run check - it should handle any input gracefully
		_ = check.Run(ctx, []string{testFile})

		// Verify file still exists and is readable after processing
		if _, statErr := os.Stat(testFile); statErr != nil {
			t.Errorf("File became inaccessible after processing: %v", statErr)
		}

		// Read the file to ensure it's still valid
		if _, readErr := os.ReadFile(testFile); readErr != nil { //nolint:gosec // test file path is controlled
			t.Errorf("File became unreadable after processing: %v", readErr)
		}
	})
}

// FuzzEOFCheck tests the EOF check with various file contents
func FuzzEOFCheck(f *testing.F) {
	// Seed corpus with various EOF scenarios
	f.Add("normal file\n")
	f.Add("no newline")
	f.Add("")
	f.Add("\n")
	f.Add("multiple\n\n\n")
	f.Add("binary\x00data\xff")
	f.Add("very long line: " + string(make([]byte, 10000)))
	f.Add("unicode: 🚀🎯⚡")
	f.Add("control chars: \r\n\t\v\f")

	f.Fuzz(func(t *testing.T, content string) {
		// Create temporary file with fuzzed content
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.go") // Use .go extension to ensure processing

		err := os.WriteFile(testFile, []byte(content), 0o600)
		if err != nil {
			t.Skip("Failed to create test file")
		}

		// Test EOF check - should never panic
		check := NewEOFCheckWithTimeout(5 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Run check - it should handle any input gracefully
		_ = check.Run(ctx, []string{testFile})

		// Verify file still exists and is readable after processing
		if _, statErr := os.Stat(testFile); statErr != nil {
			t.Errorf("File became inaccessible after processing: %v", statErr)
		}

		// Read the file to ensure it's still valid
		finalContent, readErr := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
		if readErr != nil {
			t.Errorf("File became unreadable after processing: %v", readErr)
		}

		// File should not be corrupted or empty (unless it started small/empty)
		if len(content) > 5 && len(finalContent) == 0 {
			t.Error("Non-empty file became empty after EOF check")
		}
	})
}

// FuzzMultipleChecks tests running multiple checks on the same fuzzed content
func FuzzMultipleChecks(f *testing.F) {
	// Seed with problematic content combinations
	f.Add("spaces   \nno newline at end")
	f.Add("\t\t\nnormal\nlines   ")
	f.Add("mixed\t  \n\n  \t")
	f.Add("")
	f.Add("single line no newline")

	f.Fuzz(func(t *testing.T, content string) {
		// Create temporary file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.go")

		err := os.WriteFile(testFile, []byte(content), 0o600)
		if err != nil {
			t.Skip("Failed to create test file")
		}

		// Create both checks with short timeouts
		whitespaceCheck := NewWhitespaceCheckWithTimeout(3 * time.Second)
		eofCheck := NewEOFCheckWithTimeout(3 * time.Second)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Run both checks sequentially - should handle any content
		files := []string{testFile}

		// First check: whitespace
		_ = whitespaceCheck.Run(ctx, files)

		// Verify file integrity between checks
		if _, statErr := os.Stat(testFile); statErr != nil {
			t.Errorf("File became inaccessible after whitespace check: %v", statErr)
		}

		// Second check: EOF
		_ = eofCheck.Run(ctx, files)

		// Final verification
		finalContent, readErr := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
		if readErr != nil {
			t.Errorf("File became unreadable after EOF check: %v", readErr)
		}

		// Content should be processable - allow some transformations but not complete loss
		// Only flag as error if we had substantial content that completely disappeared
		if len(content) > 5 && len(finalContent) == 0 {
			t.Error("File content was completely lost during processing")
		}
	})
}

// FuzzCheckWithBinaryData tests checks against binary/non-text data
func FuzzCheckWithBinaryData(f *testing.F) {
	// Seed with binary-like content
	f.Add([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE})
	f.Add([]byte{0x89, 0x50, 0x4E, 0x47}) // PNG header
	f.Add([]byte{0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Skip extremely large inputs to avoid resource exhaustion
		if len(data) > 100000 {
			t.Skip("Skipping very large input")
		}

		// Create temporary file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "binary.dat")

		err := os.WriteFile(testFile, data, 0o600)
		if err != nil {
			t.Skip("Failed to create test file")
		}

		// Test both checks on binary data
		whitespaceCheck := NewWhitespaceCheckWithTimeout(2 * time.Second)
		eofCheck := NewEOFCheckWithTimeout(2 * time.Second)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		files := []string{testFile}

		// Checks should handle binary data gracefully without corruption
		_ = whitespaceCheck.Run(ctx, files)

		// Verify no corruption occurred
		afterWhitespace, err := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
		if err != nil {
			t.Errorf("Could not read file after whitespace check: %v", err)
		}

		_ = eofCheck.Run(ctx, files)

		// Final verification - file should still be accessible
		finalData, err := os.ReadFile(testFile) //nolint:gosec // test file path is controlled
		if err != nil {
			t.Errorf("Could not read file after EOF check: %v", err)
		}

		// For binary data, we mainly care that it doesn't crash or corrupt
		if len(data) > 0 && len(finalData) == 0 && len(afterWhitespace) > 0 {
			t.Error("Binary file was corrupted or emptied during EOF processing")
		}
	})
}
