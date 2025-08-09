package validation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-pre-commit/internal/config"
	"github.com/mrz1836/go-pre-commit/internal/runner"
)

var errParallelGitRootNotFound = errors.New("git root not found")

// ParallelSafetyTestSuite validates thread safety and parallel execution safety
type ParallelSafetyTestSuite struct {
	suite.Suite

	tempDir    string
	envFile    string
	originalWD string
	testFiles  []string
}

// SetupSuite initializes the test environment
func (s *ParallelSafetyTestSuite) SetupSuite() {
	// Robust working directory capture for CI environments
	s.originalWD = s.getSafeWorkingDirectory()

	// Create temporary directory structure
	s.tempDir = s.T().TempDir()

	// Create .github directory
	githubDir := filepath.Join(s.tempDir, ".github")
	s.Require().NoError(os.MkdirAll(githubDir, 0o750))

	// Create comprehensive .env.shared file for parallel testing
	s.envFile = filepath.Join(githubDir, ".env.shared")
	envContent := `# Test environment configuration for parallel safety testing
ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=info
GO_PRE_COMMIT_ENABLE_FUMPT=false
GO_PRE_COMMIT_ENABLE_LINT=false
GO_PRE_COMMIT_ENABLE_MOD_TIDY=false
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
GO_PRE_COMMIT_ENABLE_EOF=true
GO_PRE_COMMIT_TIMEOUT_SECONDS=120
GO_PRE_COMMIT_PARALLEL_WORKERS=4
GO_PRE_COMMIT_WHITESPACE_TIMEOUT=30
GO_PRE_COMMIT_EOF_TIMEOUT=30
`
	s.Require().NoError(os.WriteFile(s.envFile, []byte(envContent), 0o600))

	// Change to temp directory for tests
	s.Require().NoError(os.Chdir(s.tempDir))

	// Initialize git repository
	s.Require().NoError(s.initGitRepo())

	// Create test files for parallel testing
	s.testFiles = s.createTestFiles()
}

// TearDownSuite cleans up the test environment
func (s *ParallelSafetyTestSuite) TearDownSuite() {
	// Restore original working directory
	_ = os.Chdir(s.originalWD)
}

// getSafeWorkingDirectory attempts to get current working directory with fallbacks for CI
func (s *ParallelSafetyTestSuite) getSafeWorkingDirectory() string {
	// First attempt: standard os.Getwd()
	if wd, err := os.Getwd(); err == nil {
		// Verify the directory actually exists and is accessible
		if _, statErr := os.Stat(wd); statErr == nil {
			return wd
		}
	}

	// Fallback 1: Try to find git repository root
	if gitRoot, err := s.findGitRoot(); err == nil {
		// Verify git root exists and is accessible
		if _, statErr := os.Stat(gitRoot); statErr == nil {
			return gitRoot
		}
	}

	// Fallback 2: Use current user's home directory
	if homeDir, err := os.UserHomeDir(); err == nil {
		return homeDir
	}

	// Final fallback: Use temp directory
	return os.TempDir()
}

// findGitRoot attempts to find the git repository root
func (s *ParallelSafetyTestSuite) findGitRoot() (string, error) {
	// Start from current executable's directory if possible
	if exePath, err := os.Executable(); err == nil {
		dir := filepath.Dir(exePath)
		for dir != filepath.Dir(dir) { // Stop at root
			if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
				return dir, nil
			}
			dir = filepath.Dir(dir)
		}
	}

	// Try common project paths relative to GOPATH or GOMOD
	if goPath := os.Getenv("GOPATH"); goPath != "" {
		projectPath := filepath.Join(goPath, "src", "github.com", "mrz1836", "go-pre-commit")
		if _, err := os.Stat(projectPath); err == nil {
			return projectPath, nil
		}
	}

	return "", errParallelGitRootNotFound
}

// initGitRepo initializes a git repository in the temp directory
func (s *ParallelSafetyTestSuite) initGitRepo() error {
	gitDir := filepath.Join(s.tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0o600)
}

// createTestFiles creates a variety of test files for parallel processing
func (s *ParallelSafetyTestSuite) createTestFiles() []string {
	files := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`,
		"service.go": `package main

type Service struct {
	name string
}

func NewService(name string) *Service {
	return &Service{name: name}
}
`,
		"handler.go": `package main

import "net/http"

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
`,
		"model.go": `package main

type User struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}
`,
		"utils.go": `package main

