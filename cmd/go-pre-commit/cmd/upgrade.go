package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-pre-commit/internal/git"
	"github.com/mrz1836/go-pre-commit/internal/version"
)

var (
	// ErrDevVersionNoForce is returned when trying to upgrade a dev version without --force
	ErrDevVersionNoForce = errors.New("cannot upgrade development build without --force")
	// ErrVersionParseFailed is returned when version cannot be parsed from output
	ErrVersionParseFailed = errors.New("could not parse version from output")
)

// UpgradeConfig holds configuration for the upgrade command
type UpgradeConfig struct {
	Force     bool
	CheckOnly bool
	Reinstall bool
}

// BuildUpgradeCmd creates the upgrade command
func (cb *CommandBuilder) BuildUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade go-pre-commit to the latest version",
		Long: `Upgrade the Go pre-commit system to the latest version available.

This command will:
  - Check the latest version available on GitHub
  - Compare with the currently installed version
  - Upgrade if a newer version is available
  - Optionally reinstall hooks after upgrade`,
		Example: `  # Check for available updates
  go-pre-commit upgrade --check

  # Upgrade to latest version
  go-pre-commit upgrade

  # Force upgrade even if already on latest
  go-pre-commit upgrade --force

  # Upgrade and reinstall hooks
  go-pre-commit upgrade --reinstall`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			config := UpgradeConfig{}
			var err error

			config.Force, err = cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

			config.CheckOnly, err = cmd.Flags().GetBool("check")
			if err != nil {
				return err
			}

			config.Reinstall, err = cmd.Flags().GetBool("reinstall")
			if err != nil {
				return err
			}

			return cb.runUpgradeWithConfig(config)
		},
	}

	// Add flags
	cmd.Flags().BoolP("force", "f", false, "Force upgrade even if already on latest version")
	cmd.Flags().BoolP("check", "c", false, "Check for updates without upgrading")
	cmd.Flags().BoolP("reinstall", "r", false, "Reinstall hooks after upgrade")

	return cmd
}

func (cb *CommandBuilder) runUpgradeWithConfig(config UpgradeConfig) error {
	currentVersion := cb.app.version

	// Handle development version or commit hash
	if currentVersion == "dev" || currentVersion == "" || isLikelyCommitHash(currentVersion) {
		if !config.Force && !config.CheckOnly {
			printWarning("Current version appears to be a development build (%s)", currentVersion)
			printInfo("Use --force to upgrade anyway")
			return ErrDevVersionNoForce
		}
	}

	printInfo("Current version: %s", formatVersion(currentVersion))

	// Fetch latest release
	printInfo("Checking for updates...")
	release, err := version.GetLatestRelease("mrz1836", "go-pre-commit")
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	printInfo("Latest version: %s", formatVersion(latestVersion))

	// Compare versions
	isNewer := version.IsNewerVersion(currentVersion, latestVersion)

	if !isNewer && !config.Force {
		printSuccess("You are already on the latest version (%s)", formatVersion(currentVersion))
		return nil
	}

	if config.CheckOnly {
		if isNewer {
			printWarning("A newer version is available: %s → %s", formatVersion(currentVersion), formatVersion(latestVersion))
			printInfo("Run 'go-pre-commit upgrade' to upgrade")
		} else {
			printSuccess("You are on the latest version")
		}
		return nil
	}

	// Perform upgrade
	if isNewer {
		printInfo("Upgrading from %s to %s...", formatVersion(currentVersion), formatVersion(latestVersion))
	} else if config.Force {
		printInfo("Force reinstalling version %s...", formatVersion(latestVersion))
	}

	// Run go install command
	installCmd := fmt.Sprintf("github.com/mrz1836/go-pre-commit/cmd/go-pre-commit@v%s", latestVersion)

	printInfo("Running: go install %s", installCmd)

	cmd := exec.CommandContext(context.Background(), "go", "install", installCmd) //nolint:gosec // Command is constructed safely
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to upgrade: %w", err)
	}

	printSuccess("Successfully upgraded to version %s", formatVersion(latestVersion))

	// Check if we should reinstall hooks
	if config.Reinstall {
		printInfo("Reinstalling hooks...")
		if err := cb.reinstallHooks(); err != nil {
			printWarning("Failed to reinstall hooks: %v", err)
			printInfo("You may need to run 'go-pre-commit install' manually")
		} else {
			printSuccess("Hooks reinstalled successfully")
		}
	} else {
		// Check if hooks need to be reinstalled
		cb.checkHookCompatibility()
	}

	// Show release notes if available
	if release.Body != "" && cb.app.config.Verbose {
		printInfo("\nRelease notes for v%s:", latestVersion)
		lines := strings.Split(release.Body, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				printInfo("  %s", line)
			}
		}
	}

	return nil
}

