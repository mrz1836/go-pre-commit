package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/mrz1836/go-pre-commit/internal/config"
	"github.com/mrz1836/go-pre-commit/internal/git"
	"github.com/mrz1836/go-pre-commit/internal/output"
)

// Hook states reported by `status --json`
const (
	hookStateReady         = "ready"          // our current hook, executable
	hookStateOutdated      = "outdated"       // our hook, written by another release; executable
	hookStateNotExecutable = "not_executable" // our hook, but git will not run it
	hookStateOther         = "other"          // a hook that another tool installed
	hookStateMissing       = "missing"        // no hook file
	hookStateError         = "error"          // the hook file could not be read
)

// supportedHooks are the hook types status reports, in order
func supportedHooks() []string {
	return []string{hookTypePreCommit, hookTypePrePush, "commit-msg", "post-commit"}
}

// BuildStatusCmd creates the status command
func (cb *CommandBuilder) BuildStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show installation status of git hooks",
		Long: `Show the current installation status of Go pre-commit hooks.

This command displays:
  - Which hooks are installed
  - Whether they are Go pre-commit hooks or other hooks
  - File permissions and last modified time
  - Configuration status
  - Any conflicts or issues

JSON output (--json):
  Prints one JSON document on stdout and nothing else:
    version          go-pre-commit version
    enabled          ENABLE_GO_PRE_COMMIT
    config           {"mode": "modular" (.github/env/) or "legacy" (.github/.env.base),
                      "path": absolute path of that directory or file}
    repository_root  absolute path of the repository
    hooks_dir        absolute path of the directory git runs hooks from
    hooks            one entry per hook type (pre-commit, pre-push, commit-msg,
                     post-commit): name, state, path, executable, outdated,
                     and error (only when state is "error")
  A hook's state is one of:
    ready           go-pre-commit's current hook, executable
    outdated        go-pre-commit's hook from another release (run install to update)
    not_executable  go-pre-commit's hook, but not executable, so git skips it
    other           a hook another tool installed
    missing         no hook file
    error           the hook file could not be read
  If the configuration or repository cannot be found, the document is
  {"version", "error": {"code", "message"}}, with code config_not_found,
  config_invalid, or not_a_repository, and the exit status is 1.`,
		Example: `  # Show status of all hook types
  go-pre-commit status

  # Show verbose status information
  go-pre-commit status --verbose

  # Read the pre-commit hook's state from a script
  go-pre-commit status --json | jq -r '.hooks[] | select(.name == "pre-commit") | .state'`,
		RunE: cb.runStatus,
	}

	cmd.Flags().Bool(jsonFlag, false, "Print the status as one JSON document on stdout")

	return cmd
}

func (cb *CommandBuilder) runStatus(cmd *cobra.Command, _ []string) error {
	jsonOutput, err := jsonOutputRequested(cmd)
	if err != nil {
		return err
	}
	if jsonOutput {
		return cb.runStatusJSON(stdoutOf(cmd))
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		printError("Failed to load configuration: %v", err)
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get the repository root
	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		printError("Failed to find git repository: %v", err)
		return fmt.Errorf("failed to find git repository: %w", err)
	}

	if cb.app.config.Verbose {
		printInfo("Repository root: %s", repoRoot)
		printInfo("Pre-commit directory: %s", cfg.Directory)
		printInfo("System enabled: %t", cfg.Enabled)
	}

	// Create installer for status checking
	installer := git.NewInstallerWithConfig(repoRoot, cfg.Directory, cfg)

	printHeader("Go Pre-commit System Status")

	// System-level status
	if cfg.Enabled {
		printSuccess("✓ Pre-commit system is enabled")
	} else {
		printWarning("⚠ Pre-commit system is disabled (ENABLE_GO_PRE_COMMIT=false)")
	}

	// Hook status
	printSubheader("Git Hook Status")

	foundHooks := false
	for _, hookType := range supportedHooks() {
		status, err := installer.GetInstallationStatus(hookType)
		if err != nil {
			printError("Failed to check %s hook status: %v", hookType, err)
			continue
		}

		if !status.Installed && !status.ConflictingHook {
			if cb.app.config.Verbose {
				printDetail("  %s: Not installed", hookType)
			}
			continue
		}

		foundHooks = true

		if status.IsOurHook {
			if status.Executable && !status.Outdated {
				printSuccess("  ✓ %s: %s", hookType, status.Message)
			} else {
				printWarning("  ⚠ %s: %s", hookType, status.Message)
			}
		} else if status.ConflictingHook {
			printWarning("  ⚠ %s: %s", hookType, status.Message)
			if cb.app.config.Verbose {
				printDetail("    Use --force to overwrite existing hook")
			}
		}

		if cb.app.config.Verbose && (status.Installed || status.ConflictingHook) {
			printDetail("    Path: %s", status.HookPath)
			printDetail("    Permissions: %s", status.FileMode)
			printDetail("    Modified: %s", status.ModTime.Format("2006-01-02 15:04:05"))
		}
	}

	if !foundHooks {
		printWarning("No hooks are currently installed")
		printInfo("Run 'go-pre-commit install' to install pre-commit hooks")
	}

	// Configuration status
	if cb.app.config.Verbose {
		printSubheader("Configuration Status")
		printDetail("  Checks enabled:")
		printDetail("    fumpt: %t", cfg.Checks.Fumpt)
		printDetail("    lint: %t", cfg.Checks.Lint)
		printDetail("    mod-tidy: %t", cfg.Checks.ModTidy)
		printDetail("    whitespace: %t", cfg.Checks.Whitespace)
		printDetail("    eof: %t", cfg.Checks.EOF)
		printDetail("  Timeout: %d seconds", cfg.Timeout)
		printDetail("  Parallel workers: %d", cfg.Performance.ParallelWorkers)
	}

	return nil
}

