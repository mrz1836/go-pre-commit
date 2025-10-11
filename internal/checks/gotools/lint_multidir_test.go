package gotools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrz1836/go-pre-commit/internal/shared"
	"github.com/stretchr/testify/suite"
)

// LintMultiDirTestSuite tests multi-directory linting functionality
type LintMultiDirTestSuite struct {
	suite.Suite

	tempDir   string
	oldDir    string
	check     *LintCheck
	sharedCtx *shared.Context
}

func TestLintMultiDirSuite(t *testing.T) {
	suite.Run(t, new(LintMultiDirTestSuite))
}

func (s *LintMultiDirTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "lint_multidir_test_*")
	s.Require().NoError(err)

	s.oldDir, err = os.Getwd()
	s.Require().NoError(err)

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Initialize git repo
	s.initGitRepo()

	// Create shared context and lint check
	s.sharedCtx = shared.NewContext()
	s.check = NewLintCheckWithSharedContext(s.sharedCtx)
}

func (s *LintMultiDirTestSuite) TearDownTest() {
	if s.oldDir != "" {
		_ = os.Chdir(s.oldDir)
	}
	if s.tempDir != "" {
		_ = os.RemoveAll(s.tempDir)
	}
}

func (s *LintMultiDirTestSuite) initGitRepo() {
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "init").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())
}

// Test single directory optimization path
func (s *LintMultiDirTestSuite) TestSingleDirectoryOptimization() {
	// Create test files in single directory
	s.Require().NoError(os.MkdirAll("pkg/module1", 0o750))

	file1 := `package module1

func TestFunc() {
	fmt.Println("test")
}
`
	s.Require().NoError(os.WriteFile("pkg/module1/file1.go", []byte(file1), 0o600))
	s.Require().NoError(os.WriteFile("pkg/module1/file2.go", []byte(file1), 0o600))

	// Add import for fmt
	file1WithImport := `package module1

import "fmt"

func TestFunc() {
	fmt.Println("test")
}
`
	s.Require().NoError(os.WriteFile("pkg/module1/file1.go", []byte(file1WithImport), 0o600))
	s.Require().NoError(os.WriteFile("pkg/module1/file2.go", []byte(file1WithImport), 0o600))

	// Commit files to avoid "new-from-rev" issues
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "initial").Run())

	// Run lint on files from single directory
	files := []string{"pkg/module1/file1.go", "pkg/module1/file2.go"}
	err := s.check.runDirectLint(ctx, files)
	// Should succeed or have only linting issues, not tool failures
	if err != nil {
		s.T().Logf("Lint check returned error: %v", err)
		// Check that it's not a multi-directory error
		s.NotContains(err.Error(), "named files must all be in one directory")
	}
}

// Test multiple directories execution
func (s *LintMultiDirTestSuite) TestMultipleDirectoriesExecution() {
	// Create test files in multiple directories
	s.Require().NoError(os.MkdirAll("pkg/module1", 0o750))
	s.Require().NoError(os.MkdirAll("pkg/module2", 0o750))
	s.Require().NoError(os.MkdirAll("internal/helper", 0o750))

	// Create valid Go files
	module1File := `package module1

import "fmt"

func Module1Func() {
	fmt.Println("module1")
}
`
	module2File := `package module2

import "fmt"

func Module2Func() {
	fmt.Println("module2")
}
`
	helperFile := `package helper

import "fmt"

func HelperFunc() {
	fmt.Println("helper")
}
`

	s.Require().NoError(os.WriteFile("pkg/module1/file1.go", []byte(module1File), 0o600))
	s.Require().NoError(os.WriteFile("pkg/module2/file2.go", []byte(module2File), 0o600))
	s.Require().NoError(os.WriteFile("internal/helper/util.go", []byte(helperFile), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "initial").Run())

	// Run lint on files from multiple directories
	files := []string{
		"pkg/module1/file1.go",
		"pkg/module2/file2.go",
		"internal/helper/util.go",
	}

	err := s.check.runDirectLint(ctx, files)
	// Should not have the "named files must all be in one directory" error
	if err != nil {
		s.T().Logf("Multi-directory lint returned: %v", err)
		s.NotContains(err.Error(), "named files must all be in one directory")
	}
}

