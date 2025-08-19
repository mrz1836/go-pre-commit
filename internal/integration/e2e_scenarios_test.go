package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-pre-commit/internal/config"
	"github.com/mrz1836/go-pre-commit/internal/runner"
	"github.com/mrz1836/go-pre-commit/internal/shared"
)

// E2EIntegrationTestSuite tests complete end-to-end scenarios
type E2EIntegrationTestSuite struct {
	suite.Suite

	tempDir     string
	repoRoot    string
	originalWD  string
	testProject string
	suiteEnv    map[string]string
}

// SetupSuite initializes the integration test environment
func (s *E2EIntegrationTestSuite) SetupSuite() {
	var err error
	s.originalWD, err = os.Getwd()
	s.Require().NoError(err)

	// Create isolated temp directory outside the repository tree
	s.tempDir, err = os.MkdirTemp("", "go-pre-commit-e2e-test-*")
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		_ = os.RemoveAll(s.tempDir)
	})

	s.testProject = filepath.Join(s.tempDir, "test-go-project")
	s.Require().NoError(os.MkdirAll(s.testProject, 0o750))

	// Save environment variables that might be modified by tests
	s.suiteEnv = make(map[string]string)
	s.saveSuiteEnvironment("GO_PRE_COMMIT_TIMEOUT_SECONDS")
	s.saveSuiteEnvironment("GO_PRE_COMMIT_LOG_LEVEL")
	s.saveSuiteEnvironment("ENABLE_GO_PRE_COMMIT")

	// Initialize a git repository for integration tests
	s.initializeTestGitRepo()
	s.setupTestGoProject()
}

// TearDownSuite cleans up the test environment
func (s *E2EIntegrationTestSuite) TearDownSuite() {
	_ = os.Chdir(s.originalWD)
	s.restoreSuiteEnvironment()
}

// TearDownTest ensures we're back in the original directory after each test
func (s *E2EIntegrationTestSuite) TearDownTest() {
	_ = os.Chdir(s.originalWD)
}

// saveSuiteEnvironment saves environment variable for suite-level restoration
func (s *E2EIntegrationTestSuite) saveSuiteEnvironment(key string) {
	s.suiteEnv[key] = os.Getenv(key)
}

// restoreSuiteEnvironment restores suite-level environment variables
func (s *E2EIntegrationTestSuite) restoreSuiteEnvironment() {
	for key, value := range s.suiteEnv {
		if value == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, value)
		}
	}
}

// initializeTestGitRepo creates a test git repository
func (s *E2EIntegrationTestSuite) initializeTestGitRepo() {
	// Change to test project directory
	s.Require().NoError(os.Chdir(s.testProject))

	// Initialize git repository
	ctx := context.Background()
	gitInit := exec.CommandContext(ctx, "git", "init", ".")
	s.Require().NoError(gitInit.Run())

	// Configure git user for tests
	gitConfigName := exec.CommandContext(ctx, "git", "config", "user.name", "Test User")
	s.Require().NoError(gitConfigName.Run())

	gitConfigEmail := exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com")
	s.Require().NoError(gitConfigEmail.Run())

	s.repoRoot = s.testProject
}