// statusReport is the document `status --json` prints
type statusReport struct {
	Version        string             `json:"version"`
	Enabled        bool               `json:"enabled"`
	Config         statusConfigSource `json:"config"`
	RepositoryRoot string             `json:"repository_root"`
	HooksDir       string             `json:"hooks_dir"`
	Hooks          []statusHook       `json:"hooks"`
}

// statusConfigSource names the configuration status read
type statusConfigSource struct {
	Mode string `json:"mode"`
	Path string `json:"path"`
}

// statusHook is one hook type in the status report
type statusHook struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	Path       string `json:"path"`
	Executable bool   `json:"executable"`
	Outdated   bool   `json:"outdated"`
	Error      string `json:"error,omitempty"`
}

// runStatusJSON writes the status report, or the error document, to w
func (cb *CommandBuilder) runStatusJSON(w io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return writeErrorReport(w, cb.app.version,
			withCode(configErrorCode(err), fmt.Errorf("failed to load configuration: %w", err)))
	}

	repoRoot, err := git.FindRepositoryRoot()
	if err != nil {
		return writeErrorReport(w, cb.app.version,
			withCode(errCodeNotARepository, fmt.Errorf("failed to find git repository: %w", err)))
	}

	installer := git.NewInstallerWithConfig(repoRoot, cfg.Directory, cfg)
	report := statusReport{
		Version:        cb.app.version,
		Enabled:        cfg.Enabled,
		Config:         statusConfigSource{Mode: cfg.Source.Mode, Path: cfg.Source.Path},
		RepositoryRoot: repoRoot,
		Hooks:          make([]statusHook, 0, len(supportedHooks())),
	}
	if hooksDir, hooksDirErr := installer.HooksDir(); hooksDirErr == nil {
		report.HooksDir = hooksDir
	}
	for _, hookType := range supportedHooks() {
		report.Hooks = append(report.Hooks, newStatusHook(installer, hookType))
	}

	return writeJSON(w, report)
}

// newStatusHook reports the state of one hook type
func newStatusHook(installer *git.Installer, hookType string) statusHook {
	status, err := installer.GetInstallationStatus(hookType)
	hook := statusHook{Name: hookType, Path: status.HookPath}
	if err != nil {
		hook.State = hookStateError
		hook.Error = err.Error()
		return hook
	}

	hook.Executable = status.Executable
	hook.Outdated = status.Outdated
	switch {
	case status.ConflictingHook:
		hook.State = hookStateOther
	case !status.Installed:
		hook.State = hookStateMissing
	case !status.Executable:
		hook.State = hookStateNotExecutable
	case status.Outdated:
		hook.State = hookStateOutdated
	default:
		hook.State = hookStateReady
	}
	return hook
}

func printHeader(text string) {
	formatter := output.NewDefault()
	formatter.Info("\n=== %s ===", text)
}

func printSubheader(text string) {
	formatter := output.NewDefault()
	formatter.Info("\n%s:", text)
}

func printDetail(format string, args ...any) {
	formatter := output.NewDefault()
	formatter.Info(format, args...)
}
