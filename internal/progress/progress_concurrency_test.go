package progress

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// concurrencyOp is the operation label reused across the tracker concurrency tests.
const concurrencyOp = "test"

func TestTracker_ConcurrentStartStop(t *testing.T) {
	tracker := New(Options{
		Operation:      concurrencyOp,
		Timeout:        time.Minute,
		Interval:       time.Millisecond,
		SuppressOutput: true,
	})

	var wg sync.WaitGroup
	ctx := context.Background()
	for range 20 {
		wg.Add(2)
		go func() { defer wg.Done(); tracker.Start(ctx) }()
		go func() { defer wg.Done(); tracker.Stop() }()
	}
	wg.Wait()
	// Must not panic, deadlock, or race; a final Stop is safe.
	require.NotPanics(t, tracker.Stop)
}

func TestTracker_DoubleStop(t *testing.T) {
	tracker := New(Options{Operation: concurrencyOp, Timeout: time.Minute, SuppressOutput: true})
	tracker.Start(context.Background())
	require.NotPanics(t, func() {
		tracker.Stop()
		tracker.Stop()
		tracker.Stop()
	})
}

func TestTracker_StopDuringTracking(t *testing.T) {
	tracker := New(Options{
		Operation:      concurrencyOp,
		Timeout:        time.Minute,
		Interval:       time.Millisecond, // ticks fire while we stop
		SuppressOutput: true,
	})
	tracker.Start(context.Background())
	time.Sleep(5 * time.Millisecond) // let a few ticks fire
	require.NotPanics(t, tracker.Stop)
}

func TestTracker_ContextCancellationStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tracker := New(Options{
		Operation:      concurrencyOp,
		Timeout:        time.Minute,
		Interval:       time.Millisecond,
		SuppressOutput: true,
	})
	tracker.Start(ctx)
	cancel() // the tracking goroutine should return on ctx.Done
	time.Sleep(5 * time.Millisecond)
	require.NotPanics(t, tracker.Stop)
}

func TestTracker_ZeroIntervalDefaults(t *testing.T) {
	tracker := New(Options{Operation: concurrencyOp, Timeout: time.Minute, Interval: 0})
	// New must guard interval==0 so trackProgress does not spin in a tight loop.
	assert.Equal(t, 10*time.Second, tracker.interval)
}

func TestTracker_ConcurrentGetElapsed(t *testing.T) {
	tracker := New(Options{Operation: concurrencyOp, Timeout: time.Minute, SuppressOutput: true})
	tracker.Start(context.Background())
	defer tracker.Stop()

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() { defer wg.Done(); _ = tracker.GetElapsed() }()
	}
	wg.Wait()
}
