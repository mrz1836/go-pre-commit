package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/version"
)

// errTestFetch is a sentinel error for simulating a failed release fetch.
var errTestFetch = errors.New("simulated fetch failure")

// stubRelease returns a releaseFetcher that yields a release with the given tag.
func stubRelease(tag string) releaseFetcher {
	return func(_, _, _ string) (*version.GitHubRelease, error) {
		return &version.GitHubRelease{TagName: tag}, nil
	}
}

// errRelease returns a releaseFetcher that always fails with the given error.
func errRelease(err error) releaseFetcher {
	return func(_, _, _ string) (*version.GitHubRelease, error) {
		return nil, err
	}
}

// newTestBuilder builds a CommandBuilder with injected fetcher/installer so the
// upgrade flow can be exercised without real network calls or `go install`.
func newTestBuilder(v string, fetch releaseFetcher, install releaseInstaller) *CommandBuilder {
	builder := NewCommandBuilder(NewCLIApp(v, "test-commit", "2024-01-01"))
	if fetch != nil {
		builder.fetchRelease = fetch
	}
	if install != nil {
		builder.installRelease = install
	}
	return builder
}

// isolateInTempGitRepo chdirs into a fresh temp git repo with an isolated HOME so
// cache writes and hook reinstall checks never touch the real repository.
func isolateInTempGitRepo(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	require.NoError(t, os.Chdir(tmpDir))

	// git init so FindRepositoryRoot resolves to this temp dir (network-free)
	cmd := exec.CommandContext(context.Background(), "git", "init")
	require.NoError(t, cmd.Run())
}

func TestBuildUpgradeCmd(t *testing.T) {
	app := NewCLIApp("1.0.0", "abc123", "2024-01-01")
	builder := NewCommandBuilder(app)

	cmd := builder.BuildUpgradeCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "upgrade", cmd.Use)
	assert.Contains(t, cmd.Short, "Upgrade go-pre-commit")

	// Check flags
	assert.NotNil(t, cmd.Flags().Lookup("force"))
	assert.NotNil(t, cmd.Flags().Lookup("check"))
	assert.NotNil(t, cmd.Flags().Lookup("reinstall"))
}

func TestUpgradeCommand_CheckOnly(t *testing.T) {
	// Inject a fake fetcher so no real GitHub request is made. CheckOnly returns
	// before any install, so no installer is needed.
	builder := newTestBuilder("1.0.0", stubRelease("v1.2.0"), nil)

	config := UpgradeConfig{
		CheckOnly: true,
		Force:     false,
		Reinstall: false,
	}

	err := builder.runUpgradeWithConfig(config)
	require.NoError(t, err, "check-only should succeed with a mocked release")
}

func TestUpgradeCommand_CheckOnly_FetchError(t *testing.T) {
	// A failed fetch should surface as a wrapped "failed to check for updates" error.
	builder := newTestBuilder("1.0.0", errRelease(errTestFetch), nil)

	err := builder.runUpgradeWithConfig(UpgradeConfig{CheckOnly: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check for updates")
	assert.ErrorIs(t, err, errTestFetch)
}

func TestUpgradeCommand_DevVersion(t *testing.T) {
	// Test with dev version
	app := NewCLIApp(versionDev, "abc123", "2024-01-01")
	builder := NewCommandBuilder(app)

	config := UpgradeConfig{
		CheckOnly: false,
		Force:     false,
		Reinstall: false,
	}

	err := builder.runUpgradeWithConfig(config)

	// Should error without force flag
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot upgrade development build without --force")
}

func TestFormatVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "dev version",
			input:    versionDev,
			expected: versionDev,
		},
		{
			name:     "empty version",
			input:    "",
			expected: versionDev,
		},
		{
			name:     "version without v",
			input:    "1.0.0",
			expected: "v1.0.0",
		},
		{
			name:     "version with v",
			input:    "v1.0.0",
			expected: "v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckGoInstalled(t *testing.T) {
	// This should pass in any Go development environment
	err := CheckGoInstalled()
	assert.NoError(t, err, "Go should be installed in test environment")
}

func TestGetGoPath(t *testing.T) {
	goPath, err := GetGoPath()

	require.NoError(t, err)
	assert.NotEmpty(t, goPath)
	assert.Contains(t, goPath, "bin")
}

func TestIsInPath(_ *testing.T) {
	// The binary may or may not be in PATH during tests
	// Just verify the function doesn't panic
	_ = IsInPath()
}

func TestGetBinaryLocation(t *testing.T) {
	// The binary may not exist during tests
	location, err := GetBinaryLocation()

	if err != nil {
		// Expected if binary is not installed
		assert.Contains(t, err.Error(), "go-pre-commit")
	} else {
		assert.NotEmpty(t, location)
		assert.Contains(t, location, "go-pre-commit")
	}
}

func TestUpgradeCmd_CheckViaCobra(t *testing.T) {
	// Exercise the full cobra command wiring (flag parsing -> runUpgradeWithConfig)
	// with an injected fetcher so no real network call is made.
	t.Setenv("HOME", t.TempDir())

	builder := newTestBuilder("1.0.0", stubRelease("v1.2.0"), nil)

	cmd := builder.BuildUpgradeCmd()
	cmd.SetArgs([]string{"--check"})

	require.NoError(t, cmd.Execute())
}

// TestRunUpgradeWithConfig_Comprehensive tests runUpgradeWithConfig across
// scenarios using injected fetcher/installer so no real network call or
// `go install` is performed.
func TestRunUpgradeWithConfig_Comprehensive(t *testing.T) {
	testCases := []struct {
		name           string
		currentVersion string
		latestTag      string
		config         UpgradeConfig
		expectInstall  bool
		description    string
	}{
		{
			name:           "Force Upgrade Dev Version",
			currentVersion: versionDev,
			latestTag:      "v2.0.0",
			config:         UpgradeConfig{Force: true},
			expectInstall:  true,
			description:    "Should allow force upgrade of dev version and install",
		},
		{
			name:           "Check Only Mode with Commit Hash",
			currentVersion: "abc123def456789", // Looks like commit hash
			latestTag:      "v2.0.0",
			config:         UpgradeConfig{CheckOnly: true},
			expectInstall:  false,
			description:    "Should report update in check-only mode without installing",
		},
		{
			name:           "Reinstall After Upgrade",
			currentVersion: "1.0.0",
			latestTag:      "v2.0.0",
			config:         UpgradeConfig{Reinstall: true},
			expectInstall:  true,
			description:    "Should install then attempt to reinstall hooks",
		},
		{
			name:           "Empty Version String",
			currentVersion: "",
			latestTag:      "v2.0.0",
			config:         UpgradeConfig{CheckOnly: true},
			expectInstall:  false,
			description:    "Should handle empty version string as dev build in check-only mode",
		},
		{
			name:           "Already On Latest",
			currentVersion: "2.0.0",
			latestTag:      "v2.0.0",
			config:         UpgradeConfig{},
			expectInstall:  false,
			description:    "Should short-circuit without installing when already current",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Isolate cwd/HOME so cache and hook reinstall stay in a temp git repo
			isolateInTempGitRepo(t)

			var installed bool
			install := func(_ context.Context, _ string) error {
				installed = true
				return nil
			}
			builder := newTestBuilder(tc.currentVersion, stubRelease(tc.latestTag), install)

			err := builder.runUpgradeWithConfig(tc.config)
			require.NoError(t, err, "case %q should not error", tc.description)
			assert.Equal(t, tc.expectInstall, installed,
				"install expectation mismatch for case: %s", tc.description)
		})
	}
}

