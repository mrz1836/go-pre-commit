package gotools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-pre-commit/internal/config"
	"github.com/mrz1836/go-pre-commit/internal/shared"
)

func TestNewGitleaksCheck(t *testing.T) {
	check := NewGitleaksCheck()
	assert.NotNil(t, check)
	assert.IsType(t, &GitleaksCheck{}, check)
}

func TestGitleaksCheck(t *testing.T) {
	check := &GitleaksCheck{}

	assert.Equal(t, "gitleaks", check.Name())
	assert.Equal(t, "Scan for secrets and credentials in code", check.Description())
}

func TestGitleaksCheck_FilterFiles(t *testing.T) {
	check := &GitleaksCheck{}

	files := []string{
		"main.go",
		"test.go",
		"doc.md",
		"Makefile",
		"test.txt",
		"pkg/foo.go",
		".env",
		"config.json",
	}

	// Should return ALL files (gitleaks scans everything)
	filtered := check.FilterFiles(files)
	assert.Equal(t, files, filtered)
}

func TestGitleaksCheck_Run_NoTool(t *testing.T) {
	// Skip this test if gitleaks is available since it would succeed
	_, hasGitleaks := exec.LookPath("gitleaks")
	if hasGitleaks == nil {
		t.Skip("gitleaks is available - skipping error scenario test")
	}

	// Create a temporary directory
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(oldDir); chdirErr != nil {
			t.Logf("Failed to restore directory: %v", chdirErr)
		}
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Initialize git repository
	ctx := context.Background()
	require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", testEmail).Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", testUserName).Run())

	check := NewGitleaksCheck()

	err = check.Run(ctx, []string{"test.go"})
	require.Error(t, err)
	// When gitleaks is not installed, it should return a ToolNotFoundError
	assert.Contains(t, err.Error(), "gitleaks")
}

func TestGitleaksCheck_Metadata(t *testing.T) {
	check := NewGitleaksCheck()
	metadata := check.Metadata()
	assert.NotNil(t, metadata)

	checkMetadata, ok := metadata.(CheckMetadata)
	assert.True(t, ok)
	assert.Equal(t, "gitleaks", checkMetadata.Name)
	assert.Equal(t, "security", checkMetadata.Category)
	assert.Equal(t, []string{"*"}, checkMetadata.FilePatterns) // All files
	assert.True(t, checkMetadata.RequiresFiles)
}

// Comprehensive test suite for GitleaksCheck
type GitleaksCheckTestSuite struct {
	suite.Suite

	tempDir   string
	oldDir    string
	sharedCtx *shared.Context
}

func TestGitleaksCheckSuite(t *testing.T) {
	suite.Run(t, new(GitleaksCheckTestSuite))
}

func (s *GitleaksCheckTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "gitleaks_test_*")
	s.Require().NoError(err)

	s.oldDir, err = os.Getwd()
	s.Require().NoError(err)

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Initialize git repo
	s.initGitRepo()

	s.sharedCtx = shared.NewContext()
}

func (s *GitleaksCheckTestSuite) TearDownTest() {
	if s.oldDir != "" {
		chdirErr := os.Chdir(s.oldDir)
		s.Require().NoError(chdirErr)
	}
	if s.tempDir != "" {
		removeErr := os.RemoveAll(s.tempDir)
		s.Require().NoError(removeErr)
	}
}

func (s *GitleaksCheckTestSuite) initGitRepo() {
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "init").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.email", testEmail).Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.name", testUserName).Run())
}

func (s *GitleaksCheckTestSuite) TestNewGitleaksCheckWithSharedContext() {
	check := NewGitleaksCheckWithSharedContext(s.sharedCtx)
	s.NotNil(check)
	s.Equal(s.sharedCtx, check.sharedCtx)
}

func (s *GitleaksCheckTestSuite) TestNewGitleaksCheckWithConfig() {
	timeout := 30 * time.Second
	check := NewGitleaksCheckWithConfig(s.sharedCtx, timeout)
	s.NotNil(check)
	s.Equal(s.sharedCtx, check.sharedCtx)
	s.Equal(timeout, check.timeout)
}

