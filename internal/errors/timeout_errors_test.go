// Package errors provides comprehensive timeout error testing
package errors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *TimeoutError
		expected string
	}{
		{
			name: "basic timeout error",
			err: &TimeoutError{
				Err:       ErrTimeout,
				Operation: "Tool installation",
				Timeout:   5 * time.Minute,
			},
			expected: "Tool installation timed out after 5m0s",
		},
		{
			name: "timeout error with context",
			err: &TimeoutError{
				Err:       ErrTimeout,
				Operation: "Tool installation",
				Context:   "golangci-lint",
				Timeout:   2 * time.Minute,
			},
			expected: "Tool installation (golangci-lint) timed out after 2m0s",
		},
		{
			name: "timeout error with config var",
			err: &TimeoutError{
				Err:       ErrTimeout,
				Operation: "Check execution",
				Context:   "lint",
				Timeout:   60 * time.Second,
				ConfigVar: "GO_PRE_COMMIT_LINT_TIMEOUT",
			},
			expected: "Check execution (lint) timed out after 1m0s. Consider increasing GO_PRE_COMMIT_LINT_TIMEOUT",
		},
		{
			name: "timeout error with suggested timeout",
			err: &TimeoutError{
				Err:              ErrTimeout,
				Operation:        "Tool installation",
				Context:          "golangci-lint",
				Timeout:          5 * time.Minute,
				ConfigVar:        "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT",
				SuggestedTimeout: 10 * time.Minute,
			},
			expected: "Tool installation (golangci-lint) timed out after 5m0s. Consider increasing GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT (suggested: 10m0s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTimeoutError_Unwrap(t *testing.T) {
	baseErr := ErrToolNotFound
	timeoutErr := &TimeoutError{
		Err: baseErr,
	}

	unwrapped := timeoutErr.Unwrap()
	assert.Equal(t, baseErr, unwrapped)
}

func TestTimeoutError_Is(t *testing.T) {
	baseErr := ErrToolNotFound
	timeoutErr := &TimeoutError{
		Err: baseErr,
	}

	assert.True(t, timeoutErr.Is(baseErr))
	assert.False(t, timeoutErr.Is(ErrFmtIssues))
}

func TestNewTimeoutError(t *testing.T) {
	operation := "Tool installation"
	context := "golangci-lint"
	timeout := 5 * time.Minute
	elapsed := 3 * time.Minute
	configVar := "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT"

	err := NewTimeoutError(operation, context, timeout, elapsed, configVar)

	assert.Equal(t, ErrTimeout, err.Err)
	assert.Equal(t, operation, err.Operation)
	assert.Equal(t, context, err.Context)
	assert.Equal(t, timeout, err.Timeout)
	assert.Equal(t, elapsed, err.Elapsed)
	assert.Equal(t, configVar, err.ConfigVar)

	// Test suggested timeout calculation (1.5x current timeout or 2x elapsed, whichever is larger)
	expected1_5xTimeout := timeout * 3 / 2          // 7.5 minutes
	expected2xElapsed := elapsed * 2                // 6 minutes
	expectedSuggestedTimeout := expected1_5xTimeout // 1.5x timeout (7.5m) > 2x elapsed (6m)
	if expected2xElapsed > expected1_5xTimeout {
		expectedSuggestedTimeout = expected2xElapsed
	}
	assert.Equal(t, expectedSuggestedTimeout, err.SuggestedTimeout)
}

func TestNewTimeoutError_SuggestedTimeoutCalculation(t *testing.T) {
	tests := []struct {
		name              string
		timeout           time.Duration
		elapsed           time.Duration
		expectedSuggested time.Duration
	}{
		{
			name:              "elapsed is 0, use 1.5x timeout",
			timeout:           60 * time.Second,
			elapsed:           0,
			expectedSuggested: 90 * time.Second, // 1.5 * 60
		},
		{
			name:              "2x elapsed is larger than 1.5x timeout",
			timeout:           60 * time.Second,
			elapsed:           50 * time.Second,
			expectedSuggested: 100 * time.Second, // 2 * 50 > 1.5 * 60 (90)
		},
		{
			name:              "1.5x timeout is larger than 2x elapsed",
			timeout:           120 * time.Second,
			elapsed:           30 * time.Second,
			expectedSuggested: 180 * time.Second, // 1.5 * 120 (180) > 2 * 30 (60)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTimeoutError("test", "test", tt.timeout, tt.elapsed, "TEST_VAR")
			assert.Equal(t, tt.expectedSuggested, err.SuggestedTimeout)
		})
	}
}