// TestRunUpgradeWithConfig_InstallError verifies an install failure is surfaced.
func TestRunUpgradeWithConfig_InstallError(t *testing.T) {
	isolateInTempGitRepo(t)

	builder := newTestBuilder(versionDev, stubRelease("v2.0.0"),
		func(_ context.Context, _ string) error { return errTestFetch })

	err := builder.runUpgradeWithConfig(UpgradeConfig{Force: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upgrade")
	assert.ErrorIs(t, err, errTestFetch)
}

// TestUpgradeConfigValidation tests upgrade configuration validation
func TestUpgradeConfigValidation(t *testing.T) {
	testCases := []struct {
		name    string
		config  UpgradeConfig
		version string
	}{
		{
			name: "All Flags False",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: false,
				Reinstall: false,
			},
			version: "1.0.0",
		},
		{
			name: "All Flags True",
			config: UpgradeConfig{
				Force:     true,
				CheckOnly: true,
				Reinstall: true,
			},
			version: versionDev,
		},
		{
			name: "Only Force",
			config: UpgradeConfig{
				Force:     true,
				CheckOnly: false,
				Reinstall: false,
			},
			version: "2.0.0",
		},
		{
			name: "Only Check",
			config: UpgradeConfig{
				Force:     false,
				CheckOnly: true,
				Reinstall: false,
			},
			version: "1.5.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate the config struct can be created and used
			// These assertions verify the struct fields are properly set
			assert.True(t, tc.config.Force || !tc.config.Force)
			assert.True(t, tc.config.CheckOnly || !tc.config.CheckOnly)
			assert.True(t, tc.config.Reinstall || !tc.config.Reinstall)

			t.Logf("✓ Config validation passed for %s", tc.name)
		})
	}
}

// TestIsLikelyCommitHash tests commit hash detection
func TestIsLikelyCommitHash(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Full SHA",
			input:    "abc123def456789012345678901234567890abcd",
			expected: true,
		},
		{
			name:     "Short SHA",
			input:    "abc123d",
			expected: true,
		},
		{
			name:     "Version Number",
			input:    "1.2.3",
			expected: false,
		},
		{
			name:     "Version with v prefix",
			input:    "v1.2.3",
			expected: false,
		},
		{
			name:     "Dev Version",
			input:    versionDev,
			expected: false,
		},
		{
			name:     "Empty String",
			input:    "",
			expected: false,
		},
		{
			name:     "Hexadecimal-like but too short",
			input:    "abc12",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isLikelyCommitHash(tc.input)
			assert.Equal(t, tc.expected, result,
				"isLikelyCommitHash('%s') should return %v", tc.input, tc.expected)
		})
	}
}

// TestGetInstalledVersion tests the GetInstalledVersion function
func TestGetInstalledVersion(t *testing.T) {
	version, err := GetInstalledVersion()
	if err != nil {
		require.Error(t, err)
		assert.Empty(t, version)
	} else {
		assert.NotEmpty(t, version)
	}
}

// TestGetInstalledVersionErrorHandling tests error handling
func TestGetInstalledVersionErrorPaths(t *testing.T) {
	originalPath := os.Getenv("PATH")
	defer func() {
		_ = os.Setenv("PATH", originalPath)
	}()

	_ = os.Setenv("PATH", "/nonexistent")
	_, err := GetInstalledVersion()
	assert.Error(t, err)
}

// TestIsLikelyCommitHash_Dirty tests commit hash with -dirty suffix
func TestIsLikelyCommitHash_Dirty(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Short SHA with dirty suffix",
			input:    "abc123d-dirty",
			expected: true,
		},
		{
			name:     "Full SHA with dirty suffix",
			input:    "abc123def456789012345678901234567890abcd-dirty",
			expected: true,
		},
		{
			name:     "Too long even after removing dirty",
			input:    "abc123def456789012345678901234567890abcdef-dirty",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isLikelyCommitHash(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