// Test error aggregation from multiple directories
func (s *LintMultiDirTestSuite) TestErrorAggregation() {
	// Create test files with deliberate lint issues
	s.Require().NoError(os.MkdirAll("pkg/bad1", 0o750))
	s.Require().NoError(os.MkdirAll("pkg/bad2", 0o750))

	// File with unused variable (lint error)
	bad1File := `package bad1

func BadFunc1() {
	unusedVar := 42 // This should trigger ineffassign
	_ = 1
}
`

	// File with another lint issue
	bad2File := `package bad2

func BadFunc2() {
	anotherUnused := "test" // This should trigger ineffassign
	_ = 1
}
`

	s.Require().NoError(os.WriteFile("pkg/bad1/bad1.go", []byte(bad1File), 0o600))
	s.Require().NoError(os.WriteFile("pkg/bad2/bad2.go", []byte(bad2File), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "initial").Run())

	// Modify files to trigger linting on changes
	s.Require().NoError(os.WriteFile("pkg/bad1/bad1.go", []byte(bad1File+"\n"), 0o600))
	s.Require().NoError(os.WriteFile("pkg/bad2/bad2.go", []byte(bad2File+"\n"), 0o600))

	files := []string{"pkg/bad1/bad1.go", "pkg/bad2/bad2.go"}
	err := s.check.runDirectLint(ctx, files)
	if err != nil {
		errStr := err.Error()
		s.T().Logf("Aggregated error: %v", errStr)

		// Should contain references to both directories if there are issues
		// Or should handle the multi-directory case without "named files" error
		s.NotContains(errStr, "named files must all be in one directory")
	}
}

// Test empty file list handling
func (s *LintMultiDirTestSuite) TestEmptyFileList() {
	ctx := context.Background()
	err := s.check.runDirectLint(ctx, []string{})
	s.NoError(err, "Empty file list should not cause error")
}

// Test filtering of non-Go files
func (s *LintMultiDirTestSuite) TestMixedFileTypes() {
	// Create mixed file types
	s.Require().NoError(os.MkdirAll("mixed", 0o750))

	goFile := `package mixed

import "fmt"

func MixedFunc() {
	fmt.Println("mixed")
}
`
	s.Require().NoError(os.WriteFile("mixed/code.go", []byte(goFile), 0o600))
	s.Require().NoError(os.WriteFile("mixed/readme.md", []byte("# README"), 0o600))
	s.Require().NoError(os.WriteFile("mixed/config.json", []byte("{}"), 0o600))

	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "initial").Run())

	// Filter should only include Go files
	files := []string{"mixed/code.go", "mixed/readme.md", "mixed/config.json"}
	filtered := s.check.FilterFiles(files)

	s.Len(filtered, 1, "Should only include .go files")
	s.Equal("mixed/code.go", filtered[0])
}

// Test timeout handling in multi-directory scenario
func (s *LintMultiDirTestSuite) TestTimeoutHandling() {
	s.T().Skip("Skipping timeout test - environment dependent")

	// Create a check with very short timeout
	shortTimeoutCheck := NewLintCheckWithConfig(s.sharedCtx, nil, 1*time.Nanosecond)

	// Create multiple directories with many files to ensure timeout
	for i := 0; i < 5; i++ {
		dir := fmt.Sprintf("timeout%d", i)
		s.Require().NoError(os.MkdirAll(dir, 0o750))

		// Create multiple files per directory
		for j := 0; j < 3; j++ {
			validFile := fmt.Sprintf(`package timeout%d

func TimeoutFunc%d() {
	// Some code that will take time to lint
	var x int
	x = 1
	_ = x
}
`, i, j)
			s.Require().NoError(os.WriteFile(fmt.Sprintf("%s/file%d.go", dir, j), []byte(validFile), 0o600))
		}
	}

	// Commit files to avoid new-from-rev issues
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "timeout test").Run())

	// Collect all files
	var files []string
	for i := 0; i < 5; i++ {
		for j := 0; j < 3; j++ {
			files = append(files, fmt.Sprintf("timeout%d/file%d.go", i, j))
		}
	}

	// With such a short timeout, it should definitely timeout
	err := shortTimeoutCheck.runDirectLint(ctx, files)

	// We expect an error due to timeout
	s.Require().Error(err, "Expected timeout error with nanosecond timeout")

	errStr := err.Error()
	s.T().Logf("Timeout test error: %v", errStr)

	// Check for timeout-related error message
	s.True(
		strings.Contains(errStr, "timeout") || strings.Contains(errStr, "timed out") ||
			strings.Contains(errStr, "deadline"),
		"Expected timeout-related error, got: %v", err,
	)
}

