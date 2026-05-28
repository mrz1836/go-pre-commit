package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureCmdOutput redirects stdout and the color package's output/error streams
// to an in-memory buffer for the duration of fn, returning everything written.
// Color is disabled so assertions can match plain text.
func captureCmdOutput(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	oldColorOut := color.Output
	oldNoColor := color.NoColor

	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	os.Stderr = w
	color.Output = w
	color.NoColor = true

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		color.Output = oldColorOut
		color.NoColor = oldNoColor
	}()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

// setupPluginTestEnv configures an isolated configuration directory (so
// config.Load succeeds) and an empty plugin directory, with plugins enabled.
// It returns the plugin directory. All environment changes are auto-restored.
func setupPluginTestEnv(t *testing.T) (pluginDir string) {
	t.Helper()

	configDir := t.TempDir()
	ghDir := filepath.Join(configDir, ".github")
	require.NoError(t, os.MkdirAll(ghDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(ghDir, ".env.base"), []byte(""), 0o600))
	t.Setenv("GO_PRE_COMMIT_TEST_CONFIG_DIR", configDir)

	pluginDir = t.TempDir()
	t.Setenv("GO_PRE_COMMIT_ENABLE_PLUGINS", "true")
	t.Setenv("GO_PRE_COMMIT_PLUGIN_DIR", pluginDir)
	return pluginDir
}

// validManifestYAML returns a minimal, schema-valid plugin manifest body.
func validManifestYAML(name string) string {
	return `name: ` + name + `
version: "1.2.3"
description: "Test plugin ` + name + `"
author: "Tester"
category: "testing"
executable: "./check.sh"
file_patterns:
  - "*.go"
timeout: "30s"
requires_files: true
dependencies:
  - "bash"
`
}

// createPluginManifestDir writes a manifest into parent/<name>/plugin.yaml (the
// layout the registry discovers).
func createPluginManifestDir(t *testing.T, parent, name string) {
	t.Helper()
	dir := filepath.Join(parent, name)
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(validManifestYAML(name)), 0o600))
}

// createSourcePluginDir writes a manifest into a fresh source dir (the layout
// `plugin add` expects) and returns it.
func createSourcePluginDir(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(body), 0o600))
	return dir
}

// runPluginSubcmd locates a plugin subcommand, ensures the verbose/force flags
// exist, applies the given flag overrides, then invokes RunE while capturing
// output.
func runPluginSubcmd(t *testing.T, name string, flags map[string]string, args ...string) (string, error) {
	t.Helper()

	app := NewCLIApp("1.0.0", "test", "2025-01-01")
	builder := NewCommandBuilder(app)
	pluginCmd := builder.BuildPluginCmd()
	sub := findCommand(t, pluginCmd, name)
	require.NotNil(t, sub)

	if sub.Flags().Lookup("verbose") == nil {
		sub.Flags().Bool("verbose", false, "")
	}
	if sub.Flags().Lookup("force") == nil {
		sub.Flags().Bool("force", false, "")
	}
	for k, v := range flags {
		require.NoError(t, sub.Flags().Set(k, v))
	}

	var err error
	out := captureCmdOutput(t, func() {
		err = sub.RunE(sub, args)
	})
	return out, err
}

// setPromptInput overrides the interactive confirmation reader for the duration
// of a test.
func setPromptInput(t *testing.T, input string) {
	t.Helper()
	orig := promptReader
	t.Cleanup(func() { promptReader = orig })
	promptReader = strings.NewReader(input)
}