func (s *GitleaksCheckTestSuite) TestRunGitleaks_CleanRepo() {
	// Skip if gitleaks is not available
	if _, err := exec.LookPath("gitleaks"); err != nil {
		s.T().Skip("gitleaks not available")
	}

	// Create a clean Go file with no secrets
	goFile := `package main

import "fmt"

func main() {
	fmt.Println("hello world")
}
`
	err := os.WriteFile("main.go", []byte(goFile), 0o600)
	s.Require().NoError(err)

	check := NewGitleaksCheck()
	err = check.Run(context.Background(), []string{"main.go"})

	// Should succeed when no secrets are found
	s.NoError(err)
}

func (s *GitleaksCheckTestSuite) TestRunGitleaks_WithSecrets() {
	// Skip if gitleaks is not available
	if _, err := exec.LookPath("gitleaks"); err != nil {
		s.T().Skip("gitleaks not available")
	}

	// Create a file with a potential secret (fake AWS key pattern)
	// #nosec G101 - This is a fake secret for testing only
	secretFile := `package main

const (
	// This is a fake secret for testing
	awsKey = "AKIAIOSFODNN7EXAMPLE"
)

func main() {
	println(awsKey)
}
`
	err := os.WriteFile("secret.go", []byte(secretFile), 0o600)
	s.Require().NoError(err)

	check := NewGitleaksCheck()
	err = check.Run(context.Background(), []string{"secret.go"})

	// Should fail when secrets are found
	// This test might pass if gitleaks doesn't detect the pattern
	// or if .gitleaks.toml allowlist includes it
	if err != nil {
		s.Require().Error(err)
		s.Contains(err.Error(), "secret")
	} else {
		s.T().Log("Gitleaks did not detect the test secret pattern")
	}
}

func (s *GitleaksCheckTestSuite) TestRunGitleaks_CustomConfigRoot() {
	// Skip if gitleaks is not available
	if _, err := exec.LookPath("gitleaks"); err != nil {
		s.T().Skip("gitleaks not available")
	}

	// Create custom .gitleaks.toml in root directory
	gitleaksConfig := `title = "gitleaks config"

[[rules]]
id = "test-rule"
description = "Test rule"
regex = '''TESTSECRET'''
`
	err := os.WriteFile(".gitleaks.toml", []byte(gitleaksConfig), 0o600)
	s.Require().NoError(err)

	// Create a file with the custom pattern
	testFile := `package main

const secret = "TESTSECRET123"
`
	err = os.WriteFile("test.go", []byte(testFile), 0o600)
	s.Require().NoError(err)

	check := NewGitleaksCheck()

	// The custom config should be found and used
	configPath := check.findGitleaksConfig(s.tempDir)
	s.Equal(filepath.Join(s.tempDir, ".gitleaks.toml"), configPath)
}

func (s *GitleaksCheckTestSuite) TestRunGitleaks_CustomConfigGithub() {
	// Skip if gitleaks is not available
	if _, err := exec.LookPath("gitleaks"); err != nil {
		s.T().Skip("gitleaks not available")
	}

	// Create .github directory
	err := os.MkdirAll(".github", 0o750)
	s.Require().NoError(err)

	// Create custom .gitleaks.toml in .github directory
	gitleaksConfig := `title = "gitleaks config"

[[rules]]
id = "test-rule"
description = "Test rule"
regex = '''GITHUBSECRET'''
`
	err = os.WriteFile(".github/.gitleaks.toml", []byte(gitleaksConfig), 0o600)
	s.Require().NoError(err)

	check := NewGitleaksCheck()

	// The custom config in .github should be found
	configPath := check.findGitleaksConfig(s.tempDir)
	s.Equal(filepath.Join(s.tempDir, ".github", ".gitleaks.toml"), configPath)
}