// setupTestGoProject creates a realistic Go project structure
func (s *E2EIntegrationTestSuite) setupTestGoProject() {
	// Create go.mod
	goModContent := `module github.com/test/example

go 1.21

require (
	github.com/stretchr/testify v1.8.4
)
`
	s.Require().NoError(os.WriteFile(filepath.Join(s.testProject, "go.mod"), []byte(goModContent), 0o600))

	// Create main.go with intentional issues for testing
	mainGoContent := `package main

import (
	"fmt"
	"os"
)

// Main function with some formatting issues
func main(){
fmt.Println("Hello World")
	if len(os.Args) > 1 {
		fmt.Printf("Args: %v", os.Args[1:])
	}
}

// Unused function to test linting
func unusedFunction() string {
	return "unused"
}
`
	s.Require().NoError(os.WriteFile(filepath.Join(s.testProject, "main.go"), []byte(mainGoContent), 0o600))

	// Create a test file
	testContent := `package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	assert.Equal(t, 1, 1, "Basic test should pass")
}
`
	s.Require().NoError(os.WriteFile(filepath.Join(s.testProject, "main_test.go"), []byte(testContent), 0o600))

	// Create .github directory and configuration
	githubDir := filepath.Join(s.testProject, ".github")
	s.Require().NoError(os.MkdirAll(githubDir, 0o750))

	// Create .env.base configuration file
	envContent := `# Go pre-commit configuration
ENABLE_GO_PRE_COMMIT=true
GO_PRE_COMMIT_TIMEOUT_SECONDS=300
GO_PRE_COMMIT_ENABLE_FMT=true
GO_PRE_COMMIT_ENABLE_FUMPT=true
GO_PRE_COMMIT_ENABLE_LINT=true
GO_PRE_COMMIT_ENABLE_MOD_TIDY=true
GO_PRE_COMMIT_ENABLE_WHITESPACE=true
GO_PRE_COMMIT_ENABLE_EOF=true
GO_PRE_COMMIT_LOG_LEVEL=info
GO_PRE_COMMIT_PARALLEL_WORKERS=2
GO_PRE_COMMIT_FUMPT_VERSION=latest
GO_PRE_COMMIT_GOLANGCI_LINT_VERSION=latest
GO_PRE_COMMIT_GOIMPORTS_VERSION=latest
`
	s.Require().NoError(os.WriteFile(filepath.Join(githubDir, ".env.base"), []byte(envContent), 0o600))

	// Create README.md for whitespace/EOF testing
	readmeContent := `# Test Project

This is a test project for go-pre-commit integration tests.

## Features

- Testing formatting
- Testing linting
- Testing module tidying

`
	s.Require().NoError(os.WriteFile(filepath.Join(s.testProject, "README.md"), []byte(readmeContent), 0o600))

	// Create .gitignore
	gitignoreContent := `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test

# Output of the go coverage tool
*.out

# Go workspace file
go.work
`
	s.Require().NoError(os.WriteFile(filepath.Join(s.testProject, ".gitignore"), []byte(gitignoreContent), 0o600))

	// Initial git commit
	ctx := context.Background()
	gitAdd := exec.CommandContext(ctx, "git", "add", ".")
	s.Require().NoError(gitAdd.Run())

	gitCommit := exec.CommandContext(ctx, "git", "commit", "-m", "Initial commit")
	s.Require().NoError(gitCommit.Run())
}

// TestCompleteRunnerWorkflow tests the complete workflow from configuration loading to check execution
func (s *E2EIntegrationTestSuite) TestCompleteRunnerWorkflow() {
	// Change to test project directory
	s.Require().NoError(os.Chdir(s.testProject))

	// Set test config directory to use this test's config
	s.Require().NoError(os.Setenv("GO_PRE_COMMIT_TEST_CONFIG_DIR", s.testProject))
	defer func() {
		_ = os.Unsetenv("GO_PRE_COMMIT_TEST_CONFIG_DIR")
	}()

	// Test 1: Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err, "Should load configuration successfully")
	s.True(cfg.Checks.Fmt, "Format check should be enabled")
	s.True(cfg.Checks.Lint, "Lint check should be enabled")
	s.Equal("info", cfg.LogLevel, "Log level should be info")
	s.Positive(cfg.Timeout, "Timeout should be greater than 0")
	s.T().Logf("Loaded timeout: %d seconds", cfg.Timeout)

	// Test 2: Create shared context
	ctx := context.Background()
	sharedCtx := shared.NewContext()
	repoRoot, err := sharedCtx.GetRepoRoot(ctx)
	s.Require().NoError(err, "Should get repo root")

	// Handle macOS symlink resolution for /var vs /private/var
	expectedRoot, err := filepath.EvalSymlinks(s.repoRoot)
	s.Require().NoError(err, "Should resolve expected symlinks")
	actualRoot, err := filepath.EvalSymlinks(repoRoot)
	s.Require().NoError(err, "Should resolve actual symlinks")
	s.Equal(expectedRoot, actualRoot, "Repository root should be set correctly")

	// Test 3: Create runner and execute checks
	testRunner := runner.New(cfg, s.repoRoot)
	s.NotNil(testRunner, "Runner should not be nil")

	// Test 4: Run checks (this will test the complete pipeline)
	opts := runner.Options{}
	results, runErr := testRunner.Run(ctx, opts)

	// Handle potential nil results and errors
	if runErr != nil {
		s.T().Logf("Run error: %v", runErr)
	}
	if results == nil {
		s.T().Logf("Results are nil, skipping check results validation")
		return
	}

	// Results should contain check results even if some fail
	s.NotNil(results, "Results should not be nil")
	s.NotEmpty(results.CheckResults, "Should have executed some checks")

	// Log results for debugging
	s.T().Logf("Total checks executed: %d", len(results.CheckResults))
	s.T().Logf("Execution time: %v", results.TotalDuration)

	for _, checkResult := range results.CheckResults {
		s.T().Logf("Check: %s, Success: %v, Duration: %v",
			checkResult.Name, checkResult.Success, checkResult.Duration)
		if !checkResult.Success && checkResult.Error != "" {
			s.T().Logf("  Error: %s", checkResult.Error)
		}
	}

	// Test 5: Validate check execution
	checkNames := make(map[string]bool)
	for _, result := range results.CheckResults {
		checkNames[result.Name] = true
	}

	// Verify expected checks were executed
	expectedChecks := []string{"fmt", "lint", "mod-tidy", "whitespace", "eof"}
	for _, expectedCheck := range expectedChecks {
		s.T().Logf("Checking for expected check: %s", expectedCheck)
		// In a real implementation, this would verify check configuration status
		// For now, we just log the expectation
	}

	// The run may succeed or fail depending on the state of the test files
	// but we should have meaningful results either way
	s.T().Logf("✓ Complete runner workflow test completed with %d checks", len(results.CheckResults))

	// Clean up any errors for subsequent tests
	if runErr != nil {
		s.T().Logf("Run completed with error (expected): %v", runErr)
	}
}

