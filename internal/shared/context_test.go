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

func (s *ContextTestSuite) createMakefile(targets []string) {
	makefileContent := "# Test Makefile\n\n"
	for _, target := range targets {
		makefileContent += target + ":\n\t@echo \"Running " + target + "\"\n\n"
	}

	makefilePath := filepath.Join(s.tempDir, "Makefile")
	err := os.WriteFile(makefilePath, []byte(makefileContent), 0o600)
	s.Require().NoError(err)
}

// TestNewContext tests the constructor
func (s *ContextTestSuite) TestNewContext() {
	ctx := NewContext()
	s.NotNil(ctx)
	s.NotNil(ctx.makeTargets)
	s.Empty(ctx.repoRoot)
}

// TestGetRepoRoot tests repository root discovery
func (s *ContextTestSuite) TestGetRepoRoot() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		chdirErr := os.Chdir(oldDir)
		s.Require().NoError(chdirErr)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Test successful repository root discovery
	root, err := ctx.GetRepoRoot(context.Background())
	s.Require().NoError(err)

	// Resolve symlinks to handle macOS /var -> /private/var
	expectedPath, err := filepath.EvalSymlinks(s.tempDir)
	s.Require().NoError(err)
	actualPath, err := filepath.EvalSymlinks(root)
	s.Require().NoError(err)
	s.Equal(expectedPath, actualPath)

	// Test that subsequent calls return cached result
	root2, err2 := ctx.GetRepoRoot(context.Background())
	s.Require().NoError(err2)
	s.Equal(root, root2)
}

// TestGetRepoRootError tests error handling when not in git repository
func (s *ContextTestSuite) TestGetRepoRootError() {
	ctx := NewContext()

	// Create a non-git directory
	nonGitDir, err := os.MkdirTemp("", "non_git_*")
	s.Require().NoError(err)
	defer func() {
		removeErr := os.RemoveAll(nonGitDir)
		s.Require().NoError(removeErr)
	}()

	// Change to non-git directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		chdirErr := os.Chdir(oldDir)
		s.Require().NoError(chdirErr)
	}()

	err = os.Chdir(nonGitDir)
	s.Require().NoError(err)

	// Test error case
	root, err := ctx.GetRepoRoot(context.Background())
	s.Require().Error(err)
	s.Empty(root)

	// Test that error is cached
	root2, err2 := ctx.GetRepoRoot(context.Background())
	s.Require().Error(err2)
	s.Empty(root2)
	s.Equal(err, err2)
}

// TestGetRepoRootTimeout tests timeout handling
func (s *ContextTestSuite) TestGetRepoRootTimeout() {
	ctx := NewContext()

	// Create a context that times out immediately
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Allow some time for the context to timeout
	time.Sleep(10 * time.Millisecond)

	root, err := ctx.GetRepoRoot(timeoutCtx)
	s.Require().Error(err)
	s.Empty(root)
}

// TestHasMakeTarget tests make target detection
func (s *ContextTestSuite) TestHasMakeTarget() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		chdirErr := os.Chdir(oldDir)
		s.Require().NoError(chdirErr)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create Makefile with test targets
	s.createMakefile([]string{"lint", "test", "build"})

	// Test existing target
	hasLint := ctx.HasMakeTarget(context.Background(), "lint")
	s.True(hasLint)

	// Test non-existing target
	hasFoo := ctx.HasMakeTarget(context.Background(), "nonexistent")
	s.False(hasFoo)

	// Test that results are cached
	hasLint2 := ctx.HasMakeTarget(context.Background(), "lint")
	s.True(hasLint2)

	hasFoo2 := ctx.HasMakeTarget(context.Background(), "nonexistent")
	s.False(hasFoo2)
}

// TestHasMakeTargetNoGitRepo tests behavior when not in git repository
func (s *ContextTestSuite) TestHasMakeTargetNoGitRepo() {
	ctx := NewContext()

	// Create a non-git directory
	nonGitDir, err := os.MkdirTemp("", "non_git_*")
	s.Require().NoError(err)
	defer func() {
		removeErr := os.RemoveAll(nonGitDir)
		s.Require().NoError(removeErr)
	}()

	// Change to non-git directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		chdirErr := os.Chdir(oldDir)
		s.Require().NoError(chdirErr)
	}()

	err = os.Chdir(nonGitDir)
	s.Require().NoError(err)

	// Test that it returns false when git repo root cannot be found
	hasTarget := ctx.HasMakeTarget(context.Background(), "lint")
	s.False(hasTarget)
}

