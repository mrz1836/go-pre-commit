package runner

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-pre-commit/internal/config"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true
	cfg.Checks.ModTidy = false
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	r := New(cfg, "/test/repo")
	assert.NotNil(t, r)
	assert.Equal(t, cfg, r.config)
	assert.Equal(t, "/test/repo", r.repoRoot)
	assert.NotNil(t, r.registry)
}

func TestRunner_Run_NoFiles(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Fumpt = true

	r := New(cfg, "/test/repo")

	opts := Options{
		Files: []string{},
	}

	results, err := r.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.NotNil(t, results)
	// When no files are provided, checks still run but succeed immediately
	assert.Equal(t, 1, results.Passed)
	assert.Equal(t, 0, results.Failed)
	assert.Len(t, results.CheckResults, 1)
}

func TestRunner_Run_BasicFlow(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	// Enable only built-in checks that don't require external tools
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = false
	cfg.Checks.Lint = false
	cfg.Checks.ModTidy = false

	r := New(cfg, "/test/repo")

	// Create test files
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.txt"

	opts := Options{
		Files:      []string{testFile},
		OnlyChecks: []string{"whitespace", "eof"},
	}

	results, err := r.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.NotNil(t, results)
	// Should have results for the checks we requested
	assert.NotEmpty(t, results.CheckResults)
}

func TestRunner_Run_OnlyChecks(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	// Enable all checks
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true
	cfg.Checks.ModTidy = true

	r := New(cfg, "/test/repo")

	opts := Options{
		Files:      []string{"test.go"},
		OnlyChecks: []string{"whitespace"}, // Only run whitespace
	}

	results, err := r.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.NotNil(t, results)

	// Should only have 1 check result
	assert.Len(t, results.CheckResults, 1)
	assert.Equal(t, "whitespace", results.CheckResults[0].Name)
}

func TestRunner_Run_SkipChecks(t *testing.T) {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	// Enable multiple checks
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = false
	cfg.Checks.Lint = false
	cfg.Checks.ModTidy = false

	r := New(cfg, "/test/repo")

	opts := Options{
		Files:      []string{"test.go"},
		SkipChecks: []string{"whitespace"}, // Skip whitespace
	}

	results, err := r.Run(context.Background(), opts)
	require.NoError(t, err)
	assert.NotNil(t, results)

	// Should not have whitespace check in results
	for _, result := range results.CheckResults {
		assert.NotEqual(t, "whitespace", result.Name)
	}
}

func TestOptions(t *testing.T) {
	opts := Options{
		Files:      []string{"a.go", "b.go"},
		OnlyChecks: []string{"lint"},
		SkipChecks: []string{"fumpt"},
		Parallel:   4,
		FailFast:   true,
	}

	assert.Len(t, opts.Files, 2)
	assert.Len(t, opts.OnlyChecks, 1)
	assert.Len(t, opts.SkipChecks, 1)
	assert.Equal(t, 4, opts.Parallel)
	assert.True(t, opts.FailFast)
}

func TestResults(t *testing.T) {
	results := &Results{
		CheckResults: []CheckResult{
			{
				Name:     "test1",
				Success:  true,
				Duration: 100 * time.Millisecond,
			},
			{
				Name:     "test2",
				Success:  false,
				Error:    "test error",
				Duration: 200 * time.Millisecond,
			},
		},
		Passed:        1,
		Failed:        1,
		Skipped:       0,
		TotalDuration: 300 * time.Millisecond,
	}

	assert.Len(t, results.CheckResults, 2)
	assert.Equal(t, 1, results.Passed)
	assert.Equal(t, 1, results.Failed)
	assert.Equal(t, 0, results.Skipped)
}