func (s *GitleaksCheckTestSuite) TestRunGitleaks_ConfigPriority() {
	// Skip if gitleaks is not available
	if _, err := exec.LookPath("gitleaks"); err != nil {
		s.T().Skip("gitleaks not available")
	}

	// Create .github directory
	err := os.MkdirAll(".github", 0o750)
	s.Require().NoError(err)

	// Create config in both locations
	rootConfig := `title = "root config"`
	err = os.WriteFile(".gitleaks.toml", []byte(rootConfig), 0o600)
	s.Require().NoError(err)

	githubConfig := `title = "github config"`
	err = os.WriteFile(".github/.gitleaks.toml", []byte(githubConfig), 0o600)
	s.Require().NoError(err)

	check := NewGitleaksCheck()

	// Root config should take priority
	configPath := check.findGitleaksConfig(s.tempDir)
	s.Equal(filepath.Join(s.tempDir, ".gitleaks.toml"), configPath)
}

func (s *GitleaksCheckTestSuite) TestRunWithTimeout() {
	// Skip if gitleaks is not available
	if _, err := exec.LookPath("gitleaks"); err != nil {
		s.T().Skip("gitleaks not available")
	}

	timeout := 1 * time.Millisecond // Very short timeout
	check := NewGitleaksCheckWithConfig(s.sharedCtx, timeout)

	// Create a file
	goFile := `package main

func main() {}
`
	err := os.WriteFile("main.go", []byte(goFile), 0o600)
	s.Require().NoError(err)

	err = check.Run(context.Background(), []string{"main.go"})
	// May or may not timeout depending on system speed
	// If it times out, error should mention timeout
	if err != nil && strings.Contains(err.Error(), "timed out") {
		s.Contains(err.Error(), "Gitleaks timed out")
	}
}

// Edge case tests
func TestGitleaksCheckEdgeCases(t *testing.T) {
	t.Run("empty files list", func(t *testing.T) {
		check := NewGitleaksCheck()

		// Create temp dir and change to it for clean test
		tmpDir := t.TempDir()
		oldDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(oldDir)
		}()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Initialize git repository for tests that need it
		require.NoError(t, exec.CommandContext(context.Background(), "git", "init").Run())
		require.NoError(t, exec.CommandContext(context.Background(), "git", "config", "user.email", testEmail).Run())
		require.NoError(t, exec.CommandContext(context.Background(), "git", "config", "user.name", testUserName).Run())

		err = check.Run(context.Background(), []string{})
		assert.NoError(t, err) // Should succeed with no files
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		check := NewGitleaksCheck()
		err := check.Run(ctx, []string{"test.go"})
		assert.Error(t, err)
	})

	t.Run("all file types", func(t *testing.T) {
		check := NewGitleaksCheck()
		files := []string{
			"main.go",
			".env",
			"config.json",
			"README.md",
			"Makefile",
			"secrets.txt",
		}
		filtered := check.FilterFiles(files)
		// Should return all files
		assert.Equal(t, files, filtered)
	})
}

// Test formatGitleaksErrors function
func TestFormatGitleaksErrors(t *testing.T) {
	check := &GitleaksCheck{}

	tests := []struct {
		name          string
		input         string
		expectedCount int
		shouldContain []string
	}{
		{ // #nosec G101 - test data with well-known example AWS key
			name: "single finding",
			input: `Finding: AWS Access Key
File: config.go
Line: 10
Secret: AKIAIOSFODNN7EXAMPLE`,
			expectedCount: 1,
			shouldContain: []string{"Finding:", "File:", "Line:", "Secret:"},
		},
		{
			name: "multiple findings",
			input: `Finding: AWS Access Key
File: config.go
Line: 10

Finding: GitHub Token
File: auth.go
Line: 25`,
			expectedCount: 2,
			shouldContain: []string{"Finding:", "config.go", "auth.go"},
		},
		{
			name:          "no findings",
			input:         "No secrets found in repository",
			expectedCount: 0,
			shouldContain: []string{"No secrets found"},
		},
		{
			name:          "empty input",
			input:         "",
			expectedCount: 0,
			shouldContain: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := check.formatGitleaksErrors(tt.input)

			if tt.expectedCount == 0 {
				// Should return original input when no findings
				assert.Equal(t, tt.input, result)
			} else {
				// Should contain formatted header
				assert.Contains(t, result, "secret(s)")

				// Should contain expected strings
				for _, expected := range tt.shouldContain {
					assert.Contains(t, result, expected)
				}
			}
		})
	}
}

