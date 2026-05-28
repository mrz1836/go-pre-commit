package update

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/version"
)

// errTestNetwork is a sentinel error used by fake fetchers to simulate a
// network/API failure without making real network calls.
var errTestNetwork = errors.New("simulated network failure")

// writeExpiredCache writes a cache entry directly to disk with a back-dated
// CheckedAt timestamp. WriteCache always resets CheckedAt to time.Now(), so it
// cannot be used to create an expired entry; this helper bypasses that to force
// the fetch path during tests.
func writeExpiredCache(t *testing.T) {
	t.Helper()
	dir, err := GetCacheDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0o700))

	filePath, err := getCacheFilePath()
	require.NoError(t, err)

	entry := &CacheEntry{
		CheckedAt:      time.Now().Add(-25 * time.Hour),
		CurrentVersion: testVersionCurrent,
		LatestVersion:  testVersionCurrent,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filePath, data, 0o600))
}

// stubFetcher returns a ReleaseFetcher that yields a release with the given tag.
func stubFetcher(tag string) ReleaseFetcher {
	return func(_, _, _ string) (*version.GitHubRelease, error) {
		return &version.GitHubRelease{TagName: tag}, nil
	}
}

// errFetcher returns a ReleaseFetcher that always fails with the given error.
func errFetcher(err error) ReleaseFetcher {
	return func(_, _, _ string) (*version.GitHubRelease, error) {
		return nil, err
	}
}

// failingFetcher returns a ReleaseFetcher that fails the test if it is ever
// invoked. Use it when a valid cache should short-circuit the network path.
func failingFetcher(t *testing.T) ReleaseFetcher {
	t.Helper()
	return func(_, _, _ string) (*version.GitHubRelease, error) {
		t.Error("release fetcher should not be called when cache is valid")
		return nil, errTestNetwork
	}
}

// blockingFetcher returns a ReleaseFetcher that blocks until the test completes.
// This makes context timeout/cancellation paths deterministic: the fetcher never
// wins the select against an already-done context. The fetcher returns once the
// test's cleanup unblocks it (its buffered result is discarded, so no leak).
func blockingFetcher(t *testing.T) ReleaseFetcher {
	t.Helper()
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	return func(_, _, _ string) (*version.GitHubRelease, error) {
		<-release
		return nil, errTestNetwork
	}
}

func TestStartBackgroundCheckDisabled(t *testing.T) {
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "1")

	resultChan := StartBackgroundCheck(context.Background(), testVersionCurrent)

	// Channel should close without sending a result
	result, ok := <-resultChan
	assert.Nil(t, result, "Should not receive result when disabled")
	assert.False(t, ok, "Channel should be closed")
}

func TestStartBackgroundCheckDevVersion(t *testing.T) {
	// Clear disable flags
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "")
	t.Setenv("CI", "")

	resultChan := StartBackgroundCheck(context.Background(), "dev")

	// Channel should close without sending a result (dev builds skip check)
	result, ok := <-resultChan
	assert.Nil(t, result, "Should not receive result for dev version")
	assert.False(t, ok, "Channel should be closed")
}

func TestStartBackgroundCheckEmptyVersion(t *testing.T) {
	// Clear disable flags
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "")
	t.Setenv("CI", "")

	resultChan := StartBackgroundCheck(context.Background(), "")

	// Channel should close without sending a result (empty version skips check)
	result, ok := <-resultChan
	assert.Nil(t, result, "Should not receive result for empty version")
	assert.False(t, ok, "Channel should be closed")
}

