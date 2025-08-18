// Package tools provides automatic installation and management of required Go tools
package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Error variables for tool installation
var (
	// ErrUnknownTool is returned when a tool is not recognized
	ErrUnknownTool = errors.New("unknown tool")
	// ErrInstallFailed is returned when tool installation fails
	ErrInstallFailed = errors.New("tool installation failed")
	// ErrToolNotInPath is returned when a tool is installed but not found in PATH
	ErrToolNotInPath = errors.New("tool installed but not found in PATH")
)

// Tool represents a Go tool that can be installed
type Tool struct {
	Name       string
	ImportPath string
	Version    string
	Binary     string
}

// Common tools used by go-pre-commit
// These globals are necessary for maintaining tool registry and installation state
//
//nolint:gochecknoglobals // Tool registry requires package-level state for singleton pattern
var (
	toolsMu sync.RWMutex
	tools   = map[string]*Tool{
		"golangci-lint": {
			Name:       "golangci-lint",
			ImportPath: "github.com/golangci/golangci-lint/cmd/golangci-lint",
			Version:    "", // Will be loaded from env
			Binary:     "golangci-lint",
		},
		"gofumpt": {
			Name:       "gofumpt",
			ImportPath: "mvdan.cc/gofumpt",
			Version:    "", // Will be loaded from env
			Binary:     "gofumpt",
		},
		"goimports": {
			Name:       "goimports",
			ImportPath: "golang.org/x/tools/cmd/goimports",
			Version:    "latest",
			Binary:     "goimports",
		},
	}

	//nolint:gochecknoglobals // Installation cache requires package-level state
	installedTools = make(map[string]bool)
	//nolint:gochecknoglobals // Mutex for thread-safe access to installation cache
	installMu sync.Mutex
)

// LoadVersionsFromEnv loads tool versions from environment variables
func LoadVersionsFromEnv() {
	toolsMu.Lock()
	defer toolsMu.Unlock()

	// Load versions from environment
	if v := os.Getenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION"); v != "" {
		if t, ok := tools["golangci-lint"]; ok {
			t.Version = v
		}
	} else if v := os.Getenv("GOLANGCI_LINT_VERSION"); v != "" {
		if t, ok := tools["golangci-lint"]; ok {
			t.Version = v
		}
	}

	if v := os.Getenv("GO_PRE_COMMIT_FUMPT_VERSION"); v != "" {
		if t, ok := tools["gofumpt"]; ok {
			t.Version = v
		}
	} else if v := os.Getenv("GOFUMPT_VERSION"); v != "" {
		if t, ok := tools["gofumpt"]; ok {
			t.Version = v
		}
	}

	if v := os.Getenv("GO_PRE_COMMIT_GOIMPORTS_VERSION"); v != "" {
		if t, ok := tools["goimports"]; ok {
			t.Version = v
		}
	}

	// Set defaults if not specified
	if tools["golangci-lint"].Version == "" {
		tools["golangci-lint"].Version = "v2.4.0"
	}
	if tools["gofumpt"].Version == "" {
		tools["gofumpt"].Version = "v0.8.0"
	}
}

// IsInstalled checks if a tool is installed and available in PATH
func IsInstalled(toolName string) bool {
	installMu.Lock()
	if installed, ok := installedTools[toolName]; ok {
		installMu.Unlock()
		return installed
	}
	installMu.Unlock()

	toolsMu.RLock()
	tool, exists := tools[toolName]
	toolsMu.RUnlock()

	if !exists {
		return false
	}

	_, err := exec.LookPath(tool.Binary)
	installed := err == nil

	installMu.Lock()
	installedTools[toolName] = installed
	installMu.Unlock()

	return installed
}

// EnsureInstalled ensures a tool is installed, installing it if necessary
func EnsureInstalled(ctx context.Context, toolName string) error {
	// Ensure versions are loaded from environment
	LoadVersionsFromEnv()

	// Check if already installed
	if IsInstalled(toolName) {
		return nil
	}

	toolsMu.RLock()
	tool, exists := tools[toolName]
	toolsMu.RUnlock()

	if !exists {
		return fmt.Errorf("%w: %s", ErrUnknownTool, toolName)
	}

	// Install the tool
	return InstallTool(ctx, tool)
}

