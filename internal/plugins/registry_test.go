package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry("/tmp/plugins")
	assert.NotNil(t, registry)
	assert.Equal(t, "/tmp/plugins", registry.directory)
	assert.NotNil(t, registry.plugins)
	assert.Empty(t, registry.plugins)
}

func TestRegistryLoadPlugins(t *testing.T) {
	// Create temporary plugin directory
	tmpDir := t.TempDir()

	// Test with non-existent directory (should not error)
	registry := NewRegistry(filepath.Join(tmpDir, "nonexistent"))
	err := registry.LoadPlugins()
	require.NoError(t, err)
	assert.Empty(t, registry.GetAll())

	// Create plugin directory with valid plugin
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	err = os.MkdirAll(pluginDir, 0o750)
	require.NoError(t, err)

	// Create plugin manifest
	manifest := &PluginManifest{
		Name:         "test-plugin",
		Version:      "1.0.0",
		Description:  "Test plugin",
		Executable:   "./test.sh",
		FilePatterns: []string{"*.go"},
		Timeout:      "30s",
		Category:     "testing",
	}

	manifestData, err := yaml.Marshal(manifest)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), manifestData, 0o600)
	require.NoError(t, err)

	// Create executable
	scriptContent := `#!/bin/bash
echo '{"success": true}'
`
	err = os.WriteFile(filepath.Join(pluginDir, "test.sh"), []byte(scriptContent), 0o600)
	require.NoError(t, err)

	// Load plugins
	registry = NewRegistry(tmpDir)
	err = registry.LoadPlugins()
	require.NoError(t, err)

	// Verify plugin was loaded
	plugins := registry.GetAll()
	assert.Len(t, plugins, 1)
	assert.Equal(t, "test-plugin", plugins[0].Name())

	// Test Get
	plugin, found := registry.Get("test-plugin")
	assert.True(t, found)
	assert.NotNil(t, plugin)
	assert.Equal(t, "test-plugin", plugin.Name())

	// Test Names
	names := registry.Names()
	assert.Equal(t, []string{"test-plugin"}, names)
}

func TestRegistryLoadPluginsWithErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create plugin directory with invalid manifest
	pluginDir := filepath.Join(tmpDir, "bad-plugin")
	err := os.MkdirAll(pluginDir, 0o750)
	require.NoError(t, err)

	// Create invalid YAML
	err = os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte("invalid: yaml: content:"), 0o600)
	require.NoError(t, err)

	registry := NewRegistry(tmpDir)
	err = registry.LoadPlugins()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load some plugins")
}

