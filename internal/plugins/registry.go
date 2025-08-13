package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Define registry errors
var (
	ErrPluginAlreadyRegistered = errors.New("plugin already registered")
	ErrPluginNil               = errors.New("plugin cannot be nil")
	ErrNoManifestFound         = errors.New("no manifest file found (looked for plugin.yaml, plugin.yml, plugin.json)")
	ErrPluginsLoadFailed       = errors.New("failed to load some plugins")
)

// Registry manages plugin discovery and loading
type Registry struct {
	plugins   map[string]*Plugin
	directory string
	mu        sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry(directory string) *Registry {
	return &Registry{
		plugins:   make(map[string]*Plugin),
		directory: directory,
	}
}

// LoadPlugins discovers and loads all plugins from the registry directory
func (r *Registry) LoadPlugins() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if plugin directory exists
	if _, err := os.Stat(r.directory); os.IsNotExist(err) {
		// Plugin directory doesn't exist, which is fine (no plugins)
		return nil
	}

	// Walk through the plugin directory
	entries, err := os.ReadDir(r.directory)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	var loadErrors []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(r.directory, entry.Name())

		// Try to load plugin from this directory
		if err := r.loadPlugin(pluginDir); err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", entry.Name(), err))
		}
	}

	if len(loadErrors) > 0 {
		return fmt.Errorf("%w:\n%s", ErrPluginsLoadFailed, strings.Join(loadErrors, "\n"))
	}

	return nil
}

// loadPlugin loads a single plugin from a directory
func (r *Registry) loadPlugin(directory string) error {
	// Look for manifest file (plugin.yaml or plugin.json)
	manifest, err := r.loadManifest(directory)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Create plugin instance
	plugin, err := NewPlugin(manifest, directory)
	if err != nil {
		return fmt.Errorf("failed to create plugin: %w", err)
	}

	// Check for duplicate names
	if _, exists := r.plugins[plugin.Name()]; exists {
		return fmt.Errorf("%w: %s", ErrPluginAlreadyRegistered, plugin.Name())
	}

	// Register the plugin
	r.plugins[plugin.Name()] = plugin

	return nil
}

// loadManifest loads a plugin manifest from a directory
func (r *Registry) loadManifest(directory string) (*PluginManifest, error) {
	// Clean the directory path to prevent path traversal
	directory = filepath.Clean(directory)

	// Try YAML first
	yamlPath := filepath.Join(directory, "plugin.yaml")
	// #nosec G304 - Path is safely constructed with known filename
	if data, err := os.ReadFile(yamlPath); err == nil {
		var manifest PluginManifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse plugin.yaml: %w", err)
		}
		return &manifest, nil
	}

	// Try alternative YAML extension
	ymlPath := filepath.Join(directory, "plugin.yml")
	// #nosec G304 - Path is safely constructed with known filename
	if data, err := os.ReadFile(ymlPath); err == nil {
		var manifest PluginManifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse plugin.yml: %w", err)
		}
		return &manifest, nil
	}

	// Try JSON
	jsonPath := filepath.Join(directory, "plugin.json")
	// #nosec G304 - Path is safely constructed with known filename
	if data, err := os.ReadFile(jsonPath); err == nil {
		var manifest PluginManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse plugin.json: %w", err)
		}
		return &manifest, nil
	}

	return nil, ErrNoManifestFound
}

// Get returns a plugin by name
func (r *Registry) Get(name string) (*Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	plugin, ok := r.plugins[name]
	return plugin, ok
}

// GetAll returns all loaded plugins
func (r *Registry) GetAll() []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// Names returns the names of all loaded plugins
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

// AddPlugin manually adds a plugin to the registry
func (r *Registry) AddPlugin(plugin *Plugin) error {
	if plugin == nil {
		return ErrPluginNil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[plugin.Name()]; exists {
		return fmt.Errorf("%w: %s", ErrPluginAlreadyRegistered, plugin.Name())
	}

	r.plugins[plugin.Name()] = plugin
	return nil
}

// RemovePlugin removes a plugin from the registry
func (r *Registry) RemovePlugin(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; exists {
		delete(r.plugins, name)
		return true
	}
	return false
}

// ValidateManifest validates a plugin manifest without loading it
func ValidateManifest(manifest *PluginManifest) []string {
	var errors []string

	if manifest.Name == "" {
		errors = append(errors, "plugin name is required")
	}

	if manifest.Version == "" {
		errors = append(errors, "plugin version is required")
	}

	if manifest.Description == "" {
		errors = append(errors, "plugin description is required")
	}

	if manifest.Executable == "" {
		errors = append(errors, "plugin executable is required")
	}

	if len(manifest.FilePatterns) == 0 {
		errors = append(errors, "at least one file pattern is required")
	}

	// Validate timeout format if specified
	if manifest.Timeout != "" {
		if _, err := time.ParseDuration(manifest.Timeout); err != nil {
			errors = append(errors, fmt.Sprintf("invalid timeout format: %v", err))
		}
	}

	// Validate category if specified
	validCategories := []string{"formatting", "linting", "security", "testing", "documentation", "custom"}
	if manifest.Category != "" {
		valid := false
		for _, cat := range validCategories {
			if manifest.Category == cat {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("invalid category '%s', must be one of: %v", manifest.Category, validCategories))
		}
	}

	return errors
}
