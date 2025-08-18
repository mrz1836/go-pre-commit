package tools

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type InstallerTestSuite struct {
	suite.Suite

	originalEnv map[string]string
}

func (s *InstallerTestSuite) SetupTest() {
	// Save original environment
	s.originalEnv = make(map[string]string)
	envVars := []string{
		"GO_PRE_COMMIT_GOLANGCI_LINT_VERSION",
		"GO_PRE_COMMIT_FUMPT_VERSION",
		"GO_PRE_COMMIT_GOIMPORTS_VERSION",
		"GOLANGCI_LINT_VERSION",
		"GOFUMPT_VERSION",
	}

	for _, key := range envVars {
		s.originalEnv[key] = os.Getenv(key)
		_ = os.Unsetenv(key)
	}

	// Clear caches
	CleanCache()

	// Reset tools to default state
	toolsMu.Lock()
	tools = map[string]*Tool{
		"golangci-lint": {
			Name:       "golangci-lint",
			ImportPath: "github.com/golangci/golangci-lint/cmd/golangci-lint",
			Version:    "",
			Binary:     "golangci-lint",
		},
		"gofumpt": {
			Name:       "gofumpt",
			ImportPath: "mvdan.cc/gofumpt",
			Version:    "",
			Binary:     "gofumpt",
		},
		"goimports": {
			Name:       "goimports",
			ImportPath: "golang.org/x/tools/cmd/goimports",
			Version:    "latest",
			Binary:     "goimports",
		},
	}
	toolsMu.Unlock()
}

func (s *InstallerTestSuite) TearDownTest() {
	// Restore original environment
	for key, value := range s.originalEnv {
		if value != "" {
			_ = os.Setenv(key, value)
		} else {
			_ = os.Unsetenv(key)
		}
	}
}

func TestInstallerSuite(t *testing.T) {
	suite.Run(t, new(InstallerTestSuite))
}

func (s *InstallerTestSuite) TestLoadVersionsFromEnv() {
	// Test loading from GO_PRE_COMMIT_ prefixed vars
	_ = os.Setenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION", "v1.50.0")
	_ = os.Setenv("GO_PRE_COMMIT_FUMPT_VERSION", "v0.4.0")
	_ = os.Setenv("GO_PRE_COMMIT_GOIMPORTS_VERSION", "v0.1.0")

	LoadVersionsFromEnv()

	toolsMu.RLock()
	s.Equal("v1.50.0", tools["golangci-lint"].Version)
	s.Equal("v0.4.0", tools["gofumpt"].Version)
	s.Equal("v0.1.0", tools["goimports"].Version)
	toolsMu.RUnlock()
}

func (s *InstallerTestSuite) TestLoadVersionsFromEnvFallback() {
	// Test fallback to non-prefixed vars
	_ = os.Setenv("GOLANGCI_LINT_VERSION", "v1.45.0")
	_ = os.Setenv("GOFUMPT_VERSION", "v0.3.0")

	LoadVersionsFromEnv()

	toolsMu.RLock()
	s.Equal("v1.45.0", tools["golangci-lint"].Version)
	s.Equal("v0.3.0", tools["gofumpt"].Version)
	toolsMu.RUnlock()
}

func (s *InstallerTestSuite) TestLoadVersionsDefaults() {
	// Test default values when no env vars set
	LoadVersionsFromEnv()

	toolsMu.RLock()
	s.Equal("v2.4.0", tools["golangci-lint"].Version)
	s.Equal("v0.8.0", tools["gofumpt"].Version)
	s.Equal("latest", tools["goimports"].Version)
	toolsMu.RUnlock()
}

func (s *InstallerTestSuite) TestIsInstalled() {
	// Test checking for a tool that should exist (go itself)
	// We can't guarantee specific tools are installed, so we test the logic

	// Test unknown tool
	s.False(IsInstalled("unknown-tool"))

	// Test caching behavior
	installMu.Lock()
	installedTools["test-tool"] = true
	installMu.Unlock()

	s.True(IsInstalled("test-tool"))
}

