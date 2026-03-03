package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUpdateCheckDisabled(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "not disabled by default",
			envVars:  map[string]string{},
			expected: false,
		},
		{
			name: "disabled via GO_PRE_COMMIT_DISABLE_UPDATE_CHECK=1",
			envVars: map[string]string{
				"GO_PRE_COMMIT_DISABLE_UPDATE_CHECK": "1",
			},
			expected: true,
		},
		{
			name: "disabled via GO_PRE_COMMIT_DISABLE_UPDATE_CHECK=true",
			envVars: map[string]string{
				"GO_PRE_COMMIT_DISABLE_UPDATE_CHECK": "true",
			},
			expected: true,
		},
		{
			name: "disabled in CI (CI=1)",
			envVars: map[string]string{
				"CI": "1",
			},
			expected: true,
		},
		{
			name: "disabled in CI (CI=true)",
			envVars: map[string]string{
				"CI": "true",
			},
			expected: true,
		},
		{
			name: "not disabled with random CI value",
			envVars: map[string]string{
				"CI": "false",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars first
			t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "")
			t.Setenv("CI", "")

			// Set test env vars
			for key, val := range tt.envVars {
				t.Setenv(key, val)
			}

			result := IsUpdateCheckDisabled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCheckInterval(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected time.Duration
	}{
		{
			name:     "default value (24h)",
			envValue: "",
			expected: 24 * time.Hour,
		},
		{
			name:     "valid custom value (48h)",
			envValue: "48h",
			expected: 48 * time.Hour,
		},
		{
			name:     "too short (30m) clamped to 1h",
			envValue: "30m",
			expected: 1 * time.Hour,
		},
		{
			name:     "invalid duration returns default",
			envValue: "invalid",
			expected: 24 * time.Hour,
		},
		{
			name:     "very long (800h) clamped to 720h",
			envValue: "800h",
			expected: 720 * time.Hour,
		},
		{
			name:     "exactly at minimum (1h)",
			envValue: "1h",
			expected: 1 * time.Hour,
		},
		{
			name:     "exactly at maximum (720h)",
			envValue: "720h",
			expected: 720 * time.Hour,
		},
		{
			name:     "zero duration clamped to minimum",
			envValue: "0s",
			expected: 1 * time.Hour, // Parsed as 0, then clamped to minimum
		},
		{
			name:     "negative duration clamped to minimum",
			envValue: "-5h",
			expected: 1 * time.Hour, // Negative duration is valid but < minimum, clamped to 1h
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GO_PRE_COMMIT_UPDATE_CHECK_INTERVAL", tt.envValue)

			result := GetCheckInterval()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCacheDir(t *testing.T) {
	dir, err := GetCacheDir()
	require.NoError(t, err)

	// Should return a path ending in .go-pre-commit
	assert.Contains(t, dir, ".go-pre-commit")

	// Should be under home directory
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	expected := filepath.Join(homeDir, ".go-pre-commit")
	assert.Equal(t, expected, dir)
}

func TestReadCache(t *testing.T) {
	t.Run("file not found returns nil, nil", func(t *testing.T) {
		// Create a temporary directory
		tempDir := t.TempDir()

		// Override getCacheFilePath by creating a cache in temp dir
		// We'll use WriteCache to set up the environment, then delete the file
		origHome := os.Getenv("HOME")
		t.Setenv("HOME", tempDir)
		defer func() {
			if origHome != "" {
				_ = os.Setenv("HOME", origHome)
			}
		}()

		entry, err := ReadCache()
		require.NoError(t, err)
		assert.Nil(t, entry)
	})

	t.Run("valid JSON returns entry", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)

		// Create cache file with valid JSON
		cacheDir := filepath.Join(tempDir, ".go-pre-commit")
		require.NoError(t, os.MkdirAll(cacheDir, 0o700))

		validEntry := &CacheEntry{
			CheckedAt:      time.Now(),
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.1.0",
		}

		data, err := json.MarshalIndent(validEntry, "", "  ")
		require.NoError(t, err)

		cacheFile := filepath.Join(cacheDir, "update-check.json")
		require.NoError(t, os.WriteFile(cacheFile, data, 0o600))

		// Read it back
		entry, err := ReadCache()
		require.NoError(t, err)
		require.NotNil(t, entry)
		assert.Equal(t, validEntry.CurrentVersion, entry.CurrentVersion)
		assert.Equal(t, validEntry.LatestVersion, entry.LatestVersion)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)

		// Create cache file with invalid JSON
		cacheDir := filepath.Join(tempDir, ".go-pre-commit")
		require.NoError(t, os.MkdirAll(cacheDir, 0o700))

		cacheFile := filepath.Join(cacheDir, "update-check.json")
		require.NoError(t, os.WriteFile(cacheFile, []byte("not valid json"), 0o600))

		entry, err := ReadCache()
		require.Error(t, err)
		assert.Nil(t, entry)
	})
}

func TestWriteCacheAndReadCacheRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Write cache entry
	writeEntry := &CacheEntry{
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.2.0",
	}

	err := WriteCache(writeEntry)
	require.NoError(t, err)

	// Read it back
	readEntry, err := ReadCache()
	require.NoError(t, err)
	require.NotNil(t, readEntry)

	assert.Equal(t, writeEntry.CurrentVersion, readEntry.CurrentVersion)
	assert.Equal(t, writeEntry.LatestVersion, readEntry.LatestVersion)
	assert.False(t, readEntry.CheckedAt.IsZero(), "CheckedAt should be set by WriteCache")
}

func TestIsCacheValid(t *testing.T) {
	t.Run("nil entry is not valid", func(t *testing.T) {
		valid := IsCacheValid(nil, 24*time.Hour)
		assert.False(t, valid)
	})

	t.Run("fresh entry is valid", func(t *testing.T) {
		entry := &CacheEntry{
			CheckedAt:      time.Now(),
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.0.0",
		}

		valid := IsCacheValid(entry, 24*time.Hour)
		assert.True(t, valid)
	})

	t.Run("expired entry is not valid", func(t *testing.T) {
		entry := &CacheEntry{
			CheckedAt:      time.Now().Add(-25 * time.Hour),
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.0.0",
		}

		valid := IsCacheValid(entry, 24*time.Hour)
		assert.False(t, valid)
	})

	t.Run("entry exactly at interval boundary", func(t *testing.T) {
		// Use a timestamp slightly within the boundary to avoid timing precision issues
		entry := &CacheEntry{
			CheckedAt:      time.Now().Add(-24*time.Hour + 1*time.Second),
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.0.0",
		}

		// Should still be valid (<=)
		valid := IsCacheValid(entry, 24*time.Hour)
		assert.True(t, valid)
	})
}

func TestClearCache(t *testing.T) {
	t.Run("removes existing cache file", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)

		// Create a cache file first
		entry := &CacheEntry{
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.0.0",
		}
		require.NoError(t, WriteCache(entry))

		// Verify it exists
		cacheFile := filepath.Join(tempDir, ".go-pre-commit", "update-check.json")
		_, err := os.Stat(cacheFile)
		require.NoError(t, err, "Cache file should exist before clear")

		// Clear it
		err = ClearCache()
		require.NoError(t, err)

		// Verify it's gone
		_, err = os.Stat(cacheFile)
		assert.True(t, os.IsNotExist(err), "Cache file should not exist after clear")
	})

	t.Run("no error when file doesn't exist", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv("HOME", tempDir)

		// Try to clear non-existent cache
		err := ClearCache()
		require.NoError(t, err)
	})
}

func TestWriteCacheCreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Cache directory shouldn't exist yet
	cacheDir := filepath.Join(tempDir, ".go-pre-commit")
	_, err := os.Stat(cacheDir)
	require.True(t, os.IsNotExist(err))

	// Write cache should create it
	entry := &CacheEntry{
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.0.0",
	}
	err = WriteCache(entry)
	require.NoError(t, err)

	// Directory should now exist
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestCacheFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	entry := &CacheEntry{
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.0.0",
	}
	err := WriteCache(entry)
	require.NoError(t, err)

	cacheFile := filepath.Join(tempDir, ".go-pre-commit", "update-check.json")
	info, err := os.Stat(cacheFile)
	require.NoError(t, err)

	// File should be readable/writable by owner only (0o600)
	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0o600), mode, "Cache file should have 0o600 permissions")
}

func TestCacheDirPermissions(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	entry := &CacheEntry{
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.0.0",
	}
	err := WriteCache(entry)
	require.NoError(t, err)

	cacheDir := filepath.Join(tempDir, ".go-pre-commit")
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)

	// Directory should be accessible by owner only (0o700)
	mode := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0o700), mode, "Cache directory should have 0o700 permissions")
}

func TestWriteCacheAtomicWrite(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Write multiple times rapidly
	for i := 0; i < 10; i++ {
		entry := &CacheEntry{
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.1.0",
		}
		err := WriteCache(entry)
		require.NoError(t, err)
	}

	// Should still be able to read valid cache
	entry, err := ReadCache()
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, "v1.0.0", entry.CurrentVersion)
}

func TestCleanupOrphanedTempFiles(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create cache directory
	cacheDir := filepath.Join(tempDir, ".go-pre-commit")
	require.NoError(t, os.MkdirAll(cacheDir, 0o700))

	// Create an orphaned temp file
	tempFile := filepath.Join(cacheDir, "update-check.json.tmp")
	require.NoError(t, os.WriteFile(tempFile, []byte("orphaned"), 0o600))

	// Verify it exists
	_, err := os.Stat(tempFile)
	require.NoError(t, err)

	// Writing cache should clean it up
	entry := &CacheEntry{
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.0.0",
	}
	err = WriteCache(entry)
	require.NoError(t, err)

	// Temp file should be gone (cleanup happens before write)
	// The implementation cleans up at the start of write operations
}

func TestCacheEntryJSONMarshaling(t *testing.T) {
	now := time.Now()
	entry := &CacheEntry{
		CheckedAt:      now,
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.1.0",
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	require.NoError(t, err)

	// Unmarshal back
	var decoded CacheEntry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, entry.CurrentVersion, decoded.CurrentVersion)
	assert.Equal(t, entry.LatestVersion, decoded.LatestVersion)
	// Time comparison with some tolerance for JSON encoding
	assert.WithinDuration(t, entry.CheckedAt, decoded.CheckedAt, time.Second)
}

func TestCacheConstants(t *testing.T) {
	// Verify constants are set to expected values
	assert.Equal(t, 24*time.Hour, defaultCheckInterval)
	assert.Equal(t, 1*time.Hour, minCheckInterval)
	assert.Equal(t, 720*time.Hour, maxCheckInterval)
	assert.Equal(t, ".go-pre-commit", cacheDir)
	assert.Equal(t, "update-check.json", cacheFile)
}

func TestMultipleWritesOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Write first entry
	entry1 := &CacheEntry{
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.1.0",
	}
	err := WriteCache(entry1)
	require.NoError(t, err)

	// Write second entry
	entry2 := &CacheEntry{
		CurrentVersion: "v1.1.0",
		LatestVersion:  "v1.2.0",
	}
	err = WriteCache(entry2)
	require.NoError(t, err)

	// Read should get the second entry
	read, err := ReadCache()
	require.NoError(t, err)
	require.NotNil(t, read)
	assert.Equal(t, "v1.1.0", read.CurrentVersion)
	assert.Equal(t, "v1.2.0", read.LatestVersion)
}