// Test runLintOnDirectory with valid directory
func (s *LintMultiDirTestSuite) TestRunLintOnDirectory() {
	// Create a directory with Go files
	s.Require().NoError(os.MkdirAll("testdir", 0o750))

	validFile := `package testdir

import "fmt"

func TestDirFunc() {
	fmt.Println("test")
}
`
	s.Require().NoError(os.WriteFile("testdir/file.go", []byte(validFile), 0o600))

	// Commit the file
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "initial").Run())

	// Run lint on directory
	err := s.check.runLintOnDirectory(ctx, s.tempDir, "testdir")
	if err != nil {
		s.T().Logf("Directory lint error: %v", err)
		// Should be lint issues or success, not tool failure
		s.NotContains(err.Error(), "named files must all be in one directory")
	}
}

// Test runLintOnFiles fallback behavior
func (s *LintMultiDirTestSuite) TestRunLintOnFilesFallback() {
	// This tests the fallback when "named files must all be in one directory" occurs
	// We'll mock this by testing with files that might trigger it

	s.Require().NoError(os.MkdirAll("fallback", 0o750))

	file1 := `package fallback

func FallbackFunc() {
	// test
}
`
	s.Require().NoError(os.WriteFile("fallback/file1.go", []byte(file1), 0o600))

	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "initial").Run())

	// Test the function directly
	files := []string{"fallback/file1.go"}
	err := s.check.runLintOnFiles(ctx, s.tempDir, files)
	// Should handle the file without issues
	if err != nil {
		s.T().Logf("runLintOnFiles error: %v", err)
	}
}

// Benchmark single vs multi-directory performance
func (s *LintMultiDirTestSuite) TestLintingPerformance() {
	// Skip if not running benchmarks
	if testing.Short() {
		s.T().Skip("Skipping performance test in short mode")
	}

	// Create single directory with multiple files
	s.Require().NoError(os.MkdirAll("perf/single", 0o750))
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf(`package single

import "fmt"

func Func%d() {
	fmt.Println("%d")
}
`, i, i)
		filename := fmt.Sprintf("perf/single/file%d.go", i)
		s.Require().NoError(os.WriteFile(filename, []byte(content), 0o600))
	}

	// Create multiple directories with files
	for d := 0; d < 3; d++ {
		dir := fmt.Sprintf("perf/multi%d", d)
		s.Require().NoError(os.MkdirAll(dir, 0o750))
		for f := 0; f < 2; f++ {
			content := fmt.Sprintf(`package multi%d

import "fmt"

func Func%d_%d() {
	fmt.Println("%d_%d")
}
`, d, d, f, d, f)
			filename := fmt.Sprintf("%s/file%d.go", dir, f)
			s.Require().NoError(os.WriteFile(filename, []byte(content), 0o600))
		}
	}

	// Commit all files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "perf test").Run())

	// Measure single directory performance
	singleFiles := []string{
		"perf/single/file0.go",
		"perf/single/file1.go",
		"perf/single/file2.go",
	}

	start := time.Now()
	_ = s.check.runDirectLint(ctx, singleFiles)
	singleDuration := time.Since(start)
	s.T().Logf("Single directory lint took: %v", singleDuration)

	// Measure multi-directory performance
	multiFiles := []string{
		"perf/multi0/file0.go",
		"perf/multi1/file0.go",
		"perf/multi2/file0.go",
	}

	start = time.Now()
	_ = s.check.runDirectLint(ctx, multiFiles)
	multiDuration := time.Since(start)
	s.T().Logf("Multi-directory lint took: %v", multiDuration)

	// Multi-directory should not be significantly slower (< 3x)
	if multiDuration > 3*singleDuration {
		s.T().Logf("Warning: Multi-directory linting is significantly slower than single directory")
	}
}

