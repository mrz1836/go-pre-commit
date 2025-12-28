package checks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/plugins"
)

func TestRegistryLoadPluginsNoRegistry(t *testing.T) {
	registry := &Registry{checks: make(map[string]Check)}

	require.NoError(t, registry.LoadPlugins())
	require.Empty(t, registry.GetChecks())
}

func TestRegistryLoadPluginsRegistersChecks(t *testing.T) {
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "example-plugin")

	require.NoError(t, os.MkdirAll(pluginDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`name: example-plugin
version: "1.0.0"
description: Example plugin
executable: plugin.sh
file_patterns:
  - "*.go"
`), 0o600))

	pluginRegistry := plugins.NewRegistry(tempDir)
	registry := &Registry{checks: make(map[string]Check), pluginRegistry: pluginRegistry}

	require.NoError(t, registry.LoadPlugins())

	loaded, ok := registry.Get("example-plugin")
	require.True(t, ok)
	require.IsType(t, &plugins.Plugin{}, loaded)

	metadata, ok := loaded.Metadata().(plugins.PluginMetadata)
	require.True(t, ok)
	require.Equal(t, "example-plugin", metadata.Name)
	require.Equal(t, "Example plugin", metadata.Description)
}

func TestRegistryLoadPluginsPropagatesErrors(t *testing.T) {
	tempDir := t.TempDir()
	pluginDir := filepath.Join(tempDir, "broken-plugin")

	require.NoError(t, os.MkdirAll(pluginDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`name: broken-plugin
version: "1.0.0"
description: Broken plugin
file_patterns:
  - "*.go"
`), 0o600))

	pluginRegistry := plugins.NewRegistry(tempDir)
	registry := &Registry{checks: make(map[string]Check), pluginRegistry: pluginRegistry}

	err := registry.LoadPlugins()
	require.Error(t, err)
	require.ErrorIs(t, err, plugins.ErrPluginsLoadFailed)
}
