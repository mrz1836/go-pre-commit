package cmd

import (
	"fmt"

	"github.com/mrz1836/go-pre-commit/internal/git"
	"github.com/spf13/cobra"
)

// BuildUninstallCmd creates the uninstall command
func (cb *CommandBuilder) BuildUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall git pre-commit hooks",
		Long: `Uninstall the Go pre-commit system hooks from your git repository.

This command will:
  - Remove .git/hooks/pre-commit (or other specified hook types)
  - Only remove hooks that were installed by Go pre-commit system
  - Preserve any hooks not created by this tool`,
		Example: `  # Uninstall pre-commit hook
  go-pre-commit uninstall

  # Uninstall multiple hook types
  go-pre-commit uninstall --hook-type pre-commit --hook-type pre-push`,
		RunE: func(cmd *cobra.Command, args []string) error {
			hookTypes, err := cmd.Flags().GetStringSlice("hook-type")
			if err != nil {
				return err
			}
			return cb.runUninstallWithHooks(hookTypes, cmd, args)
		},
	}

	cmd.Flags().StringSlice("hook-type", []string{"pre-commit"}, "Hook types to uninstall")
	return cmd
}

func (cb *CommandBuilder) runUninstallWithHooks(hookTypes []string, _ *cobra.Command, _ []string) error {
	// Get the repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	if cb.app.config.Verbose {
		printInfo("Repository root: %s", repoRoot)
	}

	// Create installer (also handles uninstallation)
	installer := git.NewInstaller(repoRoot, "")

	// Uninstall each hook type
	var uninstalled []string
	var notFound []string

	for _, hookType := range hookTypes {
		if cb.app.config.Verbose {
			printInfo("Uninstalling %s hook...", hookType)
		}

		removed, err := installer.UninstallHook(hookType)
		if err != nil {
			return fmt.Errorf("failed to uninstall %s hook: %w", hookType, err)
		}

		if removed {
			uninstalled = append(uninstalled, hookType)
		} else {
			notFound = append(notFound, hookType)
		}
	}

	// Summary
	if len(uninstalled) > 0 {
		printSuccess("Successfully uninstalled hooks: %v", uninstalled)
	}
	if len(notFound) > 0 {
		printInfo("Hooks not found or not managed by Go pre-commit: %v", notFound)
	}
	if len(uninstalled) == 0 && len(notFound) == 0 {
		printWarning("No hooks were uninstalled")
	}

	return nil
}