// TestHasMakeTargetTimeout tests timeout handling
func (s *ContextTestSuite) TestHasMakeTargetTimeout() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		chdirErr := os.Chdir(oldDir)
		s.Require().NoError(chdirErr)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create Makefile
	s.createMakefile([]string{"lint"})

	// Test with a very short timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Allow some time for the context to timeout
	time.Sleep(10 * time.Millisecond)

	hasTarget := ctx.HasMakeTarget(timeoutCtx, "lint")
	// Should return false due to timeout
	s.False(hasTarget)
}

// TestExecuteMakeTarget tests make target execution
func (s *ContextTestSuite) TestExecuteMakeTarget() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		chdirErr := os.Chdir(oldDir)
		s.Require().NoError(chdirErr)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create Makefile with test targets
	s.createMakefile([]string{"test", "build"})

	// Test successful execution
	err = ctx.ExecuteMakeTarget(context.Background(), "test", 5*time.Second)
	s.Require().NoError(err)

	// Test execution of non-existent target
	err = ctx.ExecuteMakeTarget(context.Background(), "nonexistent", 5*time.Second)
	s.Error(err)
}

// TestExecuteMakeTargetNoGitRepo tests execution when not in git repository
func (s *ContextTestSuite) TestExecuteMakeTargetNoGitRepo() {
	ctx := NewContext()

	// Create a non-git directory
	nonGitDir, err := os.MkdirTemp("", "non_git_*")
	s.Require().NoError(err)
	defer func() {
		removeErr := os.RemoveAll(nonGitDir)
		s.Require().NoError(removeErr)
	}()

	// Change to non-git directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		chdirErr := os.Chdir(oldDir)
		s.Require().NoError(chdirErr)
	}()

	err = os.Chdir(nonGitDir)
	s.Require().NoError(err)

	// Test that it returns error when git repo root cannot be found
	err = ctx.ExecuteMakeTarget(context.Background(), "test", 5*time.Second)
	s.Error(err)
}

// TestExecuteMakeTargetTimeout tests timeout handling
func (s *ContextTestSuite) TestExecuteMakeTargetTimeout() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		chdirErr := os.Chdir(oldDir)
		s.Require().NoError(chdirErr)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create Makefile with a slow target
	makefileContent := `# Test Makefile

slow:
	@sleep 10
`
	makefilePath := filepath.Join(s.tempDir, "Makefile")
	err = os.WriteFile(makefilePath, []byte(makefileContent), 0o600)
	s.Require().NoError(err)

	// Test execution with short timeout
	err = ctx.ExecuteMakeTarget(context.Background(), "slow", 100*time.Millisecond)
	s.Error(err)
}

// TestConcurrentAccess tests concurrent access to the context
func (s *ContextTestSuite) TestConcurrentAccess() {
	ctx := NewContext()

	// Change to temp directory
	oldDir, err := os.Getwd()
	s.Require().NoError(err)
	defer func() {
		chdirErr := os.Chdir(oldDir)
		s.Require().NoError(chdirErr)
	}()

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Create Makefile
	s.createMakefile([]string{"lint", "test", "build"})

	// Run multiple goroutines checking make targets
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			target := "lint"
			if i%2 == 0 {
				target = "test"
			}
			hasTarget := ctx.HasMakeTarget(context.Background(), target)
			s.True(hasTarget)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			s.Fail("Timeout waiting for goroutines")
		}
	}
}

// Unit tests for simple cases
func TestNewContextUnit(t *testing.T) {
	ctx := NewContext()
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.makeTargets)
	assert.Empty(t, ctx.repoRoot)
}

func TestContextCaching(t *testing.T) {
	ctx := NewContext()

	// Test that make targets cache is properly initialized
	assert.NotNil(t, ctx.makeTargets)
	assert.Empty(t, ctx.makeTargets)
}

// Benchmark tests
func BenchmarkHasMakeTarget(b *testing.B) {
	ctx := NewContext()

	// Create temporary git repo
	tempDir, err := os.MkdirTemp("", "bench_*")
	require.NoError(b, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Initialize git repo
	oldDir, err := os.Getwd()
	require.NoError(b, err)
	defer func() {
		_ = os.Chdir(oldDir)
	}()

	require.NoError(b, os.Chdir(tempDir))
	require.NoError(b, exec.CommandContext(context.Background(), "git", "init").Run())
	require.NoError(b, exec.CommandContext(context.Background(), "git", "config", "user.email", "test@example.com").Run())
	require.NoError(b, exec.CommandContext(context.Background(), "git", "config", "user.name", "Test User").Run())

	// Create Makefile
	makefileContent := "lint:\n\t@echo lint\n"
	require.NoError(b, os.WriteFile(filepath.Join(tempDir, "Makefile"), []byte(makefileContent), 0o600))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.HasMakeTarget(context.Background(), "lint")
	}
}

