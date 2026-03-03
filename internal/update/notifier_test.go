package update

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartBackgroundCheckDisabled(t *testing.T) {
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "1")

	resultChan := StartBackgroundCheck(context.Background(), "v1.0.0")

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
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.1.0",
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	resultChan := StartBackgroundCheck(context.Background(), "v1.0.0")

	// Should receive cached result
	select {
	case result := <-resultChan:
		require.NotNil(t, result)
		assert.True(t, result.FromCache, "Result should be from cache")
		assert.Equal(t, "v1.0.0", result.CurrentVersion)
		assert.Equal(t, "v1.1.0", result.LatestVersion)
		assert.True(t, result.UpdateAvailable)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for result")
	}
}

func TestStartBackgroundCheckWithExpiredCache(t *testing.T) {
	// Skip if no GitHub token available (would make real API call)
	if os.Getenv("GITHUB_TOKEN") == "" &&
		os.Getenv("GH_TOKEN") == "" &&
		os.Getenv("GO_PRE_COMMIT_GITHUB_TOKEN") == "" {
		t.Skip("Skipping test that requires GitHub token")
	}

	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "")
	t.Setenv("CI", "")

	// Pre-populate cache with expired entry
	cacheEntry := &CacheEntry{
		CheckedAt:      time.Now().Add(-25 * time.Hour),
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.0.0",
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resultChan := StartBackgroundCheck(ctx, "v1.0.0")

	// Should receive result (from API call or error)
	select {
	case result := <-resultChan:
		// Result might be nil if API call failed
		if result != nil {
			assert.False(t, result.FromCache, "Result should not be from cache")
		}
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
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.2.0",
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	result := checkForUpdate(context.Background(), "v1.0.0")
	require.NotNil(t, result)

	assert.True(t, result.FromCache)
	assert.Equal(t, "v1.0.0", result.CurrentVersion)
	assert.Equal(t, "v1.2.0", result.LatestVersion)
	assert.True(t, result.UpdateAvailable)
	assert.NoError(t, result.Error)
}

func TestCheckForUpdateWithExpiredCacheAndNoToken(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)
	// Clear all token env vars
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GO_PRE_COMMIT_GITHUB_TOKEN", "")

	// Pre-populate cache with expired entry
	cacheEntry := &CacheEntry{
		CheckedAt:      time.Now().Add(-25 * time.Hour),
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.0.0",
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result := checkForUpdate(ctx, "v1.0.0")

	// Should return a result (possibly with error due to rate limiting)
	require.NotNil(t, result)
	// API call without token might hit rate limit or succeed
	// The result may or may not be from cache depending on API success
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
		CurrentVersion:  "v1.0.0",
		LatestVersion:   "v1.1.0",
		UpdateAvailable: true,
		CheckedAt:       now,
		FromCache:       false,
		Error:           nil,
	}

	assert.Equal(t, "v1.0.0", result.CurrentVersion)
	assert.Equal(t, "v1.1.0", result.LatestVersion)
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

	// Create expired cache to force API call
	cacheEntry := &CacheEntry{
		CheckedAt:      time.Now().Add(-25 * time.Hour),
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.0.0",
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	resultChan := StartBackgroundCheck(ctx, "v1.0.0")

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
	resultChan := StartBackgroundCheck(context.Background(), "v1.0.0")

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
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.1.0",
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	before := time.Now()
	result := checkForUpdate(context.Background(), "v1.0.0")
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

	// Create expired cache to force API call
	cacheEntry := &CacheEntry{
		CheckedAt:      time.Now().Add(-25 * time.Hour),
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.0.0",
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	// Use a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait a bit to ensure timeout
	time.Sleep(10 * time.Millisecond)

	result := checkForUpdate(ctx, "v1.0.0")

	// Should return result with timeout error
	require.NotNil(t, result)
	if result.Error != nil {
		// Timeout or deadline exceeded
		assert.False(t, result.FromCache)
	}
}

func TestGitHubConstants(t *testing.T) {
	// Verify constants are set correctly
	assert.Equal(t, "mrz1836", gitHubOwner)
	assert.Equal(t, "go-pre-commit", gitHubRepo)
	assert.Equal(t, 5*time.Second, updateCheckTimeout)
}

func TestCheckForUpdateWritesCache(t *testing.T) {
	// Skip if no GitHub token available
	if os.Getenv("GITHUB_TOKEN") == "" &&
		os.Getenv("GH_TOKEN") == "" &&
		os.Getenv("GO_PRE_COMMIT_GITHUB_TOKEN") == "" {
		t.Skip("Skipping test that requires GitHub token")
	}

	tempDir := t.TempDir()
	t.Setenv("HOME", tempDir)

	// Create expired cache to force API call
	cacheEntry := &CacheEntry{
		CheckedAt:      time.Now().Add(-25 * time.Hour),
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.0.0",
	}
	err := WriteCache(cacheEntry)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := checkForUpdate(ctx, "v1.0.0")
	require.NotNil(t, result)

	// If successful, cache should be updated
	if result.Error == nil && !result.FromCache {
		// Read cache back
		cached, err := ReadCache()
		require.NoError(t, err)
		require.NotNil(t, cached)

		// Cache should be newer than the old entry
		assert.True(t, cached.CheckedAt.After(cacheEntry.CheckedAt))
	}
}

func TestStartBackgroundCheckChannelBehavior(t *testing.T) {
	t.Setenv("GO_PRE_COMMIT_DISABLE_UPDATE_CHECK", "1")

	resultChan := StartBackgroundCheck(context.Background(), "v1.0.0")

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
			currentVersion:  "v1.0.0",
			cachedLatest:    "v1.1.0",
			expectAvailable: true,
		},
		{
			name:            "same version",
			currentVersion:  "v1.0.0",
			cachedLatest:    "v1.0.0",
			expectAvailable: false,
		},
		{
			name:            "current version ahead",
			currentVersion:  "v2.0.0",
			cachedLatest:    "v1.0.0",
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

			result := checkForUpdate(context.Background(), tt.currentVersion)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectAvailable, result.UpdateAvailable,
				"UpdateAvailable mismatch for current=%s, latest=%s",
				tt.currentVersion, tt.cachedLatest)
		})
	}
}