func TestNewToolInstallTimeoutError(t *testing.T) {
	toolName := "golangci-lint"
	timeout := 5 * time.Minute
	elapsed := 3 * time.Minute

	err := NewToolInstallTimeoutError(toolName, timeout, elapsed)

	assert.Equal(t, "Tool installation", err.Operation)
	assert.Equal(t, toolName, err.Context)
	assert.Equal(t, timeout, err.Timeout)
	assert.Equal(t, elapsed, err.Elapsed)
	assert.Equal(t, "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT", err.ConfigVar)
}

func TestNewCheckTimeoutError(t *testing.T) {
	tests := []struct {
		name              string
		checkName         string
		expectedConfigVar string
	}{
		{
			name:              "fmt check",
			checkName:         "fmt",
			expectedConfigVar: "GO_PRE_COMMIT_FMT_TIMEOUT",
		},
		{
			name:              "fumpt check",
			checkName:         "fumpt",
			expectedConfigVar: "GO_PRE_COMMIT_FUMPT_TIMEOUT",
		},
		{
			name:              "lint check",
			checkName:         "lint",
			expectedConfigVar: "GO_PRE_COMMIT_LINT_TIMEOUT",
		},
		{
			name:              "mod-tidy check",
			checkName:         "mod-tidy",
			expectedConfigVar: "GO_PRE_COMMIT_MOD_TIDY_TIMEOUT",
		},
		{
			name:              "whitespace check",
			checkName:         "whitespace",
			expectedConfigVar: "GO_PRE_COMMIT_WHITESPACE_TIMEOUT",
		},
		{
			name:              "eof check",
			checkName:         "eof",
			expectedConfigVar: "GO_PRE_COMMIT_EOF_TIMEOUT",
		},
		{
			name:              "ai_detection check",
			checkName:         "ai_detection",
			expectedConfigVar: "GO_PRE_COMMIT_AI_DETECTION_TIMEOUT",
		},
		{
			name:              "unknown check defaults to global",
			checkName:         "unknown",
			expectedConfigVar: "GO_PRE_COMMIT_TIMEOUT_SECONDS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := 60 * time.Second
			elapsed := 30 * time.Second

			err := NewCheckTimeoutError(tt.checkName, timeout, elapsed)

			assert.Equal(t, "Check execution", err.Operation)
			assert.Equal(t, tt.checkName, err.Context)
			assert.Equal(t, timeout, err.Timeout)
			assert.Equal(t, elapsed, err.Elapsed)
			assert.Equal(t, tt.expectedConfigVar, err.ConfigVar)
		})
	}
}

func TestTimeoutError_Integration(t *testing.T) {
	// Test that TimeoutError can be used with standard Go error handling patterns
	toolTimeoutErr := NewToolInstallTimeoutError("gofumpt", 2*time.Minute, 90*time.Second)

	// Test error wrapping
	wrappedErr := fmt.Errorf("installation failed: %w", toolTimeoutErr)

	// Test error unwrapping
	var timeoutErr *TimeoutError
	require.ErrorAs(t, wrappedErr, &timeoutErr)
	assert.Equal(t, "gofumpt", timeoutErr.Context)
	assert.Equal(t, "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT", timeoutErr.ConfigVar)

	// Test Is behavior
	require.ErrorIs(t, wrappedErr, ErrTimeout)
	require.ErrorIs(t, toolTimeoutErr, ErrTimeout)

	// Test that the error message is helpful
	errorMsg := toolTimeoutErr.Error()
	assert.Contains(t, errorMsg, "gofumpt")
	assert.Contains(t, errorMsg, "2m0s")
	assert.Contains(t, errorMsg, "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT")
	assert.Contains(t, errorMsg, "suggested: 3m0s")
}

// TestTimeoutError_ContextTimeout tests timeout error creation from context.DeadlineExceeded
func TestTimeoutError_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Wait for timeout
	<-ctx.Done()

	// Verify context was canceled due to deadline
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())

	// Create timeout error as would happen in real usage
	timeoutErr := NewToolInstallTimeoutError("test-tool", 100*time.Millisecond, 100*time.Millisecond)

	assert.Equal(t, "Tool installation", timeoutErr.Operation)
	assert.Equal(t, "test-tool", timeoutErr.Context)
	assert.Equal(t, 100*time.Millisecond, timeoutErr.Timeout)
}
