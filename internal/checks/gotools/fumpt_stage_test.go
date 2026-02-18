package gotools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-pre-commit/internal/config"
	"github.com/mrz1836/go-pre-commit/internal/shared"
)

// FumptStageTestSuite tests the stageFiles functionality of FumptCheck
type FumptStageTestSuite struct {
	suite.Suite

	tempDir    string
	repoRoot   string
	testFile   string
	sharedCtx  *shared.Context
	originalWD string
}

// SetupSuite initializes the test environment
func (s *FumptStageTestSuite) SetupSuite() {
	var err error
	s.originalWD, err = os.Getwd()
	s.Require().NoError(err)

	// Create temporary directory
	s.tempDir = s.T().TempDir()
	s.repoRoot = s.tempDir

	// Initialize git repository
	s.Require().NoError(s.initGitRepo())

	// Change to temp directory
	s.Require().NoError(os.Chdir(s.tempDir))

	// Create test Go file
	s.testFile = filepath.Join(s.tempDir, "test.go")
	testContent := `package main

import "fmt"

func main() {
fmt.Println("Hello, World!")
}
`
	s.Require().NoError(os.WriteFile(s.testFile, []byte(testContent), 0o600))

	// Create shared context
	s.sharedCtx = shared.NewContext()
}

// TearDownSuite cleans up the test environment
func (s *FumptStageTestSuite) TearDownSuite() {
	_ = os.Chdir(s.originalWD)
}

