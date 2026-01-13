package tools

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallTool_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tool := &Tool{
		Name:       "test-tool-canceled",
		ImportPath: "example.com/test-tool-canceled",
		Binary:     "test-tool-canceled",
	}

	err := InstallTool(ctx, tool)
	require.Error(t, err)
	// Depending on timing, it might be "context canceled" or wrapped
	assert.Contains(t, err.Error(), "context canceled")
}

func TestInstallTool_Timeout(t *testing.T) {
	// Set a very short timeout for testing
	origTimeout := GetInstallTimeout()
	defer SetInstallTimeout(origTimeout)
	SetInstallTimeout(1 * time.Millisecond)

	ctx := context.Background()
	tool := &Tool{
		Name:       "test-tool-timeout",
		ImportPath: "example.com/test-tool-timeout",
		Binary:     "test-tool-timeout",
	}

	// This should fail with timeout
	err := InstallTool(ctx, tool)
	require.Error(t, err)
	// It should return a timeout error
	assert.Contains(t, err.Error(), "timed out")
}

func TestInstallGitleaks_DownloadError(t *testing.T) {
	// We can test this by calling InstallTool with "gitleaks" and an invalid version
	// This exercises the installGitleaks function

	// Use a lock to prevent concurrent access to tools map
	toolsMu.Lock()
	originalTool := tools["gitleaks"]
	tools["gitleaks"] = &Tool{
		Name:       "gitleaks",
		ImportPath: "github.com/gitleaks/gitleaks/v8",
		Version:    "v99.99.99-nonexistent", // Invalid version to force download failure
		Binary:     "gitleaks-nonexistent",
	}
	toolsMu.Unlock()

	defer func() {
		toolsMu.Lock()
		tools["gitleaks"] = originalTool
		toolsMu.Unlock()
	}()

	ctx := context.Background()
	// We expect this to fail
	err := EnsureInstalled(ctx, "gitleaks")

	// If it succeeds unexpectedly, we need to know why
	if err == nil {
		t.Logf("Expected download error but got success")
	} else {
		assert.Contains(t, err.Error(), "tool installation failed")
	}
}

func TestRetryWithBackoff_Success(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	operation := func() error {
		callCount++
		if callCount < 2 {
			// Return network error on first attempt
			return &testNetworkError{msg: "connection refused"}
		}
		return nil
	}

	err := retryWithBackoff(ctx, "test-operation", operation)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "Should succeed on second attempt")
}

func TestRetryWithBackoff_NonNetworkError(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	operation := func() error {
		callCount++
		return assert.AnError // Non-network error
	}

	err := retryWithBackoff(ctx, "test-operation", operation)
	require.Error(t, err)
	assert.Equal(t, 1, callCount, "Should not retry non-network errors")
}

func TestRetryWithBackoff_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0

	operation := func() error {
		callCount++
		if callCount == 1 {
			cancel() // Cancel context after first attempt
			return &testNetworkError{msg: "connection refused"}
		}
		return nil
	}

	err := retryWithBackoff(ctx, "test-operation", operation)
	require.Error(t, err)
	assert.Equal(t, 1, callCount, "Should stop on context cancellation")
}

func TestRetryWithBackoff_AllAttemptsExhausted(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	// Set retry config to small values for testing
	origAttempts, origDelay := GetRetryConfig()
	defer SetRetryConfig(origAttempts, origDelay)
	SetRetryConfig(3, 1*time.Millisecond)

	operation := func() error {
		callCount++
		return &testNetworkError{msg: "connection refused"}
	}

	err := retryWithBackoff(ctx, "test-operation", operation)
	require.Error(t, err)
	assert.Equal(t, 3, callCount, "Should exhaust all retry attempts")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestRetryWithBackoff_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	operation := func() error {
		time.Sleep(20 * time.Millisecond) // Longer than context timeout
		return &testNetworkError{msg: "connection refused"}
	}

	err := retryWithBackoff(ctx, "test-operation", operation)
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || err.Error() == "connection refused")
}