func add(a, b int) int {
	return a + b
}
`,
		"README.md": `# Test Project

This is a test project for parallel safety validation.

## Features

- Parallel execution testing
- Thread safety validation
- Resource management testing
`,
		"CHANGELOG.md": `# Changelog

## v1.0.0
- Initial release
- Parallel execution support
`,
		"config.yaml": `
app:
  name: test-app
  version: 1.0.0
  parallel_workers: 4
`,
		"script.sh": `#!/bin/bash
echo "Test script for parallel execution"
exit 0
`,
		"data.txt": `Line 1
Line 2
Line 3
Line 4
Line 5
`,
		"go.mod": `module test-project

go 1.21
`,
	}

	createdFiles := make([]string, 0, len(files))
	for filename, content := range files {
		filePath := filepath.Join(s.tempDir, filename)
		s.Require().NoError(os.WriteFile(filePath, []byte(content), 0o600))
		createdFiles = append(createdFiles, filename)
	}

	return createdFiles
}

// TestConcurrentRunnerExecution validates that multiple runner instances can execute safely
func (s *ParallelSafetyTestSuite) TestConcurrentRunnerExecution() {
	const numGoroutines = 10
	const numIterations = 5

	// Load configuration once
	cfg, err := config.Load()
	s.Require().NoError(err)

	var wg sync.WaitGroup
	results := make(chan *runner.Results, numGoroutines*numIterations)
	errors := make(chan error, numGoroutines*numIterations)

	// Launch multiple goroutines running the same checks concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				// Create a new runner for each execution
				r := runner.New(cfg, s.tempDir)

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				result, err := r.Run(ctx, runner.Options{
					Files:    s.testFiles,
					Parallel: 2, // Use parallel execution within each runner
				})
				cancel()

				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(results)
	close(errors)

	// Validate results
	allResults := make([]*runner.Results, 0, numGoroutines*numIterations)
	for result := range results {
		allResults = append(allResults, result)
	}

	allErrors := make([]error, 0, numGoroutines*numIterations)
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	// Should have no errors
	s.Empty(allErrors, "Concurrent execution should not produce errors")

	// Should have expected number of results
	expectedResults := numGoroutines * numIterations
	s.Len(allResults, expectedResults, "Should have all expected results")

	// All results should be valid
	for i, result := range allResults {
		s.NotNil(result, "Result %d should not be nil", i)
		s.Positive(result.TotalDuration, "Result %d should have positive duration", i)
	}

	s.T().Logf("Concurrent execution test completed: %d goroutines × %d iterations = %d total executions",
		numGoroutines, numIterations, len(allResults))
}

// TestParallelCheckExecution validates internal parallel check execution safety
func (s *ParallelSafetyTestSuite) TestParallelCheckExecution() {
	testCases := []struct {
		name            string
		parallelWorkers int
		description     string
	}{
		{
			name:            "Single Worker",
			parallelWorkers: 1,
			description:     "Sequential execution for baseline",
		},
		{
			name:            "Multiple Workers",
			parallelWorkers: 4,
			description:     "Parallel execution with multiple workers",
		},
		{
			name:            "Max Workers",
			parallelWorkers: runtime.NumCPU(),
			description:     "Parallel execution with CPU count workers",
		},
		{
			name:            "Excessive Workers",
			parallelWorkers: runtime.NumCPU() * 2,
			description:     "Parallel execution with more workers than CPUs",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Load configuration
			cfg, err := config.Load()
			s.Require().NoError(err)

			// Create runner
			r := runner.New(cfg, s.tempDir)

			// Execute with specified parallelism
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files:    s.testFiles,
				Parallel: tc.parallelWorkers,
			})
			duration := time.Since(start)

			// Validate results
			s.Require().NoError(err, tc.description)
			s.NotNil(result, "Result should not be nil")
			s.Positive(duration, "Execution should take measurable time")

			s.T().Logf("%s: %d workers, duration=%v, checks=%d",
				tc.name, tc.parallelWorkers, duration, len(result.CheckResults))
		})
	}
}

// TestMemoryUsageUnderParallelExecution validates memory usage and cleanup
func (s *ParallelSafetyTestSuite) TestMemoryUsageUnderParallelExecution() {
	// Record initial memory stats
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	const numIterations = 20

	// Run multiple iterations to test memory cleanup
	for i := 0; i < numIterations; i++ {
		r := runner.New(cfg, s.tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, err := r.Run(ctx, runner.Options{
			Files:    s.testFiles,
			Parallel: 4,
		})
		cancel()

		s.Require().NoError(err, "Iteration %d should not fail", i)
		s.Require().NotNil(result, "Result %d should not be nil", i)

		// Occasional GC to help with memory measurement
		if i%5 == 0 {
			runtime.GC()
		}
	}

	// Force GC and measure final memory
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Calculate memory differences (handle case where GC might have run)
	var allocDiff int64
	if memAfter.Alloc > memBefore.Alloc {
		diff := memAfter.Alloc - memBefore.Alloc
		if diff > uint64(int64(^uint64(0)>>1)) { // Check for int64 overflow
			allocDiff = int64(^uint64(0) >> 1) // Max int64 value
		} else {
			allocDiff = int64(diff)
		}
	} else {
		allocDiff = 0 // Memory decreased due to GC, which is fine
	}
	totalAllocDiff := memAfter.TotalAlloc - memBefore.TotalAlloc

	s.T().Logf("Memory usage: before=%d, after=%d, diff=%d, total_alloc_diff=%d",
		memBefore.Alloc, memAfter.Alloc, allocDiff, totalAllocDiff)

	// Memory should not grow excessively (allow reasonable buffer)
	maxAllowedGrowth := int64(50 * 1024 * 1024) // 50MB
	s.Less(allocDiff, maxAllowedGrowth,
		"Memory growth should be reasonable: %d bytes (max: %d)", allocDiff, maxAllowedGrowth)
}

// TestResourceCleanupUnderParallelExecution validates resource cleanup
func (s *ParallelSafetyTestSuite) TestResourceCleanupUnderParallelExecution() {
	// Count initial goroutines
	initialGoroutines := runtime.NumGoroutine()

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	const numIterations = 10

	// Run multiple iterations with parallel execution
	for i := 0; i < numIterations; i++ {
		r := runner.New(cfg, s.tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, err := r.Run(ctx, runner.Options{
			Files:    s.testFiles,
			Parallel: 4,
		})
		cancel()

		s.Require().NoError(err, "Iteration %d should not fail", i)
		s.Require().NotNil(result, "Result %d should not be nil", i)

		// Brief pause to allow cleanup
		time.Sleep(10 * time.Millisecond)
	}

	// Allow time for cleanup
	time.Sleep(100 * time.Millisecond)
	runtime.GC()

	// Count final goroutines
	finalGoroutines := runtime.NumGoroutine()

	s.T().Logf("Goroutines: initial=%d, final=%d, diff=%d",
		initialGoroutines, finalGoroutines, finalGoroutines-initialGoroutines)

	// Goroutine count should not grow significantly
	// Allow some tolerance for test environment variance
	maxAllowedGrowth := 5
	goroutineGrowth := finalGoroutines - initialGoroutines
	s.LessOrEqual(goroutineGrowth, maxAllowedGrowth,
		"Goroutine count should not grow excessively: %d (max: %d)",
		goroutineGrowth, maxAllowedGrowth)
}

// TestRaceConditionDetection validates absence of race conditions
func (s *ParallelSafetyTestSuite) TestRaceConditionDetection() {
	// This test should be run with -race flag to detect race conditions
	// go test -race ./internal/validation

	const numGoroutines = 20
	var wg sync.WaitGroup

	// Load configuration once and share among goroutines
	cfg, err := config.Load()
	s.Require().NoError(err)

	// Shared state that might cause race conditions
	var executionCount int64
	var mutex sync.Mutex
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Create runner (this should be safe)
			r := runner.New(cfg, s.tempDir)

			// Execute check (this should be safe)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			result, err := r.Run(ctx, runner.Options{
				Files:    s.testFiles,
				Parallel: 2,
			})
			cancel()

			// Update shared state safely
			mutex.Lock()
			executionCount++
			mutex.Unlock()

			// Check error outside goroutine to avoid testifylint go-require violation
			if err != nil {
				errors <- fmt.Errorf("goroutine %d failed: %w", id, err)
				return
			}
			if result == nil {
				errors <- fmt.Errorf("result from goroutine %d should not be nil", id) //nolint:err113 // test-specific error
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	errorList := make([]error, 0, numGoroutines)
	for err := range errors {
		errorList = append(errorList, err)
	}
	s.Empty(errorList, "No goroutines should have errors")

	// Validate final state
	mutex.Lock()
	finalCount := executionCount
	mutex.Unlock()

	s.Equal(int64(numGoroutines), finalCount,
		"All goroutines should have executed successfully")

	s.T().Logf("Race condition test completed: %d concurrent executions", finalCount)
}

// TestContextCancellationSafety validates proper context cancellation handling
func (s *ParallelSafetyTestSuite) TestContextCancellationSafety() {
	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	testCases := []struct {
		name        string
		timeout     time.Duration
		description string
	}{
		{
			name:        "Immediate Cancellation",
			timeout:     1 * time.Millisecond,
			description: "Context canceled almost immediately",
		},
		{
			name:        "Short Timeout",
			timeout:     100 * time.Millisecond,
			description: "Context canceled after short timeout",
		},
		{
			name:        "Medium Timeout",
			timeout:     1 * time.Second,
			description: "Context canceled after medium timeout",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			r := runner.New(cfg, s.tempDir)

			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files:    s.testFiles,
				Parallel: 4,
			})
			duration := time.Since(start)

			// Should handle cancellation gracefully
			if err != nil {
				// Context cancellation is expected and acceptable
				s.Contains(err.Error(), "context",
					"Error should be context-related")
			}

			// Should not take significantly longer than timeout
			maxDuration := tc.timeout + 2*time.Second // Allow reasonable buffer
			s.LessOrEqual(duration, maxDuration,
				"Execution should respect timeout: %v (max: %v)", duration, maxDuration)

			s.T().Logf("%s: timeout=%v, duration=%v, canceled=%v",
				tc.name, tc.timeout, duration, err != nil)

			// Result might be nil or partial on cancellation - both are valid
			if result != nil {
				s.T().Logf("Partial result received with %d checks", len(result.CheckResults))
			}
		})
	}
}

// TestParallelExecutionConsistency validates that parallel execution produces consistent results
func (s *ParallelSafetyTestSuite) TestParallelExecutionConsistency() {
	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	const numRuns = 10
	var results []*runner.Results

	// Run the same checks multiple times with parallel execution
	for i := 0; i < numRuns; i++ {
		r := runner.New(cfg, s.tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		result, err := r.Run(ctx, runner.Options{
			Files:    s.testFiles,
			Parallel: 4,
		})
		cancel()

		s.Require().NoError(err, "Run %d should not fail", i)
		s.Require().NotNil(result, "Result %d should not be nil", i)

		results = append(results, result)
	}

	// Validate consistency across runs
	firstResult := results[0]
	for i, result := range results[1:] {
		// Should have same number of checks
		s.Len(result.CheckResults, len(firstResult.CheckResults),
			"Run %d should have same number of checks as first run", i+1)

		// Should have same total file count
		s.Equal(firstResult.TotalFiles, result.TotalFiles,
			"Run %d should process same number of files", i+1)

		// Check results should be consistent (names should exist, order doesn't matter)
		firstCheckNames := make(map[string]bool)
		for _, checkResult := range firstResult.CheckResults {
			firstCheckNames[checkResult.Name] = true
		}

		currentCheckNames := make(map[string]bool)
		for _, checkResult := range result.CheckResults {
			currentCheckNames[checkResult.Name] = true
		}

		s.Equal(firstCheckNames, currentCheckNames,
			"Run %d should have same check names as first run (order may vary)", i+1)
	}

	s.T().Logf("Consistency test completed: %d runs, %d checks per run",
		numRuns, len(firstResult.CheckResults))
}

// TestParallelExecutionUnderLoad validates behavior under high load
func (s *ParallelSafetyTestSuite) TestParallelExecutionUnderLoad() {
	// Create additional test files to increase load
	largeTestFiles := make([]string, 0, len(s.testFiles)+20)
	largeTestFiles = append(largeTestFiles, s.testFiles...)

	// Generate additional files
	for i := 0; i < 20; i++ {
		filename := filepath.Join(s.tempDir, "generated_"+string(rune('A'+i))+".md")
		content := "# Generated Test File " + string(rune('A'+i)) + "\n\nContent for testing.\n"
		s.Require().NoError(os.WriteFile(filename, []byte(content), 0o600))
		largeTestFiles = append(largeTestFiles, "generated_"+string(rune('A'+i))+".md")
	}

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	// Test with different load levels
	testCases := []struct {
		name            string
		files           []string
		parallelWorkers int
		description     string
	}{
		{
			name:            "Normal Load",
			files:           s.testFiles,
			parallelWorkers: 2,
			description:     "Normal file count with moderate parallelism",
		},
		{
			name:            "High Load - Many Files",
			files:           largeTestFiles,
			parallelWorkers: 4,
			description:     "High file count with high parallelism",
		},
		{
			name:            "High Load - Max Workers",
			files:           largeTestFiles,
			parallelWorkers: runtime.NumCPU(),
			description:     "High file count with maximum workers",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			r := runner.New(cfg, s.tempDir)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			start := time.Now()
			result, err := r.Run(ctx, runner.Options{
				Files:    tc.files,
				Parallel: tc.parallelWorkers,
			})
			duration := time.Since(start)

			// Should complete successfully even under load
			s.Require().NoError(err, tc.description)
			s.NotNil(result, "Result should not be nil")
			s.Positive(duration, "Should have measurable duration")

			s.T().Logf("%s: %d files, %d workers, duration=%v",
				tc.name, len(tc.files), tc.parallelWorkers, duration)
		})
	}
}

// TestParallelExecutionErrorHandling validates error handling in parallel scenarios
func (s *ParallelSafetyTestSuite) TestParallelExecutionErrorHandling() {
	// Create configuration with very short timeouts to trigger errors
	githubDir := filepath.Join(s.tempDir, ".github")
	envFile := filepath.Join(githubDir, ".env.shared")
	shortTimeoutConfig := `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=info
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
GO_PRE_COMMIT_ENABLE_EOF=true
GO_PRE_COMMIT_TIMEOUT_SECONDS=1
GO_PRE_COMMIT_WHITESPACE_TIMEOUT=1
GO_PRE_COMMIT_EOF_TIMEOUT=1
GO_PRE_COMMIT_PARALLEL_WORKERS=4
`
	s.Require().NoError(os.WriteFile(envFile, []byte(shortTimeoutConfig), 0o600))

	// Load the configuration with short timeouts
	cfg, err := config.Load()
	s.Require().NoError(err)

	const numGoroutines = 5
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	results := make(chan *runner.Results, numGoroutines)

	// Launch multiple goroutines that may encounter timeouts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()

			r := runner.New(cfg, s.tempDir)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := r.Run(ctx, runner.Options{
				Files:    s.testFiles,
				Parallel: 4,
			})

			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(results)

	// Collect results
	allErrors := make([]error, 0, numGoroutines)
	allResults := make([]*runner.Results, 0, numGoroutines)

	for err := range errors {
		allErrors = append(allErrors, err)
	}

	for result := range results {
		allResults = append(allResults, result)
	}

	// Should handle errors gracefully without crashing
	totalExecutions := len(allErrors) + len(allResults)
	s.Equal(numGoroutines, totalExecutions,
		"All executions should complete (with success or error)")

	s.T().Logf("Error handling test: %d errors, %d successes out of %d executions",
		len(allErrors), len(allResults), numGoroutines)

	// Restore original configuration
	originalConfig := `ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_LOG_LEVEL=info
GO_PRE_COMMIT_ENABLE_FUMPT=false
GO_PRE_COMMIT_ENABLE_LINT=false
GO_PRE_COMMIT_ENABLE_MOD_TIDY=false
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
GO_PRE_COMMIT_ENABLE_EOF=true
GO_PRE_COMMIT_TIMEOUT_SECONDS=120
GO_PRE_COMMIT_PARALLEL_WORKERS=4
`
	s.Require().NoError(os.WriteFile(envFile, []byte(originalConfig), 0o600))
}

// TestSuite runs the parallel safety test suite
func TestParallelSafetyTestSuite(t *testing.T) {
	suite.Run(t, new(ParallelSafetyTestSuite))
}
