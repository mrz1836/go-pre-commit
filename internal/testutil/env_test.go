package testutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	envChanged = "GO_PRE_COMMIT_TESTUTIL_CHANGED"
	envRemoved = "GO_PRE_COMMIT_TESTUTIL_REMOVED"
	envEmpty   = "GO_PRE_COMMIT_TESTUTIL_EMPTY"
	envAdded   = "GO_PRE_COMMIT_TESTUTIL_ADDED"
)

func TestIsolateEnv_RestoresEnvironment(t *testing.T) {
	t.Setenv(envChanged, "original")
	t.Setenv(envRemoved, "keep-me")
	t.Setenv(envEmpty, "")
	require.NoError(t, os.Unsetenv(envAdded))

	t.Run("test that mutates the environment", func(t *testing.T) {
		IsolateEnv(t)

		require.NoError(t, os.Setenv(envChanged, "changed"))
		require.NoError(t, os.Unsetenv(envRemoved))
		require.NoError(t, os.Unsetenv(envEmpty))
		require.NoError(t, os.Setenv(envAdded, "added"))
	})

	assert.Equal(t, "original", os.Getenv(envChanged))
	assert.Equal(t, "keep-me", os.Getenv(envRemoved))

	value, ok := os.LookupEnv(envEmpty)
	assert.True(t, ok, "a variable set to an empty value must be restored, not left unset")
	assert.Empty(t, value)

	_, ok = os.LookupEnv(envAdded)
	assert.False(t, ok, "a variable added by the test must be unset")
}

func TestEnvironMap(t *testing.T) {
	env := environMap([]string{"A=1", "B=x=y", "EMPTY=", "=C:=C:\\", "NOVALUE"})

	assert.Equal(t, map[string]string{"A": "1", "B": "x=y", "EMPTY": "", "NOVALUE": ""}, env)
}

func TestClearConfigEnv(t *testing.T) {
	t.Setenv("ENABLE_GO_PRE_COMMIT", "true")
	t.Setenv("GO_PRE_COMMIT_TIMEOUT_SECONDS", "720")
	t.Setenv("GO_PRE_COMMIT_FUMPT_VERSION_LATEST", "v0.12.0")
	t.Setenv("MAGE_X_GOFUMPT_VERSION", "v0.11.0")

	require.NoError(t, ClearConfigEnv())

	for _, key := range []string{"ENABLE_GO_PRE_COMMIT", "GO_PRE_COMMIT_TIMEOUT_SECONDS", "GO_PRE_COMMIT_FUMPT_VERSION_LATEST"} {
		_, ok := os.LookupEnv(key)
		assert.False(t, ok, "%s should be cleared", key)
	}
	assert.Equal(t, "v0.11.0", os.Getenv("MAGE_X_GOFUMPT_VERSION"), "unrelated variables must be kept")
}

func TestIsConfigEnvVar(t *testing.T) {
	for _, key := range []string{"ENABLE_GO_PRE_COMMIT", "GO_PRE_COMMIT_LOG_LEVEL", "GO_PRE_COMMIT_FUMPT_VERSION_LATEST_MIN_GO"} {
		assert.True(t, IsConfigEnvVar(key), key)
	}
	for _, key := range []string{"ENABLE_GO_PRE_COMMIT_EXTRA", "GO_PRE_COMMIT", "MAGE_X_GOFUMPT_VERSION", "CI", "PATH"} {
		assert.False(t, IsConfigEnvVar(key), key)
	}
}