// InstallTool installs a specific tool
func InstallTool(ctx context.Context, tool *Tool) error {
	installMu.Lock()
	defer installMu.Unlock()

	// Double-check if installed (in case of concurrent calls)
	if _, err := exec.LookPath(tool.Binary); err == nil {
		installedTools[tool.Name] = true
		return nil
	}

	// Log installation status (output is intentional for user feedback)
	if _, err := fmt.Fprintf(os.Stdout, "%s Installing %s@%s...\n",
		color.YellowString("→"),
		tool.Name,
		tool.Version); err != nil {
		// Ignore write error to stdout
		_ = err
	}

	// Build install command
	installPath := tool.ImportPath
	if tool.Version != "" && tool.Version != "latest" {
		installPath = fmt.Sprintf("%s@%s", tool.ImportPath, tool.Version)
	}

	// Create timeout context
	installCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// Special handling for golangci-lint which has a custom installer
	if tool.Name == "golangci-lint" {
		return installGolangciLint(installCtx, tool.Version)
	}

	// Standard go install for other tools
	cmd := exec.CommandContext(installCtx, "go", "install", installPath) //nolint:gosec // installPath is constructed from trusted tool config
	cmd.Env = append(os.Environ(), "GO111MODULE=on")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w for %s: %w\nOutput: %s", ErrInstallFailed, tool.Name, err, output)
	}

	// Verify installation
	if _, err := exec.LookPath(tool.Binary); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrToolNotInPath, tool.Name, err)
	}

	installedTools[tool.Name] = true
	// Log success status (output is intentional for user feedback)
	if _, err := fmt.Fprintf(os.Stdout, "%s Successfully installed %s\n",
		color.GreenString("✓"),
		tool.Name); err != nil {
		// Ignore write error to stdout
		_ = err
	}

	return nil
}

// installGolangciLint uses the official installer script for golangci-lint
func installGolangciLint(ctx context.Context, version string) error {
	// Use the official installation method
	installScript := `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin %s`

	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf(installScript, version)) //nolint:gosec // version comes from trusted config
	cmd.Env = append(os.Environ(), "GO111MODULE=on")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to go install method
		installPath := fmt.Sprintf("github.com/golangci/golangci-lint/cmd/golangci-lint@%s", version)
		fallbackCmd := exec.CommandContext(ctx, "go", "install", installPath) //nolint:gosec // installPath is constructed from trusted tool config
		fallbackCmd.Env = append(os.Environ(), "GO111MODULE=on")

		fallbackOutput, fallbackErr := fallbackCmd.CombinedOutput()
		if fallbackErr != nil {
			return fmt.Errorf("%w for golangci-lint: %w\nScript output: %s\nGo install output: %s",
				ErrInstallFailed, err, output, fallbackOutput)
		}
	}

	return nil
}

// GetToolPath returns the full path to a tool binary
func GetToolPath(toolName string) (string, error) {
	toolsMu.RLock()
	tool, exists := tools[toolName]
	toolsMu.RUnlock()

	if !exists {
		return "", fmt.Errorf("%w: %s", ErrUnknownTool, toolName)
	}

	return exec.LookPath(tool.Binary)
}

// InstallAllTools installs all required tools
func InstallAllTools(ctx context.Context) error {
	LoadVersionsFromEnv()

	var wg sync.WaitGroup
	errCh := make(chan error, len(tools))

	for name := range tools {
		wg.Add(1)
		go func(toolName string) {
			defer wg.Done()
			if err := EnsureInstalled(ctx, toolName); err != nil {
				errCh <- fmt.Errorf("%s: %w", toolName, err)
			}
		}(name)
	}

	wg.Wait()
	close(errCh)

	errors := make([]string, 0, len(tools))
	for err := range errCh {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("%w:\n%s", ErrInstallFailed, strings.Join(errors, "\n"))
	}

	return nil
}

// CleanCache clears the installed tools cache
func CleanCache() {
	installMu.Lock()
	installedTools = make(map[string]bool)
	installMu.Unlock()
}

// GetGoPath returns the GOPATH or default if not set
func GetGoPath() string {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return gopath
	}

	// Default GOPATH
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go")
}

// GetGoBin returns the Go bin directory
func GetGoBin() string {
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return gobin
	}
	return filepath.Join(GetGoPath(), "bin")
}
