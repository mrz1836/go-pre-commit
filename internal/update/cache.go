// Package update provides update checking and caching functionality for go-pre-commit
package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Cache constants
const (
	// cacheDir is the directory name under home for go-pre-commit cache
	cacheDir = ".go-pre-commit"

	// cacheFile is the filename for the update check cache
	cacheFile = "update-check.json"

	// defaultCheckInterval is the default time between version checks
	defaultCheckInterval = 24 * time.Hour

	// minCheckInterval is the minimum allowed check interval to prevent API abuse
	minCheckInterval = 1 * time.Hour

	// maxCheckInterval is the maximum allowed check interval (720 hours = 30 days)
	maxCheckInterval = 720 * time.Hour
)

// CacheEntry represents the cached update check data persisted to disk
type CacheEntry struct {
	CheckedAt      time.Time `json:"checked_at"`
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
}

// IsUpdateCheckDisabled returns true if update checking is disabled
// This happens when:
// - GO_PRE_COMMIT_DISABLE_UPDATE_CHECK is set to "1" or "true"
// - CI environment variable is set to "1" or "true"
func IsUpdateCheckDisabled() bool {
	// Check explicit disable flag
	if val := os.Getenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK"); val == "1" || val == "true" {
		return true
	}

	// Disable in CI environments
	if val := os.Getenv("CI"); val == "1" || val == "true" {
		return true
	}

	return false
}

// GetCheckInterval returns the configured update check interval
// Reads from GO_PRE_COMMIT_UPDATE_CHECK_INTERVAL env var
// Defaults to 24h, minimum 1h, maximum 720h
func GetCheckInterval() time.Duration {
	intervalStr := os.Getenv("GO_PRE_COMMIT_UPDATE_CHECK_INTERVAL")
	if intervalStr == "" {
		return defaultCheckInterval
	}

	duration, err := time.ParseDuration(intervalStr)
	if err != nil {
		return defaultCheckInterval
	}

	// Enforce minimum interval to prevent API abuse
	if duration < minCheckInterval {
		return minCheckInterval
	}

	// Enforce maximum interval (30 days)
	if duration > maxCheckInterval {
		return maxCheckInterval
	}

	return duration
}

// GetCacheDir returns the cache directory path (~/.go-pre-commit/)
func GetCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, cacheDir), nil
}

// getCacheFilePath returns the full path to the cache file
func getCacheFilePath() (string, error) {
	dir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, cacheFile), nil
}

// ReadCache reads the cached update check data
// Returns nil and no error if cache file doesn't exist
func ReadCache() (*CacheEntry, error) {
	// Clean up any orphaned temp files
	cleanupOrphanedTempFiles()

	filePath, err := getCacheFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath) // #nosec G304 -- filePath is constructed from user home dir and constant path
	if err != nil {
		if os.IsNotExist(err) {
			// No cache file exists yet - not an error
			return nil, nil //nolint:nilnil // nil entry with nil error is the expected return for missing cache
		}
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// WriteCache saves the update check data to the cache file
// Uses atomic write via temp file + rename
func WriteCache(entry *CacheEntry) error {
	// Clean up any orphaned temp files
	cleanupOrphanedTempFiles()

	dir, err := GetCacheDir()
	if err != nil {
		return err
	}

	// Ensure cache directory exists
	if mkdirErr := os.MkdirAll(dir, 0o700); mkdirErr != nil {
		return mkdirErr
	}

	filePath, err := getCacheFilePath()
	if err != nil {
		return err
	}

	// Set the check time
	entry.CheckedAt = time.Now()

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	// Write atomically by writing to temp file first, then renaming
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0o600); err != nil {
		return err
	}

	return os.Rename(tempFile, filePath)
}

// IsCacheValid checks if the cache entry is still valid based on the interval
// Returns false if entry is nil or if the cache has expired
func IsCacheValid(entry *CacheEntry, interval time.Duration) bool {
	if entry == nil {
		return false
	}
	return time.Since(entry.CheckedAt) <= interval
}

// ClearCache removes the cache file
func ClearCache() error {
	filePath, err := getCacheFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// cleanupOrphanedTempFiles removes any .tmp files left over from crashed writes
// This is called before read/write operations to prevent accumulation of orphaned temp files
func cleanupOrphanedTempFiles() {
	filePath, err := getCacheFilePath()
	if err != nil {
		return
	}

	tempFile := filePath + ".tmp"
	// Best effort cleanup - ignore errors as this is not critical
	_ = os.Remove(tempFile)
}
