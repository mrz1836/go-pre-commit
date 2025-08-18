package shared

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ContextTestSuite struct {
	suite.Suite

	tempDir string
}

func TestContextSuite(t *testing.T) {
	suite.Run(t, new(ContextTestSuite))
}

func (s *ContextTestSuite) SetupTest() {
	// Create a temporary directory for testing
	var err error
	s.tempDir, err = os.MkdirTemp("", "context_test_*")
	s.Require().NoError(err)

	// Initialize git repository
	s.Require().NoError(s.initGitRepo())
}

func (s *ContextTestSuite) TearDownTest() {
	if s.tempDir != "" {
		err := os.RemoveAll(s.tempDir)
		s.Require().NoError(err)
	}
}

func (s *ContextTestSuite) initGitRepo() error {
	ctx := context.Background()
	// Change to temp directory
	oldDir, err := os.Getwd()
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	if err := os.Chdir(s.tempDir); err != nil {
		return err
	}

	// Initialize git repo
	if err := exec.CommandContext(ctx, "git", "init").Run(); err != nil {
		return err
	}

	// Configure git
	if err := exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run(); err != nil {
		return err
	}
	if err := exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run(); err != nil {
		return err
	}

	// Create and add a test file
	testFile := filepath.Join(s.tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
		return err
	}

	if err := exec.CommandContext(context.Background(), "git", "add", "test.txt").Run(); err != nil {
		return err
	}

	if err := exec.CommandContext(context.Background(), "git", "commit", "-m", "Initial commit").Run(); err != nil {
		return err
	}

	return nil
}

// TestNewContext tests the constructor
func (s *ContextTestSuite) TestNewContext() {
	ctx := NewContext()
	s.NotNil(ctx)
	s.Empty(ctx.repoRoot)
	s.NoError(ctx.repoRootErr)
}

// TestGetRepoRoot tests repository root detection
func (s *ContextTestSuite) TestGetRepoRoot() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			s.Require().NoError(chErr)
		}
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Test getting repo root
	repoRoot, err := ctx.GetRepoRoot(context.Background())
	s.Require().NoError(err)
	s.NotEmpty(repoRoot)
	// On macOS, /var is a symlink to /private/var, so we need to resolve symlinks
	expectedPath, _ := filepath.EvalSymlinks(s.tempDir)
	actualPath, _ := filepath.EvalSymlinks(repoRoot)
	s.Equal(expectedPath, actualPath)

	// Test caching - should return same result
	repoRoot2, err2 := ctx.GetRepoRoot(context.Background())
	s.Require().NoError(err2)
	s.Equal(repoRoot, repoRoot2)
}

// TestGetRepoRootNoGitRepo tests behavior when not in git repository
func (s *ContextTestSuite) TestGetRepoRootNoGitRepo() {
	ctx := NewContext()

	// Create a non-git directory
	nonGitDir, err := os.MkdirTemp("", "non_git_*")
	s.Require().NoError(err)
	defer func() {
		_ = os.RemoveAll(nonGitDir)
	}()

	// Change to non-git directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			s.Require().NoError(chErr)
		}
	}()

	err = os.Chdir(nonGitDir)
	s.Require().NoError(err)

	// Test getting repo root should fail
	repoRoot, err := ctx.GetRepoRoot(context.Background())
	s.Require().Error(err)
	s.Empty(repoRoot)
}

// TestGetRepoRootTimeout tests timeout handling
func (s *ContextTestSuite) TestGetRepoRootTimeout() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			s.Require().NoError(chErr)
		}
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create a context with very short timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Sleep to ensure timeout
	time.Sleep(10 * time.Millisecond)

	// Should still work because GetRepoRoot creates its own timeout context
	repoRoot, err := ctx.GetRepoRoot(timeoutCtx)
	// The result depends on whether git command completes before parent context timeout
	// Both success and timeout are acceptable outcomes
	if err != nil {
		s.Contains(err.Error(), "context")
	} else {
		s.NotEmpty(repoRoot)
	}
}

// TestConcurrentGetRepoRoot tests concurrent access to GetRepoRoot
func (s *ContextTestSuite) TestConcurrentGetRepoRoot() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		if chErr := os.Chdir(oldDir); chErr != nil {
			s.Require().NoError(chErr)
		}
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Run multiple goroutines getting repo root
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			repoRoot, err := ctx.GetRepoRoot(context.Background())
			s.NoError(err)
			s.NotEmpty(repoRoot)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			s.Fail("Timeout waiting for goroutines")
		}
	}
}

// TestRepoRootCaching tests that repo root is properly cached
func TestRepoRootCaching(t *testing.T) {
	// Create a temporary git repository
	tempDir, err := os.MkdirTemp("", "cache_test_*")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Initialize git repo
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())

	// Create shared context
	sharedCtx := NewContext()

	// First call should execute git command
	start := time.Now()
	repoRoot1, err := sharedCtx.GetRepoRoot(context.Background())
	require.NoError(t, err)
	firstCallDuration := time.Since(start)

	// Second call should use cache and be much faster
	start = time.Now()
	repoRoot2, err := sharedCtx.GetRepoRoot(context.Background())
	require.NoError(t, err)
	secondCallDuration := time.Since(start)

	// Both calls should return the same result
	assert.Equal(t, repoRoot1, repoRoot2)

	// Second call should be significantly faster (cached)
	// Allow some tolerance for timing variations
	if firstCallDuration > 10*time.Millisecond {
		assert.Less(t, secondCallDuration.Nanoseconds(), firstCallDuration.Nanoseconds()/2)
	}
}

// Benchmark tests
func BenchmarkGetRepoRoot(b *testing.B) {
	// Setup
	tempDir, err := os.MkdirTemp("", "bench_*")
	require.NoError(b, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	oldDir, err := os.Getwd()
	require.NoError(b, err)
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	err = os.Chdir(tempDir)
	require.NoError(b, err)

	ctx := context.Background()
	require.NoError(b, exec.CommandContext(ctx, "git", "init").Run())

	sharedCtx := NewContext()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sharedCtx.GetRepoRoot(context.Background())
	}
}

func BenchmarkGetRepoRootConcurrent(b *testing.B) {
	// Setup
	tempDir, err := os.MkdirTemp("", "bench_concurrent_*")
	require.NoError(b, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	oldDir, err := os.Getwd()
	require.NoError(b, err)
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	err = os.Chdir(tempDir)
	require.NoError(b, err)

	ctx := context.Background()
	require.NoError(b, exec.CommandContext(ctx, "git", "init").Run())

	sharedCtx := NewContext()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = sharedCtx.GetRepoRoot(context.Background())
		}
	})
}