func TestPluginListCmd_RunE(t *testing.T) {
	t.Run("plugins disabled", func(t *testing.T) {
		setupPluginTestEnv(t)
		t.Setenv("GO_PRE_COMMIT_ENABLE_PLUGINS", "false")

		out, err := runPluginSubcmd(t, "list", nil)
		require.NoError(t, err)
		assert.Contains(t, out, "Plugins are disabled")
	})

	t.Run("empty plugin directory", func(t *testing.T) {
		setupPluginTestEnv(t)

		out, err := runPluginSubcmd(t, "list", nil)
		require.NoError(t, err)
		assert.Contains(t, out, "No plugins found")
	})

	t.Run("plugin directory missing", func(t *testing.T) {
		setupPluginTestEnv(t)
		t.Setenv("GO_PRE_COMMIT_PLUGIN_DIR", filepath.Join(t.TempDir(), "does-not-exist"))

		out, err := runPluginSubcmd(t, "list", nil)
		require.NoError(t, err)
		assert.Contains(t, out, "No plugins found")
	})

	t.Run("single plugin table view", func(t *testing.T) {
		dir := setupPluginTestEnv(t)
		createPluginManifestDir(t, dir, "alpha")

		out, err := runPluginSubcmd(t, "list", nil)
		require.NoError(t, err)
		assert.Contains(t, out, "NAME")
		assert.Contains(t, out, "alpha")
		assert.Contains(t, out, "1.2.3")
	})

	t.Run("multiple plugins table view", func(t *testing.T) {
		dir := setupPluginTestEnv(t)
		createPluginManifestDir(t, dir, "alpha")
		createPluginManifestDir(t, dir, "beta")

		out, err := runPluginSubcmd(t, "list", nil)
		require.NoError(t, err)
		assert.Contains(t, out, "alpha")
		assert.Contains(t, out, "beta")
	})

	t.Run("verbose view", func(t *testing.T) {
		dir := setupPluginTestEnv(t)
		createPluginManifestDir(t, dir, "alpha")

		out, err := runPluginSubcmd(t, "list", map[string]string{"verbose": "true"})
		require.NoError(t, err)
		assert.Contains(t, out, "Plugin: alpha")
		assert.Contains(t, out, "Version: 1.2.3")
	})

	t.Run("config load failure", func(t *testing.T) {
		// Point the config override at an empty dir lacking .github/.env.base.
		t.Setenv("GO_PRE_COMMIT_TEST_CONFIG_DIR", t.TempDir())

		_, err := runPluginSubcmd(t, "list", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	t.Run("malformed manifest aborts load", func(t *testing.T) {
		dir := setupPluginTestEnv(t)
		bad := filepath.Join(dir, "broken")
		require.NoError(t, os.MkdirAll(bad, 0o750))
		require.NoError(t, os.WriteFile(filepath.Join(bad, "plugin.yaml"), []byte("name: [unterminated"), 0o600))

		_, err := runPluginSubcmd(t, "list", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load plugins")
	})
}

func TestPluginValidateCmd_RunE(t *testing.T) {
	t.Run("valid manifest with arg", func(t *testing.T) {
		dir := createSourcePluginDir(t, validManifestYAML("alpha"))

		out, err := runPluginSubcmd(t, "validate", nil, dir)
		require.NoError(t, err)
		assert.Contains(t, out, "valid")
		assert.Contains(t, out, "alpha")
	})

	t.Run("valid manifest in current directory", func(t *testing.T) {
		dir := createSourcePluginDir(t, validManifestYAML("alpha"))
		t.Chdir(dir)

		out, err := runPluginSubcmd(t, "validate", nil)
		require.NoError(t, err)
		assert.Contains(t, out, "valid")
	})

	t.Run("missing manifest", func(t *testing.T) {
		_, err := runPluginSubcmd(t, "validate", nil, t.TempDir())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no manifest file found")
	})

	t.Run("validation failure missing fields", func(t *testing.T) {
		dir := createSourcePluginDir(t, "name: x\n")

		out, err := runPluginSubcmd(t, "validate", nil, dir)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrValidationFailed)
		assert.Contains(t, out, "validation failed")
	})

	t.Run("invalid category", func(t *testing.T) {
		body := `name: x
version: "1.0.0"
description: "desc"
executable: "./c.sh"
file_patterns: ["*.go"]
category: "not-a-category"
`
		dir := createSourcePluginDir(t, body)

		_, err := runPluginSubcmd(t, "validate", nil, dir)
		require.ErrorIs(t, err, ErrValidationFailed)
	})

	t.Run("invalid timeout format", func(t *testing.T) {
		body := `name: x
version: "1.0.0"
description: "desc"
executable: "./c.sh"
file_patterns: ["*.go"]
timeout: "not-a-duration"
`
		dir := createSourcePluginDir(t, body)

		_, err := runPluginSubcmd(t, "validate", nil, dir)
		require.ErrorIs(t, err, ErrValidationFailed)
	})

	t.Run("malformed yaml", func(t *testing.T) {
		dir := createSourcePluginDir(t, "name: [unterminated")

		_, err := runPluginSubcmd(t, "validate", nil, dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})
}

func TestPluginAddCmd_RunE(t *testing.T) {
	t.Run("add valid local directory", func(t *testing.T) {
		pluginDir := setupPluginTestEnv(t)
		source := createSourcePluginDir(t, validManifestYAML("alpha"))

		out, err := runPluginSubcmd(t, "add", nil, source)
		require.NoError(t, err)
		assert.Contains(t, out, "installed successfully")
		assert.DirExists(t, filepath.Join(pluginDir, "alpha"))
		assert.FileExists(t, filepath.Join(pluginDir, "alpha", "plugin.yaml"))
	})

	t.Run("no source argument", func(t *testing.T) {
		setupPluginTestEnv(t)
		_, err := runPluginSubcmd(t, "add", nil)
		require.ErrorIs(t, err, ErrPluginSourceRequired)
	})

	t.Run("source is a file", func(t *testing.T) {
		setupPluginTestEnv(t)
		f := filepath.Join(t.TempDir(), "not-a-dir.txt")
		require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))

		_, err := runPluginSubcmd(t, "add", nil, f)
		require.ErrorIs(t, err, ErrDirectoryOnly)
	})

	t.Run("missing manifest in source", func(t *testing.T) {
		setupPluginTestEnv(t)
		_, err := runPluginSubcmd(t, "add", nil, t.TempDir())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no manifest file found")
	})

	t.Run("manifest validation failure", func(t *testing.T) {
		setupPluginTestEnv(t)
		source := createSourcePluginDir(t, "name: x\nversion: \"1.0.0\"\n")

		out, err := runPluginSubcmd(t, "add", nil, source)
		require.ErrorIs(t, err, ErrInvalidPlugin)
		assert.Contains(t, out, "validation failed")
	})

	t.Run("plugin already exists", func(t *testing.T) {
		pluginDir := setupPluginTestEnv(t)
		createPluginManifestDir(t, pluginDir, "alpha") // pre-existing install
		source := createSourcePluginDir(t, validManifestYAML("alpha"))

		_, err := runPluginSubcmd(t, "add", nil, source)
		require.ErrorIs(t, err, ErrPluginAlreadyExists)
	})

	t.Run("plugin directory creation failure", func(t *testing.T) {
		setupPluginTestEnv(t)
		// Point the plugin dir at an existing regular file so MkdirAll fails.
		blocker := filepath.Join(t.TempDir(), "blocker")
		require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
		t.Setenv("GO_PRE_COMMIT_PLUGIN_DIR", blocker)
		source := createSourcePluginDir(t, validManifestYAML("alpha"))

		_, err := runPluginSubcmd(t, "add", nil, source)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create plugin directory")
	})

	t.Run("config load failure", func(t *testing.T) {
		t.Setenv("GO_PRE_COMMIT_TEST_CONFIG_DIR", t.TempDir())
		source := createSourcePluginDir(t, validManifestYAML("alpha"))

		_, err := runPluginSubcmd(t, "add", nil, source)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})
}

func TestPluginRemoveCmd_RunE(t *testing.T) {
	t.Run("force removes plugin", func(t *testing.T) {
		pluginDir := setupPluginTestEnv(t)
		createPluginManifestDir(t, pluginDir, "alpha")

		out, err := runPluginSubcmd(t, "remove", map[string]string{"force": "true"}, "alpha")
		require.NoError(t, err)
		assert.Contains(t, out, "removed successfully")
		assert.NoDirExists(t, filepath.Join(pluginDir, "alpha"))
	})

	t.Run("confirmation yes removes plugin", func(t *testing.T) {
		pluginDir := setupPluginTestEnv(t)
		createPluginManifestDir(t, pluginDir, "alpha")
		setPromptInput(t, "y\n")

		out, err := runPluginSubcmd(t, "remove", nil, "alpha")
		require.NoError(t, err)
		assert.Contains(t, out, "removed successfully")
		assert.NoDirExists(t, filepath.Join(pluginDir, "alpha"))
	})

	t.Run("confirmation no keeps plugin", func(t *testing.T) {
		pluginDir := setupPluginTestEnv(t)
		createPluginManifestDir(t, pluginDir, "alpha")
		setPromptInput(t, "n\n")

		out, err := runPluginSubcmd(t, "remove", nil, "alpha")
		require.NoError(t, err)
		assert.Contains(t, out, "Canceled")
		assert.DirExists(t, filepath.Join(pluginDir, "alpha"))
	})

	t.Run("confirmation empty keeps plugin", func(t *testing.T) {
		pluginDir := setupPluginTestEnv(t)
		createPluginManifestDir(t, pluginDir, "alpha")
		setPromptInput(t, "\n")

		out, err := runPluginSubcmd(t, "remove", nil, "alpha")
		require.NoError(t, err)
		assert.Contains(t, out, "Canceled")
		assert.DirExists(t, filepath.Join(pluginDir, "alpha"))
	})

	t.Run("no name argument", func(t *testing.T) {
		setupPluginTestEnv(t)
		_, err := runPluginSubcmd(t, "remove", nil)
		require.ErrorIs(t, err, ErrPluginNameRequired)
	})

	t.Run("plugin not found", func(t *testing.T) {
		setupPluginTestEnv(t)
		_, err := runPluginSubcmd(t, "remove", map[string]string{"force": "true"}, "ghost")
		require.ErrorIs(t, err, ErrPluginNotFound)
	})

	t.Run("config load failure", func(t *testing.T) {
		t.Setenv("GO_PRE_COMMIT_TEST_CONFIG_DIR", t.TempDir())
		_, err := runPluginSubcmd(t, "remove", map[string]string{"force": "true"}, "alpha")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})

	t.Run("force flag is registered and parses end-to-end", func(t *testing.T) {
		pluginDir := setupPluginTestEnv(t)
		createPluginManifestDir(t, pluginDir, "alpha")

		app := NewCLIApp("1.0.0", "test", "2025-01-01")
		pluginCmd := NewCommandBuilder(app).BuildPluginCmd()
		removeCmd := findCommand(t, pluginCmd, "remove")
		require.NotNil(t, removeCmd.Flags().Lookup("force"))
		require.NotNil(t, removeCmd.Flags().ShorthandLookup("f"))

		// Drive the real cobra flag parsing via the parent command.
		pluginCmd.SetArgs([]string{"remove", "--force", "alpha"})
		out := captureCmdOutput(t, func() {
			require.NoError(t, pluginCmd.Execute())
		})
		assert.Contains(t, out, "removed successfully")
		assert.NoDirExists(t, filepath.Join(pluginDir, "alpha"))
	})
}

func TestPluginInfoCmd_RunE(t *testing.T) {
	t.Run("displays plugin details", func(t *testing.T) {
		pluginDir := setupPluginTestEnv(t)
		createPluginManifestDir(t, pluginDir, "alpha")

		out, err := runPluginSubcmd(t, "info", nil, "alpha")
		require.NoError(t, err)
		assert.Contains(t, out, "Plugin: alpha")
		assert.Contains(t, out, "Version: 1.2.3")
	})

	t.Run("no name argument", func(t *testing.T) {
		setupPluginTestEnv(t)
		_, err := runPluginSubcmd(t, "info", nil)
		require.ErrorIs(t, err, ErrPluginNameRequired)
	})

	t.Run("plugin not found", func(t *testing.T) {
		setupPluginTestEnv(t)
		_, err := runPluginSubcmd(t, "info", nil, "ghost")
		require.ErrorIs(t, err, ErrPluginNotFound)
	})

	t.Run("config load failure", func(t *testing.T) {
		t.Setenv("GO_PRE_COMMIT_TEST_CONFIG_DIR", t.TempDir())
		_, err := runPluginSubcmd(t, "info", nil, "alpha")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")
	})
}