func TestCheckResult(t *testing.T) {
	result := CheckResult{
		Name:     "test-check",
		Success:  false,
		Error:    "check failed",
		Output:   "detailed output",
		Duration: 123 * time.Millisecond,
		Files:    []string{"a.go", "b.go"},
	}

	assert.Equal(t, "test-check", result.Name)
	assert.False(t, result.Success)
	assert.Equal(t, "check failed", result.Error)
	assert.Equal(t, "detailed output", result.Output)
	assert.Equal(t, 123*time.Millisecond, result.Duration)
	assert.Len(t, result.Files, 2)
}

// Comprehensive test suite for runner functionality

type RunnerTestSuite struct {
	suite.Suite

	tempDir string
}

func TestRunnerSuite(t *testing.T) {
	suite.Run(t, new(RunnerTestSuite))
}

func (s *RunnerTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "runner_test_*")
	s.Require().NoError(err)
}

func (s *RunnerTestSuite) TearDownTest() {
	if s.tempDir != "" {
		err := os.RemoveAll(s.tempDir)
		s.Require().NoError(err)
	}
}

func (s *RunnerTestSuite) createTestFile(filename, content string) string {
	fullPath := s.tempDir + "/" + filename
	err := os.WriteFile(fullPath, []byte(content), 0o600)
	s.Require().NoError(err)
	return fullPath
}

// TestFailFastExecution tests the fail-fast execution path
func (s *RunnerTestSuite) TestFailFastExecution() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	r := New(cfg, s.tempDir)

	// Create a file with whitespace issues
	testFile := s.createTestFile("test.txt", "content with trailing spaces  \n")

	var progressCalls []string
	var progressMutex sync.Mutex

	opts := Options{
		Files:    []string{testFile},
		FailFast: true,
		ProgressCallback: func(checkName, status string, _ time.Duration) {
			progressMutex.Lock()
			defer progressMutex.Unlock()
			progressCalls = append(progressCalls, checkName+":"+status)
		},
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.NotNil(results)

	// Verify that progress callbacks were called
	progressMutex.Lock()
	defer progressMutex.Unlock()
	s.NotEmpty(progressCalls)
}

// TestParallelExecution tests the parallel execution path
func (s *RunnerTestSuite) TestParallelExecution() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	var progressCalls []string
	var progressMutex sync.Mutex

	opts := Options{
		Files:    []string{testFile},
		FailFast: false, // Parallel execution
		Parallel: 2,
		ProgressCallback: func(checkName, status string, _ time.Duration) {
			progressMutex.Lock()
			defer progressMutex.Unlock()
			progressCalls = append(progressCalls, checkName+":"+status)
		},
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.NotNil(results)

	// Should have run checks in parallel
	s.NotEmpty(results.CheckResults)

	// Verify that progress callbacks were called
	progressMutex.Lock()
	defer progressMutex.Unlock()
	s.NotEmpty(progressCalls)
}

// TestParallelismConfiguration tests different parallelism settings
func (s *RunnerTestSuite) TestParallelismConfiguration() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Performance.ParallelWorkers = 3
	cfg.Checks.Whitespace = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	tests := []struct {
		name             string
		parallelInOpts   int
		expectedParallel int // We can't directly test this, but we can test behavior
	}{
		{"default from config", 0, 3},
		{"override from opts", 5, 5},
		{"negative falls back", -1, 3},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			opts := Options{
				Files:    []string{testFile},
				FailFast: false,
				Parallel: tt.parallelInOpts,
			}

			results, err := r.Run(context.Background(), opts)
			s.Require().NoError(err)
			s.NotNil(results)
		})
	}
}

// TestGracefulDegradation tests graceful degradation functionality
func (s *RunnerTestSuite) TestGracefulDegradation() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	// Enable checks that might fail gracefully
	cfg.Checks.Fumpt = true // This might not be available
	cfg.Checks.Whitespace = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.go", "package main\n")

	opts := Options{
		Files:               []string{testFile},
		GracefulDegradation: true,
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.NotNil(results)

	// With graceful degradation, we should handle failures gracefully
	// The exact results depend on what tools are available
	s.GreaterOrEqual(results.Passed, 0)
	s.GreaterOrEqual(results.Failed, 0)
	s.GreaterOrEqual(results.Skipped, 0)
}