// initGitRepo initializes a git repository in the temp directory
func (s *FumptStageTestSuite) initGitRepo() error {
	// Initialize git repo
	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = s.tempDir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Set git config
	configCmds := [][]string{
		{"config", "user.name", "Test User"},
		{"config", "user.email", "test@example.com"},
		{"config", "init.defaultBranch", "main"},
	}

	for _, args := range configCmds {
		// #nosec G204 - git command with controlled arguments from test
		cmd := exec.CommandContext(context.Background(), "git", args...)
		cmd.Dir = s.tempDir
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

// TestFumptCheck_StageFiles_EmptyFileList tests stageFiles with empty file list
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_EmptyFileList() {
	check := NewFumptCheckWithSharedContext(s.sharedCtx)

	err := check.stageFiles(context.Background(), []string{})
	s.NoError(err, "stageFiles should handle empty file list gracefully")
}

// TestFumptCheck_StageFiles_SingleFile tests stageFiles with a single file
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_SingleFile() {
	if !s.isGitAvailable() {
		s.T().Skip("Git not available, skipping git staging test")
	}

	check := NewFumptCheckWithSharedContext(s.sharedCtx)

	// Ensure file is not staged initially
	s.resetGitHead(s.testFile)

	// Stage the file
	err := check.stageFiles(context.Background(), []string{s.testFile})
	s.Require().NoError(err, "stageFiles should stage the file successfully")

	// Verify file was staged
	s.verifyFileStaged(s.testFile)
}

// TestFumptCheck_StageFiles_MultipleFiles tests stageFiles with multiple files
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_MultipleFiles() {
	if !s.isGitAvailable() {
		s.T().Skip("Git not available, skipping git staging test")
	}

	// Create additional test files
	testFile2 := filepath.Join(s.tempDir, "test2.go")
	testFile3 := filepath.Join(s.tempDir, "utils.go")

	testFiles := []string{s.testFile, testFile2, testFile3}

	for _, file := range testFiles[1:] { // Skip first file as it already exists
		content := `package main

func helper() {
	return
}
`
		s.Require().NoError(os.WriteFile(file, []byte(content), 0o600))
	}

	check := NewFumptCheckWithSharedContext(s.sharedCtx)

	// Ensure files are not staged initially
	for _, file := range testFiles {
		s.resetGitHead(file)
	}

	// Stage all files
	err := check.stageFiles(context.Background(), testFiles)
	s.Require().NoError(err, "stageFiles should stage all files successfully")

	// Verify all files were staged
	for _, file := range testFiles {
		s.verifyFileStaged(file)
	}
}

// TestFumptCheck_StageFiles_NonexistentFile tests stageFiles with nonexistent file
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_NonexistentFile() {
	if !s.isGitAvailable() {
		s.T().Skip("Git not available, skipping git staging test")
	}

	check := NewFumptCheckWithSharedContext(s.sharedCtx)

	nonexistentFile := filepath.Join(s.tempDir, "nonexistent.go")

	err := check.stageFiles(context.Background(), []string{nonexistentFile})
	s.Require().Error(err, "stageFiles should error when trying to stage nonexistent file")
	s.Contains(err.Error(), "git add failed", "Error should mention git add failure")
}

// TestFumptCheck_StageFiles_ContextCancellation tests stageFiles with context cancellation
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_ContextCancellation() {
	if !s.isGitAvailable() {
		s.T().Skip("Git not available, skipping git staging test")
	}

	check := NewFumptCheckWithSharedContext(s.sharedCtx)

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := check.stageFiles(ctx, []string{s.testFile})
	s.Error(err, "stageFiles should fail with canceled context")
}

// TestFumptCheck_StageFiles_NoGitRepo tests stageFiles behavior without git repo
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_NoGitRepo() {
	// Create a separate temp directory without git
	nonGitDir := s.T().TempDir()
	testFileNonGit := filepath.Join(nonGitDir, "test.go")
	s.Require().NoError(os.WriteFile(testFileNonGit, []byte("package main\n"), 0o600))

	check := NewFumptCheckWithSharedContext(s.sharedCtx)

	// Change to non-git directory temporarily
	originalDir, _ := os.Getwd()
	_ = os.Chdir(nonGitDir)
	defer func() { _ = os.Chdir(originalDir) }()

	err := check.stageFiles(context.Background(), []string{testFileNonGit})
	s.Error(err, "stageFiles should fail when not in a git repository")
}

// TestFumptCheck_StageFiles_WithTimeout tests stageFiles with timeout
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_WithTimeout() {
	if !s.isGitAvailable() {
		s.T().Skip("Git not available, skipping git staging test")
	}

	check := NewFumptCheckWithSharedContext(s.sharedCtx)

	// Use a very short timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to expire
	time.Sleep(1 * time.Millisecond)

	err := check.stageFiles(ctx, []string{s.testFile})
	s.Error(err, "stageFiles should fail with expired context")
}

// TestFumptCheck_StageFiles_InvalidPath tests stageFiles with invalid paths
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_InvalidPath() {
	if !s.isGitAvailable() {
		s.T().Skip("Git not available, skipping git staging test")
	}

	check := NewFumptCheckWithSharedContext(s.sharedCtx)

	// Test with invalid path characters
	invalidPaths := []string{
		"/invalid/path/that/does/not/exist.go",
		"",
		" ", // Whitespace-only path
	}

	for _, invalidPath := range invalidPaths {
		s.Run("InvalidPath_"+invalidPath, func() {
			if invalidPath == "" {
				return // Empty path handled by empty file list test
			}

			err := check.stageFiles(context.Background(), []string{invalidPath})
			s.Error(err, "stageFiles should fail with invalid path: %s", invalidPath)
		})
	}
}

// TestFumptCheck_StageFiles_MixedExistentAndNonexistent tests staging mix of files
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_MixedExistentAndNonexistent() {
	if !s.isGitAvailable() {
		s.T().Skip("Git not available, skipping git staging test")
	}

	check := NewFumptCheckWithSharedContext(s.sharedCtx)

	nonexistentFile := filepath.Join(s.tempDir, "nonexistent.go")

	files := []string{s.testFile, nonexistentFile}

	err := check.stageFiles(context.Background(), files)
	s.Error(err, "stageFiles should fail when some files don't exist")
}

// TestFumptCheck_StageFiles_RelativeAndAbsolutePaths tests staging with different path types
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_RelativeAndAbsolutePaths() {
	if !s.isGitAvailable() {
		s.T().Skip("Git not available, skipping git staging test")
	}

	check := NewFumptCheckWithSharedContext(s.sharedCtx)

	// Create file in subdirectory
	subDir := filepath.Join(s.tempDir, "subdir")
	s.Require().NoError(os.MkdirAll(subDir, 0o750))

	subFile := filepath.Join(subDir, "sub.go")
	s.Require().NoError(os.WriteFile(subFile, []byte("package sub\n"), 0o600))

	// Test with both absolute and relative paths
	files := []string{
		s.testFile,      // Absolute path
		"subdir/sub.go", // Relative path
	}

	// Reset staging
	for _, file := range files {
		s.resetGitHead(file)
	}

	err := check.stageFiles(context.Background(), files)
	s.Require().NoError(err, "stageFiles should handle both relative and absolute paths")

	// Verify files were staged
	for _, file := range files {
		s.verifyFileStaged(filepath.Base(file))
	}
}

// Helper methods

// isGitAvailable checks if git is available in the system
func (s *FumptStageTestSuite) isGitAvailable() bool {
	cmd := exec.CommandContext(context.Background(), "git", "--version")
	return cmd.Run() == nil
}

// resetGitHead resets git HEAD for a file
func (s *FumptStageTestSuite) resetGitHead(file string) {
	cmd := exec.CommandContext(context.Background(), "git", "reset", "HEAD", file) // #nosec G204 - git binary path is fixed
	cmd.Dir = s.tempDir
	_ = cmd.Run() // Ignore errors as file might not be tracked
}

// verifyFileStaged verifies that a file is staged in git
func (s *FumptStageTestSuite) verifyFileStaged(file string) {
	cmd := exec.CommandContext(context.Background(), "git", "diff", "--cached", "--name-only")
	cmd.Dir = s.tempDir
	output, err := cmd.Output()
	s.Require().NoError(err, "Should be able to check staged files")

	staged := strings.TrimSpace(string(output))
	s.Contains(staged, filepath.Base(file), "File should be staged: %s", file)
}

// TestFumptCheck_StageFiles_AutoStageIntegration tests integration with auto-stage functionality
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_AutoStageIntegration() {
	testCases := []struct {
		name        string
		autoStage   bool
		expectStage bool
		description string
	}{
		{
			name:        "Auto-stage Enabled",
			autoStage:   true,
			expectStage: true,
			description: "Should stage files when auto-stage is enabled",
		},
		{
			name:        "Auto-stage Disabled",
			autoStage:   false,
			expectStage: false,
			description: "Should not attempt auto-staging when disabled",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cfg := &config.Config{}
			cfg.CheckBehaviors.FumptAutoStage = tc.autoStage
			cfg.CheckTimeouts.Fumpt = 30

			check := NewFumptCheckWithFullConfig(s.sharedCtx, cfg)

			s.Equal(tc.autoStage, check.autoStage, "Auto-stage setting should match config")

			if tc.expectStage && s.isGitAvailable() {
				// Test that the check's stageFiles function works
				s.resetGitHead(s.testFile)

				err := check.stageFiles(context.Background(), []string{s.testFile})
				s.Require().NoError(err, "Should be able to stage files when auto-stage is enabled")

				s.verifyFileStaged(s.testFile)
			}
		})
	}
}

// TestFumptCheck_StageFiles_ErrorHandling tests comprehensive error handling
func (s *FumptStageTestSuite) TestFumptCheck_StageFiles_ErrorHandling() {
	testCases := []struct {
		name          string
		setupFunc     func() ([]string, context.Context)
		expectedError string
		description   string
	}{
		{
			name: "Nil Context",
			setupFunc: func() ([]string, context.Context) {
				return []string{s.testFile}, nil
			},
			expectedError: "context",
			description:   "Should handle nil context gracefully",
		},
		{
			name: "Empty File Paths",
			setupFunc: func() ([]string, context.Context) {
				return []string{""}, context.Background()
			},
			expectedError: "git add failed",
			description:   "Should handle empty file paths",
		},
	}

	for _, tc := range testCases {
		if !s.isGitAvailable() {
			s.T().Skip("Git not available, skipping git error handling test")
		}

		s.Run(tc.name, func() {
			check := NewFumptCheckWithSharedContext(s.sharedCtx)

			files, ctx := tc.setupFunc()

			err := check.stageFiles(ctx, files)

			if tc.expectedError != "" {
				s.Require().Error(err, tc.description)
				s.Contains(err.Error(), tc.expectedError, "Error should contain expected message")
			} else {
				s.Require().NoError(err, tc.description)
			}
		})
	}
}

// TestSuite runs the fumpt stage test suite
func TestFumptStageTestSuite(t *testing.T) {
	suite.Run(t, new(FumptStageTestSuite))
}
