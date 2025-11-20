// Package tools provides comprehensive timeout testing for tool installation
package tools

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

func TestSetGetInstallTimeout(t *testing.T) {
	originalTimeout := GetInstallTimeout()
	defer SetInstallTimeout(originalTimeout) // Restore original

	// Test setting and getting timeout
	newTimeout := 10 * time.Minute
	SetInstallTimeout(newTimeout)

	retrieved := GetInstallTimeout()
	assert.Equal(t, newTimeout, retrieved)
}

func TestSetInstallTimeout_ConcurrentAccess(t *testing.T) {
	originalTimeout := GetInstallTimeout()
	defer SetInstallTimeout(originalTimeout) // Restore original

	// Test concurrent access to timeout configuration
	done := make(chan bool, 10)

	// Start multiple goroutines that set and get timeouts
	for i := 0; i < 10; i++ {
		go func(timeout time.Duration) {
			defer func() { done <- true }()

			SetInstallTimeout(timeout)
			retrieved := GetInstallTimeout()

			// Should be able to set and get without data races
			assert.Positive(t, retrieved)
		}(time.Duration(i+1) * time.Minute)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Final timeout should be some valid value
	final := GetInstallTimeout()
	assert.Positive(t, final)
}

func TestInstallTool_TimeoutConfiguration(t *testing.T) {
	// Test that InstallTool respects the configured timeout
	originalTimeout := GetInstallTimeout()
	defer SetInstallTimeout(originalTimeout)

	// Set a short timeout for testing
	testTimeout := 100 * time.Millisecond
	SetInstallTimeout(testTimeout)

	// Create a fake tool that doesn't exist (will cause go install to hang/fail)
	fakeTool := &Tool{
		Name:       "nonexistent-tool-12345",
		Binary:     "nonexistent-tool-12345",
		ImportPath: "github.com/nonexistent/nonexistent-tool-12345",
		Version:    "latest",
	}

	ctx := context.Background()
	start := time.Now()

	err := InstallTool(ctx, fakeTool)
	elapsed := time.Since(start)

	// Should have failed due to timeout or installation error
	require.Error(t, err)

	// If it's a timeout error, check it's the right type
	var timeoutErr *prerrors.TimeoutError
	if errors.As(err, &timeoutErr) {
		assert.Equal(t, "Tool installation", timeoutErr.Operation)
		assert.Equal(t, fakeTool.Name, timeoutErr.Context)
		assert.Contains(t, timeoutErr.ConfigVar, "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT")
	}

	// Should not have taken significantly longer than the timeout
	// (allowing some buffer for context switching and cleanup)
	assert.Less(t, elapsed, testTimeout*3,
		"Installation took %v, expected less than %v", elapsed, testTimeout*3)
}

func TestInstallTool_ContextAlreadyCanceled(t *testing.T) {
	originalTimeout := GetInstallTimeout()
	defer SetInstallTimeout(originalTimeout)

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	fakeTool := &Tool{
		Name:       "test-tool",
		Binary:     "test-tool",
		ImportPath: "github.com/test/test-tool",
		Version:    "latest",
	}

	err := InstallTool(ctx, fakeTool)
	require.Error(t, err)

	// Should get an appropriate error for canceled context
	// The exact error type may vary, but it should be related to cancellation
}

func TestInstallGolangciLint_TimeoutHandling(t *testing.T) {
	originalTimeout := GetInstallTimeout()
	defer SetInstallTimeout(originalTimeout)

	// Set a very short timeout
	testTimeout := 50 * time.Millisecond
	SetInstallTimeout(testTimeout)

	// Create a context that will timeout quickly
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	start := time.Now()
	err := installGolangciLint(ctx, "latest")
	elapsed := time.Since(start)

	require.Error(t, err)

	// Check if it's a timeout error
	var timeoutErr *prerrors.TimeoutError
	if errors.As(err, &timeoutErr) {
		assert.Equal(t, "Tool installation", timeoutErr.Operation)
		assert.Equal(t, "golangci-lint", timeoutErr.Context)
		assert.Contains(t, timeoutErr.ConfigVar, "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT")
	}

	// Should have completed within reasonable time due to timeout
	// Allow up to 10 seconds for timeout handling and cleanup (increased buffer for slower systems)
	assert.Less(t, elapsed, 10*time.Second)
}

func TestInstallTool_ProgressTracking(t *testing.T) {
	// Test that progress tracking works with tool installation
	originalTimeout := GetInstallTimeout()
	defer SetInstallTimeout(originalTimeout)

	// Set a reasonable timeout
	SetInstallTimeout(2 * time.Second)

	// This test mainly ensures progress tracking doesn't break installation
	// We can't easily test the actual progress output without complex mocking

	fakeTool := &Tool{
		Name:       "quick-fail-tool",
		Binary:     "quick-fail-tool",
		ImportPath: "github.com/nonexistent/quick-fail-tool",
		Version:    "latest",
	}

	ctx := context.Background()
	err := InstallTool(ctx, fakeTool)

	// Should fail (tool doesn't exist), but progress tracking shouldn't cause panics
	require.Error(t, err)
}

func TestTimeout_ErrorWrapping(t *testing.T) {
	// Test that timeout errors can be properly unwrapped and identified
	toolTimeoutErr := prerrors.NewToolInstallTimeoutError("test-tool", 5*time.Minute, 3*time.Minute)

	// Test wrapping
	wrappedErr := fmt.Errorf("installation failed: %w", toolTimeoutErr)

	// Test unwrapping
	var timeoutErr *prerrors.TimeoutError
	require.ErrorAs(t, wrappedErr, &timeoutErr)
	assert.Equal(t, "test-tool", timeoutErr.Context)

	// Test Is behavior
	assert.ErrorIs(t, wrappedErr, prerrors.ErrTimeout)
}

func TestTimeout_DefaultValue(t *testing.T) {
	// Test that the default timeout is reasonable
	defaultTimeout := GetInstallTimeout()

	// Should be at least 1 minute but not more than 1 hour
	assert.GreaterOrEqual(t, defaultTimeout, 1*time.Minute)
	assert.LessOrEqual(t, defaultTimeout, 1*time.Hour)
}

func TestTimeout_EdgeCases(t *testing.T) {
	originalTimeout := GetInstallTimeout()
	defer SetInstallTimeout(originalTimeout)

	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "very short timeout",
			timeout: 1 * time.Millisecond,
		},
		{
			name:    "zero timeout",
			timeout: 0,
		},
		{
			name:    "negative timeout",
			timeout: -1 * time.Second,
		},
		{
			name:    "very long timeout",
			timeout: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic when setting any timeout value
			require.NotPanics(t, func() {
				SetInstallTimeout(tt.timeout)
			})

			retrieved := GetInstallTimeout()
			assert.Equal(t, tt.timeout, retrieved)
		})
	}
}