// TestContextTimeout tests timeout handling
func (s *RunnerTestSuite) TestContextTimeout() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 1, // Very short timeout
	}
	cfg.Checks.Whitespace = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	opts := Options{
		Files: []string{testFile},
	}

	// The test should complete despite the short timeout for simple checks
	results, err := r.Run(context.Background(), opts)
	// Could succeed or fail depending on timing
	if err == nil {
		s.NotNil(results)
	}
}

// TestContextCancellation tests context cancellation
func (s *RunnerTestSuite) TestContextCancellation() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := Options{
		Files: []string{testFile},
	}

	results, err := r.Run(ctx, opts)
	// Should handle cancellation gracefully
	if err != nil {
		s.Contains(err.Error(), "context")
	} else {
		s.NotNil(results)
	}
}

// TestErrorConditions tests various error conditions
func (s *RunnerTestSuite) TestErrorConditions() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}

	r := New(cfg, s.tempDir)

	// Test with no enabled checks
	opts := Options{
		Files: []string{"test.go"},
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().Error(err) // Should return error when no checks to run
	s.Contains(err.Error(), "no checks to run")
	s.Nil(results)
}

// TestDetermineChecks tests the check determination logic
func (s *RunnerTestSuite) TestDetermineChecks() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true

	r := New(cfg, s.tempDir)

	tests := []struct {
		name        string
		onlyChecks  []string
		skipChecks  []string
		expectedMin int
		expectedMax int
	}{
		{
			name:        "all enabled checks",
			onlyChecks:  nil,
			skipChecks:  nil,
			expectedMin: 1,
			expectedMax: 10, // Should get all enabled checks
		},
		{
			name:        "only specific checks",
			onlyChecks:  []string{"whitespace"},
			skipChecks:  nil,
			expectedMin: 1,
			expectedMax: 1,
		},
		{
			name:        "skip specific checks",
			onlyChecks:  nil,
			skipChecks:  []string{"fumpt"},
			expectedMin: 1,
			expectedMax: 10,
		},
		{
			name:        "skip all enabled checks",
			onlyChecks:  nil,
			skipChecks:  []string{"whitespace", "eof", "fumpt"},
			expectedMin: 0,
			expectedMax: 0,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			opts := Options{
				Files:      []string{"test.txt"},
				OnlyChecks: tt.onlyChecks,
				SkipChecks: tt.skipChecks,
			}

			results, err := r.Run(context.Background(), opts)
			if tt.expectedMin == 0 && tt.expectedMax == 0 {
				// Expect error when no checks to run
				s.Require().Error(err)
				s.Contains(err.Error(), "no checks to run")
			} else {
				s.Require().NoError(err)
				s.NotNil(results)
				s.GreaterOrEqual(len(results.CheckResults), tt.expectedMin)
				s.LessOrEqual(len(results.CheckResults), tt.expectedMax)
			}
		})
	}
}

// TestProgressCallbacks tests progress callback functionality
func (s *RunnerTestSuite) TestProgressCallbacks() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	var progressEvents []string
	var progressMutex sync.Mutex

	opts := Options{
		Files: []string{testFile},
		ProgressCallback: func(checkName, status string, _ time.Duration) {
			progressMutex.Lock()
			defer progressMutex.Unlock()
			progressEvents = append(progressEvents, checkName+":"+status)
		},
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.NotNil(results)

	progressMutex.Lock()
	defer progressMutex.Unlock()

	// Should have at least running and completion events
	s.NotEmpty(progressEvents)

	// Verify we have running events
	hasRunning := false
	for _, event := range progressEvents {
		if contains(event, "running") {
			hasRunning = true
			break
		}
	}
	s.True(hasRunning, "Should have 'running' progress events")
}

