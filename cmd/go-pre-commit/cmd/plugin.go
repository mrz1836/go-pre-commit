package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/mrz1836/go-pre-commit/internal/config"
	"github.com/mrz1836/go-pre-commit/internal/plugins"
)

// Define plugin command errors
var (
	ErrPluginSourceRequired = errors.New("plugin source required")
	ErrPluginNameRequired   = errors.New("plugin name required")
	ErrInvalidPlugin        = errors.New("invalid plugin")
	ErrDirectoryOnly        = errors.New("source must be a directory (URL support coming soon)")
	ErrPluginAlreadyExists  = errors.New("plugin already exists")
	ErrPluginNotFound       = errors.New("plugin not found")
	ErrNoManifestFile       = errors.New("no manifest file found")
	ErrValidationFailed     = errors.New("validation failed")
)

// BuildPluginCmd creates the plugin management command
func (cb *CommandBuilder) BuildPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage pre-commit plugins",
		Long: `Manage custom pre-commit plugins.

Plugins allow you to extend go-pre-commit with custom checks
implemented in any language.`,
	}

	// Add subcommands
	cmd.AddCommand(cb.buildPluginListCmd())
	cmd.AddCommand(cb.buildPluginValidateCmd())
	cmd.AddCommand(cb.buildPluginAddCmd())
	cmd.AddCommand(cb.buildPluginRemoveCmd())
	cmd.AddCommand(cb.buildPluginInfoCmd())

	return cmd
}

// buildPluginListCmd creates the plugin list command
func (cb *CommandBuilder) buildPluginListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available plugins",
		Long:  `List all available plugins in the plugin directory.`,
		Example: `  # List all plugins
  go-pre-commit plugin list

  # List plugins with verbose output
  go-pre-commit plugin list -v`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Check if plugins are enabled
			if !cfg.Plugins.Enabled {
				color.Yellow("⚠️  Plugins are disabled. Enable with GO_PRE_COMMIT_ENABLE_PLUGINS=true")
				return nil
			}

			// Create plugin registry
			pluginDir := cfg.Plugins.Directory
			if pluginDir == "" {
				pluginDir = ".pre-commit-plugins"
			}

			registry := plugins.NewRegistry(pluginDir)
			if err := registry.LoadPlugins(); err != nil {
				return fmt.Errorf("failed to load plugins: %w", err)
			}

			allPlugins := registry.GetAll()
			if len(allPlugins) == 0 {
				color.Yellow("No plugins found in %s", pluginDir)
				return nil
			}

			// Display plugins
			if verbose {
				// Detailed view
				for i, plugin := range allPlugins {
					if i > 0 {
						_, _ = color.New().Println()
					}
					displayPluginDetails(plugin)
				}
			} else {
				// Table view
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintln(w, "NAME\tVERSION\tCATEGORY\tDESCRIPTION")
				_, _ = fmt.Fprintln(w, "----\t-------\t--------\t-----------")

				for _, plugin := range allPlugins {
					metadata := plugin.Metadata().(plugins.PluginMetadata)
					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
						plugin.Name(),
						metadata.Version,
						metadata.Category,
						plugin.Description())
				}
				_ = w.Flush()
			}

			return nil
		},
	}
}

// buildPluginValidateCmd creates the plugin validate command
func (cb *CommandBuilder) buildPluginValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [plugin-dir]",
		Short: "Validate a plugin manifest",
		Long:  `Validate a plugin manifest file for correctness.`,
		Example: `  # Validate current directory
  go-pre-commit plugin validate

  # Validate specific plugin directory
  go-pre-commit plugin validate ./my-plugin`,
		RunE: func(_ *cobra.Command, args []string) error {
			pluginDir := "."
			if len(args) > 0 {
				pluginDir = args[0]
			}

			// Look for manifest file
			manifest, err := loadManifestFromDir(pluginDir)
			if err != nil {
				return err
			}

			// Validate manifest
			errors := plugins.ValidateManifest(manifest)
			if len(errors) > 0 {
				color.Red("❌ Plugin validation failed:")
				for _, err := range errors {
					color.Red("  • %s", err)
				}
				return fmt.Errorf("%w: %d error(s)", ErrValidationFailed, len(errors))
			}

			color.Green("✅ Plugin manifest is valid")
			color.Cyan("\nPlugin: %s v%s", manifest.Name, manifest.Version)
			_, _ = color.New().Printf("Description: %s\n", manifest.Description)

			return nil
		},
	}
}

// buildPluginAddCmd creates the plugin add command
func (cb *CommandBuilder) buildPluginAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <source>",
		Short: "Add a plugin from a directory or URL",
		Long: `Add a plugin to your project.

The source can be:
- A local directory path
- A GitHub repository URL (coming soon)
- A plugin archive URL (coming soon)`,
		Example: `  # Add from local directory
  go-pre-commit plugin add ./my-custom-plugin

  # Add from examples
  go-pre-commit plugin add examples/shell-plugin`,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return ErrPluginSourceRequired
			}

			source := args[0]

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			pluginDir := cfg.Plugins.Directory
			if pluginDir == "" {
				pluginDir = ".pre-commit-plugins"
			}

			// Ensure plugin directory exists
			if mkErr := os.MkdirAll(pluginDir, 0o750); mkErr != nil {
				return fmt.Errorf("failed to create plugin directory: %w", mkErr)
			}

			// For now, only support local directories
			if !isDirectory(source) {
				return ErrDirectoryOnly
			}

			// Load and validate manifest
			manifest, err := loadManifestFromDir(source)
			if err != nil {
				return err
			}

			// Validate manifest
			validationErrors := plugins.ValidateManifest(manifest)
			if len(validationErrors) > 0 {
				color.Red("❌ Plugin validation failed:")
				for _, err := range validationErrors {
					color.Red("  • %s", err)
				}
				return ErrInvalidPlugin
			}

			// Copy plugin to plugin directory
			targetDir := filepath.Join(pluginDir, manifest.Name)
			if _, err := os.Stat(targetDir); err == nil {
				return fmt.Errorf("%w: %s", ErrPluginAlreadyExists, manifest.Name)
			}

			_, _ = color.New().Printf("Installing plugin '%s' from %s...\n", manifest.Name, source)

			// Copy directory
			if err := copyDir(source, targetDir); err != nil {
				return fmt.Errorf("failed to install plugin: %w", err)
			}

			color.Green("✅ Plugin '%s' installed successfully", manifest.Name)
			color.Yellow("\nTo use this plugin, ensure GO_PRE_COMMIT_ENABLE_PLUGINS=true")

			return nil
		},
	}
}

