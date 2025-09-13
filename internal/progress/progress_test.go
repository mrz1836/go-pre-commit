// Package progress provides comprehensive timeout progress tracking tests
package progress

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	opts := Options{
		Operation: "Test operation",
		Context:   "test-context",
		Timeout:   5 * time.Minute,
		Interval:  10 * time.Second,
	}

	tracker := New(opts)

	assert.Equal(t, "Test operation", tracker.operation)
	assert.Equal(t, "test-context", tracker.context)
	assert.Equal(t, 5*time.Minute, tracker.timeout)
	assert.Equal(t, 10*time.Second, tracker.interval)
	assert.NotNil(t, tracker.progressFunc)
	assert.False(t, tracker.suppressOutput)
}

func TestNew_DefaultInterval(t *testing.T) {
	opts := Options{
		Operation: "Test operation",
		Context:   "test-context",
		Timeout:   5 * time.Minute,
		// No interval specified
	}

	tracker := New(opts)

	assert.Equal(t, 10*time.Second, tracker.interval) // Should default to 10s
}

func TestNew_DefaultProgressFunc(t *testing.T) {
	opts := Options{
		Operation: "Test operation",
		Context:   "test-context",
		Timeout:   5 * time.Minute,
		// No progress func specified
	}

	tracker := New(opts)

	// Should use default progress function
	assert.NotNil(t, tracker.progressFunc)

	// Test the default function
	message := tracker.progressFunc(30*time.Second, 60*time.Second)
	assert.Contains(t, message, "running for 30s")
	assert.Contains(t, message, "60s remaining")
}

func TestNew_CustomProgressFunc(t *testing.T) {
	customFunc := func(_, _ time.Duration) string {
		return "custom message"
	}

	opts := Options{
		Operation:    "Test operation",
		Context:      "test-context",
		Timeout:      5 * time.Minute,
		ProgressFunc: customFunc,
	}

	tracker := New(opts)

	message := tracker.progressFunc(30*time.Second, 60*time.Second)
	assert.Equal(t, "custom message", message)
}

func TestTracker_StartStop(t *testing.T) {
	opts := Options{
		Operation:      "Test operation",
		Context:        "test-context",
		Timeout:        5 * time.Minute,
		Interval:       50 * time.Millisecond, // Fast interval for testing
		SuppressOutput: true,                  // Don't print during tests
	}

	tracker := New(opts)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start tracking
	tracker.Start(ctx)

	// Wait a bit to ensure it started
	time.Sleep(10 * time.Millisecond)

	// Check that start time is set
	assert.False(t, tracker.startTime.IsZero())

	// Stop tracking
	tracker.Stop()

	// Verify it's canceled
	assert.True(t, tracker.canceled)
}

func TestTracker_GetElapsed(t *testing.T) {
	opts := Options{
		Operation:      "Test operation",
		Context:        "test-context",
		Timeout:        5 * time.Minute,
		SuppressOutput: true,
	}

	tracker := New(opts)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracker.Start(ctx)

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	elapsed := tracker.GetElapsed()
	assert.GreaterOrEqual(t, elapsed, 40*time.Millisecond) // Account for some variability
	assert.Less(t, elapsed, 200*time.Millisecond)

	tracker.Stop()
}

func TestTracker_ContextCancellation(t *testing.T) {
	opts := Options{
		Operation:      "Test operation",
		Context:        "test-context",
		Timeout:        5 * time.Minute,
		Interval:       10 * time.Millisecond,
		SuppressOutput: true,
	}

	tracker := New(opts)
	ctx, cancel := context.WithCancel(context.Background())

	tracker.Start(ctx)

	// Cancel context
	cancel()

	// Wait a bit to ensure goroutine exits
	time.Sleep(50 * time.Millisecond)

	// Tracker should still be accessible
	elapsed := tracker.GetElapsed()
	assert.Positive(t, elapsed)
}

func TestDefaultProgressMessage(t *testing.T) {
	tests := []struct {
		name      string
		elapsed   time.Duration
		remaining time.Duration
		contains  []string
	}{
		{
			name:      "short durations",
			elapsed:   30 * time.Second,
			remaining: 45 * time.Second,
			contains:  []string{"running for 30s", "45s remaining"},
		},
		{
			name:      "long remaining time",
			elapsed:   45 * time.Second,
			remaining: 2*time.Minute + 30*time.Second,
			contains:  []string{"running for 45s", "2m 30s remaining"},
		},
		{
			name:      "very long remaining time",
			elapsed:   30 * time.Second,
			remaining: 5*time.Minute + 0*time.Second,
			contains:  []string{"running for 30s", "5m 0s remaining"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := defaultProgressMessage(tt.elapsed, tt.remaining)
			for _, expected := range tt.contains {
				assert.Contains(t, message, expected)
			}
		})
	}
}