func (cb *CommandBuilder) reinstallHooks() error {
	// Get the repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	// Create installer
	installer := git.NewInstaller(repoRoot, "")

	// Check which hooks are installed and reinstall them
	hookTypes := []string{"pre-commit", "pre-push", "commit-msg", "post-commit"}
	reinstalled := 0

	for _, hookType := range hookTypes {
		if installer.IsHookInstalled(hookType) {
			if err := installer.InstallHook(hookType, true); err != nil {
				return fmt.Errorf("failed to reinstall %s hook: %w", hookType, err)
			}
			reinstalled++
		}
	}

	if reinstalled == 0 {
		printInfo("No hooks were installed, skipping reinstall")
	} else {
		printSuccess("Reinstalled %d hook(s)", reinstalled)
	}

	return nil
}

func (cb *CommandBuilder) checkHookCompatibility() {
	// Get the repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		// Not in a git repo, skip check
		return
	}

	installer := git.NewInstaller(repoRoot, "")

	// Check if any hooks are installed
	hookTypes := []string{"pre-commit", "pre-push", "commit-msg", "post-commit"}
	hasHooks := false

	for _, hookType := range hookTypes {
		if installer.IsHookInstalled(hookType) {
			hasHooks = true
			break
		}
	}

	if hasHooks && cb.app.config.Verbose {
		printInfo("Existing hooks detected. They should continue to work with the new version.")
		printInfo("If you experience issues, run 'go-pre-commit install --force' to update them.")
	}
}

func formatVersion(v string) string {
	if v == "dev" || v == "" {
		return "dev"
	}
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

// GetInstalledVersion attempts to get the version of the installed binary
func GetInstalledVersion() (string, error) {
	// Try to run the binary with --version flag
	cmd := exec.CommandContext(context.Background(), "go-pre-commit", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version: %w", err)
	}

	// Parse the version from output
	// Expected format: "go-pre-commit version X.Y.Z (commit: abc123, built: date)"
	outputStr := strings.TrimSpace(string(output))
	parts := strings.Fields(outputStr)

	for i, part := range parts {
		if part == "version" && i+1 < len(parts) {
			version := parts[i+1]
			// Clean up version string
			version = strings.TrimPrefix(version, "v")
			return version, nil
		}
	}

	return "", fmt.Errorf("%w: %s", ErrVersionParseFailed, outputStr)
}

// CheckGoInstalled verifies that Go is installed and available
func CheckGoInstalled() error {
	cmd := exec.CommandContext(context.Background(), "go", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go is not installed or not in PATH: %w", err)
	}
	return nil
}

// GetGoPath returns the GOPATH/bin directory where binaries are installed
func GetGoPath() (string, error) {
	cmd := exec.CommandContext(context.Background(), "go", "env", "GOPATH")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GOPATH: %w", err)
	}

	gopath := strings.TrimSpace(string(output))
	if gopath == "" {
		// Use default GOPATH
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		gopath = fmt.Sprintf("%s/go", home)
	}

	return fmt.Sprintf("%s/bin", gopath), nil
}

// IsInPath checks if go-pre-commit binary is in PATH
func IsInPath() bool {
	_, err := exec.LookPath("go-pre-commit")
	return err == nil
}

// GetBinaryLocation returns the location of the go-pre-commit binary
func GetBinaryLocation() (string, error) {
	if runtime.GOOS == "windows" {
		return exec.LookPath("go-pre-commit.exe")
	}
	return exec.LookPath("go-pre-commit")
}

// isLikelyCommitHash checks if a version string looks like a commit hash
func isLikelyCommitHash(version string) bool {
	// Remove any -dirty suffix
	version = strings.TrimSuffix(version, "-dirty")

	// Commit hashes are typically 7-40 hex characters
	if len(version) < 7 || len(version) > 40 {
		return false
	}

	// Check if all characters are valid hex
	for _, c := range version {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}

	return true
}
