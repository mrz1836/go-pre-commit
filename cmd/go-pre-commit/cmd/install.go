// Package cmd implements the CLI commands for go-pre-commit
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-pre-commit/internal/config"
	"github.com/mrz1836/go-pre-commit/internal/git"
)

// InstallConfig holds configuration for the install command
type InstallConfig struct {
	Force     bool
	HookTypes []string
}

// BuildInstallCmd creates the install command
func (cb *CommandBuilder) BuildInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install git pre-commit hooks",
		Long: `Install the Go pre-commit system hooks into your git repository.

This command will:
  - Create .git/hooks/pre-commit (or other specified hook types)
  - Make the hook executable
  - Preserve any existing hooks (unless --force is used)
  - Configure the hook to use the Go pre-commit system`,
		Example: `  # Install pre-commit hook
  go-pre-commit install

  # Force install, overwriting existing hooks
  go-pre-commit install --force

  # Install multiple hook types
  go-pre-commit install --hook-type pre-commit --hook-type pre-push`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flags and create config
			config := InstallConfig{}
			var err error

			config.Force, err = cmd.Flags().GetBool("force")
			if err != nil {
				return err
			}

			config.HookTypes, err = cmd.Flags().GetStringSlice("hook-type")
			if err != nil {
				return err
			}

			return cb.runInstallWithConfig(config, cmd, args)
		},
	}

	// Add flags
	cmd.Flags().BoolP("force", "f", false, "Force installation, overwriting existing hooks")
	cmd.Flags().StringSlice("hook-type", []string{"pre-commit"}, "Hook types to install")

	return cmd
}

func (cb *CommandBuilder) runInstallWithConfig(installConfig InstallConfig, _ *cobra.Command, _ []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Check if pre-commit system is enabled
	if !cfg.Enabled {
		printWarning("Pre-commit system is disabled in configuration (ENABLE_GO_PRE_COMMIT=false)")
		printInfo("To enable, set ENABLE_GO_PRE_COMMIT=true in .github/.env.shared")
		return nil
	}

	// Get the repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	if cb.app.config.Verbose {
		printInfo("Repository root: %s", repoRoot)
	}

	// Create installer with configuration for enhanced validation
	installer := git.NewInstallerWithConfig(repoRoot, "", cfg)

	// Install each hook type
	installed := make([]string, 0, len(installConfig.HookTypes))
	for _, hookType := range installConfig.HookTypes {
		if cb.app.config.Verbose {
			printInfo("Installing %s hook...", hookType)
		}

		err := installer.InstallHook(hookType, installConfig.Force)
		if err != nil {
			if !installConfig.Force && os.IsExist(err) {
				printWarning("Hook already exists: %s (use --force to overwrite)", hookType)
				continue
			}
			return fmt.Errorf("failed to install %s hook: %w", hookType, err)
		}

		installed = append(installed, hookType)
	}

	// Summary
	if len(installed) > 0 {
		printSuccess("Successfully installed hooks: %v", installed)
		printInfo("Run 'git commit' to test the pre-commit hook")
		printInfo("Run '%s run --help' to see available checks", filepath.Base(os.Args[0]))
	} else {
		printWarning("No hooks were installed")
	}

	return nil
}
