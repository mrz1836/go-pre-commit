// Package version provides version comparison and GitHub release fetching utilities
package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
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
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set user agent to avoid rate limiting
	req.Header.Set("User-Agent", fmt.Sprintf("go-pre-commit/%s (%s/%s)", "dev", runtime.GOOS, runtime.GOARCH))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d: %s", ErrGitHubAPIFailed, resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &release, nil
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

	// Handle development versions
	if v1 == "dev" || v1 == "" {
		return -1 // dev is always considered older
	}
	if v2 == "dev" || v2 == "" {
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
