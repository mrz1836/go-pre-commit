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
