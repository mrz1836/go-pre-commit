// Package update provides update checking and caching functionality for go-pre-commit
package update

import (
	"context"
	"os"
	"time"

	"github.com/mrz1836/go-pre-commit/internal/version"
)

// GitHub repository constants
const (
	// gitHubOwner is the GitHub owner for go-pre-commit releases
	gitHubOwner = "mrz1836"

	// gitHubRepo is the GitHub repository for go-pre-commit releases
	gitHubRepo = "go-pre-commit"

	// updateCheckTimeout is the maximum time for an update check API call
	updateCheckTimeout = 5 * time.Second
)

// CheckResult contains the result of an update check
type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	CheckedAt       time.Time
	FromCache       bool
	Error           error
}

// StartBackgroundCheck starts an asynchronous update check
// Returns a channel that receives the result when complete
// The check is non-blocking and runs in a goroutine
// The channel is closed after sending the result or on early return
func StartBackgroundCheck(ctx context.Context, currentVersion string) <-chan *CheckResult {
	resultChan := make(chan *CheckResult, 1)

	go func() {
		defer close(resultChan)

		// Recover from any panics to prevent crashing the CLI
		defer func() {
			if r := recover(); r != nil {
				// Panic recovered, return nothing to prevent crash
				// In production, this would ideally log somewhere
				_ = r
			}
		}()

		// Skip if update checking is disabled
		if IsUpdateCheckDisabled() {
			return
		}

		// Skip if version is dev or empty (development builds)
		if currentVersion == "" || currentVersion == "dev" {
			return
		}

		result := checkForUpdate(ctx, currentVersion)
		if result != nil {
			resultChan <- result
		}
	}()

	return resultChan
}

// checkForUpdate performs the update check with cache logic
func checkForUpdate(ctx context.Context, currentVersion string) *CheckResult {
	// Check cache first
	cached, err := ReadCache()
	if err == nil && IsCacheValid(cached, GetCheckInterval()) {
		// Return cached result if valid
		return &CheckResult{
			CurrentVersion:  currentVersion,
			LatestVersion:   cached.LatestVersion,
			UpdateAvailable: version.IsNewerVersion(currentVersion, cached.LatestVersion),
			CheckedAt:       cached.CheckedAt,
			FromCache:       true,
		}
	}

	// Create timeout context for API call
	checkCtx, cancel := context.WithTimeout(ctx, updateCheckTimeout)
	defer cancel()

	// Fetch latest release from GitHub with timeout
	// Set GITHUB_TOKEN temporarily if GO_PRE_COMMIT_GITHUB_TOKEN is set
	// This ensures our custom token priority is respected by version.GetLatestReleaseWithVersion
	token := getGitHubToken()
	if token != "" {
		// Check if this is from GO_PRE_COMMIT_GITHUB_TOKEN (custom priority)
		if customToken := os.Getenv("GO_PRE_COMMIT_GITHUB_TOKEN"); customToken != "" && customToken == token {
			// Temporarily override GITHUB_TOKEN so version package uses our custom token
			originalToken := os.Getenv("GITHUB_TOKEN")
			_ = os.Setenv("GITHUB_TOKEN", token)
			defer func() {
				if originalToken != "" {
					_ = os.Setenv("GITHUB_TOKEN", originalToken)
				} else {
					_ = os.Unsetenv("GITHUB_TOKEN")
				}
			}()
		}
	}

	// Use goroutine to respect context timeout since version.GetLatestReleaseWithVersion
	// doesn't accept a context parameter
	type apiResult struct {
		release *version.GitHubRelease
		err     error
	}
	resultChan := make(chan apiResult, 1)

	go func() { //nolint:contextcheck // GetLatestReleaseWithVersion doesn't accept context; timeout enforced via select
		rel, err := version.GetLatestReleaseWithVersion(gitHubOwner, gitHubRepo, currentVersion)
		resultChan <- apiResult{release: rel, err: err}
	}()

	// Wait for result or context timeout
	var release *version.GitHubRelease
	var apiErr error

	select {
	case <-checkCtx.Done():
		// Context canceled (timeout or parent cancellation)
		return &CheckResult{
			CurrentVersion: currentVersion,
			CheckedAt:      time.Now(),
			FromCache:      false,
			Error:          checkCtx.Err(),
		}
	case res := <-resultChan:
		release = res.release
		apiErr = res.err
	}

	if apiErr != nil {
		return &CheckResult{
			CurrentVersion: currentVersion,
			CheckedAt:      time.Now(),
			FromCache:      false,
			Error:          apiErr,
		}
	}

	// Build result
	latestVersion := release.TagName
	updateAvailable := version.IsNewerVersion(currentVersion, latestVersion)

	result := &CheckResult{
		CurrentVersion:  currentVersion,
		LatestVersion:   latestVersion,
		UpdateAvailable: updateAvailable,
		CheckedAt:       time.Now(),
		FromCache:       false,
	}

	// Write to cache (best effort, ignore errors)
	_ = WriteCache(&CacheEntry{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
	})

	return result
}

// getGitHubToken returns the GitHub token for update checks
// Priority: GO_PRE_COMMIT_GITHUB_TOKEN > GITHUB_TOKEN > GH_TOKEN
func getGitHubToken() string {
	if token := os.Getenv("GO_PRE_COMMIT_GITHUB_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	return os.Getenv("GH_TOKEN")
}
