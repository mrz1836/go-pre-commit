// Package progress provides timeout-aware progress indicators for long-running operations
package progress

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Tracker tracks progress of long-running operations with timeout awareness
type Tracker struct {
	operation      string
	context        string
	timeout        time.Duration
	interval       time.Duration
	startTime      time.Time
	lastUpdate     time.Time
	done           chan bool
	mu             sync.Mutex
	canceled       bool
	progressFunc   func(elapsed, remaining time.Duration) string
	suppressOutput bool
}

// Options configures a progress tracker
type Options struct {
	Operation      string                                        // e.g., "Tool installation", "Check execution"
	Context        string                                        // e.g., tool name, check name
	Timeout        time.Duration                                 // Total timeout
	Interval       time.Duration                                 // Update interval (default: 10s)
	ProgressFunc   func(elapsed, remaining time.Duration) string // Custom progress message function
	SuppressOutput bool                                          // Don't output progress messages (for testing)
}

// New creates a new progress tracker
func New(opts Options) *Tracker {
	if opts.Interval == 0 {
		opts.Interval = 10 * time.Second
	}

	if opts.ProgressFunc == nil {
		opts.ProgressFunc = defaultProgressMessage
	}

	return &Tracker{
		operation:      opts.Operation,
		context:        opts.Context,
		timeout:        opts.Timeout,
		interval:       opts.Interval,
		done:           make(chan bool, 1),
		progressFunc:   opts.ProgressFunc,
		suppressOutput: opts.SuppressOutput,
	}
}

// Start begins tracking progress in a separate goroutine
func (t *Tracker) Start(ctx context.Context) {
	t.mu.Lock()
	t.startTime = time.Now()
	t.lastUpdate = t.startTime
	t.mu.Unlock()

	go t.trackProgress(ctx)
}

// Stop stops the progress tracker
func (t *Tracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.canceled {
		t.canceled = true
		select {
		case t.done <- true:
		default:
		}
	}
}

// trackProgress runs the progress tracking loop
func (t *Tracker) trackProgress(ctx context.Context) {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.done:
			return
		case <-ticker.C:
			t.updateProgress()
		}
	}
}

// updateProgress prints a progress update
func (t *Tracker) updateProgress() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.canceled {
		return
	}

	now := time.Now()
	elapsed := now.Sub(t.startTime)
	remaining := t.timeout - elapsed

	if remaining <= 0 {
		return // Timeout should be handled by the context
	}

	message := t.progressFunc(elapsed, remaining)

	if t.suppressOutput {
		return
	}

	contextStr := ""
	if t.context != "" {
		contextStr = fmt.Sprintf(" (%s)", t.context)
	}

	_, _ = fmt.Fprintf(os.Stdout, "%s %s%s - %s\n",
		color.CyanString("⏳"),
		t.operation,
		contextStr,
		message)

	t.lastUpdate = now
}

// GetElapsed returns the time elapsed since start
func (t *Tracker) GetElapsed() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return time.Since(t.startTime)
}

// defaultProgressMessage creates a default progress message
func defaultProgressMessage(elapsed, remaining time.Duration) string {
	elapsedSeconds := int(elapsed.Seconds())
	remainingSeconds := int(remaining.Seconds())

	if remainingSeconds > 60 {
		return fmt.Sprintf("running for %ds, %dm %ds remaining",
			elapsedSeconds,
			remainingSeconds/60,
			remainingSeconds%60)
	}

	return fmt.Sprintf("running for %ds, %ds remaining",
		elapsedSeconds,
		remainingSeconds)
}

// WithContext starts a progress tracker that automatically stops when the context is done
func WithContext(ctx context.Context, opts Options) (*Tracker, context.Context) {
	tracker := New(opts)

	// Create a context that will be canceled when the original context is done
	childCtx, cancel := context.WithCancel(ctx)

	// Start tracking
	tracker.Start(childCtx)

	// Set up automatic cleanup when context is done
	go func() {
		<-childCtx.Done()
		tracker.Stop()
		cancel()
	}()

	return tracker, childCtx
}

// InstallProgressFunc creates a progress message function for tool installation
func InstallProgressFunc(toolName string) func(elapsed, remaining time.Duration) string {
	return func(elapsed, remaining time.Duration) string {
		elapsedSeconds := int(elapsed.Seconds())
		remainingSeconds := int(remaining.Seconds())

		if elapsedSeconds < 30 {
			return fmt.Sprintf("installing %s... (%ds elapsed, %ds timeout)", toolName, elapsedSeconds, int(elapsed.Seconds()+remaining.Seconds()))
		}

		return fmt.Sprintf("installing %s (this is taking longer than expected)... %ds elapsed, %ds remaining",
			toolName, elapsedSeconds, remainingSeconds)
	}
}