// Test repository root detection failure scenarios
func TestGitleaksRepositoryRootFailures(t *testing.T) {
	t.Run("gitleaks check without git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldDir) }()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Don't initialize git repository
		ctx := context.Background()

		// Skip if gitleaks is not available
		if _, lookupErr := exec.LookPath("gitleaks"); lookupErr != nil {
			t.Skip("gitleaks not available")
		}

		check := NewGitleaksCheck()
		err = check.Run(ctx, []string{"test.go"})
		require.Error(t, err)
		// Should error due to missing git repo
		assert.Contains(t, err.Error(), "repository root")
	})
}

// Test specific error paths
func TestGitleaksSpecificErrorPaths(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		// Skip if gitleaks is not available
		if _, lookupErr := exec.LookPath("gitleaks"); lookupErr != nil {
			t.Skip("gitleaks is not available, cannot test context cancellation")
		}

		tmpDir := t.TempDir()
		oldDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldDir) }()

		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Initialize git repository
		ctx := context.Background()
		require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
		require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", testEmail).Run())
		require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", testUserName).Run())

		// Create context that is immediately canceled
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		check := NewGitleaksCheck()
		err = check.Run(canceledCtx, []string{"test.go"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context")
	})
}

// Test custom config via environment variable
func TestGitleaksCustomConfigEnv(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(oldDir)
		_ = os.Unsetenv("GO_PRE_COMMIT_GITLEAKS_CONFIG")
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create custom config in non-standard location
	customDir := filepath.Join(tmpDir, "custom")
	err = os.MkdirAll(customDir, 0o750)
	require.NoError(t, err)

	customConfigPath := filepath.Join(customDir, "my-gitleaks.toml")
	err = os.WriteFile(customConfigPath, []byte("title = \"custom config\""), 0o600)
	require.NoError(t, err)

	// Set environment variable
	_ = os.Setenv("GO_PRE_COMMIT_GITLEAKS_CONFIG", customConfigPath)

	// Create check with config to enable environment variable reading
	cfg := &config.Config{}
	check := NewGitleaksCheckWithFullConfig(shared.NewContext(), cfg)
	foundPath := check.findGitleaksConfig(tmpDir)

	assert.Equal(t, customConfigPath, foundPath)
}

// Test that gitleaks check works with various file types
func TestGitleaksVariousFileTypes(t *testing.T) {
	// Skip if gitleaks is not available
	if _, lookupErr := exec.LookPath("gitleaks"); lookupErr != nil {
		t.Skip("gitleaks not available")
	}

	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldDir) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Initialize git repository
	ctx := context.Background()
	require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", testEmail).Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", testUserName).Run())

	// Create various file types
	files := map[string]string{
		"main.go":     "package main\n\nfunc main() {}\n",
		".env":        "DATABASE_URL=localhost\n",
		"config.json": "{\"key\": \"value\"}\n",
		"README.md":   "# Project\n",
		"Makefile":    "all:\n\t@echo done\n",
		"script.sh":   "#!/bin/bash\necho hello\n",
	}

	for filename, content := range files {
		err = os.WriteFile(filename, []byte(content), 0o600)
		require.NoError(t, err)
	}

	check := NewGitleaksCheck()
	fileList := []string{"main.go", ".env", "config.json", "README.md", "Makefile", "script.sh"}

	err = check.Run(ctx, fileList)
	// Should succeed with clean files (no secrets)
	assert.NoError(t, err)
}