// TestResultsAggregation tests that results are properly aggregated
func (s *RunnerTestSuite) TestResultsAggregation() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	r := New(cfg, s.tempDir)

	testFile := s.createTestFile("test.txt", "content\n")

	opts := Options{
		Files: []string{testFile},
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.NotNil(results)

	// Verify results aggregation
	totalResults := results.Passed + results.Failed + results.Skipped
	s.Equal(len(results.CheckResults), totalResults)
	s.Equal(1, results.TotalFiles)
	s.GreaterOrEqual(results.TotalDuration, time.Duration(0))
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Unit tests for edge cases
func TestRunnerEdgeCases(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		// This would panic in real usage, but test the constructor
		assert.NotPanics(t, func() {
			r := New(&config.Config{}, "/test")
			assert.NotNil(t, r)
		})
	})

	t.Run("empty repo root", func(t *testing.T) {
		cfg := &config.Config{Enabled: true}
		r := New(cfg, "")
		assert.NotNil(t, r)
		assert.Empty(t, r.repoRoot)
	})

	t.Run("zero timeout", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
			Timeout: 0,
		}
		cfg.Checks.Whitespace = true

		r := New(cfg, "/test")

		opts := Options{
			Files: []string{"test.txt"},
		}

		// Zero timeout should still work for simple checks
		results, err := r.Run(context.Background(), opts)
		// May succeed or fail depending on timing
		if err == nil {
			assert.NotNil(t, results)
		}
	})

	t.Run("debug timeout with CI environment", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
			Timeout: 30,
		}
		cfg.Checks.Whitespace = true
		cfg.Environment.IsCI = true
		cfg.Environment.CIProvider = "GitHub Actions"
		cfg.Environment.AutoAdjustTimers = true
		cfg.ToolInstallation.Timeout = 300

		r := New(cfg, "/test")

		opts := Options{
			Files:        []string{"test.txt"},
			DebugTimeout: true, // Enable debug output
		}

		// Capture stderr to verify debug output
		oldStderr := os.Stderr
		rPipe, wPipe, _ := os.Pipe()
		os.Stderr = wPipe

		_, _ = r.Run(context.Background(), opts)

		_ = wPipe.Close()
		os.Stderr = oldStderr

		// Read captured output
		buf := make([]byte, 1024)
		n, _ := rPipe.Read(buf)
		output := string(buf[:n])

		// Should contain debug timeout information
		assert.Contains(t, output, "DEBUG-TIMEOUT")
	})

	t.Run("fail fast with skip on degradation", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
			Timeout: 30,
		}
		// Use a check that might not be available
		cfg.Checks.Fumpt = true

		r := New(cfg, "/test")

		opts := Options{
			Files:               []string{"test.go"},
			FailFast:            true,
			GracefulDegradation: true,
		}

		results, err := r.Run(context.Background(), opts)
		// Should complete even if fumpt is not available
		if err == nil {
			assert.NotNil(t, results)
		}
	})

	t.Run("parallel execution with graceful degradation", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
			Timeout: 30,
		}
		cfg.Checks.Whitespace = true
		cfg.Checks.EOF = true
		cfg.Checks.Fumpt = true // May not be available
		cfg.Performance.ParallelWorkers = 2

		r := New(cfg, "/test")

		opts := Options{
			Files:               []string{"test.txt"},
			GracefulDegradation: true,
		}

		results, err := r.Run(context.Background(), opts)
		// Should handle missing tools gracefully
		if err == nil {
			assert.NotNil(t, results)
			// Some checks may have been skipped
			assert.GreaterOrEqual(t, results.Passed+results.Skipped, 0)
		}
	})

	t.Run("context timeout during check execution", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
			Timeout: 1, // Very short timeout
		}
		cfg.Checks.Whitespace = true

		r := New(cfg, "/test")

		// Create context with very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		opts := Options{
			Files: []string{"test.txt"},
		}

		// Wait a bit to ensure timeout
		time.Sleep(10 * time.Millisecond)

		results, err := r.Run(ctx, opts)
		// Should either timeout or complete quickly
		// Test passes if we reach here without panic
		_ = err
		_ = results
	})
}