func TestRegistryAddPlugin(t *testing.T) {
	registry := NewRegistry("/tmp")

	// Create a plugin
	manifest := &PluginManifest{
		Name:       "test-plugin",
		Executable: "test.sh",
	}
	plugin, err := NewPlugin(manifest, "/tmp")
	require.NoError(t, err)

	// Add plugin
	err = registry.AddPlugin(plugin)
	require.NoError(t, err)

	// Verify plugin was added
	retrieved, found := registry.Get("test-plugin")
	assert.True(t, found)
	assert.Equal(t, plugin, retrieved)

	// Try to add duplicate
	err = registry.AddPlugin(plugin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Try to add nil
	err = registry.AddPlugin(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

func TestRegistryRemovePlugin(t *testing.T) {
	registry := NewRegistry("/tmp")

	// Create and add a plugin
	manifest := &PluginManifest{
		Name:       "test-plugin",
		Executable: "test.sh",
	}
	plugin, err := NewPlugin(manifest, "/tmp")
	require.NoError(t, err)

	err = registry.AddPlugin(plugin)
	require.NoError(t, err)

	// Remove existing plugin
	removed := registry.RemovePlugin("test-plugin")
	assert.True(t, removed)

	// Verify it's gone
	_, found := registry.Get("test-plugin")
	assert.False(t, found)

	// Try to remove non-existent plugin
	removed = registry.RemovePlugin("nonexistent")
	assert.False(t, removed)
}

func TestLoadManifestFormats(t *testing.T) {
	tmpDir := t.TempDir()
	registry := NewRegistry(tmpDir)

	manifest := &PluginManifest{
		Name:         "test-plugin",
		Version:      "1.0.0",
		Description:  "Test plugin",
		Executable:   "./test.sh",
		FilePatterns: []string{"*.go"},
	}

	tests := []struct {
		name     string
		filename string
		marshal  func(interface{}) ([]byte, error)
	}{
		{
			name:     "YAML with .yaml extension",
			filename: "plugin.yaml",
			marshal:  func(v interface{}) ([]byte, error) { return yaml.Marshal(v) },
		},
		{
			name:     "YAML with .yml extension",
			filename: "plugin.yml",
			marshal:  func(v interface{}) ([]byte, error) { return yaml.Marshal(v) },
		},
		{
			name:     "JSON",
			filename: "plugin.json",
			marshal:  json.Marshal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create plugin directory
			pluginDir := filepath.Join(tmpDir, tt.name)
			err := os.MkdirAll(pluginDir, 0o750)
			require.NoError(t, err)

			// Write manifest in the specific format
			data, err := tt.marshal(manifest)
			require.NoError(t, err)

			err = os.WriteFile(filepath.Join(pluginDir, tt.filename), data, 0o600)
			require.NoError(t, err)

			// Load manifest
			loaded, err := registry.loadManifest(pluginDir)
			require.NoError(t, err)
			assert.Equal(t, manifest.Name, loaded.Name)
			assert.Equal(t, manifest.Version, loaded.Version)
			assert.Equal(t, manifest.Description, loaded.Description)
			assert.Equal(t, manifest.Executable, loaded.Executable)
			assert.Equal(t, manifest.FilePatterns, loaded.FilePatterns)
		})
	}

	// Test with no manifest file
	emptyDir := filepath.Join(tmpDir, "empty")
	err := os.MkdirAll(emptyDir, 0o750)
	require.NoError(t, err)

	_, err = registry.loadManifest(emptyDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no manifest file found")
}

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name     string
		manifest *PluginManifest
		wantErrs []string
	}{
		{
			name: "valid manifest",
			manifest: &PluginManifest{
				Name:         "test-plugin",
				Version:      "1.0.0",
				Description:  "Test plugin",
				Executable:   "./test.sh",
				FilePatterns: []string{"*.go"},
				Timeout:      "30s",
				Category:     "linting",
			},
			wantErrs: []string{},
		},
		{
			name:     "missing required fields",
			manifest: &PluginManifest{},
			wantErrs: []string{
				"plugin name is required",
				"plugin version is required",
				"plugin description is required",
				"plugin executable is required",
				"at least one file pattern is required",
			},
		},
		{
			name: "invalid timeout",
			manifest: &PluginManifest{
				Name:         "test",
				Version:      "1.0.0",
				Description:  "Test",
				Executable:   "./test",
				FilePatterns: []string{"*.go"},
				Timeout:      "invalid",
			},
			wantErrs: []string{
				"invalid timeout format",
			},
		},
		{
			name: "invalid category",
			manifest: &PluginManifest{
				Name:         "test",
				Version:      "1.0.0",
				Description:  "Test",
				Executable:   "./test",
				FilePatterns: []string{"*.go"},
				Category:     "invalid-category",
			},
			wantErrs: []string{
				"invalid category",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateManifest(tt.manifest)

			if len(tt.wantErrs) == 0 {
				assert.Empty(t, errs)
			} else {
				// Check we have at least the expected errors
				for _, wantErr := range tt.wantErrs {
					found := false
					for _, err := range errs {
						if strings.Contains(err, wantErr) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error containing %q not found in %v", wantErr, errs)
				}
			}
		})
	}
}