// buildPluginRemoveCmd creates the plugin remove command
func (cb *CommandBuilder) buildPluginRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <plugin-name>",
		Short: "Remove an installed plugin",
		Long:  `Remove an installed plugin from the plugin directory.`,
		Example: `  # Remove a plugin
  go-pre-commit plugin remove todo-checker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return ErrPluginNameRequired
			}

			pluginName := args[0]

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			pluginDir := cfg.Plugins.Directory
			if pluginDir == "" {
				pluginDir = ".pre-commit-plugins"
			}

			targetDir := filepath.Join(pluginDir, pluginName)
			if _, err := os.Stat(targetDir); os.IsNotExist(err) {
				return fmt.Errorf("%w: %s", ErrPluginNotFound, pluginName)
			}

			// Confirm removal
			force, _ := cmd.Flags().GetBool("force")
			if !force {
				_, _ = color.New().Printf("Remove plugin '%s'? [y/N]: ", pluginName)
				var response string
				_, _ = fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					color.Yellow("Canceled")
					return nil
				}
			}

			// Remove plugin directory
			if err := os.RemoveAll(targetDir); err != nil {
				return fmt.Errorf("failed to remove plugin: %w", err)
			}

			color.Green("✅ Plugin '%s' removed successfully", pluginName)
			return nil
		},
	}
}

// buildPluginInfoCmd creates the plugin info command
func (cb *CommandBuilder) buildPluginInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <plugin-name>",
		Short: "Show detailed information about a plugin",
		Long:  `Display detailed information about an installed plugin.`,
		Example: `  # Show plugin info
  go-pre-commit plugin info todo-checker`,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return ErrPluginNameRequired
			}

			pluginName := args[0]

			// Load configuration
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Create plugin registry
			pluginDir := cfg.Plugins.Directory
			if pluginDir == "" {
				pluginDir = ".pre-commit-plugins"
			}

			registry := plugins.NewRegistry(pluginDir)
			if err := registry.LoadPlugins(); err != nil {
				return fmt.Errorf("failed to load plugins: %w", err)
			}

			plugin, found := registry.Get(pluginName)
			if !found {
				return fmt.Errorf("%w: %s", ErrPluginNotFound, pluginName)
			}

			displayPluginDetails(plugin)
			return nil
		},
	}
}

// Helper functions

func loadManifestFromDir(dir string) (*plugins.PluginManifest, error) {
	// Try YAML first
	yamlPath := filepath.Join(dir, "plugin.yaml")
	// #nosec G304 - Path is safely constructed with known filename
	if data, err := os.ReadFile(yamlPath); err == nil {
		var manifest plugins.PluginManifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse plugin.yaml: %w", err)
		}
		return &manifest, nil
	}

	// Try alternative YAML extension
	ymlPath := filepath.Join(dir, "plugin.yml")
	// #nosec G304 - Path is safely constructed with known filename
	if data, err := os.ReadFile(ymlPath); err == nil {
		var manifest plugins.PluginManifest
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse plugin.yml: %w", err)
		}
		return &manifest, nil
	}

	// Try JSON
	jsonPath := filepath.Join(dir, "plugin.json")
	// #nosec G304 - Path is safely constructed with known filename
	if data, err := os.ReadFile(jsonPath); err == nil {
		var manifest plugins.PluginManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse plugin.json: %w", err)
		}
		return &manifest, nil
	}

	return nil, fmt.Errorf("%w in %s (looked for plugin.yaml, plugin.yml, plugin.json)", ErrNoManifestFile, dir)
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func copyDir(src, dst string) error {
	// Create destination directory
	if err := os.MkdirAll(dst, 0o750); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	// #nosec G304 - Source path comes from plugin installation process
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Get source file info
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, info.Mode())
}

func displayPluginDetails(plugin *plugins.Plugin) {
	metadata := plugin.Metadata().(plugins.PluginMetadata)

	color.Cyan("Plugin: %s", plugin.Name())
	_, _ = color.New().Printf("Version: %s\n", metadata.Version)
	if metadata.Author != "" {
		_, _ = color.New().Printf("Author: %s\n", metadata.Author)
	}
	_, _ = color.New().Printf("Category: %s\n", metadata.Category)
	_, _ = color.New().Printf("Description: %s\n", plugin.Description())

	if len(metadata.FilePatterns) > 0 {
		_, _ = color.New().Printf("File Patterns: %v\n", metadata.FilePatterns)
	}

	_, _ = color.New().Printf("Timeout: %v\n", metadata.DefaultTimeout)
	_, _ = color.New().Printf("Requires Files: %v\n", metadata.RequiresFiles)

	if len(metadata.Dependencies) > 0 {
		_, _ = color.New().Printf("Dependencies: %v\n", metadata.Dependencies)
	}
}
