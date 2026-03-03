// Package update provides update checking and caching functionality for go-pre-commit
package update

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

// Banner color constants
const (
	bannerColorReset  = "\033[0m"
	bannerColorYellow = "\033[33m"
)

// Banner layout constants
const (
	// upgradeCmd is the command users should run to update
	upgradeCmd = "go-pre-commit upgrade"

	// versionDisplayWidth is the fixed width for version strings in the banner
	// Chosen to accommodate typical semver versions like "v1.2.3-beta.1"
	versionDisplayWidth = 12
)

// ShowBanner displays an update banner to stderr when an update is available
// Returns early if result is nil, has an error, or no update is available
func ShowBanner(result *CheckResult) {
	if result == nil || result.Error != nil || !result.UpdateAvailable {
		return
	}

	banner := FormatBanner(result)

	if useColor() {
		fmt.Fprintf(os.Stderr, "\n%s%s%s\n", bannerColorYellow, banner, bannerColorReset)
	} else {
		fmt.Fprintf(os.Stderr, "\n%s\n", banner)
	}
}

// FormatBanner creates the update notification banner string
// Returns a formatted banner ready for display
func FormatBanner(result *CheckResult) string {
	if result == nil {
		return ""
	}

	// Choose banner style based on terminal capabilities
	if isTerminal(os.Stderr.Fd()) {
		return formatBannerFancy(result.CurrentVersion, result.LatestVersion)
	}
	return formatBannerASCII(result.CurrentVersion, result.LatestVersion)
}

// formatBannerFancy creates a fancy update notification banner with Unicode box drawing
func formatBannerFancy(current, latest string) string {
	const boxWidth = 70 // characters between │ and │

	currentPadded := padVersion(current, versionDisplayWidth)
	latestPadded := padVersion(latest, versionDisplayWidth)

	// Build version line and pad to box width
	versionLine := fmt.Sprintf("   Current: %s   Latest: %s", currentPadded, latestPadded)
	versionLine = padRight(versionLine, boxWidth)

	// Build command line and pad to box width
	cmdLine := "   " + upgradeCmd
	cmdLine = padRight(cmdLine, boxWidth)

	emptyLine := padRight("", boxWidth)

	lines := []string{
		"",
		"  ╭──────────────────────────────────────────────────────────────────────╮",
		"  │" + emptyLine + "│",
		"  │" + padRight("   A new version of GO-PRE-COMMIT is available!", boxWidth) + "│",
		"  │" + emptyLine + "│",
		"  │" + versionLine + "│",
		"  │" + emptyLine + "│",
		"  │" + padRight("   Upgrade:", boxWidth) + "│",
		"  │" + cmdLine + "│",
		"  │" + emptyLine + "│",
		"  ╰──────────────────────────────────────────────────────────────────────╯",
		"",
	}

	return strings.Join(lines, "\n")
}

// formatBannerASCII creates an ASCII-art update notification banner (no Unicode)
func formatBannerASCII(current, latest string) string {
	const boxWidth = 70 // characters between | and |

	currentPadded := padVersion(current, versionDisplayWidth)
	latestPadded := padVersion(latest, versionDisplayWidth)

	// Build version line and pad to box width
	versionLine := fmt.Sprintf("   Current: %s   Latest: %s", currentPadded, latestPadded)
	versionLine = padRight(versionLine, boxWidth)

	// Build command line and pad to box width
	cmdLine := "   " + upgradeCmd
	cmdLine = padRight(cmdLine, boxWidth)

	emptyLine := padRight("", boxWidth)

	lines := []string{
		"",
		"  +----------------------------------------------------------------------+",
		"  |" + emptyLine + "|",
		"  |" + padRight("   A new version of GO-PRE-COMMIT is available!", boxWidth) + "|",
		"  |" + emptyLine + "|",
		"  |" + versionLine + "|",
		"  |" + emptyLine + "|",
		"  |" + padRight("   Upgrade command:", boxWidth) + "|",
		"  |" + cmdLine + "|",
		"  |" + emptyLine + "|",
		"  +----------------------------------------------------------------------+",
		"",
	}

	return strings.Join(lines, "\n")
}

// padVersion pads a version string to a fixed width (Unicode-safe)
// Uses rune count for proper handling of multi-byte characters
func padVersion(version string, width int) string {
	runeCount := utf8.RuneCountInString(version)
	if runeCount >= width {
		// Truncate to width runes, not bytes
		runes := []rune(version)
		return string(runes[:width])
	}
	return version + strings.Repeat(" ", width-runeCount)
}

// padRight pads a string to a fixed width on the right (Unicode-safe)
// Uses rune count for proper handling of multi-byte characters
func padRight(s string, width int) string {
	runeCount := utf8.RuneCountInString(s)
	if runeCount >= width {
		// Truncate to width runes, not bytes
		runes := []rune(s)
		return string(runes[:width])
	}
	return s + strings.Repeat(" ", width-runeCount)
}

// useColor determines if color output should be enabled for the banner
// Checks stderr (not stdout) since banner goes to stderr
func useColor() bool {
	// Disable color in CI environments
	if os.Getenv("CI") != "" {
		return false
	}

	// Disable color if NO_COLOR is set
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Disable color if not a terminal
	return isTerminal(os.Stderr.Fd())
}

// isTerminal checks if the given file descriptor is a terminal
func isTerminal(fd uintptr) bool {
	return term.IsTerminal(int(fd)) // #nosec G115 -- fd from os.Stderr.Fd() is always a valid file descriptor, no overflow risk
}
