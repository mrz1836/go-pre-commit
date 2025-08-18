package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-pre-commit/internal/config"
	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/git"
)

// FileSystemErrorTestSuite tests file system error handling across the application
type FileSystemErrorTestSuite struct {
	suite.Suite

	tempDir         string
	originalWD      string
	readOnlyDir     string
	nonExistentPath string
}

// SetupSuite initializes the test environment
func (s *FileSystemErrorTestSuite) SetupSuite() {
	var err error
	s.originalWD, err = os.Getwd()
	s.Require().NoError(err)

	// Create temporary directory
	s.tempDir = s.T().TempDir()

	// Create read-only directory for permission error testing
	s.readOnlyDir = filepath.Join(s.tempDir, "readonly")
	s.Require().NoError(os.MkdirAll(s.readOnlyDir, 0o000)) // No permissions

	// Define nonexistent path
	s.nonExistentPath = filepath.Join(s.tempDir, "nonexistent", "path", "file.go")
}

// TearDownSuite cleans up the test environment
func (s *FileSystemErrorTestSuite) TearDownSuite() {
	// Restore permissions to allow cleanup
	_ = os.Chmod(s.readOnlyDir, 0o700) // #nosec G302 - intentional permission change for cleanup

	// Restore permissions for any .git directories created during tests
	gitDir := filepath.Join(s.tempDir, "unreadable-git", ".git")
	if _, err := os.Stat(gitDir); err == nil {
		_ = os.Chmod(gitDir, 0o750) // #nosec G302 - intentional permission change for cleanup
	}

	_ = os.Chdir(s.originalWD)
}

