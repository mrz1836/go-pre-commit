// Package version provides version comparison and GitHub release fetching utilities
package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ErrGitHubAPIFailed is returned when GitHub API returns a non-200 status
var ErrGitHubAPIFailed = errors.New("GitHub API request failed")

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	Body        string    `json:"body"`
}

// Info contains version information
type Info struct {
	Current string
	Latest  string
	IsNewer bool
}

// GetLatestRelease fetches the latest release from GitHub
func GetLatestRelease(owner, repo string) (*GitHubRelease, error) {
	return GetLatestReleaseWithVersion(owner, repo, "dev")
}

// HTTPClient interface for dependency injection
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// GetLatestReleaseWithVersion fetches the latest release from GitHub with version info for User-Agent
func GetLatestReleaseWithVersion(owner, repo, currentVersion string) (*GitHubRelease, error) {
	return GetLatestReleaseWithVersionAndClient(owner, repo, currentVersion, nil)
}

// GetLatestReleaseWithVersionAndClient fetches the latest release with optional custom HTTP client
func GetLatestReleaseWithVersionAndClient(owner, repo, currentVersion string, client HTTPClient) (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	if client == nil {
		client = &http.Client{
			Timeout: 15 * time.Second, // Increased timeout for better reliability
		}
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set descriptive user agent
	userAgent := fmt.Sprintf("go-pre-commit/%s (%s/%s)", currentVersion, runtime.GOOS, runtime.GOARCH)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Add authentication if available
	if token := getGitHubToken(); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %s", ErrGitHubAPIFailed, formatGitHubError(resp.StatusCode, string(body), resp.Header))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &release, nil
}

// getGitHubToken returns GitHub token from environment variables
func getGitHubToken() string {
	// Try common GitHub token environment variables
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}
	return ""
}

// formatGitHubError formats GitHub API errors with helpful suggestions
func formatGitHubError(statusCode int, body string, headers http.Header) string {
	var msg strings.Builder
	fmt.Fprintf(&msg, "status %d: %s", statusCode, body) // #nosec G705 - msg is a strings.Builder, not an HTTP response writer

	// Handle rate limiting specifically
	if statusCode == 403 && strings.Contains(body, "rate limit") {
		msg.WriteString("\n\nTo avoid rate limits:")
		msg.WriteString("\n• Set GITHUB_TOKEN environment variable with a GitHub personal access token")
		msg.WriteString("\n• Or set GH_TOKEN if using GitHub CLI")
		msg.WriteString("\n• Authenticated requests have 5,000 requests/hour vs 60 for unauthenticated")

		// Show rate limit info if available
		if limit := headers.Get("X-RateLimit-Limit"); limit != "" {
			fmt.Fprintf(&msg, "\n• Current limit: %s requests/hour", limit) // #nosec G705 - msg is a strings.Builder, not an HTTP response writer
		}
		if remaining := headers.Get("X-RateLimit-Remaining"); remaining != "" {
			fmt.Fprintf(&msg, "\n• Remaining: %s requests", remaining) // #nosec G705 - msg is a strings.Builder, not an HTTP response writer
		}
		if reset := headers.Get("X-RateLimit-Reset"); reset != "" {
			if resetTime, err := strconv.ParseInt(reset, 10, 64); err == nil {
				resetAt := time.Unix(resetTime, 0)
				fmt.Fprintf(&msg, "\n• Rate limit resets at: %s", resetAt.Format("15:04:05 MST")) // #nosec G705 - msg is a strings.Builder, not an HTTP response writer
			}
		}
	}

	return msg.String()
}

// CompareVersions compares two version strings
// Returns:
//   - 1 if v1 > v2
//   - 0 if v1 == v2
//   - -1 if v1 < v2
func CompareVersions(v1, v2 string) int {
	// Clean versions (remove 'v' prefix if present)
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Handle development versions and commit hashes
	// Check if v1 is a development version or commit hash
	isV1Dev := v1 == "dev" || v1 == "" || isCommitHash(v1)
	// Check if v2 is a development version or commit hash
	isV2Dev := v2 == "dev" || v2 == "" || isCommitHash(v2)

	if isV1Dev && isV2Dev {
		// Both are dev/commit versions, consider them equal
		return 0
	}
	if isV1Dev {
		return -1 // dev/commit is always considered older than a release
	}
	if isV2Dev {
		return 1
	}

	// Split versions into parts
	parts1 := parseVersion(v1)
	parts2 := parseVersion(v2)

	// Compare major, minor, patch
	for i := 0; i < 3; i++ {
		if i >= len(parts1) && i >= len(parts2) {
			break
		}
		val1 := 0
		val2 := 0
		if i < len(parts1) {
			val1 = parts1[i]
		}
		if i < len(parts2) {
			val2 = parts2[i]
		}

		if val1 > val2 {
			return 1
		}
		if val1 < val2 {
			return -1
		}
	}

	return 0
}

// parseVersion parses a version string into major, minor, patch integers
func parseVersion(version string) []int {
	// Remove any suffixes like -dirty, -rc1, etc.
	if idx := strings.IndexAny(version, "-+"); idx != -1 {
		version = version[:idx]
	}

	parts := strings.Split(version, ".")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		var num int
		if _, err := fmt.Sscanf(part, "%d", &num); err == nil {
			result = append(result, num)
		}
	}

	return result
}

// IsNewerVersion checks if latestVersion is newer than currentVersion
func IsNewerVersion(currentVersion, latestVersion string) bool {
	return CompareVersions(latestVersion, currentVersion) > 0
}

// NormalizeVersion ensures version strings are in a consistent format
func NormalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")

	// Remove any git suffixes
	if idx := strings.Index(version, "-"); idx != -1 {
		version = version[:idx]
	}

	return version
}

// isCommitHash checks if a string looks like a git commit hash
func isCommitHash(s string) bool {
	// Commit hashes are typically 7-40 hex characters
	if len(s) < 7 || len(s) > 40 {
		return false
	}

	// Check if all characters are valid hex
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}

	return true
}