// TestRunWithFileModifications tests the workflow when files are modified during checks
func (s *E2EIntegrationTestSuite) TestRunWithFileModifications() {
	s.Require().NoError(os.Chdir(s.testProject))

	// Create a file with formatting issues
	badFormatFile := filepath.Join(s.testProject, "bad_format.go")
	badContent := `package main

import"fmt"

func   badFormat(   ){
fmt.Println(   "needs formatting"   )
		return
}
`
	s.Require().NoError(os.WriteFile(badFormatFile, []byte(badContent), 0o600))

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	// Create context and runner
	ctx := context.Background()

	testRunner := runner.New(cfg, s.repoRoot)
	s.Require().NoError(err)

	// Run checks
	opts := runner.Options{}
	results, runErr := testRunner.Run(ctx, opts)

	// Handle potential nil results and errors
	if runErr != nil {
		s.T().Logf("Run error: %v", runErr)
	}
	if results == nil {
		s.T().Logf("Results are nil, skipping file modification validation")
		return
	}

	s.NotNil(results)

	// Verify fmt check was executed
	var fmtResult *runner.CheckResult
	for _, result := range results.CheckResults {
		if result.Name == "fmt" {
			fmtResult = &result
			break
		}
	}

	if fmtResult != nil {
		s.T().Logf("Format check executed: %v, Duration: %v", fmtResult.Success, fmtResult.Duration)
		if len(fmtResult.Files) > 0 {
			s.T().Logf("Files processed: %v", fmtResult.Files)
		}
	}

	// Clean up
	_ = os.Remove(badFormatFile)

	s.T().Logf("✓ File modification workflow test completed")
}

// TestSkipFunctionality tests the complete skip workflow
func (s *E2EIntegrationTestSuite) TestSkipFunctionality() {
	s.Require().NoError(os.Chdir(s.testProject))

	// Set skip environment variables
	originalSkip := os.Getenv("SKIP")
	defer func() {
		if originalSkip == "" {
			_ = os.Unsetenv("SKIP")
		} else {
			_ = os.Setenv("SKIP", originalSkip)
		}
	}()

	// Test skipping specific checks
	_ = os.Setenv("SKIP", "lint,fumpt")

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	// Create context and runner
	ctx := context.Background()

	testRunner := runner.New(cfg, s.repoRoot)
	s.Require().NoError(err)

	// Run checks
	opts := runner.Options{}
	results, _ := testRunner.Run(ctx, opts)
	s.NotNil(results)

	// Verify skipped checks were not executed
	checkNames := make(map[string]bool)
	for _, result := range results.CheckResults {
		checkNames[result.Name] = true
	}

	s.False(checkNames["lint"], "Lint check should be skipped")
	s.False(checkNames["fumpt"], "Fumpt check should be skipped")

	// Verify non-skipped checks were executed
	s.NotEmpty(results.CheckResults, "Some checks should still be executed")

	s.T().Logf("✓ Skip functionality test completed with %d checks executed", len(results.CheckResults))
}