func TestStartBackgroundCheckWithValidCache(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "")
	t.Setenv("CI", "")

	// Pre-populate cache with valid entry
	cacheEntry := &CacheEntry{
		CheckedAt:      time.Now(),
		CurrentVersion: testVersionCurrent,
		LatestVersion:  testVersionLatest,
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	resultChan := StartBackgroundCheck(context.Background(), testVersionCurrent)

	// Should receive cached result
	select {
	case result := <-resultChan:
		require.NotNil(t, result)
		assert.True(t, result.FromCache, "Result should be from cache")
		assert.Equal(t, testVersionCurrent, result.CurrentVersion)
		assert.Equal(t, testVersionLatest, result.LatestVersion)
		assert.True(t, result.UpdateAvailable)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for result")
	}
}

func TestStartBackgroundCheckWithExpiredCache(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "")
	t.Setenv("CI", "")

	// Pre-populate cache with an expired entry so the fetcher path is exercised
	writeExpiredCache(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Inject a fake fetcher so no real network call is made
	resultChan := startBackgroundCheck(ctx, testVersionCurrent, stubFetcher(testVersionLatest))

	select {
	case result := <-resultChan:
		require.NotNil(t, result, "Should receive a fresh result")
		assert.False(t, result.FromCache, "Result should not be from cache")
		assert.Equal(t, testVersionLatest, result.LatestVersion)
		assert.True(t, result.UpdateAvailable, "Newer version should be available")
		require.NoError(t, result.Error)
	case <-time.After(11 * time.Second):
		t.Fatal("Timeout waiting for result")
	}
}