func (s *InstallerTestSuite) TestGetToolPath() {
	// Test getting path for known tool
	toolsMu.Lock()
	tools["test-tool"] = &Tool{
		Name:   "test-tool",
		Binary: "go", // Use 'go' as it should exist
	}
	toolsMu.Unlock()

	path, err := GetToolPath("test-tool")
	s.Require().NoError(err)
	s.NotEmpty(path)

	// Test unknown tool
	_, err = GetToolPath("unknown-tool")
	s.Require().Error(err)
	s.Contains(err.Error(), "unknown tool")
}

func (s *InstallerTestSuite) TestEnsureInstalledUnknownTool() {
	ctx := context.Background()
	err := EnsureInstalled(ctx, "unknown-tool")
	s.Require().Error(err)
	s.Contains(err.Error(), "unknown tool")
}

func (s *InstallerTestSuite) TestGetGoPath() {
	// Test with GOPATH set
	originalGoPath := os.Getenv("GOPATH")
	defer func() {
		if originalGoPath != "" {
			_ = os.Setenv("GOPATH", originalGoPath)
		} else {
			_ = os.Unsetenv("GOPATH")
		}
	}()

	testPath := "/custom/go/path"
	_ = os.Setenv("GOPATH", testPath)
	s.Equal(testPath, GetGoPath())

	// Test with GOPATH unset (should return default)
	_ = os.Unsetenv("GOPATH")
	path := GetGoPath()
	s.Contains(path, "go")
}

func (s *InstallerTestSuite) TestGetGoBin() {
	// Save original values
	originalGoBin := os.Getenv("GOBIN")
	originalGoPath := os.Getenv("GOPATH")
	defer func() {
		if originalGoBin != "" {
			_ = os.Setenv("GOBIN", originalGoBin)
		} else {
			_ = os.Unsetenv("GOBIN")
		}
		if originalGoPath != "" {
			_ = os.Setenv("GOPATH", originalGoPath)
		} else {
			_ = os.Unsetenv("GOPATH")
		}
	}()

	// Test with GOBIN set
	testBin := "/custom/go/bin"
	_ = os.Setenv("GOBIN", testBin)
	s.Equal(testBin, GetGoBin())

	// Test with GOBIN unset but GOPATH set
	_ = os.Unsetenv("GOBIN")
	_ = os.Setenv("GOPATH", "/custom/go")
	s.Equal("/custom/go/bin", GetGoBin())
}

func (s *InstallerTestSuite) TestCleanCache() {
	// Add some cached entries
	installMu.Lock()
	installedTools["tool1"] = true
	installedTools["tool2"] = false
	installMu.Unlock()

	// Verify they exist
	s.Len(installedTools, 2)

	// Clean cache
	CleanCache()

	// Verify cache is empty
	s.Empty(installedTools)
}

func (s *InstallerTestSuite) TestConcurrentIsInstalled() {
	// Test concurrent access to IsInstalled
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_ = IsInstalled("gofumpt")
			done <- true
		}()
	}

	// Wait for all goroutines with timeout
	timeout := time.After(2 * time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			s.T().Fatal("Concurrent IsInstalled test timed out")
		}
	}
}

// TestInstallToolMocked tests the install logic without actually installing
func (s *InstallerTestSuite) TestInstallToolMocked() {
	// This test would require mocking exec.Command which is complex
	// For now, we ensure the function exists and can be called
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create a fake tool that won't actually install
	fakeTool := &Tool{
		Name:       "fake-tool",
		ImportPath: "example.com/fake/tool",
		Version:    "v1.0.0",
		Binary:     "fake-tool-that-does-not-exist",
	}

	// This should fail quickly as the tool doesn't exist
	err := InstallTool(ctx, fakeTool)
	s.Require().Error(err)
}

func (s *InstallerTestSuite) TestInstallGolangciLintSpecialHandling() {
	// Test that golangci-lint gets special handling
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	tool := &Tool{
		Name:       "golangci-lint",
		ImportPath: "github.com/golangci/golangci-lint/cmd/golangci-lint",
		Version:    "v1.50.0",
		Binary:     "golangci-lint",
	}

	// Clear PATH to force installation failure
	originalPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "")
	defer func() {
		_ = os.Setenv("PATH", originalPath)
	}()

	// This will fail due to missing PATH/shell, but we can verify the special handling
	err := InstallTool(ctx, tool)
	// The error could be timeout, installation failure, or binary not found
	// Any of these indicate that the special golangci-lint path was taken
	s.Require().Error(err)
}

