package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-pre-commit/internal/config"
)

// FuzzRunnerOptions tests the runner with various option configurations
func FuzzRunnerOptions(f *testing.F) {
	// Seed corpus with various option scenarios
	f.Add(int(0), false, "fumpt", "")               // No parallel, no fail-fast
	f.Add(int(1), true, "fumpt,lint", "whitespace") // Single worker, fail-fast
	f.Add(int(-1), false, "", "fumpt,lint")         // Invalid parallel count
	f.Add(int(100), true, "invalid", "")            // Too many workers
	f.Add(int(2), false, "", "")                    // Normal case

	f.Fuzz(func(t *testing.T, parallel int, failFast bool, onlyChecks, skipChecks string) {
		// Create temporary directory with test files
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.go")
		err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0o600)
		if err != nil {
			t.Skip("Failed to create test file")
		}

		// Create configuration
		cfg := &config.Config{
			Enabled: true,
			Timeout: 30,
		}
		cfg.Checks.Fumpt = true
		cfg.Checks.Lint = false // Disable lint to avoid external dependencies
		cfg.Checks.ModTidy = false
		cfg.Checks.Whitespace = true
		cfg.Checks.EOF = true

		// Create runner
		runner := New(cfg, tmpDir)

		// Parse check lists (handle invalid input gracefully)
		var onlyList, skipList []string
		if strings.TrimSpace(onlyChecks) != "" {
			onlyList = strings.Split(onlyChecks, ",")
		}
		if strings.TrimSpace(skipChecks) != "" {
			skipList = strings.Split(skipChecks, ",")
		}

		// Create options with fuzzed parameters
		opts := Options{
			Files:      []string{testFile},
			OnlyChecks: onlyList,
			SkipChecks: skipList,
			Parallel:   parallel,
			FailFast:   failFast,
		}

		// Run should handle any configuration gracefully
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		results, err := runner.Run(ctx, opts)
		// Verify runner doesn't crash with invalid configs
		if err != nil {
			// Some errors are expected for invalid configurations
			if strings.Contains(err.Error(), "panic") {
				t.Errorf("Runner panicked with options: parallel=%d, failFast=%v, only=%s, skip=%s: %v",
					parallel, failFast, onlyChecks, skipChecks, err)
			}
		}

		// If successful, verify results structure
		if results != nil {
			if results.Passed < 0 || results.Failed < 0 || results.Skipped < 0 {
				t.Error("Results contain negative counts")
			}

			if results.TotalDuration < 0 {
				t.Error("Results contain negative duration")
			}
		}
	})
}

// FuzzRunnerWithInvalidFiles tests runner behavior with malformed file paths
func FuzzRunnerWithInvalidFiles(f *testing.F) {
	// Seed with various file path scenarios
	f.Add("normal.go")
	f.Add("")
	f.Add("nonexistent.go")
	f.Add("../../../etc/passwd")
	f.Add("file with spaces.go")
	f.Add("unicode🚀file.go")
	f.Add("file1.go,file2.go,nonexistent.go")

	f.Fuzz(func(t *testing.T, filePathsStr string) {
		// Parse comma-separated file paths
		var filePaths []string
		if strings.TrimSpace(filePathsStr) != "" {
			filePaths = strings.Split(filePathsStr, ",")
		}

		// Skip extremely large file lists
		if len(filePaths) > 100 {
			t.Skip("Skipping very large file list")
		}

		// Create configuration
		cfg := &config.Config{
			Enabled: true,
			Timeout: 10,
		}
		cfg.Checks.Whitespace = true
		cfg.Checks.EOF = true

		// Create temporary directory
		tmpDir := t.TempDir()
		runner := New(cfg, tmpDir)

		// Process file paths - create valid ones, leave invalid ones as-is
		var testFiles []string
		for _, filePath := range filePaths {
			if filePath == "" || strings.Contains(filePath, "\x00") {
				// Add invalid path directly to test error handling
				testFiles = append(testFiles, filePath)
				continue
			}

			// Create actual file for valid-looking paths
			if !strings.Contains(filePath, "../") && !strings.HasPrefix(filePath, "/") {
				cleanPath := filepath.Clean(filepath.Base(filePath))
				if cleanPath != "" && cleanPath != "." && cleanPath != ".." {
					testFile := filepath.Join(tmpDir, cleanPath)
					_ = os.WriteFile(testFile, []byte("test content\n"), 0o600)
					testFiles = append(testFiles, testFile)
				}
			} else {
				// Add problematic path to test error handling
				testFiles = append(testFiles, filePath)
			}
		}

		// Create options
		opts := Options{
			Files:    testFiles,
			Parallel: 1,
			FailFast: false,
		}

		// Run should handle invalid file lists gracefully
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results, err := runner.Run(ctx, opts)

		// Runner should not panic regardless of input
		if err != nil && strings.Contains(err.Error(), "panic") {
			t.Errorf("Runner panicked with files %v: %v", filePathsStr, err)
		}

		// If results returned, verify basic structure
		if results != nil && results.TotalFiles < 0 {
			t.Error("Results reported negative file count")
		}
	})
}

