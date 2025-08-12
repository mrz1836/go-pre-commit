package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	// Create app with a specific version
	app := NewCLIApp("1.0.0", "abc123", "2024-01-01")
	builder := NewCommandBuilder(app)

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	config := UpgradeConfig{
		CheckOnly: true,
		Force:     false,
		Reinstall: false,
	}

	// This will try to fetch from GitHub
	// In a real test environment, we'd mock this
	err := builder.runUpgradeWithConfig(config)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// The command should not error when checking
	// It may fail to fetch from GitHub in test environment
	if err != nil {
		assert.Contains(t, err.Error(), "failed to check for updates")
	} else {
		assert.Contains(t, output, "version")
	}
}

func TestUpgradeCommand_DevVersion(t *testing.T) {
	// Test with dev version
	app := NewCLIApp("dev", "abc123", "2024-01-01")
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
			input:    "dev",
			expected: "dev",
		},
		{
			name:     "empty version",
			input:    "",
			expected: "dev",
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

func TestUpgradeCmd_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()

	err := os.Chdir(tmpDir)
	require.NoError(t, err)

	// Initialize as a git repo
	err = os.MkdirAll(".git/hooks", 0o750)
	require.NoError(t, err)

	// Create app and builder
	app := NewCLIApp("1.0.0", "test", "2024-01-01")
	builder := NewCommandBuilder(app)

	// Build the command
	cmd := builder.BuildUpgradeCmd()

	// Set check-only flag
	cmd.SetArgs([]string{"--check"})

	// Execute command (may fail due to network)
	err = cmd.Execute()
	// We don't fail the test if network is unavailable
	if err != nil {
		t.Logf("Command execution failed (may be offline): %v", err)
	}
}
