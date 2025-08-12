package main

import (
	"runtime/debug"
	"strings"
)

// Build-time variables injected via ldflags
//
//nolint:gochecknoglobals // These are build-time injected variables
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// GetVersion returns the version information with fallback to BuildInfo
func GetVersion() string {
	// If version was set via ldflags, use it
	if Version != "dev" && Version != "" {
		return Version
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

// GetCommit returns the commit hash with fallback to BuildInfo
func GetCommit() string {
	// If commit was set via ldflags, use it
	if Commit != "none" && Commit != "" {
		return Commit
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

// GetBuildDate returns the build date with fallback to BuildInfo
func GetBuildDate() string {
	// If build date was set via ldflags, use it
	if BuildDate != "unknown" && BuildDate != "" {
		return BuildDate
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

// IsModified returns true if the build has uncommitted changes
func IsModified() bool {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.modified" {
				return setting.Value == "true"
			}
		}
	}
	return false
}
