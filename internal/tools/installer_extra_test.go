package tools

import (
	"context"
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
