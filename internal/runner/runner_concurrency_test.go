package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/config"
)

// mockCheck is a configurable Check used for concurrency and panic tests.
type mockCheck struct {
	name string
	run  func(ctx context.Context, files []string) error
}

func (m *mockCheck) Name() string                        { return m.name }
func (m *mockCheck) Description() string                 { return m.name }
func (m *mockCheck) Metadata() any                       { return nil }
func (m *mockCheck) FilterFiles(files []string) []string { return files }
func (m *mockCheck) Run(ctx context.Context, files []string) error {
	if m.run == nil {
		return nil
	}
	return m.run(ctx, files)
}

// errMockCheckFailed is a static sentinel returned by failing mock checks.
var errMockCheckFailed = errors.New("mock check failed")

// knownCheckNames returns every check name the runner recognizes as enableable.
func knownCheckNames() []string {
	return []string{
		checkNameFumpt, checkNameGitleaks,
		checkNameLint, checkNameModTidy, checkNameEOF, checkNameWhitespace,
	}
}

func enableAllChecks(cfg *config.Config) {
	cfg.Checks.Fumpt = true
	cfg.Checks.Gitleaks = true
	cfg.Checks.Lint = true
	cfg.Checks.ModTidy = true
	cfg.Checks.EOF = true
	cfg.Checks.Whitespace = true
}

func tempFile(t *testing.T) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "f.txt")
	require.NoError(t, os.WriteFile(f, []byte("x\n"), 0o600))
	return f
}

func TestRunCheck_RecoversFromPanic(t *testing.T) {
	cfg := &config.Config{Enabled: true, Timeout: 60}
	cfg.Checks.Lint = true

	r := New(cfg, t.TempDir())
	r.registry.Register(&mockCheck{name: checkNameLint, run: func(context.Context, []string) error {
		panic("boom")
	}})

	results, err := r.Run(context.Background(), Options{Files: []string{tempFile(t)}})
	require.NoError(t, err, "a panicking check must not crash the run")
	require.Len(t, results.CheckResults, 1)
	assert.False(t, results.CheckResults[0].Success)
	assert.Contains(t, results.CheckResults[0].Error, "check panicked")
	assert.Equal(t, 1, results.Failed)
}

func TestRunParallel_ManyChecksUnderRace(t *testing.T) {
	cfg := &config.Config{Enabled: true, Timeout: 60}
	enableAllChecks(cfg)

	names := knownCheckNames()
	r := New(cfg, t.TempDir())
	var counter int64
	for _, name := range names {
		r.registry.Register(&mockCheck{name: name, run: func(context.Context, []string) error {
			atomic.AddInt64(&counter, 1)
			time.Sleep(time.Millisecond) // widen the race window
			return nil
		}})
	}

	results, err := r.Run(context.Background(), Options{Files: []string{tempFile(t)}, Parallel: 8})
	require.NoError(t, err)
	assert.Len(t, results.CheckResults, len(names))
	assert.Equal(t, len(names), results.Passed)
	assert.Equal(t, int64(len(names)), atomic.LoadInt64(&counter))
}

func TestRunParallel_LargeWorkerCount(t *testing.T) {
	cfg := &config.Config{Enabled: true, Timeout: 60}
	enableAllChecks(cfg)

	names := knownCheckNames()
	r := New(cfg, t.TempDir())
	for _, name := range names {
		r.registry.Register(&mockCheck{name: name})
	}

	// A worker count far larger than the number of checks must not deadlock or
	// over-spawn; every check still runs exactly once.
	results, err := r.Run(context.Background(), Options{Files: []string{tempFile(t)}, Parallel: 1000})
	require.NoError(t, err)
	assert.Equal(t, len(names), results.Passed)
}

func TestRunParallel_MixedPanicAndSuccess(t *testing.T) {
	cfg := &config.Config{Enabled: true, Timeout: 60}
	cfg.Checks.Lint = true
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	r := New(cfg, t.TempDir())
	r.registry.Register(&mockCheck{name: checkNameLint, run: func(context.Context, []string) error {
		panic("lint exploded")
	}})
	r.registry.Register(&mockCheck{name: checkNameWhitespace})
	r.registry.Register(&mockCheck{name: checkNameEOF, run: func(context.Context, []string) error {
		return errMockCheckFailed
	}})

	results, err := r.Run(context.Background(), Options{Files: []string{tempFile(t)}, Parallel: 4})
	require.NoError(t, err)
	assert.Len(t, results.CheckResults, 3)
	assert.Equal(t, 1, results.Passed) // whitespace
	assert.Equal(t, 2, results.Failed) // lint (panic) + eof
}