// Tests for GetAvailableMakeTargets function (currently 0% coverage)
func TestGetAvailableMakeTargets(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldDir) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Initialize git repository
	ctx := context.Background()
	require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())

	t.Run("with valid makefile", func(t *testing.T) {
		// Create Makefile with various target types
		makefileContent := `# Test Makefile

help:
	@echo "Available targets:"

build: ## Build the project
	@echo "Building..."

test: ## Run tests
	@echo "Testing..."

# This is a comment
clean:
	@echo "Cleaning..."

# Hidden target
.PHONY: help build test clean

# Variable assignment
VERSION = 1.0.0

# Target with dependency
install: build
	@echo "Installing..."
`
		err := os.WriteFile("Makefile", []byte(makefileContent), 0o600)
		require.NoError(t, err)

		sharedCtx := NewContext()
		targets, err := sharedCtx.GetAvailableMakeTargets(ctx)
		require.NoError(t, err)

		// Should contain the main targets but not hidden ones
		expectedTargets := []string{"help", "build", "test", "clean", "install"}
		for _, expected := range expectedTargets {
			assert.Contains(t, targets, expected, "Missing target: %s", expected)
		}

		// Should not contain hidden or invalid targets
		for _, target := range targets {
			assert.NotEmpty(t, target)
			assert.NotContains(t, target, ".") // No .PHONY etc
			assert.NotContains(t, target, "/") // No paths
			assert.NotContains(t, target, "%") // No patterns
		}
	})

	t.Run("with make command failure", func(t *testing.T) {
		// Remove Makefile to cause make -qp to fail
		_ = os.Remove("Makefile")

		sharedCtx := NewContext()
		targets, err := sharedCtx.GetAvailableMakeTargets(ctx)

		// Should return fallback targets when make fails
		require.NoError(t, err)
		expectedFallbacks := []string{"help", "build", "test", "clean", "install"}
		assert.ElementsMatch(t, expectedFallbacks, targets)
	})

	t.Run("without git repository", func(t *testing.T) {
		// Move to non-git directory
		nonGitDir := t.TempDir()
		err := os.Chdir(nonGitDir)
		require.NoError(t, err)

		sharedCtx := NewContext()
		targets, err := sharedCtx.GetAvailableMakeTargets(ctx)

		// Should fail when not in git repository
		require.Error(t, err)
		assert.Nil(t, targets)
		assert.Contains(t, err.Error(), "failed to find repository root")
	})

	t.Run("with timeout", func(t *testing.T) {
		// Back to git repo
		err := os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create a simple Makefile
		err = os.WriteFile("Makefile", []byte("test:\n\t@echo test"), 0o600)
		require.NoError(t, err)

		// Test with very short timeout
		shortCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure timeout

		sharedCtx := NewContext()
		targets, err := sharedCtx.GetAvailableMakeTargets(shortCtx)

		// Should return fallback targets on timeout
		require.NoError(t, err) // Function handles timeout gracefully
		expectedFallbacks := []string{"help", "build", "test", "clean", "install"}
		assert.ElementsMatch(t, expectedFallbacks, targets)
	})
}