func TestCheckForUpdateWithValidCache(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Pre-populate cache with valid entry
	cacheEntry := &CacheEntry{
		CheckedAt:      time.Now(),
		CurrentVersion: testVersionCurrent,
		LatestVersion:  "v1.2.0",
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	// Valid cache should short-circuit before the fetcher is ever called
	result := checkForUpdate(context.Background(), testVersionCurrent, failingFetcher(t))
	require.NotNil(t, result)

	assert.True(t, result.FromCache)
	assert.Equal(t, testVersionCurrent, result.CurrentVersion)
	assert.Equal(t, "v1.2.0", result.LatestVersion)
	assert.True(t, result.UpdateAvailable)
	assert.NoError(t, result.Error)
}

func TestCheckForUpdateWithExpiredCacheFetchError(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	// Clear all token env vars
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GO_PRE_COMMIT_GITHUB_TOKEN", "")

	// Pre-populate cache with an expired entry to force the fetch path
	writeExpiredCache(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Inject a fetcher that fails (simulating a network/API error)
	result := checkForUpdate(ctx, testVersionCurrent, errFetcher(errTestNetwork))

	require.NotNil(t, result)
	require.Error(t, result.Error, "Fetch failure should be surfaced on the result")
	require.ErrorIs(t, result.Error, errTestNetwork)
	assert.False(t, result.FromCache, "Failed fetch should not report a cached result")
}

func TestGetGitHubToken(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "no token set",
			envVars:  map[string]string{},
			expected: "",
		},
		{
			name: "GO_PRE_COMMIT_GITHUB_TOKEN set",
			envVars: map[string]string{
				"GO_PRE_COMMIT_GITHUB_TOKEN": "custom_token",
			},
			expected: "custom_token",
		},
		{
			name: "GITHUB_TOKEN set",
			envVars: map[string]string{
				"GITHUB_TOKEN": "github_token",
			},
			expected: "github_token",
		},
		{
			name: "GH_TOKEN set",
			envVars: map[string]string{
				"GH_TOKEN": "gh_token",
			},
			expected: "gh_token",
		},
		{
			name: "GO_PRE_COMMIT_GITHUB_TOKEN takes priority over GITHUB_TOKEN",
			envVars: map[string]string{
				"GO_PRE_COMMIT_GITHUB_TOKEN": "custom_token",
				"GITHUB_TOKEN":               "github_token",
			},
			expected: "custom_token",
		},
		{
			name: "GO_PRE_COMMIT_GITHUB_TOKEN takes priority over GH_TOKEN",
			envVars: map[string]string{
				"GO_PRE_COMMIT_GITHUB_TOKEN": "custom_token",
				"GH_TOKEN":                   "gh_token",
			},
			expected: "custom_token",
		},
		{
			name: "GITHUB_TOKEN takes priority over GH_TOKEN",
			envVars: map[string]string{
				"GITHUB_TOKEN": "github_token",
				"GH_TOKEN":     "gh_token",
			},
			expected: "github_token",
		},
		{
			name: "all tokens set - GO_PRE_COMMIT_GITHUB_TOKEN wins",
			envVars: map[string]string{
				"GO_PRE_COMMIT_GITHUB_TOKEN": "custom_token",
				"GITHUB_TOKEN":               "github_token",
				"GH_TOKEN":                   "gh_token",
			},
			expected: "custom_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all token env vars
			t.Setenv("GO_PRE_COMMIT_GITHUB_TOKEN", "")
			t.Setenv("GITHUB_TOKEN", "")
			t.Setenv("GH_TOKEN", "")

			// Set test env vars
			for key, val := range tt.envVars {
				t.Setenv(key, val)
			}

			result := getGitHubToken()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckResultStructure(t *testing.T) {
	now := time.Now()
	result := &CheckResult{
		CurrentVersion:  testVersionCurrent,
		LatestVersion:   testVersionLatest,
		UpdateAvailable: true,
		CheckedAt:       now,
		FromCache:       false,
		Error:           nil,
	}

	assert.Equal(t, testVersionCurrent, result.CurrentVersion)
	assert.Equal(t, testVersionLatest, result.LatestVersion)
	assert.True(t, result.UpdateAvailable)
	assert.Equal(t, now, result.CheckedAt)
	assert.False(t, result.FromCache)
	assert.NoError(t, result.Error)
}

func TestStartBackgroundCheckContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "")
	t.Setenv("CI", "")

	// Create an expired cache to force the fetch path so cancellation matters
	writeExpiredCache(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// blockingFetcher guarantees the canceled context wins the select
	resultChan := startBackgroundCheck(ctx, testVersionCurrent, blockingFetcher(t))

	// Should receive result or channel closes
	select {
	case result := <-resultChan:
		// Might get nil or a result with error
		if result != nil && result.Error != nil {
			// Context cancellation might be reflected in error
			t.Logf("Got error as expected: %v", result.Error)
		}
	case <-time.After(5 * time.Second):
		// Channel should close or return quickly
		t.Fatal("Expected quick return on canceled context")
	}
}

func TestStartBackgroundCheckRecoversFromPanic(t *testing.T) {
	// This test verifies the panic recovery in StartBackgroundCheck
	// We can't easily trigger a panic without modifying code, but we can
	// verify the function completes without panicking

	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "")
	t.Setenv("CI", "")

	// This should not panic regardless of what happens internally
	resultChan := startBackgroundCheck(context.Background(), testVersionCurrent, stubFetcher(testVersionLatest))

	select {
	case <-resultChan:
		// Channel closed or received result
	case <-time.After(10 * time.Second):
		// Timeout is acceptable - we're just verifying no panic
	}
}

func TestCheckForUpdateSetsCheckedAt(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Pre-populate cache with valid entry
	cacheEntry := &CacheEntry{
		CheckedAt:      time.Now().Add(-1 * time.Hour),
		CurrentVersion: testVersionCurrent,
		LatestVersion:  testVersionLatest,
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	// Cache is 1 hour old (within the default check interval) so it is valid
	// and the fetcher must not be called.
	before := time.Now()
	result := checkForUpdate(context.Background(), testVersionCurrent, failingFetcher(t))
	after := time.Now()

	require.NotNil(t, result)
	// CheckedAt should be from the cache (1 hour ago)
	assert.WithinDuration(t, cacheEntry.CheckedAt, result.CheckedAt, 2*time.Second)

	// But if it's from cache, CheckedAt is the cache timestamp
	if result.FromCache {
		assert.True(t, result.CheckedAt.Before(before))
	} else {
		// If it made an API call, CheckedAt should be recent
		assert.True(t, result.CheckedAt.After(before) || result.CheckedAt.Equal(before))
		assert.True(t, result.CheckedAt.Before(after) || result.CheckedAt.Equal(after))
	}
}

func TestCheckForUpdateTimeout(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create an expired cache to force the fetch path
	writeExpiredCache(t)

	// Use a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait a bit to ensure timeout
	time.Sleep(10 * time.Millisecond)

	// blockingFetcher never returns before the test ends, so the already-expired
	// context deterministically produces a timeout result with no network call.
	result := checkForUpdate(ctx, testVersionCurrent, blockingFetcher(t))

	// Should return result with timeout error
	require.NotNil(t, result)
	require.Error(t, result.Error, "expired context should produce an error")
	require.ErrorIs(t, result.Error, context.DeadlineExceeded)
	assert.False(t, result.FromCache)
}

func TestGitHubConstants(t *testing.T) {
	// Verify constants are set correctly
	assert.Equal(t, "mrz1836", gitHubOwner)
	assert.Equal(t, "go-pre-commit", gitHubRepo)
	assert.Equal(t, 5*time.Second, updateCheckTimeout)
}

func TestCheckForUpdateWritesCache(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create an expired cache to force the fetch path
	oldCheckedAt := time.Now().Add(-25 * time.Hour)
	writeExpiredCache(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Inject a stub fetcher returning a newer version; no real network call
	result := checkForUpdate(ctx, testVersionCurrent, stubFetcher(testVersionLatest))
	require.NotNil(t, result)
	require.NoError(t, result.Error)
	assert.False(t, result.FromCache)
	assert.Equal(t, testVersionLatest, result.LatestVersion)

	// Cache should be updated with the freshly fetched version
	cached, err := ReadCache()
	require.NoError(t, err)
	require.NotNil(t, cached)
	assert.Equal(t, testVersionLatest, cached.LatestVersion)
	assert.True(t, cached.CheckedAt.After(oldCheckedAt))
}

func TestStartBackgroundCheckChannelBehavior(t *testing.T) {
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "1")

	resultChan := StartBackgroundCheck(context.Background(), testVersionCurrent)

	// Verify channel is buffered and closed
	_, ok := <-resultChan
	assert.False(t, ok, "Channel should be closed")

	// Reading from closed channel should immediately return zero value
	result, ok := <-resultChan
	assert.Nil(t, result)
	assert.False(t, ok)
}

func TestCheckForUpdateUpdateAvailableLogic(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	tests := []struct {
		name            string
		currentVersion  string
		cachedLatest    string
		expectAvailable bool
	}{
		{
			name:            "newer version available",
			currentVersion:  testVersionCurrent,
			cachedLatest:    testVersionLatest,
			expectAvailable: true,
		},
		{
			name:            "same version",
			currentVersion:  testVersionCurrent,
			cachedLatest:    testVersionCurrent,
			expectAvailable: false,
		},
		{
			name:            "current version ahead",
			currentVersion:  "v2.0.0",
			cachedLatest:    testVersionCurrent,
			expectAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean cache for each test
			_ = ClearCache()

			cacheEntry := &CacheEntry{
				CheckedAt:      time.Now(),
				CurrentVersion: tt.currentVersion,
				LatestVersion:  tt.cachedLatest,
			}
			err := WriteCache(cacheEntry)
			require.NoError(t, err)

			// Fresh, valid cache: the fetcher must not be called
			result := checkForUpdate(context.Background(), tt.currentVersion, failingFetcher(t))
			require.NotNil(t, result)

			assert.Equal(t, tt.expectAvailable, result.UpdateAvailable,
				"UpdateAvailable mismatch for current=%s, latest=%s",
				tt.currentVersion, tt.cachedLatest)
		})
	}
}