func TestInstallTool_RealScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real installation test in short mode")
	}

	originalTimeout := GetInstallTimeout()
	defer SetInstallTimeout(originalTimeout)

	// Set a reasonable timeout for a real installation
	SetInstallTimeout(30 * time.Second)

	// Test with a real, lightweight tool that should install quickly
	realTool := &Tool{
		Name:       "hello",
		Binary:     "hello",
		ImportPath: "golang.org/x/example/hello",
		Version:    "latest",
	}

	ctx := context.Background()

	// Clean up before test
	_ = exec.CommandContext(ctx, "go", "clean", "-i", realTool.ImportPath).Run() //nolint:gosec // Test cleanup command with known input

	err := InstallTool(ctx, realTool)
	// This might still fail due to network issues, but shouldn't timeout
	if err != nil {
		// If it fails, it should not be due to our timeout mechanism
		var timeoutErr *prerrors.TimeoutError
		assert.NotErrorAs(t, err, &timeoutErr,
			"Should not fail due to timeout with 30s limit: %v", err)
	}
}

// Benchmark timeout configuration access
func BenchmarkTimeout_Access(b *testing.B) {
	originalTimeout := GetInstallTimeout()
	defer SetInstallTimeout(originalTimeout)

	b.ResetTimer()

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GetInstallTimeout()
		}
	})

	b.Run("Set", func(b *testing.B) {
		timeout := 5 * time.Minute
		for i := 0; i < b.N; i++ {
			SetInstallTimeout(timeout)
		}
	})

	b.Run("ConcurrentAccess", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				SetInstallTimeout(5 * time.Minute)
				_ = GetInstallTimeout()
			}
		})
	})
}
