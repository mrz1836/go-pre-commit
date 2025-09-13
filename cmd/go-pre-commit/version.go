package main

import (
	"errors"
	"runtime/debug"
	"strings"
	"time"
)

// Build-time variables injected via ldflags
// These are package-level but unexported to reduce global exposure
//
//nolint:gochecknoglobals // These are build-time injected variables, required for ldflags
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

// BuildInfo encapsulates build-time information
type BuildInfo struct {
	version   string
	commit    string
	buildDate string
}

// NewBuildInfo creates a new BuildInfo instance with build-time injected values
// and fallbacks to runtime build information
func NewBuildInfo() *BuildInfo {
	return &BuildInfo{
		version:   getVersionWithFallback(),
		commit:    getCommitWithFallback(),
		buildDate: getBuildDateWithFallback(),
	}
}

// Version returns the version string
func (b *BuildInfo) Version() string {
	return b.version
}

// Commit returns the commit hash
func (b *BuildInfo) Commit() string {
	return b.commit
}

// BuildDate returns the build date
func (b *BuildInfo) BuildDate() string {
	return b.buildDate
}

// IsModified returns true if the build has uncommitted changes
func (b *BuildInfo) IsModified() bool {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.modified" {
				return setting.Value == "true"
			}
		}
	}
	return false
}

// getVersionWithFallback returns the version information with fallback to BuildInfo
func getVersionWithFallback() string {
	// If version was set via ldflags and it's not a template placeholder, use it
	if version != "dev" && version != "" && !isTemplateString(version) {
		return version
	}

	// Try to get version from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		// Check if there's a module version (from go install @version)
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			// For go install @version, use the version as-is (already includes 'v' prefix)
			return info.Main.Version
		}

		// Try to get VCS revision as fallback for development builds
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				// Use short commit hash for readability
				if len(setting.Value) > 7 {
					return setting.Value[:7]
				}
				return setting.Value
			}
		}
	}

	// Default to "dev" if nothing else is available
	return "dev"
}

// getCommitWithFallback returns the commit hash with fallback to BuildInfo
func getCommitWithFallback() string {
	// If commit was set via ldflags and it's not a template placeholder, use it
	if commit != "none" && commit != "" && !isTemplateString(commit) {
		return commit
	}

	// Try to get from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				// For commit display, use short hash for readability
				if len(setting.Value) > 7 {
					return setting.Value[:7]
				}
				return setting.Value
			}
		}

		// For go install builds, try to extract commit from module sum if available
		if info.Main.Sum != "" {
			// Module sum format: h1:base64hash - extract first 7 chars of hash
			if parts := strings.Split(info.Main.Sum, ":"); len(parts) == 2 && len(parts[1]) >= 7 {
				return parts[1][:7]
			}
		}
	}

	return "none"
}

// getBuildDateWithFallback returns the build date with fallback to BuildInfo
func getBuildDateWithFallback() string {
	// If build date was set via ldflags and it's not a template placeholder, use it
	if buildDate != "unknown" && buildDate != "" && !isTemplateString(buildDate) {
		return buildDate
	}

	// Try to get from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.time" && setting.Value != "" {
				// VCS time is in RFC3339 format, convert to a more readable format
				if t, err := parseTime(setting.Value); err == nil {
					return t.Format("2006-01-02_15:04:05_UTC")
				}
				return setting.Value
			}
		}

		// For go install builds without VCS info, use a generic marker
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return "go-install"
		}
	}

	return "unknown"
}

// ErrUnableToParseTime is returned when time string cannot be parsed
var ErrUnableToParseTime = errors.New("unable to parse time")

// parseTime attempts to parse time from various formats
func parseTime(timeStr string) (time.Time, error) {
	// Try RFC3339 format first (Git's default)
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t.UTC(), nil
	}

	// Try other common formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, ErrUnableToParseTime
}

// Legacy compatibility functions - these wrap the new BuildInfo pattern
// to maintain backward compatibility during migration

// GetVersion returns the version information (legacy compatibility)
func GetVersion() string {
	return getVersionWithFallback()
}

// GetCommit returns the commit hash (legacy compatibility)
func GetCommit() string {
	return getCommitWithFallback()
}

// GetBuildDate returns the build date (legacy compatibility)
func GetBuildDate() string {
	return getBuildDateWithFallback()
}

// IsModified returns true if the build has uncommitted changes (legacy compatibility)
func IsModified() bool {
	bi := &BuildInfo{}
	return bi.IsModified()
}

// isTemplateString checks if a string contains unsubstituted template syntax
func isTemplateString(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}
