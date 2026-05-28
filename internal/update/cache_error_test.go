package update

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteCache_MkdirAllError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Place a regular file where the cache directory should be so MkdirAll fails.
	require.NoError(t, os.WriteFile(filepath.Join(home, cacheDir), []byte("x"), 0o600))

	err := WriteCache(&CacheEntry{CurrentVersion: "1.0.0", LatestVersion: "1.1.0"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create cache dir")
}

func TestClearCache_RemoveError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission errors are not enforced for root")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create a valid cache file, then make its directory read-only so removal fails.
	require.NoError(t, WriteCache(&CacheEntry{CurrentVersion: "1.0.0", LatestVersion: "1.1.0"}))
	dir := filepath.Join(home, cacheDir)
	require.NoError(t, os.Chmod(dir, 0o500))       //nolint:gosec // intentionally read-only to trigger a remove failure
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:gosec // restore for cleanup

	err := ClearCache()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove cache file")
}

func TestReadCache_CorruptJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, cacheDir)
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, cacheFile), []byte("{not valid json"), 0o600))

	_, err := ReadCache()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse cache file")
}

func TestClearCache_NotExistIsNotError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// No cache file written; ClearCache should treat a missing file as success.
	require.NoError(t, ClearCache())
}
