package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallCmd_ParseFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T, cmd *cobra.Command)
	}{
		{
			name: "force flag",
			args: []string{"--force"},
			validate: func(t *testing.T, cmd *cobra.Command) {
				force, err := cmd.Flags().GetBool("force")
				require.NoError(t, err)
				assert.True(t, force)
			},
		},
		{
			name: "force flag short",
			args: []string{"-f"},
			validate: func(t *testing.T, cmd *cobra.Command) {
				force, err := cmd.Flags().GetBool("force")
				require.NoError(t, err)
				assert.True(t, force)
			},
		},
		{
			name: "hook-type flag single",
			args: []string{"--hook-type", "pre-push"},
			validate: func(t *testing.T, cmd *cobra.Command) {
				hookTypes, err := cmd.Flags().GetStringSlice("hook-type")
				require.NoError(t, err)
				assert.Equal(t, []string{"pre-push"}, hookTypes)
			},
		},
		{
			name: "hook-type flag multiple",
			args: []string{"--hook-type", "pre-commit", "--hook-type", "pre-push"},
			validate: func(t *testing.T, cmd *cobra.Command) {
				hookTypes, err := cmd.Flags().GetStringSlice("hook-type")
				require.NoError(t, err)
				assert.Equal(t, []string{"pre-commit", "pre-push"}, hookTypes)
			},
		},
		{
			name: "default hook type",
			args: []string{},
			validate: func(t *testing.T, cmd *cobra.Command) {
				hookTypes, err := cmd.Flags().GetStringSlice("hook-type")
				require.NoError(t, err)
				assert.Equal(t, []string{"pre-commit"}, hookTypes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh CLI app and command builder for each test case
			app := NewCLIApp("test", "test-commit", "test-date")
			builder := NewCommandBuilder(app)
			installCmd := builder.BuildInstallCmd()

			// Parse the flags
			err := installCmd.ParseFlags(tt.args)
			require.NoError(t, err)

			// Validate
			tt.validate(t, installCmd)
		})
	}
}

func TestInstallCmd_CommandStructure(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	cmd := builder.BuildInstallCmd()

	// Verify command has correct structure
	assert.Equal(t, "install", cmd.Name())
	assert.Contains(t, cmd.Short, "Install")

	// Check flags exist
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)

	hookTypeFlag := cmd.Flags().Lookup("hook-type")
	assert.NotNil(t, hookTypeFlag)
}
