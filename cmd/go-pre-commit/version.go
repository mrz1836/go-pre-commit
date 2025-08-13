package main

import (
	"runtime/debug"
	"strings"
)

// Build-time variables injected via ldflags
// These are package-level but unexported to reduce global exposure
//
//nolint:gochecknoglobals // These are build-time injected variables, required for ldflags
var (
	injectedVersion   = "dev"
	injectedCommit    = "none"
	injectedBuildDate = "unknown"
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
	// If version was set via ldflags, use it
	if injectedVersion != "dev" && injectedVersion != "" {
		return injectedVersion
	}

	// Try to get version from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		// Check if there's a module version (from go install @version)
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			// Clean up the version string
			version := info.Main.Version
			// Remove 'v' prefix if present for consistency
			version = strings.TrimPrefix(version, "v")
			return version
		}

		// Try to get VCS revision as fallback
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				// Use short commit hash like we do in Makefile
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
	// If commit was set via ldflags, use it
	if injectedCommit != "none" && injectedCommit != "" {
		return injectedCommit
	}

	// Try to get from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				return setting.Value
			}
		}
	}

	return "none"
}

// getBuildDateWithFallback returns the build date with fallback to BuildInfo
func getBuildDateWithFallback() string {
	// If build date was set via ldflags, use it
	if injectedBuildDate != "unknown" && injectedBuildDate != "" {
		return injectedBuildDate
	}

	// Try to get from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.time" && setting.Value != "" {
				return setting.Value
			}
		}
	}

	return "unknown"
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
