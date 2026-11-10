package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const envGitDir = "GIT_DIR"

func TestIsolateGit_RestoresEnvironment(t *testing.T) {
	home := os.Getenv("HOME")
	t.Setenv(envGitDir, "/enclosing/hook/.git")

	t.Run("test that runs git", func(t *testing.T) {
		IsolateGit(t)

		_, ok := os.LookupEnv(envGitDir)
		assert.False(t, ok, "an inherited GIT_* variable must be unset")
		assert.NotEqual(t, home, os.Getenv("HOME"), "HOME must not be the developer's")
		assert.DirExists(t, os.Getenv("HOME"))
		assert.Equal(t, os.DevNull, os.Getenv("GIT_CONFIG_GLOBAL"))
		assert.Equal(t, "1", os.Getenv("GIT_CONFIG_NOSYSTEM"))
	})

	assert.Equal(t, home, os.Getenv("HOME"))
	assert.Equal(t, "/enclosing/hook/.git", os.Getenv(envGitDir))
}

func TestNewGitRepo(t *testing.T) {
	root := NewGitRepo(t)

	assert.DirExists(t, filepath.Join(root, ".git"))

	output, err := runGit(t, root, "config", "--show-origin", "--get-regexp", ".")
	require.NoError(t, err, output)
	assert.NotContains(t, output, "global", "git must not read a global configuration")
	assert.NotContains(t, output, "user.", "git must not see the developer's identity")
}