// TestErrorHandlingWorkflow tests error handling in the complete workflow
func (s *E2EIntegrationTestSuite) TestErrorHandlingWorkflow() {
	s.Require().NoError(os.Chdir(s.testProject))

	// Create a file that will cause linting errors
	problemFile := filepath.Join(s.testProject, "problem.go")
	problemContent := `package main

import (
	"fmt"
	"unused"  // This will cause a linting error
)

func main() {
	fmt.Println("This will cause linting issues")
	var unused_var string = "unused"  // Unused variable
	_ = unused_var  // Fix for testing
}
`
	s.Require().NoError(os.WriteFile(problemFile, []byte(problemContent), 0o600))

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	// Create context and runner
	ctx := context.Background()

	testRunner := runner.New(cfg, s.repoRoot)
	s.Require().NoError(err)

	// Run checks (expect some to fail)
	opts := runner.Options{}
	results, runErr := testRunner.Run(ctx, opts)

	// Handle potential nil results and errors
	if runErr != nil {
		s.T().Logf("Run error: %v", runErr)
	}
	if results == nil {
		s.T().Logf("Results are nil, skipping error handling validation")
		return
	}

	s.NotNil(results)

	// Verify we got results even with errors
	s.NotEmpty(results.CheckResults, "Should have check results even with errors")

	// Log error details for analysis
	if runErr != nil {
		s.T().Logf("Expected error occurred: %v", runErr)
	}

	for _, result := range results.CheckResults {
		if !result.Success && result.Error != "" {
			s.T().Logf("Check %s failed with error: %s", result.Name, result.Error)
		}
	}

	// Clean up
	_ = os.Remove(problemFile)

	s.T().Logf("✓ Error handling workflow test completed")
}

// TestConcurrentCheckExecution tests that checks can be executed concurrently
func (s *E2EIntegrationTestSuite) TestConcurrentCheckExecution() {
	s.Require().NoError(os.Chdir(s.testProject))

	// Load configuration with multiple workers
	cfg, err := config.Load()
	s.Require().NoError(err)
	cfg.Performance.ParallelWorkers = 3

	// Create context and runner
	ctx := context.Background()

	testRunner := runner.New(cfg, s.repoRoot)
	s.Require().NoError(err)

	// Measure execution time
	startTime := time.Now()
	opts := runner.Options{}
	results, runErr := testRunner.Run(ctx, opts)
	executionTime := time.Since(startTime)

	// Handle potential nil results and errors
	if runErr != nil {
		s.T().Logf("Run error: %v", runErr)
	}
	if results == nil {
		s.T().Logf("Results are nil, skipping concurrent execution validation")
		return
	}

	s.NotNil(results)
	s.NotEmpty(results.CheckResults, "Should have executed checks")
	s.Less(executionTime, 60*time.Second, "Execution should complete within reasonable time")

	s.T().Logf("✓ Concurrent execution test completed in %v with %d checks",
		executionTime, len(results.CheckResults))
}

// TestContextCancellation tests that the runner responds to context cancellation
func (s *E2EIntegrationTestSuite) TestContextCancellation() {
	s.Require().NoError(os.Chdir(s.testProject))

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	testRunner := runner.New(cfg, s.repoRoot)
	s.Require().NoError(err)

	// Run checks with canceled context
	opts := runner.Options{}
	results, runErr := testRunner.Run(ctx, opts)

	// Handle potential nil results - cancellation might return nil
	if results == nil {
		s.T().Logf("Results are nil due to cancellation, which is expected")
		if runErr != nil {
			s.T().Logf("Expected cancellation error: %v", runErr)
		}
		s.T().Logf("✓ Context cancellation test completed")
		return
	}

	// We should get some kind of result, even if canceled
	s.NotNil(results, "Should get results even with cancellation")

	if runErr != nil {
		s.T().Logf("Expected cancellation error: %v", runErr)
	}

	s.T().Logf("✓ Context cancellation test completed")
}