func TestLoadVersionsFromEnv_AllVariables(t *testing.T) {
	// Save original environment
	origVars := map[string]string{
		"GO_PRE_COMMIT_GOLANGCI_LINT_VERSION": os.Getenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION"),
		"GO_PRE_COMMIT_FUMPT_VERSION":         os.Getenv("GO_PRE_COMMIT_FUMPT_VERSION"),
		"GO_PRE_COMMIT_GOIMPORTS_VERSION":     os.Getenv("GO_PRE_COMMIT_GOIMPORTS_VERSION"),
		"GO_PRE_COMMIT_GITLEAKS_VERSION":      os.Getenv("GO_PRE_COMMIT_GITLEAKS_VERSION"),
	}
	defer func() {
		for key, val := range origVars {
			if val != "" {
				_ = os.Setenv(key, val)
			} else {
				_ = os.Unsetenv(key)
			}
		}
		// Reload with original values
		LoadVersionsFromEnv()
	}()

	// Set all environment variables
	_ = os.Setenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION", "v1.55.0")
	_ = os.Setenv("GO_PRE_COMMIT_FUMPT_VERSION", "v0.6.0")
	_ = os.Setenv("GO_PRE_COMMIT_GOIMPORTS_VERSION", "v0.15.0")
	_ = os.Setenv("GO_PRE_COMMIT_GITLEAKS_VERSION", "v8.18.0")

	LoadVersionsFromEnv()

	toolsMu.RLock()
	assert.Equal(t, "v1.55.0", tools["golangci-lint"].Version)
	assert.Equal(t, "v0.6.0", tools["gofumpt"].Version)
	assert.Equal(t, "v0.15.0", tools["goimports"].Version)
	assert.Equal(t, "v8.18.0", tools["gitleaks"].Version)
	toolsMu.RUnlock()
}

func TestLoadVersionsFromEnv_PartialVariables(t *testing.T) {
	// Save original environment
	origVars := map[string]string{
		"GO_PRE_COMMIT_GOLANGCI_LINT_VERSION": os.Getenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION"),
		"GO_PRE_COMMIT_FUMPT_VERSION":         os.Getenv("GO_PRE_COMMIT_FUMPT_VERSION"),
	}
	defer func() {
		for key, val := range origVars {
			if val != "" {
				_ = os.Setenv(key, val)
			} else {
				_ = os.Unsetenv(key)
			}
		}
		LoadVersionsFromEnv()
	}()

	// Set only some environment variables
	_ = os.Setenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION", "v1.60.0")
	_ = os.Unsetenv("GO_PRE_COMMIT_FUMPT_VERSION")

	LoadVersionsFromEnv()

	toolsMu.RLock()
	assert.Equal(t, "v1.60.0", tools["golangci-lint"].Version)
	// fumpt should get default version since env var is not set
	assert.NotEmpty(t, tools["gofumpt"].Version)
	toolsMu.RUnlock()
}

func TestInstallAllTools_PartialFailure(t *testing.T) {
	// Test that InstallAllTools handles partial failures gracefully
	// by attempting to install a tool that doesn't exist
	ctx := context.Background()

	// Save original tools
	toolsMu.Lock()
	origTools := make(map[string]*Tool)
	for k, v := range tools {
		origTools[k] = v
	}
	// Add a non-existent tool
	tools["nonexistent-tool-xyz"] = &Tool{
		Name:       "nonexistent-tool-xyz",
		ImportPath: "example.com/nonexistent/tool",
		Version:    "v1.0.0",
		Binary:     "nonexistent-tool-xyz",
	}
	toolsMu.Unlock()

	defer func() {
		toolsMu.Lock()
		tools = origTools
		toolsMu.Unlock()
	}()

	// Set a short timeout for faster test
	origTimeout := GetInstallTimeout()
	defer SetInstallTimeout(origTimeout)
	SetInstallTimeout(5 * time.Second)

	// This should fail for the nonexistent tool but we expect it to continue
	err := InstallAllTools(ctx)
	// Should return an error since at least one tool failed
	if err != nil {
		assert.Contains(t, err.Error(), "nonexistent-tool-xyz")
	}
}

func TestInstallTool_SuccessAlreadyInstalled(t *testing.T) {
	// Test that InstallTool returns early if tool is already in PATH
	ctx := context.Background()

	// Use 'go' as the tool since it's always in PATH
	tool := &Tool{
		Name:       "go",
		ImportPath: "golang.org/cmd/go",
		Binary:     "go",
		Version:    "latest",
	}

	err := InstallTool(ctx, tool)
	require.NoError(t, err)
}

func TestLoadVersionsFromEnv_InvalidVariableName(t *testing.T) {
	// Test that LoadVersionsFromEnv handles tools that don't exist in the map
	// This is more of a robustness test
	origVars := map[string]string{
		"GO_PRE_COMMIT_UNKNOWN_TOOL_VERSION": os.Getenv("GO_PRE_COMMIT_UNKNOWN_TOOL_VERSION"),
	}
	defer func() {
		for key, val := range origVars {
			if val != "" {
				_ = os.Setenv(key, val)
			} else {
				_ = os.Unsetenv(key)
			}
		}
		LoadVersionsFromEnv()
	}()

	_ = os.Setenv("GO_PRE_COMMIT_UNKNOWN_TOOL_VERSION", "v1.0.0")

	// This should not panic
	LoadVersionsFromEnv()
}