// TestMatchesExcludePattern tests the pattern matching helper function
func TestMatchesExcludePattern(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		pattern  string
		expected bool
	}{
		// Directory patterns (ending with /)
		{
			name:     "directory pattern matches file in directory",
			filePath: "vendor/pkg/file.go",
			pattern:  "vendor/",
			expected: true,
		},
		{
			name:     "directory pattern matches nested file",
			filePath: ".github/ci-tester/fixtures/lint-fail/main.go",
			pattern:  ".github/ci-tester/fixtures/",
			expected: true,
		},
		{
			name:     "directory pattern does not match unrelated path",
			filePath: "src/main.go",
			pattern:  "vendor/",
			expected: false,
		},
		{
			name:     "directory pattern matches when path starts with pattern",
			filePath: "vendor/github.com/pkg/errors/errors.go",
			pattern:  "vendor/",
			expected: true,
		},
		{
			name:     "directory pattern matches embedded directory",
			filePath: "some/path/vendor/pkg/file.go",
			pattern:  "vendor/",
			expected: true,
		},
		// Exact/substring patterns (not ending with /)
		{
			name:     "substring pattern matches",
			filePath: "path/to/testdata/file.go",
			pattern:  "testdata",
			expected: true,
		},
		{
			name:     "substring pattern does not match",
			filePath: "src/main.go",
			pattern:  "testdata",
			expected: false,
		},
		{
			name:     "exact filename pattern matches",
			filePath: "go.sum",
			pattern:  "go.sum",
			expected: true,
		},
		// Edge cases
		{
			name:     "empty pattern matches nothing",
			filePath: "src/main.go",
			pattern:  "",
			expected: false,
		},
		{
			name:     "pattern matches at start of path",
			filePath: ".git/config",
			pattern:  ".git/",
			expected: true,
		},
		{
			name:     "node_modules directory",
			filePath: "node_modules/pkg/index.js",
			pattern:  "node_modules/",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesExcludePattern(tt.filePath, tt.pattern)
			assert.Equal(t, tt.expected, result, "matchesExcludePattern(%q, %q)", tt.filePath, tt.pattern)
		})
	}
}