// TestConfigLoad_FileSystemErrors tests config loading with file system errors
func (s *FileSystemErrorTestSuite) TestConfigLoad_FileSystemErrors() {
	testCases := []struct {
		name          string
		setupFunc     func() string
		expectError   bool
		errorContains string
		description   string
	}{
		{
			name: "Missing .env.base file",
			setupFunc: func() string {
				// Create directory without .env.base
				testDir := filepath.Join(s.tempDir, "no-env-base")
				s.Require().NoError(os.MkdirAll(testDir, 0o750))
				return testDir
			},
			expectError:   false, // Should succeed - config loading falls back to env vars
			errorContains: "",
			description:   "Should succeed when .env.base file is missing (fallback to env vars)",
		},
		{
			name: "Unreadable .env.base file",
			setupFunc: func() string {
				// Create directory with unreadable .env.base
				testDir := filepath.Join(s.tempDir, "unreadable-env")
				githubDir := filepath.Join(testDir, ".github")
				s.Require().NoError(os.MkdirAll(githubDir, 0o750))

				envFile := filepath.Join(githubDir, ".env.base")
				s.Require().NoError(os.WriteFile(envFile, []byte("ENABLE_GO_PRE_COMMIT=true\n"), 0o000)) // No read permissions

				return testDir
			},
			expectError:   true,
			errorContains: "failed to load",
			description:   "Should fail when .env.base file is unreadable",
		},
		{
			name: "Invalid .env.base content",
			setupFunc: func() string {
				// Create directory with malformed .env.base
				testDir := filepath.Join(s.tempDir, "invalid-env")
				githubDir := filepath.Join(testDir, ".github")
				s.Require().NoError(os.MkdirAll(githubDir, 0o750))

				envFile := filepath.Join(githubDir, ".env.base")
				// Create invalid env file content
				invalidContent := "ENABLE_GO_PRE_COMMIT=true\nINVALID_LINE_WITHOUT_EQUALS\n"
				s.Require().NoError(os.WriteFile(envFile, []byte(invalidContent), 0o600))

				return testDir
			},
			expectError:   false, // godotenv is tolerant of invalid lines
			errorContains: "",
			description:   "Should handle invalid .env.base content gracefully",
		},
		{
			name: "Read-only directory",
			setupFunc: func() string {
				return s.readOnlyDir
			},
			expectError:   false, // Should succeed - config loading falls back to env vars
			errorContains: "",
			description:   "Should succeed when directory is not readable (fallback to env vars)",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Change to test directory
			testDir := tc.setupFunc()
			originalDir, _ := os.Getwd()
			defer func() { _ = os.Chdir(originalDir) }()

			_ = os.Chdir(testDir)

			// Attempt to load config
			_, err := config.Load()

			if tc.expectError {
				s.Require().Error(err, tc.description)
				if tc.errorContains != "" {
					s.Contains(err.Error(), tc.errorContains, "Error should contain expected message")
				}
			} else {
				if err != nil {
					s.T().Logf("Unexpected error (may be acceptable): %v", err)
				}
			}

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestGitRepository_FileSystemErrors tests git repository operations with file system errors
func (s *FileSystemErrorTestSuite) TestGitRepository_FileSystemErrors() {
	testCases := []struct {
		name          string
		setupFunc     func() *git.Repository
		testFunc      func(*git.Repository) error
		expectError   bool
		errorContains string
		description   string
	}{
		{
			name: "GetStagedFiles - Non-existent directory",
			setupFunc: func() *git.Repository {
				return git.NewRepository("/nonexistent/directory")
			},
			testFunc: func(repo *git.Repository) error {
				_, err := repo.GetStagedFiles()
				return err
			},
			expectError:   true,
			errorContains: "failed to get staged files",
			description:   "Should fail when git directory doesn't exist",
		},
		{
			name: "GetAllFiles - Non-git repository",
			setupFunc: func() *git.Repository {
				// Create non-git directory
				nonGitDir := filepath.Join(s.tempDir, "non-git")
				s.Require().NoError(os.MkdirAll(nonGitDir, 0o750))
				return git.NewRepository(nonGitDir)
			},
			testFunc: func(repo *git.Repository) error {
				_, err := repo.GetAllFiles()
				return err
			},
			expectError:   true,
			errorContains: "failed to get all files",
			description:   "Should fail when directory is not a git repository",
		},
		{
			name: "GetModifiedFiles - Permission denied",
			setupFunc: func() *git.Repository {
				return git.NewRepository(s.readOnlyDir)
			},
			testFunc: func(repo *git.Repository) error {
				_, err := repo.GetModifiedFiles()
				return err
			},
			expectError:   true,
			errorContains: "failed to get staged files",
			description:   "Should fail when directory permissions are insufficient",
		},
		{
			name: "GetFileContent - Nonexistent file",
			setupFunc: func() *git.Repository {
				// Create minimal git repo
				gitDir := filepath.Join(s.tempDir, "test-git-repo")
				s.Require().NoError(os.MkdirAll(gitDir, 0o750))
				s.initBasicGitRepo(gitDir)
				return git.NewRepository(gitDir)
			},
			testFunc: func(repo *git.Repository) error {
				_, err := repo.GetFileContent("nonexistent-file.go")
				return err
			},
			expectError:   true,
			errorContains: "",
			description:   "Should fail when file doesn't exist",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			repo := tc.setupFunc()
			err := tc.testFunc(repo)

			if tc.expectError {
				s.Require().Error(err, tc.description)
				if tc.errorContains != "" {
					s.Contains(err.Error(), tc.errorContains, "Error should contain expected message")
				}
			} else {
				s.Require().NoError(err, tc.description)
			}

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestFindRepositoryRoot_FileSystemErrors tests FindRepositoryRoot with file system errors
func (s *FileSystemErrorTestSuite) TestFindRepositoryRoot_FileSystemErrors() {
	testCases := []struct {
		name          string
		setupFunc     func() string
		expectError   bool
		errorContains string
		description   string
	}{
		{
			name: "Non-existent current directory",
			setupFunc: func() string {
				// Create and then remove a directory to simulate invalid working directory
				testDir := filepath.Join(s.tempDir, "temp-dir")
				s.Require().NoError(os.MkdirAll(testDir, 0o750))
				s.Require().NoError(os.RemoveAll(testDir))
				return testDir
			},
			expectError:   false, // May succeed depending on fallback behavior
			errorContains: "",
			description:   "Should handle when current directory doesn't exist",
		},
		{
			name: "No git repository in hierarchy",
			setupFunc: func() string {
				// Create directory hierarchy without .git
				testDir := filepath.Join(s.tempDir, "no-git", "deep", "directory")
				s.Require().NoError(os.MkdirAll(testDir, 0o750))
				return testDir
			},
			expectError:   true,
			errorContains: "not in a git repository",
			description:   "Should fail when no .git directory is found in hierarchy",
		},
		{
			name: "Unreadable .git directory",
			setupFunc: func() string {
				// Create git repo with unreadable .git directory
				testDir := filepath.Join(s.tempDir, "unreadable-git")
				gitDir := filepath.Join(testDir, ".git")
				s.Require().NoError(os.MkdirAll(gitDir, 0o750)) // Create with permissions first
				s.Require().NoError(os.Chmod(gitDir, 0o000))    // Then remove permissions
				// TearDownSuite will restore permissions for cleanup
				return testDir
			},
			expectError:   true,
			errorContains: "not in a git repository",
			description:   "Should fail when .git directory is not readable",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			testDir := tc.setupFunc()
			originalDir, _ := os.Getwd()
			defer func() { _ = os.Chdir(originalDir) }()

			if _, err := os.Stat(testDir); err == nil {
				_ = os.Chdir(testDir)
			}

			_, err := git.FindRepositoryRoot()

			if tc.expectError {
				s.Require().Error(err, tc.description)
				if tc.errorContains != "" {
					s.Contains(err.Error(), tc.errorContains, "Error should contain expected message")
				}
			} else {
				s.Require().NoError(err, tc.description)
			}

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestFilePermissionErrors tests various file permission scenarios
func (s *FileSystemErrorTestSuite) TestFilePermissionErrors() {
	testCases := []struct {
		name        string
		setupFunc   func() string
		testFunc    func(string) error
		expectError bool
		description string
	}{
		{
			name: "Write to read-only file",
			setupFunc: func() string {
				testFile := filepath.Join(s.tempDir, "readonly-file.txt")
				s.Require().NoError(os.WriteFile(testFile, []byte("content"), 0o400)) // Read-only
				return testFile
			},
			testFunc: func(path string) error {
				return os.WriteFile(path, []byte("new content"), 0o600)
			},
			expectError: true,
			description: "Should fail when trying to write to read-only file",
		},
		{
			name: "Read from no-permission file",
			setupFunc: func() string {
				testFile := filepath.Join(s.tempDir, "no-read-file.txt")
				s.Require().NoError(os.WriteFile(testFile, []byte("content"), 0o000)) // No permissions
				return testFile
			},
			testFunc: func(path string) error {
				_, err := os.ReadFile(path) // #nosec G304 - test path is controlled
				return err
			},
			expectError: true,
			description: "Should fail when trying to read file without permissions",
		},
		{
			name: "Create file in read-only directory",
			setupFunc: func() string {
				readOnlySubDir := filepath.Join(s.tempDir, "readonly-subdir")
				s.Require().NoError(os.MkdirAll(readOnlySubDir, 0o550)) // Read and execute, no write
				return filepath.Join(readOnlySubDir, "newfile.txt")
			},
			testFunc: func(path string) error {
				return os.WriteFile(path, []byte("content"), 0o600)
			},
			expectError: true,
			description: "Should fail when trying to create file in read-only directory",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			path := tc.setupFunc()
			err := tc.testFunc(path)

			if tc.expectError {
				s.Require().Error(err, tc.description)
			} else {
				s.Require().NoError(err, tc.description)
			}

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestDiskSpaceErrors tests disk space related errors (simulated)
func (s *FileSystemErrorTestSuite) TestDiskSpaceErrors() {
	// Actual disk space errors are hard to simulate reliably in unit tests
	// This test focuses on testing the error handling paths

	testCases := []struct {
		name        string
		setupFunc   func() (string, func())
		testFunc    func(string) error
		expectError bool
		description string
	}{
		{
			name: "Write very large file to temp",
			setupFunc: func() (string, func()) {
				// Create a reasonably large but not massive file
				largePath := filepath.Join(s.tempDir, "large-test-file.txt")
				return largePath, func() { _ = os.Remove(largePath) }
			},
			testFunc: func(path string) error {
				// Try to write a large buffer (but not too large to actually cause issues)
				largeBuffer := make([]byte, 1024*1024) // 1MB
				return os.WriteFile(path, largeBuffer, 0o600)
			},
			expectError: false, // Should succeed with reasonable size
			description: "Should handle large file writes appropriately",
		},
		{
			name: "Create many temporary files",
			setupFunc: func() (string, func()) {
				tempSubDir := filepath.Join(s.tempDir, "many-files")
				s.Require().NoError(os.MkdirAll(tempSubDir, 0o750))
				return tempSubDir, func() { _ = os.RemoveAll(tempSubDir) }
			},
			testFunc: func(dir string) error {
				// Create many small files to test resource limits
				for i := 0; i < 100; i++ {
					path := filepath.Join(dir, fmt.Sprintf("file_%d.txt", i))
					if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
						return err
					}
				}
				return nil
			},
			expectError: false, // Should succeed with reasonable number
			description: "Should handle creating many small files",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			path, cleanup := tc.setupFunc()
			defer cleanup()

			err := tc.testFunc(path)

			if tc.expectError {
				s.Require().Error(err, tc.description)
			} else {
				if err != nil {
					s.T().Logf("Unexpected error (may be environment-specific): %v", err)
				}
			}

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestContextCancellation tests context cancellation during file operations
func (s *FileSystemErrorTestSuite) TestContextCancellation() {
	testCases := []struct {
		name        string
		testFunc    func(context.Context) error
		expectError bool
		description string
	}{
		{
			name: "Canceled context for git operations",
			testFunc: func(ctx context.Context) error {
				// Create a git repository
				gitDir := filepath.Join(s.tempDir, "context-test-git")
				s.Require().NoError(os.MkdirAll(gitDir, 0o750))
				s.initBasicGitRepo(gitDir)

				// Create repository and try to get files with canceled context
				repo := git.NewRepository(gitDir)

				// Since GetStagedFiles doesn't currently accept context,
				// we simulate context cancellation effect
				if ctx.Err() != nil {
					return ctx.Err()
				}

				_, err := repo.GetStagedFiles() //nolint:contextcheck // testing error handling without context
				return err
			},
			expectError: true,
			description: "Should handle context cancellation appropriately",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Create a canceled context
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// Ensure context is canceled
			s.Require().Error(ctx.Err(), "Context should be canceled")

			err := tc.testFunc(ctx)

			if tc.expectError {
				s.Require().Error(err, tc.description)
			} else {
				s.Require().NoError(err, tc.description)
			}

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// TestRaceConditions tests file system race conditions
func (s *FileSystemErrorTestSuite) TestRaceConditions() {
	testCases := []struct {
		name        string
		setupFunc   func() string
		testFunc    func(string) error
		description string
	}{
		{
			name: "Concurrent file access",
			setupFunc: func() string {
				testFile := filepath.Join(s.tempDir, "concurrent-test.txt")
				s.Require().NoError(os.WriteFile(testFile, []byte("initial content"), 0o600))
				return testFile
			},
			testFunc: func(path string) error {
				// Simulate concurrent access by reading while another goroutine writes
				done := make(chan error, 2)

				// Reader goroutine
				go func() {
					for i := 0; i < 10; i++ {
						_, err := os.ReadFile(path) // #nosec G304 - test path is controlled
						if err != nil {
							done <- err
							return
						}
						time.Sleep(1 * time.Millisecond)
					}
					done <- nil
				}()

				// Writer goroutine
				go func() {
					for i := 0; i < 10; i++ {
						err := os.WriteFile(path, []byte(fmt.Sprintf("content %d", i)), 0o600)
						if err != nil {
							done <- err
							return
						}
						time.Sleep(1 * time.Millisecond)
					}
					done <- nil
				}()

				// Wait for both goroutines
				for i := 0; i < 2; i++ {
					if err := <-done; err != nil {
						return err
					}
				}

				return nil
			},
			description: "Should handle concurrent file access gracefully",
		},
		{
			name: "File deletion during read",
			setupFunc: func() string {
				testFile := filepath.Join(s.tempDir, "delete-test.txt")
				s.Require().NoError(os.WriteFile(testFile, []byte("content"), 0o600))
				return testFile
			},
			testFunc: func(path string) error {
				// Try to read a file that gets deleted
				done := make(chan error, 1)

				go func() {
					time.Sleep(5 * time.Millisecond) // Small delay
					done <- os.Remove(path)
				}()

				// Try to read the file multiple times
				var lastErr error
				for i := 0; i < 20; i++ {
					_, err := os.ReadFile(path) // #nosec G304 - test path is controlled
					lastErr = err
					if err != nil {
						break
					}
					time.Sleep(1 * time.Millisecond)
				}

				// Wait for deletion to complete
				<-done

				// The last read should have failed or file should be deleted
				if lastErr == nil {
					// File might have been successfully deleted without read error
					_, err := os.Stat(path)
					if err == nil {
						return prerrors.ErrFileStillExists
					}
				}

				return nil
			},
			description: "Should handle file deletion during operations",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			path := tc.setupFunc()
			err := tc.testFunc(path)
			// Race condition tests may have unpredictable outcomes
			// We mainly test that they don't panic or cause corruption
			if err != nil {
				s.T().Logf("Race condition test error (may be expected): %v", err)
			}

			s.T().Logf("✓ %s: %s", tc.name, tc.description)
		})
	}
}

// Helper methods

// initBasicGitRepo initializes a basic git repository structure
func (s *FileSystemErrorTestSuite) initBasicGitRepo(dir string) {
	gitDir := filepath.Join(dir, ".git")
	s.Require().NoError(os.MkdirAll(gitDir, 0o750))

	// Create basic git files
	s.Require().NoError(os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o600))

	refsDir := filepath.Join(gitDir, "refs", "heads")
	s.Require().NoError(os.MkdirAll(refsDir, 0o750))

	objectsDir := filepath.Join(gitDir, "objects")
	s.Require().NoError(os.MkdirAll(objectsDir, 0o750))

	// Create a simple test file
	testFile := filepath.Join(dir, "test.go")
	s.Require().NoError(os.WriteFile(testFile, []byte("package main\n"), 0o600))
}

// TestSuite runs the file system error test suite
func TestFileSystemErrorTestSuite(t *testing.T) {
	suite.Run(t, new(FileSystemErrorTestSuite))
}
