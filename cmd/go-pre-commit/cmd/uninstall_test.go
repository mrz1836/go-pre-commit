package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUninstallCmd_ParseFlags(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	uninstallCmd := builder.BuildUninstallCmd()

	// Parse the flags
	err := uninstallCmd.ParseFlags([]string{"--hook-type", "pre-push"})
	require.NoError(t, err)

	// Validate flags were parsed correctly
	hookTypes, err := uninstallCmd.Flags().GetStringSlice("hook-type")
	require.NoError(t, err)
	assert.Equal(t, []string{"pre-push"}, hookTypes)
}

func TestUninstallCmd_CommandStructure(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	cmd := builder.BuildUninstallCmd()

	// Verify command has correct structure
	assert.Equal(t, "uninstall", cmd.Name())
	assert.Contains(t, cmd.Short, "Uninstall")

	// Check flags exist
	hookTypeFlag := cmd.Flags().Lookup("hook-type")
	assert.NotNil(t, hookTypeFlag)
}