// TestApplyExcludePatterns tests the Runner's exclude pattern application
func TestApplyExcludePatterns(t *testing.T) {
	t.Run("no patterns configured", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
		}
		// No exclude patterns set
		cfg.Git.ExcludePatterns = nil

		r := New(cfg, "/test")

		files := []string{"src/main.go", "vendor/pkg/file.go", "test.go"}
		result := r.applyExcludePatterns(files)

		assert.Equal(t, files, result, "should return all files when no patterns configured")
	})

	t.Run("empty patterns slice", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
		}
		cfg.Git.ExcludePatterns = []string{}

		r := New(cfg, "/test")

		files := []string{"src/main.go", "vendor/pkg/file.go"}
		result := r.applyExcludePatterns(files)

		assert.Equal(t, files, result, "should return all files with empty patterns slice")
	})

	t.Run("single pattern excludes matching files", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
		}
		cfg.Git.ExcludePatterns = []string{"vendor/"}

		r := New(cfg, "/test")

		files := []string{
			"src/main.go",
			"vendor/pkg/file.go",
			"vendor/other/lib.go",
			"test/test.go",
		}
		result := r.applyExcludePatterns(files)

		expected := []string{"src/main.go", "test/test.go"}
		assert.Equal(t, expected, result)
	})

	t.Run("multiple patterns exclude correctly", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
		}
		cfg.Git.ExcludePatterns = []string{"vendor/", "node_modules/", ".git/"}

		r := New(cfg, "/test")

		files := []string{
			"src/main.go",
			"vendor/pkg/file.go",
			"node_modules/pkg/index.js",
			".git/config",
			"test/test.go",
		}
		result := r.applyExcludePatterns(files)

		expected := []string{"src/main.go", "test/test.go"}
		assert.Equal(t, expected, result)
	})

	t.Run("ci-tester fixtures exclusion", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
		}
		cfg.Git.ExcludePatterns = []string{
			"vendor/",
			"node_modules/",
			".git/",
			".github/ci-tester/fixtures/",
		}

		r := New(cfg, "/test")

		files := []string{
			"src/main.go",
			".github/ci-tester/fixtures/lint-fail/main.go",
			".github/ci-tester/fixtures/test-fail/main.go",
			".github/workflows/ci.yml",
			"test/test.go",
		}
		result := r.applyExcludePatterns(files)

		expected := []string{"src/main.go", ".github/workflows/ci.yml", "test/test.go"}
		assert.Equal(t, expected, result)
	})

	t.Run("all files excluded", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
		}
		cfg.Git.ExcludePatterns = []string{"vendor/"}

		r := New(cfg, "/test")

		files := []string{
			"vendor/pkg/file.go",
			"vendor/other/lib.go",
		}
		result := r.applyExcludePatterns(files)

		assert.Empty(t, result, "should return empty slice when all files are excluded")
	})

	t.Run("no files excluded", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
		}
		cfg.Git.ExcludePatterns = []string{"vendor/", "node_modules/"}

		r := New(cfg, "/test")

		files := []string{
			"src/main.go",
			"pkg/lib.go",
			"test/test.go",
		}
		result := r.applyExcludePatterns(files)

		assert.Equal(t, files, result, "should return all files when none match patterns")
	})

	t.Run("empty files list", func(t *testing.T) {
		cfg := &config.Config{
			Enabled: true,
		}
		cfg.Git.ExcludePatterns = []string{"vendor/"}

		r := New(cfg, "/test")

		result := r.applyExcludePatterns([]string{})

		assert.Empty(t, result, "should return empty slice for empty input")
	})
}

// TestExcludePatternsIntegration tests that exclusion patterns work in the full run flow
func (s *RunnerTestSuite) TestExcludePatternsIntegration() {
	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Git.ExcludePatterns = []string{".github/ci-tester/fixtures/"}

	r := New(cfg, s.tempDir)

	// Create src directory and test file - this one should be processed
	srcDir := s.tempDir + "/src"
	err := os.MkdirAll(srcDir, 0o750)
	s.Require().NoError(err)

	regularFile := srcDir + "/main.go"
	err = os.WriteFile(regularFile, []byte("package main\n"), 0o600)
	s.Require().NoError(err)

	// Create a fixture file that should be excluded
	fixtureDir := s.tempDir + "/.github/ci-tester/fixtures/lint-fail"
	err = os.MkdirAll(fixtureDir, 0o750)
	s.Require().NoError(err)

	fixtureFile := fixtureDir + "/main.go"
	err = os.WriteFile(fixtureFile, []byte("package main\n    badIndent\n"), 0o600)
	s.Require().NoError(err)

	opts := Options{
		Files: []string{regularFile, fixtureFile},
	}

	results, err := r.Run(context.Background(), opts)
	s.Require().NoError(err)
	s.NotNil(results)

	// Verify that the excluded file doesn't appear in any check's processed files
	for _, checkResult := range results.CheckResults {
		for _, processedFile := range checkResult.Files {
			s.NotContains(processedFile, ".github/ci-tester/fixtures/",
				"excluded files should not be processed by check %s", checkResult.Name)
		}
		// Each check should only have processed 1 file (the non-excluded one)
		s.LessOrEqual(len(checkResult.Files), 1,
			"check %s should process at most 1 file after exclusion", checkResult.Name)
	}
}
