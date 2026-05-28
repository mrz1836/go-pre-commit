package envfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDir_ErrorPaths(t *testing.T) {
	t.Run("path is not a directory", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "afile")
		require.NoError(t, os.WriteFile(f, []byte("X=1\n"), 0o600))

		err := LoadDir(f, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotDirectory)
	})

	t.Run("missing directory", func(t *testing.T) {
		err := LoadDir(filepath.Join(t.TempDir(), "nope"), false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "env directory not found")
	})

	t.Run("directory with no env files", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("x"), 0o600))

		err := LoadDir(dir, false)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoEnvFiles)
	})

	t.Run("skipLocal skips 99-local.env", func(t *testing.T) {
		dir := t.TempDir()
		// Only a local file exists; skipping it yields no loaded files.
		require.NoError(t, os.WriteFile(filepath.Join(dir, "99-local.env"),
			[]byte("GO_PRE_COMMIT_ROBUSTNESS_LOCAL=1\n"), 0o600))

		err := LoadDir(dir, true)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoEnvFiles)
	})
}

func TestLoad_LineEndingsAndComments(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	// CRLF line endings, a comment, and a quoted value with surrounding spaces.
	content := "# a comment\r\nGO_PRE_COMMIT_RB_CRLF=value1\r\nGO_PRE_COMMIT_RB_QUOTED=\"spaced value\"\r\n"
	require.NoError(t, os.WriteFile(envFile, []byte(content), 0o600))

	t.Setenv("GO_PRE_COMMIT_RB_CRLF", "")
	t.Setenv("GO_PRE_COMMIT_RB_QUOTED", "")
	require.NoError(t, Overload(envFile))

	// The carriage return must not leak into the parsed value.
	assert.Equal(t, "value1", os.Getenv("GO_PRE_COMMIT_RB_CRLF"))
	assert.Equal(t, "spaced value", os.Getenv("GO_PRE_COMMIT_RB_QUOTED"))
}

func TestLoad_MissingFile(t *testing.T) {
	err := Load(filepath.Join(t.TempDir(), "absent.env"))
	require.Error(t, err)
}