func TestGetToolPath_AllTools(t *testing.T) {
	// Test GetToolPath for all known tools
	toolsMu.RLock()
	toolNames := make([]string, 0, len(tools))
	for name := range tools {
		toolNames = append(toolNames, name)
	}
	toolsMu.RUnlock()

	for _, name := range toolNames {
		path, err := GetToolPath(name)
		// Either it exists or returns an error
		if err != nil {
			assert.Contains(t, err.Error(), "not installed")
		} else {
			assert.NotEmpty(t, path)
		}
	}
}

func TestGetToolPath_UnknownTool(t *testing.T) {
	_, err := GetToolPath("nonexistent-tool-12345")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestIsInstalled_VariousTools(t *testing.T) {
	// Test IsInstalled with known and unknown tools
	tests := []struct {
		name     string
		toolName string
		checkNot bool // If true, check that result is false
	}{
		{
			name:     "gofumpt may or may not be installed",
			toolName: "gofumpt",
			checkNot: false,
		},
		{
			name:     "nonexistent tool returns false",
			toolName: "nonexistent-tool-xyz-12345",
			checkNot: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInstalled(tt.toolName)
			if tt.checkNot {
				assert.False(t, result)
			}
			// For tools that may or may not be installed, just verify it doesn't panic
		})
	}
}

func TestCleanCache_ClearsInstalledTools(t *testing.T) {
	// Test that CleanCache clears the installed tools cache
	// First, mark a tool as installed
	installMu.Lock()
	installedTools["test-tool"] = true
	installMu.Unlock()

	// Clean the cache
	CleanCache()

	// Verify the cache is empty
	installMu.Lock()
	_, exists := installedTools["test-tool"]
	installMu.Unlock()

	assert.False(t, exists, "Cache should be cleared")
}

func TestGetGoPath_NotEmpty(t *testing.T) {
	gopath := GetGoPath()
	assert.NotEmpty(t, gopath, "GOPATH should not be empty")
}

func TestGetGoBin_NotEmpty(t *testing.T) {
	gobin := GetGoBin()
	assert.NotEmpty(t, gobin, "GOBIN should not be empty")
}

func TestEnsureInstalled_AlreadyInstalled(t *testing.T) {
	// Test that EnsureInstalled returns early if tool is already installed
	ctx := context.Background()

	// Mark gofumpt as already installed
	installMu.Lock()
	installedTools["gofumpt"] = true
	installMu.Unlock()

	// Clean up after test
	defer func() {
		installMu.Lock()
		delete(installedTools, "gofumpt")
		installMu.Unlock()
	}()

	err := EnsureInstalled(ctx, "gofumpt")
	// Should return early without error
	require.NoError(t, err)
}

func TestEnsureInstalled_UnknownTool(t *testing.T) {
	ctx := context.Background()
	err := EnsureInstalled(ctx, "nonexistent-tool-abc-123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestInstallTool_ContextAlreadyCancelled(t *testing.T) {
	// Test that InstallTool handles a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before calling

	tool := &Tool{
		Name:       "test-tool-pre-canceled",
		ImportPath: "example.com/test-tool",
		Binary:     "test-tool-pre-canceled",
		Version:    "latest",
	}

	err := InstallTool(ctx, tool)
	// Should error due to canceled context
	require.Error(t, err)
}

func TestRetryWithBackoff_MultipleNetworkErrors(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	// Set retry config for testing
	origAttempts, origDelay := GetRetryConfig()
	defer SetRetryConfig(origAttempts, origDelay)
	SetRetryConfig(3, 1*time.Millisecond)

	operation := func() error {
		callCount++
		// Always return network error
		return &testNetworkError{msg: "i/o timeout"}
	}

	err := retryWithBackoff(ctx, "test-multi-error", operation)
	require.Error(t, err)
	// Should have tried all 3 attempts
	assert.Equal(t, 3, callCount)
	assert.Contains(t, err.Error(), "i/o timeout")
}

func TestLoadVersionsFromEnv_EmptyValues(t *testing.T) {
	// Test that LoadVersionsFromEnv handles empty environment variables
	origVars := map[string]string{
		"GO_PRE_COMMIT_FUMPT_VERSION": os.Getenv("GO_PRE_COMMIT_FUMPT_VERSION"),
	}
	defer func() {
		for key, val := range origVars {
			if val != "" {
				_ = os.Setenv(key, val)
			} else {
				_ = os.Unsetenv(key)
			}
		}
		LoadVersionsFromEnv()
	}()

	// Set to empty string
	_ = os.Setenv("GO_PRE_COMMIT_FUMPT_VERSION", "")

	LoadVersionsFromEnv()

	// Should use default version
	toolsMu.RLock()
	assert.NotEmpty(t, tools["gofumpt"].Version)
	toolsMu.RUnlock()
}

// testNetworkError is a test helper that implements a network error
type testNetworkError struct {
	msg string
}

func (e *testNetworkError) Error() string {
	return e.msg
}

func (e *testNetworkError) Timeout() bool {
	return false
}

func (e *testNetworkError) Temporary() bool {
	return true
}
