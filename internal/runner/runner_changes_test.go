package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/config"
)

// mockFixer is a check that rewrites files and reports which ones it changed
type mockFixer struct {
	mockCheck

	fix    func(ctx context.Context, files []string) ([]string, error)
	filter func(files []string) []string
}

func (m *mockFixer) FilterFiles(files []string) []string {
	if m.filter != nil {
		return m.filter(files)
	}
	return files
}

func (m *mockFixer) RunWithChanges(ctx context.Context, files []string) ([]string, error) {
	return m.fix(ctx, files)
}

// resultsByName indexes check results by check name
func resultsByName(results *Results) map[string]CheckResult {
	byName := make(map[string]CheckResult, len(results.CheckResults))
	for _, result := range results.CheckResults {
		byName[result.Name] = result
	}
	return byName
}

// TestRunRecordsChangedFiles tests that the runner records the files a fixing
// check changed (an empty list when it changed nothing, including when it had
// no files to process) and leaves ChangedFiles nil for checks that cannot tell
func TestRunRecordsChangedFiles(t *testing.T) {
	cfg := &config.Config{Enabled: true, Timeout: 60}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true
	cfg.Checks.Fumpt = true
	cfg.Checks.Lint = true

	file := tempFile(t)
	r := New(cfg, t.TempDir())
	r.registry.Register(&mockFixer{
		mockCheck: mockCheck{name: checkNameWhitespace},
		fix: func(_ context.Context, files []string) ([]string, error) {
			return files, errMockCheckFailed
		},
	})
	r.registry.Register(&mockFixer{
		mockCheck: mockCheck{name: checkNameEOF},
		fix:       func(context.Context, []string) ([]string, error) { return nil, nil },
	})
	r.registry.Register(&mockFixer{
		mockCheck: mockCheck{name: checkNameFumpt},
		fix: func(context.Context, []string) ([]string, error) {
			panic("Run must not be called without files")
		},
		filter: func([]string) []string { return nil },
	})
	r.registry.Register(&mockCheck{name: checkNameLint})

	for _, failFast := range []bool{false, true} {
		results, err := r.Run(t.Context(), Options{Files: []string{file}, FailFast: failFast, OnlyChecks: []string{checkNameWhitespace}})
		require.NoError(t, err)
		require.Len(t, results.CheckResults, 1)
		assert.False(t, results.CheckResults[0].Success)
		assert.Equal(t, []string{file}, results.CheckResults[0].ChangedFiles, "fail-fast=%t", failFast)
	}

	results, err := r.Run(t.Context(), Options{Files: []string{file}, SkipChecks: []string{checkNameWhitespace}})
	require.NoError(t, err)
	byName := resultsByName(results)
	require.Len(t, byName, 3)

	assert.NotNil(t, byName[checkNameEOF].ChangedFiles, "a fixer that changed nothing reports an empty list")
	assert.Empty(t, byName[checkNameEOF].ChangedFiles)

	assert.NotNil(t, byName[checkNameFumpt].ChangedFiles, "a fixer with no files to process changed nothing")
	assert.Empty(t, byName[checkNameFumpt].ChangedFiles)

	assert.Nil(t, byName[checkNameLint].ChangedFiles, "a check that does not report changes has none recorded")
}

// TestRunRecordsChangedFilesWhenAFixerPanics tests that a panicking fixer is a
// failed result that changed nothing it could report
func TestRunRecordsChangedFilesWhenAFixerPanics(t *testing.T) {
	cfg := &config.Config{Enabled: true, Timeout: 60}
	cfg.Checks.EOF = true

	r := New(cfg, t.TempDir())
	r.registry.Register(&mockFixer{
		mockCheck: mockCheck{name: checkNameEOF},
		fix:       func(context.Context, []string) ([]string, error) { panic("boom") },
	})

	results, err := r.Run(t.Context(), Options{Files: []string{tempFile(t)}})
	require.NoError(t, err)
	require.Len(t, results.CheckResults, 1)
	assert.False(t, results.CheckResults[0].Success)
	assert.Contains(t, results.CheckResults[0].Error, "check panicked")
	assert.Equal(t, []string{}, results.CheckResults[0].ChangedFiles)
}