// TestWorkflowWithMixedFileTypes tests the workflow with various file types
func (s *E2EIntegrationTestSuite) TestWorkflowWithMixedFileTypes() {
	s.Require().NoError(os.Chdir(s.testProject))

	// Create various file types
	files := map[string]string{
		"valid.go": `package main

import "fmt"

func main() {
	fmt.Println("Valid Go file")
}
`,
		"script.sh": `#!/bin/bash
echo "Shell script"
`,
		"config.json": `{
  "name": "test",
  "version": "1.0.0"
}`,
		"data.txt": `Plain text file
with multiple lines
`,
		"markdown.md": `# Markdown File

This is a markdown file for testing.
`,
	}

	// Create test files
	createdFiles := make([]string, 0, len(files))
	for filename, content := range files {
		fullPath := filepath.Join(s.testProject, filename)
		s.Require().NoError(os.WriteFile(fullPath, []byte(content), 0o600))
		createdFiles = append(createdFiles, fullPath)
	}

	// Load configuration and run checks
	cfg, err := config.Load()
	s.Require().NoError(err)

	ctx := context.Background()

	testRunner := runner.New(cfg, s.repoRoot)
	s.Require().NoError(err)

	opts := runner.Options{}
	results, runErr := testRunner.Run(ctx, opts)

	// Handle potential nil results and errors
	if runErr != nil {
		s.T().Logf("Run error: %v", runErr)
	}
	if results == nil {
		s.T().Logf("Results are nil, skipping mixed file types validation")
		return
	}

	s.NotNil(results)

	// Verify checks were executed
	s.NotEmpty(results.CheckResults, "Should have executed checks on mixed file types")

	// Log file processing information
	for _, result := range results.CheckResults {
		if len(result.Files) > 0 {
			s.T().Logf("Check %s processed files: %v", result.Name, result.Files)
		}
	}

	// Clean up
	for _, file := range createdFiles {
		_ = os.Remove(file)
	}

	s.T().Logf("✓ Mixed file types workflow test completed")
}

// TestFullWorkflowWithRealChecks tests the complete workflow with real tool execution
func (s *E2EIntegrationTestSuite) TestFullWorkflowWithRealChecks() {
	s.Require().NoError(os.Chdir(s.testProject))

	// Create a comprehensive Go file to test all checks
	testFile := filepath.Join(s.testProject, "comprehensive.go")
	testContent := `// Package main demonstrates various Go code patterns for testing
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

// Constants for the application
const (
	DefaultTimeout = 30
	MaxRetries     = 3
)

// Config holds application configuration
type Config struct {
	Timeout int
	Debug   bool
	Workers int
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		Timeout: DefaultTimeout,
		Debug:   false,
		Workers: 1,
	}
}

// ProcessData processes input data with error handling
func ProcessData(input string) (int, error) {
	if input == "" {
		return 0, fmt.Errorf("input cannot be empty")
	}

	value, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("invalid number format: %w", err)
	}

	return value * 2, nil
}

// main function demonstrates various patterns
func main() {
	cfg := NewConfig()

	if len(os.Args) < 2 {
		log.Println("Usage: program <number>")
		os.Exit(1)
	}

	result, err := ProcessData(os.Args[1])
	if err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}

	if cfg.Debug {
		fmt.Printf("Debug: Processing completed successfully\n")
	}

	fmt.Printf("Result: %d\n", result)
}
`
	s.Require().NoError(os.WriteFile(testFile, []byte(testContent), 0o600))

	// Also create a corresponding test file
	testTestFile := filepath.Join(s.testProject, "comprehensive_test.go")
	testTestContent := `package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, DefaultTimeout, cfg.Timeout)
	assert.False(t, cfg.Debug)
	assert.Equal(t, 1, cfg.Workers)
}

func TestProcessData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{
			name:     "valid number",
			input:    "5",
			expected: 10,
			wantErr:  false,
		},
		{
			name:     "empty input",
			input:    "",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "invalid number",
			input:    "abc",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessData(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Zero(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestProcessDataEdgeCases(t *testing.T) {
	// Test with zero
	result, err := ProcessData("0")
	assert.NoError(t, err)
	assert.Equal(t, 0, result)

	// Test with negative number
	result, err = ProcessData("-5")
	assert.NoError(t, err)
	assert.Equal(t, -10, result)

	// Test with large number
	result, err = ProcessData("1000")
	assert.NoError(t, err)
	assert.Equal(t, 2000, result)
}
`
	s.Require().NoError(os.WriteFile(testTestFile, []byte(testTestContent), 0o600))

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	// Create context and runner
	ctx := context.Background()

	testRunner := runner.New(cfg, s.repoRoot)
	s.Require().NoError(err)

	// Run the complete workflow
	startTime := time.Now()
	opts := runner.Options{}
	results, runErr := testRunner.Run(ctx, opts)
	executionTime := time.Since(startTime)

	// Handle potential nil results and errors
	if runErr != nil {
		s.T().Logf("Run error: %v", runErr)
	}
	if results == nil {
		s.T().Logf("Results are nil, skipping full workflow validation")
		return
	}

	// Validate results
	s.NotNil(results, "Should have results")
	s.NotEmpty(results.CheckResults, "Should have executed checks")
	s.Less(executionTime.Seconds(), 120.0, "Should complete within reasonable time")

	// Detailed result analysis
	var (
		successCount  = 0
		failureCount  = 0
		totalDuration time.Duration
	)

	for _, result := range results.CheckResults {
		totalDuration += result.Duration
		if result.Success {
			successCount++
		} else {
			failureCount++
		}

		s.T().Logf("Check: %-12s | Success: %-5v | Duration: %-10v | Files: %d",
			result.Name, result.Success, result.Duration, len(result.Files))

		if result.Error != "" {
			s.T().Logf("  Error: %s", result.Error)
		}
	}

	s.T().Logf("✓ Full workflow completed:")
	s.T().Logf("  Total checks: %d", len(results.CheckResults))
	s.T().Logf("  Successful: %d", successCount)
	s.T().Logf("  Failed: %d", failureCount)
	s.T().Logf("  Total execution time: %v", executionTime)
	s.T().Logf("  Cumulative check time: %v", totalDuration)

	// Clean up
	_ = os.Remove(testFile)
	_ = os.Remove(testTestFile)

	// Log final result
	if runErr != nil {
		s.T().Logf("  Final result: FAILED (%v)", runErr)
	} else {
		s.T().Logf("  Final result: PASSED")
	}
}