// Tests for parseMakeTargets function (currently 0% coverage)
func TestParseMakeTargets(t *testing.T) {
	sharedCtx := NewContext()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "basic targets",
			input: `build:
test:
clean:`,
			expected: []string{"build", "test", "clean"},
		},
		{
			name: "targets with dependencies",
			input: `build: deps
test: build
clean:`,
			expected: []string{"build", "test", "clean"},
		},
		{
			name: "mixed content with variables and comments",
			input: `# This is a comment
VERSION = 1.0.0

build: ## Build target
	@echo building

# Another comment
test:
	@echo testing

.PHONY: build test`,
			expected: []string{"build", "test"},
		},
		{
			name: "skip hidden and special targets",
			input: `.PHONY: all
.DEFAULT_GOAL := build

all: build test
build:
test:
%pattern:
target/with/slash:
	@echo should skip`,
			expected: []string{"all", "build", "test"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name: "no targets",
			input: `# Just comments
VERSION = 1.0.0
# More comments`,
			expected: []string{},
		},
		{
			name: "duplicate targets",
			input: `build:
test:
build: ## Duplicate build target
clean:`,
			expected: []string{"build", "test", "clean"}, // Should deduplicate
		},
		{
			name: "targets with whitespace",
			input: `  build  :
		test		:
clean	:`,
			expected: []string{"build", "test", "clean"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sharedCtx.parseMakeTargets(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

// Tests for extractTargetDescription function (currently 45% coverage)
func TestExtractTargetDescription(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldDir) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Initialize git repository
	ctx := context.Background()
	require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())

	sharedCtx := NewContext()

	t.Run("with help target that provides descriptions", func(t *testing.T) {
		// Create Makefile with help target
		makefileContent := `help:
	@echo "Available targets:"
	@echo "  build    - Build the project"
	@echo "  test     - Run tests"
	@echo "  clean    - Clean build artifacts"

build:
	@echo "Building..."

test:
	@echo "Testing..."

clean:
	@echo "Cleaning..."
`
		err := os.WriteFile("Makefile", []byte(makefileContent), 0o600)
		require.NoError(t, err)

		// Test extracting description for existing target
		desc := sharedCtx.extractTargetDescription(ctx, tmpDir, "build")
		assert.Contains(t, desc, "Build the project")

		desc = sharedCtx.extractTargetDescription(ctx, tmpDir, "test")
		assert.Contains(t, desc, "Run tests")

		desc = sharedCtx.extractTargetDescription(ctx, tmpDir, "clean")
		assert.Contains(t, desc, "Clean build artifacts")
	})

	t.Run("without help target", func(t *testing.T) {
		// Create Makefile without help target
		makefileContent := `build:
	@echo "Building..."

test:
	@echo "Testing..."
`
		err := os.WriteFile("Makefile", []byte(makefileContent), 0o600)
		require.NoError(t, err)

		// Should return empty description when help fails
		desc := sharedCtx.extractTargetDescription(ctx, tmpDir, "build")
		assert.Empty(t, desc)
	})

	t.Run("for non-existent target", func(t *testing.T) {
		// Create Makefile with help that doesn't mention the target
		makefileContent := `help:
	@echo "Available targets:"
	@echo "  build - Build the project"

build:
	@echo "Building..."
`
		err := os.WriteFile("Makefile", []byte(makefileContent), 0o600)
		require.NoError(t, err)

		// Should return empty description for non-mentioned target
		desc := sharedCtx.extractTargetDescription(ctx, tmpDir, "nonexistent")
		assert.Empty(t, desc)
	})

	t.Run("with timeout", func(t *testing.T) {
		// Test timeout behavior by using a very short timeout
		shortCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()

		// Create Makefile with normal help target
		makefileContent := `help:
	@echo "Available targets:"
	@echo "  build - Build the project"

build:
	@echo "Building..."
`
		err := os.WriteFile("Makefile", []byte(makefileContent), 0o600)
		require.NoError(t, err)

		// Should handle timeout gracefully
		start := time.Now()
		desc := sharedCtx.extractTargetDescription(shortCtx, tmpDir, "build")
		elapsed := time.Since(start)

		// Should return empty on timeout and complete quickly
		assert.Empty(t, desc)
		assert.Less(t, elapsed, 1*time.Second)
	})

	t.Run("with various description formats", func(t *testing.T) {
		// Test different formats of help output
		makefileContent := `help:
	@echo "build: Build the application"
	@echo "test - Run all tests"
	@echo "clean:      Clean up files"
	@echo "  deploy  -  Deploy to production"
	@echo "format		Format the code"

build:
	@echo "Building..."

test:
	@echo "Testing..."

clean:
	@echo "Cleaning..."

deploy:
	@echo "Deploying..."

format:
	@echo "Formatting..."
`
		err := os.WriteFile("Makefile", []byte(makefileContent), 0o600)
		require.NoError(t, err)

		// Test different description extraction patterns
		desc := sharedCtx.extractTargetDescription(ctx, tmpDir, "build")
		assert.NotEmpty(t, desc)
		assert.Contains(t, desc, "Build the application")

		desc = sharedCtx.extractTargetDescription(ctx, tmpDir, "test")
		assert.Contains(t, desc, "Run all tests")

		desc = sharedCtx.extractTargetDescription(ctx, tmpDir, "clean")
		assert.Contains(t, desc, "Clean up files")

		desc = sharedCtx.extractTargetDescription(ctx, tmpDir, "deploy")
		assert.Contains(t, desc, "Deploy to production")

		desc = sharedCtx.extractTargetDescription(ctx, tmpDir, "format")
		assert.Contains(t, desc, "Format the code")
	})
}
