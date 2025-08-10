package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCmd_CommandStructure(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	runCmd := builder.BuildRunCmd()

	// Test basic command properties
	assert.Equal(t, "run [check-name] [flags] [files...]", runCmd.Use)
	assert.Contains(t, runCmd.Short, "Run pre-commit checks")

	// Test that all expected flags exist
	expectedFlags := []string{
		"all-files", "files", "skip", "only", "parallel",
		"fail-fast", "show-checks", "graceful", "progress", "quiet",
	}

	for _, flagName := range expectedFlags {
		flag := runCmd.Flags().Lookup(flagName)
		assert.NotNil(t, flag, "Flag %s should exist", flagName)
	}
}

func TestRunCmd_FlagParsing(t *testing.T) {
	// Create CLI app and command builder
	app := NewCLIApp("test", "test-commit", "test-date")
	builder := NewCommandBuilder(app)
	runCmd := builder.BuildRunCmd()

	// Test parsing various flags
	testCases := []struct {
		name     string
		args     []string
		flagName string
		expected interface{}
	}{
		{
			name:     "all-files flag",
			args:     []string{"--all-files"},
			flagName: "all-files",
			expected: true,
		},
		{
			name:     "parallel flag",
			args:     []string{"--parallel", "4"},
			flagName: "parallel",
			expected: 4,
		},
		{
			name:     "files flag",
			args:     []string{"--files", "file1.go,file2.go"},
			flagName: "files",
			expected: []string{"file1.go", "file2.go"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := runCmd.ParseFlags(tc.args)
			require.NoError(t, err)

			switch tc.flagName {
			case "all-files":
				value, err := runCmd.Flags().GetBool(tc.flagName)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, value)
			case "parallel":
				value, err := runCmd.Flags().GetInt(tc.flagName)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, value)
			case "files":
				value, err := runCmd.Flags().GetStringSlice(tc.flagName)
				require.NoError(t, err)
				assert.Equal(t, tc.expected, value)
			}
		})
	}
}