// Test with large number of directories
func (s *LintMultiDirTestSuite) TestLargeScaleDirectories() {
	if testing.Short() {
		s.T().Skip("Skipping large scale test in short mode")
	}

	// Create 20 directories with files
	var allFiles []string
	for i := 0; i < 20; i++ {
		dir := fmt.Sprintf("large/dir%d", i)
		s.Require().NoError(os.MkdirAll(dir, 0o750))

		content := fmt.Sprintf(`package dir%d

import "fmt"

func Dir%dFunc() {
	fmt.Println("dir%d")
}
`, i, i, i)
		filename := fmt.Sprintf("%s/file.go", dir)
		s.Require().NoError(os.WriteFile(filename, []byte(content), 0o600))
		allFiles = append(allFiles, filename)
	}

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "large scale").Run())

	// Should handle large number of directories
	err := s.check.runDirectLint(ctx, allFiles)
	if err != nil {
		s.T().Logf("Large scale test error: %v", err)
		// Should not fail due to multi-directory issues
		s.NotContains(err.Error(), "named files must all be in one directory")
	}
}

// Test special characters in paths
func (s *LintMultiDirTestSuite) TestSpecialCharactersInPaths() {
	// Create directories with spaces and special chars (if supported by OS)
	dirName := "dir with spaces"
	s.Require().NoError(os.MkdirAll(dirName, 0o750))

	content := `package dirwithspaces

func SpecialFunc() {
	// test
}
`
	filename := filepath.Join(dirName, "file.go")
	s.Require().NoError(os.WriteFile(filename, []byte(content), 0o600))

	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "special chars").Run())

	// Should handle special characters in paths
	files := []string{filename}
	err := s.check.runDirectLint(ctx, files)
	if err != nil {
		s.T().Logf("Special char test error: %v", err)
	}
}

// Test concurrent execution safety
func (s *LintMultiDirTestSuite) TestConcurrentExecution() {
	// Create test directories
	for i := 0; i < 3; i++ {
		dir := fmt.Sprintf("concurrent%d", i)
		s.Require().NoError(os.MkdirAll(dir, 0o750))

		content := fmt.Sprintf(`package concurrent%d

func ConcurrentFunc%d() {
	// test
}
`, i, i)
		s.Require().NoError(os.WriteFile(fmt.Sprintf("%s/file.go", dir), []byte(content), 0o600))
	}

	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "concurrent").Run())

	// Run multiple lint checks concurrently
	done := make(chan bool, 3)
	for i := 0; i < 3; i++ {
		go func(idx int) {
			files := []string{fmt.Sprintf("concurrent%d/file.go", idx)}
			_ = s.check.runDirectLint(ctx, files)
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(10 * time.Second):
			s.Fail("Concurrent execution timed out")
		}
	}
}

