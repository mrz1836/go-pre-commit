package builtin

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

// changeReporter is the method the runner uses to learn which files a fixing
// check rewrote (checks.ChangeReporter, restated here to avoid an import cycle)
type changeReporter interface {
	RunWithChanges(ctx context.Context, files []string) ([]string, error)
}

// TestFixingChecksReportChangedFiles tests that the whitespace and EOF checks
// report exactly the files they rewrote, in the form they were given, and
// fail or pass just as Run does
func TestFixingChecksReportChangedFiles(t *testing.T) {
	tests := []struct {
		name        string
		check       changeReporter
		files       map[string]string
		wantChanged []string
		wantErr     error
	}{
		{
			name:  "whitespace rewrites only the files with trailing whitespace",
			check: NewWhitespaceCheck(),
			files: map[string]string{
				"clean.go":  "package main\n",
				"dirty.md":  "hello   \n",
				"dirty2.go": "package main \t\n",
			},
			wantChanged: []string{"dirty.md", "dirty2.go"},
			wantErr:     prerrors.ErrWhitespaceIssues,
		},
		{
			name:        "whitespace reports an empty list when nothing changed",
			check:       NewWhitespaceCheck(),
			files:       map[string]string{"clean.go": "package main\n"},
			wantChanged: []string{},
		},
		{
			name:  "eof rewrites only the files missing a final newline",
			check: NewEOFCheck(),
			files: map[string]string{
				"clean.md": "done\n",
				"short.md": "no newline",
			},
			wantChanged: []string{"short.md"},
			wantErr:     prerrors.ErrEOFIssues,
		},
		{
			name:        "eof reports an empty list when nothing changed",
			check:       NewEOFCheck(),
			files:       map[string]string{"clean.md": "done\n"},
			wantChanged: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			paths := make([]string, 0, len(tt.files))
			for _, name := range slices.Sorted(maps.Keys(tt.files)) {
				path := filepath.Join(dir, name)
				require.NoError(t, os.WriteFile(path, []byte(tt.files[name]), 0o600))
				paths = append(paths, path)
			}

			changed, err := tt.check.RunWithChanges(t.Context(), paths)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.NotNil(t, changed, "a fixing check reports an empty list, not nil, when it changed nothing")
			wantPaths := make([]string, 0, len(tt.wantChanged))
			for _, name := range tt.wantChanged {
				wantPaths = append(wantPaths, filepath.Join(dir, name))
			}
			assert.Equal(t, wantPaths, changed)
		})
	}
}

// TestFixingChecksReportChangesBeforeAnError tests that a file the check could
// not read fails the check, while the files it did rewrite are still reported
func TestFixingChecksReportChangesBeforeAnError(t *testing.T) {
	for _, check := range []changeReporter{NewWhitespaceCheck(), NewEOFCheck()} {
		dir := t.TempDir()
		fixed := filepath.Join(dir, "fixed.md")
		require.NoError(t, os.WriteFile(fixed, []byte("text  "), 0o600))
		missing := filepath.Join(dir, "missing.md")

		changed, err := check.RunWithChanges(t.Context(), []string{fixed, missing})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing.md")
		assert.Equal(t, []string{fixed}, changed)
	}
}