// FuzzProgressCallback tests progress callback handling with various inputs
func FuzzProgressCallback(f *testing.F) {
	// Seed with various callback scenarios
	f.Add("fumpt", "starting")
	f.Add("", "")
	f.Add("very long check name that might cause issues", "status")
	f.Add("check\x00name", "status\x00with\x00nulls")
	f.Add("unicode🚀check", "unicode🎯status")

	f.Fuzz(func(t *testing.T, checkName, status string) {
		// Skip very long inputs
		if len(checkName) > 1000 || len(status) > 1000 {
			t.Skip("Skipping very long inputs")
		}

		// Create configuration
		cfg := &config.Config{
			Enabled: true,
			Timeout: 5,
		}
		cfg.Checks.Whitespace = true

		// Create temporary file and runner
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test content\n"), 0o600)
		if err != nil {
			t.Skip("Failed to create test file")
		}

		runner := New(cfg, tmpDir)

		// Create progress callback that shouldn't panic
		callbackCalled := false
		progressCallback := func(name, stat string) {
			callbackCalled = true
			// Callback should handle any input without panicking
			// Just use the variables to avoid unused warnings
			_ = name
			_ = stat
		}

		// Create options with callback
		opts := Options{
			Files:            []string{testFile},
			Parallel:         1,
			ProgressCallback: progressCallback,
		}

		// Run with progress callback
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, err = runner.Run(ctx, opts)

		// Should not panic regardless of callback behavior
		if err != nil && strings.Contains(err.Error(), "panic") {
			t.Errorf("Runner panicked with progress callback: %v", err)
		}

		// Use the callback variable to avoid unused variable warning
		_ = callbackCalled
	})
}

// FuzzRunnerTimeout tests timeout handling with various timeout values
func FuzzRunnerTimeout(f *testing.F) {
	// Seed with various timeout scenarios
	f.Add(int64(0))      // No timeout
	f.Add(int64(1))      // Very short timeout
	f.Add(int64(-1))     // Invalid timeout
	f.Add(int64(300))    // Normal timeout
	f.Add(int64(999999)) // Very long timeout

	f.Fuzz(func(t *testing.T, timeoutSeconds int64) {
		// Skip extreme values to avoid resource issues
		if timeoutSeconds > 3600 || timeoutSeconds < -100 {
			t.Skip("Skipping extreme timeout values")
		}

		// Create configuration with fuzzed timeout
		cfg := &config.Config{
			Enabled: true,
			Timeout: int(timeoutSeconds),
		}
		cfg.Checks.Whitespace = true

		// Create test file
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		err := os.WriteFile(testFile, []byte("content\n"), 0o600)
		if err != nil {
			t.Skip("Failed to create test file")
		}

		runner := New(cfg, tmpDir)

		// Use reasonable context timeout regardless of config timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		opts := Options{
			Files:    []string{testFile},
			Parallel: 1,
		}

		// Runner should handle invalid timeout configurations gracefully
		results, err := runner.Run(ctx, opts)

		// Should not panic with timeout issues
		if err != nil && strings.Contains(err.Error(), "panic") {
			t.Errorf("Runner panicked with timeout %d: %v", timeoutSeconds, err)
		}

		// Verify results structure if returned
		if results != nil && results.TotalDuration < 0 {
			t.Error("Results contain negative total duration")
		}
	})
}

// FuzzRunnerConcurrency tests concurrent runner execution
func FuzzRunnerConcurrency(f *testing.F) {
	// Seed with various concurrency scenarios
	f.Add(int(1))  // Single threaded
	f.Add(int(2))  // Normal parallel
	f.Add(int(0))  // Auto-detect
	f.Add(int(-1)) // Invalid
	f.Add(int(50)) // High concurrency

	f.Fuzz(func(t *testing.T, workers int) {
		// Limit workers to reasonable range
		if workers > 20 || workers < -5 {
			t.Skip("Skipping extreme worker count")
		}

		// Create configuration
		cfg := &config.Config{
			Enabled: true,
			Timeout: 10,
		}
		cfg.Performance.ParallelWorkers = workers
		cfg.Checks.Whitespace = true
		cfg.Checks.EOF = true

		// Create multiple test files
		tmpDir := t.TempDir()
		var testFiles []string
		for i := 0; i < 3; i++ {
			testFile := filepath.Join(tmpDir, "test"+string(rune(i))+"test.txt")
			err := os.WriteFile(testFile, []byte("content\n"), 0o600)
			if err != nil {
				continue
			}
			testFiles = append(testFiles, testFile)
		}

		if len(testFiles) == 0 {
			t.Skip("Failed to create test files")
		}

		runner := New(cfg, tmpDir)

		opts := Options{
			Files:    testFiles,
			Parallel: workers,
		}

		// Test concurrent execution
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		results, err := runner.Run(ctx, opts)

		// Should handle concurrency without panicking
		if err != nil && strings.Contains(err.Error(), "panic") {
			t.Errorf("Runner panicked with %d workers: %v", workers, err)
		}

		// Basic result validation
		if results != nil {
			if results.Passed < 0 || results.Failed < 0 || results.Skipped < 0 {
				t.Error("Invalid result counts from concurrent execution")
			}
		}
	})
}