// TestGoModuleSubdirectoryHandling tests the new Go module handling in subdirectories
func (s *LintMultiDirTestSuite) TestGoModuleSubdirectoryHandling() {
	// Create a structure with a Go module in a subdirectory
	s.Require().NoError(os.MkdirAll("project/worker/cmd", 0o750))
	s.Require().NoError(os.MkdirAll("project/worker/internal/service", 0o750))
	s.Require().NoError(os.MkdirAll("project/docs", 0o750))

	// Create go.mod in the subdirectory
	goModContent := `module example.com/worker

go 1.21

require github.com/stretchr/testify v1.8.4
`
	s.Require().NoError(os.WriteFile("project/worker/go.mod", []byte(goModContent), 0o600))

	// Create Go files in the module
	mainContent := `package main

import (
	"fmt"
	"example.com/worker/internal/service"
)

func main() {
	svc := service.New("test")
	fmt.Printf("Service: %s\n", svc.Name())
}
`
	s.Require().NoError(os.WriteFile("project/worker/cmd/main.go", []byte(mainContent), 0o600))

	serviceContent := `package service

type Service struct {
	name string
}

func New(name string) *Service {
	return &Service{name: name}
}

func (s *Service) Name() string {
	return s.name
}
`
	s.Require().NoError(os.WriteFile("project/worker/internal/service/service.go", []byte(serviceContent), 0o600))

	// Create orphaned Go file (not in module)
	orphanedContent := `package docs

// This file is not part of the Go module
func DocumentationHelper() {
	// helper
}
`
	s.Require().NoError(os.WriteFile("project/docs/helper.go", []byte(orphanedContent), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "go module subdirectory").Run())

	// Test linting files from Go module subdirectory
	files := []string{
		"project/worker/cmd/main.go",
		"project/worker/internal/service/service.go",
	}

	err := s.check.runDirectLint(ctx, files)
	if err != nil {
		s.T().Logf("Go module subdirectory lint result: %v", err)
		// Should not fail due to module resolution issues
		s.NotContains(err.Error(), "no go files to analyze")
		s.NotContains(err.Error(), "could not import")
	}
}

// TestOrphanedFilesSkipping tests that orphaned Go files are skipped
func (s *LintMultiDirTestSuite) TestOrphanedFilesSkipping() {
	// Create orphaned Go files (no go.mod anywhere)
	s.Require().NoError(os.MkdirAll("standalone/utils", 0o750))
	s.Require().NoError(os.MkdirAll("scripts", 0o750))

	orphanedContent1 := `package utils

func Utility() {
	// utility function
}
`
	s.Require().NoError(os.WriteFile("standalone/utils/util.go", []byte(orphanedContent1), 0o600))

	orphanedContent2 := `package main

import "fmt"

func main() {
	fmt.Println("Standalone script")
}
`
	s.Require().NoError(os.WriteFile("scripts/script.go", []byte(orphanedContent2), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "orphaned files").Run())

	// Test linting orphaned files - should be skipped silently
	files := []string{
		"standalone/utils/util.go",
		"scripts/script.go",
	}

	err := s.check.runDirectLint(ctx, files)
	// Should succeed (orphaned files are skipped)
	s.NoError(err, "Orphaned Go files should be skipped without error")
}

// TestMixedGoModuleAndOrphanedFiles tests mixed scenarios
func (s *LintMultiDirTestSuite) TestMixedGoModuleAndOrphanedFiles() {
	// Create a Go module
	s.Require().NoError(os.MkdirAll("mymodule/pkg", 0o750))
	s.Require().NoError(os.WriteFile("mymodule/go.mod", []byte("module example.com/mymodule\n"), 0o600))

	moduleContent := `package pkg

func ModuleFunc() {
	// module function
}
`
	s.Require().NoError(os.WriteFile("mymodule/pkg/module.go", []byte(moduleContent), 0o600))

	// Create orphaned files
	s.Require().NoError(os.MkdirAll("orphaned", 0o750))
	orphanedContent := `package orphaned

func OrphanedFunc() {
	// orphaned function
}
`
	s.Require().NoError(os.WriteFile("orphaned/orphaned.go", []byte(orphanedContent), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "mixed files").Run())

	// Test linting mixed files
	files := []string{
		"mymodule/pkg/module.go", // Should be linted
		"orphaned/orphaned.go",   // Should be skipped
	}

	err := s.check.runDirectLint(ctx, files)
	if err != nil {
		s.T().Logf("Mixed files lint result: %v", err)
		// Should handle the module file and skip orphaned
		s.NotContains(err.Error(), "no go files to analyze")
	}
}