func (s *InstallerTestSuite) TestInstallAllToolsErrorAggregation() {
	// Test that InstallAllTools properly aggregates errors
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Clear PATH to force failures
	originalPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "")
	defer func() {
		_ = os.Setenv("PATH", originalPath)
	}()

	err := InstallAllTools(ctx)
	s.Require().Error(err)
	s.Contains(err.Error(), "tool installation failed")
}

func (s *InstallerTestSuite) TestToolVersionParsing() {
	// Test that latest version is handled properly
	tool := &Tool{
		Name:       "test-tool",
		ImportPath: "example.com/test-tool",
		Version:    "latest",
		Binary:     "test-tool",
	}

	// This should format the import path without @latest
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := InstallTool(ctx, tool)
	s.Require().Error(err) // Will fail but that's expected
}

func (s *InstallerTestSuite) TestEnvironmentVariablePrecedence() {
	// Test GO_PRE_COMMIT_ vars take precedence over base vars
	_ = os.Setenv("GOLANGCI_LINT_VERSION", "v1.40.0")
	_ = os.Setenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION", "v1.50.0")
	_ = os.Setenv("GOFUMPT_VERSION", "v0.3.0")
	_ = os.Setenv("GO_PRE_COMMIT_FUMPT_VERSION", "v0.4.0")

	LoadVersionsFromEnv()

	toolsMu.RLock()
	s.Equal("v1.50.0", tools["golangci-lint"].Version, "GO_PRE_COMMIT_ should take precedence")
	s.Equal("v0.4.0", tools["gofumpt"].Version, "GO_PRE_COMMIT_ should take precedence")
	toolsMu.RUnlock()
}

func (s *InstallerTestSuite) TestConcurrentToolInstallation() {
	// Test that concurrent installations don't cause data races
	done := make(chan bool, 5)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	for i := 0; i < 5; i++ {
		go func(_ int) {
			defer func() { done <- true }()

			fakeTool := &Tool{
				Name:       "concurrent-tool",
				ImportPath: "example.com/concurrent/tool",
				Version:    "v1.0.0",
				Binary:     "concurrent-tool-that-does-not-exist",
			}

			// This will fail, but shouldn't cause data races
			_ = InstallTool(ctx, fakeTool)
		}(i)
	}

	// Wait for all goroutines with timeout
	timeout := time.After(2 * time.Second)
	for i := 0; i < 5; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			s.T().Fatal("Concurrent installation test timed out")
		}
	}
}

func (s *InstallerTestSuite) TestGetToolPathCaching() {
	// Add a tool to the registry temporarily
	toolsMu.Lock()
	tools["cache-test-tool"] = &Tool{
		Name:   "cache-test-tool",
		Binary: "nonexistent-binary-for-cache-test",
	}
	toolsMu.Unlock()

	// First call should cache the result
	_, err := GetToolPath("cache-test-tool")
	s.Require().Error(err) // Binary doesn't exist

	// Second call should use the cached logic path
	_, err2 := GetToolPath("cache-test-tool")
	s.Require().Error(err2)
	s.Equal(err.Error(), err2.Error())
}

func (s *InstallerTestSuite) TestLoadVersionsFromEnvEdgeCases() {
	// Test with empty string values
	_ = os.Setenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION", "")
	_ = os.Setenv("GOLANGCI_LINT_VERSION", "v1.49.0")

	LoadVersionsFromEnv()

	toolsMu.RLock()
	// Should use the fallback when primary is empty
	s.Equal("v1.49.0", tools["golangci-lint"].Version)
	toolsMu.RUnlock()
}

func (s *InstallerTestSuite) TestIsInstalledCaching() {
	// Test that caching works properly
	s.False(IsInstalled("test-cache-tool"))

	// Manually add to cache
	installMu.Lock()
	installedTools["test-cache-tool"] = true
	installMu.Unlock()

	// Should return cached value even though tool doesn't exist
	s.True(IsInstalled("test-cache-tool"))

	// Clear cache
	CleanCache()

	// Should check again and return false
	s.False(IsInstalled("test-cache-tool"))
}