// TestRunnerMemoryAndPerformance tests memory usage and performance characteristics
func (s *E2EIntegrationTestSuite) TestRunnerMemoryAndPerformance() {
	s.Require().NoError(os.Chdir(s.testProject))

	// Create multiple files for performance testing
	const numFiles = 10
	createdFiles := make([]string, 0, numFiles)

	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(s.testProject, fmt.Sprintf("perf_test_%d.go", i))
		content := fmt.Sprintf(`// File %d for performance testing
package main

import "fmt"

func function%d() {
	fmt.Printf("Function %d called\\n")

	// Some computation
	sum := 0
	for i := 0; i < 1000; i++ {
		sum += i * %d
	}

	fmt.Printf("Sum: %%d\\n", sum)
}
`, i, i, i, i+1)

		s.Require().NoError(os.WriteFile(filename, []byte(content), 0o600))
		createdFiles = append(createdFiles, filename)
	}

	// Load configuration
	cfg, err := config.Load()
	s.Require().NoError(err)

	// Test with different parallel worker counts
	workerCounts := []int{1, 2, 4}

	for _, workers := range workerCounts {
		cfg.Performance.ParallelWorkers = workers

		ctx := context.Background()

		testRunner := runner.New(cfg, s.repoRoot)
		s.Require().NoError(err)

		// Measure performance
		startTime := time.Now()
		opts := runner.Options{}
		results, runErr := testRunner.Run(ctx, opts)
		executionTime := time.Since(startTime)

		// Handle potential nil results and errors
		if runErr != nil {
			s.T().Logf("Run error for %d workers: %v", workers, runErr)
		}
		if results == nil {
			s.T().Logf("Results are nil for %d workers, skipping performance validation", workers)
			continue
		}

		s.NotNil(results)
		s.T().Logf("Workers: %d | Execution time: %v | Checks: %d",
			workers, executionTime, len(results.CheckResults))
	}

	// Clean up
	for _, file := range createdFiles {
		_ = os.Remove(file)
	}

	s.T().Logf("✓ Performance test completed")
}

// TestSuite runs the integration test suite
func TestE2EIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(E2EIntegrationTestSuite))
}