func TestInstallProgressFunc(t *testing.T) {
	toolName := "golangci-lint"
	progressFunc := InstallProgressFunc(toolName)

	tests := []struct {
		name      string
		elapsed   time.Duration
		remaining time.Duration
		contains  []string
	}{
		{
			name:      "short elapsed time",
			elapsed:   15 * time.Second,
			remaining: 285 * time.Second,
			contains:  []string{"installing golangci-lint", "15s elapsed", "300s timeout"},
		},
		{
			name:      "long elapsed time",
			elapsed:   45 * time.Second,
			remaining: 255 * time.Second,
			contains:  []string{"installing golangci-lint", "taking longer than expected", "45s elapsed", "255s remaining"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := progressFunc(tt.elapsed, tt.remaining)
			for _, expected := range tt.contains {
				assert.Contains(t, message, expected)
			}
		})
	}
}

func TestWithContext(t *testing.T) {
	opts := Options{
		Operation:      "Test operation",
		Context:        "test-context",
		Timeout:        100 * time.Millisecond,
		Interval:       10 * time.Millisecond,
		SuppressOutput: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	tracker, childCtx := WithContext(ctx, opts)

	// Wait for context to timeout
	<-childCtx.Done()

	// Tracker should be stopped automatically
	time.Sleep(20 * time.Millisecond) // Give time for cleanup

	// Verify tracker was started and is now stopped
	elapsed := tracker.GetElapsed()
	assert.Positive(t, elapsed)
	assert.True(t, tracker.canceled)
}

func TestTracker_StopIdempotent(t *testing.T) {
	opts := Options{
		Operation:      "Test operation",
		Context:        "test-context",
		Timeout:        5 * time.Minute,
		SuppressOutput: true,
	}

	tracker := New(opts)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracker.Start(ctx)

	// Stop multiple times - should not panic or cause issues
	tracker.Stop()
	tracker.Stop()
	tracker.Stop()

	assert.True(t, tracker.canceled)
}

func TestTracker_ProgressInterval(t *testing.T) {
	// Test that progress updates occur at the expected interval
	var updateCount int
	var mu sync.Mutex

	opts := Options{
		Operation:      "Test operation",
		Context:        "test-context",
		Timeout:        1 * time.Second,
		Interval:       25 * time.Millisecond, // Shorter interval for testing
		SuppressOutput: true,
		ProgressFunc: func(_, _ time.Duration) string {
			mu.Lock()
			updateCount++
			mu.Unlock()
			return "test update"
		},
	}

	tracker := New(opts)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracker.Start(ctx)

	// Wait for several intervals
	time.Sleep(125 * time.Millisecond) // Should allow 4-5 updates

	tracker.Stop()

	// Get final count
	mu.Lock()
	finalCount := updateCount
	mu.Unlock()

	// Should have had at least 2 updates (allowing for timing variability)
	assert.GreaterOrEqual(t, finalCount, 2, "Expected at least 2 updates, got %d", finalCount)
}

// BenchmarkTracker_Performance tests the performance impact of the progress tracker
func BenchmarkTracker_Performance(b *testing.B) {
	opts := Options{
		Operation:      "Benchmark operation",
		Context:        "bench-context",
		Timeout:        5 * time.Minute,
		Interval:       1 * time.Second,
		SuppressOutput: true,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tracker := New(opts)
		ctx, cancel := context.WithCancel(context.Background())

		tracker.Start(ctx)
		time.Sleep(1 * time.Millisecond) // Simulate some work
		tracker.Stop()

		cancel()
	}
}

func TestTracker_ZeroTimeout(t *testing.T) {
	// Edge case: zero timeout
	opts := Options{
		Operation:      "Test operation",
		Context:        "test-context",
		Timeout:        0, // Zero timeout
		Interval:       10 * time.Millisecond,
		SuppressOutput: true,
	}

	tracker := New(opts)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Should not panic
	require.NotPanics(t, func() {
		tracker.Start(ctx)
		time.Sleep(20 * time.Millisecond)
		tracker.Stop()
	})
}
