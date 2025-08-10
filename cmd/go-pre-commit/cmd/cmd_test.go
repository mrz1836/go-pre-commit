package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	rootCmd := builder.BuildRootCmd()

	// Test root command has expected properties
	assert.Equal(t, "go-pre-commit", rootCmd.Use)
	assert.Contains(t, rootCmd.Short, "Go Pre-commit System")

	// Build full command tree for testing subcommands
	rootCmd.AddCommand(builder.BuildInstallCmd())
	rootCmd.AddCommand(builder.BuildRunCmd())
	rootCmd.AddCommand(builder.BuildUninstallCmd())
	rootCmd.AddCommand(builder.BuildStatusCmd())

	// Test subcommands are registered
	commands := rootCmd.Commands()
	cmdMap := make(map[string]bool)
	for _, cmd := range commands {
		cmdMap[cmd.Name()] = true
	}

	assert.True(t, cmdMap["install"])
	assert.True(t, cmdMap["run"])
	assert.True(t, cmdMap["uninstall"])
	assert.True(t, cmdMap["status"])
}

func TestExecute_Version(t *testing.T) {
	// Save original
	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
	}()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set version info
	SetVersionInfo("1.0.0", "abc123", "2025-01-01")

	// Run with version flag
	os.Args = []string{"go-pre-commit", "--version"}

	err := Execute()
	require.NoError(t, err)

	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Version should contain the version we set
	assert.Contains(t, output, "version")
	// The actual version might be empty in tests
}

func TestExecute_Help(t *testing.T) {
	// Save original
	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
	}()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run with help flag
	os.Args = []string{"go-pre-commit", "--help"}

	err := Execute()
	require.NoError(t, err)

	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "Go Pre-commit System")
	assert.Contains(t, output, "Available Commands:")
	assert.Contains(t, output, "install")
	assert.Contains(t, output, "run")
	assert.Contains(t, output, "uninstall")
}

func TestInstallCommand(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	installCmd := builder.BuildInstallCmd()

	// Test install command properties
	assert.Equal(t, "install", installCmd.Use)
	assert.Contains(t, installCmd.Short, "Install")

	// Test flags
	forceFlag := installCmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)

	hookTypeFlag := installCmd.Flags().Lookup("hook-type")
	assert.NotNil(t, hookTypeFlag)
}

func TestRunCommand(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	runCmd := builder.BuildRunCmd()

	// Test run command properties
	assert.Equal(t, "run [check-name] [flags] [files...]", runCmd.Use)
	assert.Contains(t, runCmd.Short, "Run pre-commit checks")

	// Test flags
	flags := []string{"all-files", "files", "skip", "only", "parallel", "fail-fast", "show-checks"}
	for _, flagName := range flags {
		flag := runCmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "Flag %s should exist", flagName)
	}
}

func TestUninstallCommand(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	uninstallCmd := builder.BuildUninstallCmd()

	// Test uninstall command properties
	assert.Equal(t, "uninstall", uninstallCmd.Use)
	assert.Contains(t, uninstallCmd.Short, "Uninstall")

	// Test flags
	hookTypeFlag := uninstallCmd.Flags().Lookup("hook-type")
	assert.NotNil(t, hookTypeFlag)
}
