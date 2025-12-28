package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationErrorFormatting(t *testing.T) {
	err := &ValidationError{Errors: []string{"first issue", "second issue"}}

	require.Equal(t, "configuration validation failed:\n  - first issue\n  - second issue", err.Error())
}

func TestGetConfigHelpIncludesSections(t *testing.T) {
	help := GetConfigHelp()

	require.Contains(t, help, "GoFortress Pre-commit System Configuration Help")
	require.Contains(t, help, "Environment Variables:")
	require.Contains(t, help, "GO_PRE_COMMIT_ENABLE_FMT")
	require.Contains(t, help, "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION")
	require.Contains(t, help, "GO_PRE_COMMIT_EXCLUDE_PATTERNS")
}

func TestStripComments(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "comment only",
			input:    "# just a comment",
			expected: "",
		},
		{
			name:     "inline comment after space",
			input:    "value #comment",
			expected: "value",
		},
		{
			name:     "inline comment after tab",
			input:    "value\t#comment",
			expected: "value",
		},
		{
			name:     "hash as part of value",
			input:    "value#notacomment",
			expected: "value#notacomment",
		},
		{
			name:     "hash with trailing space in value",
			input:    "value #comment #more",
			expected: "value",
		},
		{
			name:     "leading space before comment",
			input:    " value #comment",
			expected: "value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, stripComments(tc.input))
		})
	}
}

func TestGetStringEnvStripsComments(t *testing.T) {
	require.NoError(t, os.Setenv("TEST_STRING_WITH_COMMENT", "value #comment"))
	defer func() {
		require.NoError(t, os.Unsetenv("TEST_STRING_WITH_COMMENT"))
	}()

	require.Equal(t, "value", getStringEnv("TEST_STRING_WITH_COMMENT", "default"))
}
