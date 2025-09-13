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
	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/progress"
)

// Error variables for tool installation
var (
	// ErrUnknownTool is returned when a tool is not recognized
	ErrUnknownTool = errors.New("unknown tool")
	// ErrInstallFailed is returned when tool installation fails
	ErrInstallFailed = errors.New("tool installation failed")
	// ErrToolNotInPath is returned when a tool is installed but not found in PATH
	ErrToolNotInPath = errors.New("tool installed but not found in PATH")
	// ErrInstallTimeout is returned when tool installation times out
	ErrInstallTimeout = errors.New("tool installation timed out")
)

// Configuration for tool installation
//
//nolint:gochecknoglobals // These globals are required for configuration management
var (
	installTimeout = 5 * time.Minute // Default timeout for tool installation
	retryAttempts  = 3               // Default number of retry attempts for network operations
	retryDelay     = 2 * time.Second // Default delay between retry attempts
	configMu       sync.RWMutex      // Protects configuration
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
	}

	if v := os.Getenv("GO_PRE_COMMIT_FUMPT_VERSION"); v != "" {
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

// SetInstallTimeout configures the timeout for tool installation
func SetInstallTimeout(timeout time.Duration) {
	configMu.Lock()
	defer configMu.Unlock()
	installTimeout = timeout
}

// GetInstallTimeout returns the current tool installation timeout
func GetInstallTimeout() time.Duration {
	configMu.RLock()
	defer configMu.RUnlock()
	return installTimeout
}

// SetRetryConfig configures retry behavior for network operations
func SetRetryConfig(attempts int, delay time.Duration) {
	configMu.Lock()
	defer configMu.Unlock()
	retryAttempts = attempts
	retryDelay = delay
}

// GetRetryConfig returns the current retry configuration
func GetRetryConfig() (int, time.Duration) {
	configMu.RLock()
	defer configMu.RUnlock()
	return retryAttempts, retryDelay
}

// isNetworkError checks if an error is likely due to network issues
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := strings.ToLower(err.Error())
	networkErrorPatterns := []string{
		"connection refused",
		"connection timeout",
		"connection timed out",
		"connection reset",
		"network is unreachable",
		"no such host",
		"temporary failure in name resolution",
		"i/o timeout",
		"dial tcp",
		"tls handshake timeout",
		"proxy error",
		"bad gateway",
		"service unavailable",
		"gateway timeout",
	}

	for _, pattern := range networkErrorPatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// retryWithBackoff executes a function with exponential backoff retry logic
func retryWithBackoff(ctx context.Context, operation string, fn func() error) error {
	attempts, delay := GetRetryConfig()

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff with jitter
			backoffDelay := time.Duration(float64(delay) * (1.5 * float64(attempt)))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoffDelay):
				// Continue with retry
			}

			if _, err := fmt.Fprintf(os.Stdout, "%s Retrying %s (attempt %d/%d)...\n",
				color.YellowString("⚠"), operation, attempt+1, attempts); err != nil {
				// Ignore write error to stdout
				_ = err
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil // Success
		}

		// Only retry on network errors
		if !isNetworkError(lastErr) {
			break
		}

		// Don't retry if context is done
		if ctx.Err() != nil {
			break
		}
	}

	return lastErr
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

	// Create timeout context using configurable timeout
	timeout := GetInstallTimeout()
	installCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Start progress tracking for installations that might take a while
	tracker := progress.New(progress.Options{
		Operation:    "Tool installation",
		Context:      tool.Name,
		Timeout:      timeout,
		ProgressFunc: progress.InstallProgressFunc(tool.Name),
	})
	tracker.Start(installCtx)
	defer tracker.Stop()

	// Special handling for golangci-lint which has a custom installer
	if tool.Name == "golangci-lint" {
		return installGolangciLint(installCtx, tool.Version)
	}

	// Standard go install for other tools with retry logic
	var output []byte
	start := time.Now()

	err := retryWithBackoff(installCtx, fmt.Sprintf("installing %s", tool.Name), func() error {
		cmd := exec.CommandContext(installCtx, "go", "install", installPath) //nolint:gosec // installPath is constructed from trusted tool config
		cmd.Env = append(os.Environ(), "GO111MODULE=on")

		var cmdErr error
		output, cmdErr = cmd.CombinedOutput()
		return cmdErr
	})

	elapsed := time.Since(start)

	if err != nil {
		// Check if this was a timeout
		if errors.Is(installCtx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolInstallTimeoutError(tool.Name, timeout, elapsed)
		}

		// Check if we exhausted retries on network errors
		if isNetworkError(err) {
			attempts, _ := GetRetryConfig()
			return fmt.Errorf("%w for %s after %d attempts (network error): %w\nOutput: %s",
				ErrInstallFailed, tool.Name, attempts, err, output)
		}

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
	timeout := GetInstallTimeout()

	// Check if this was a timeout at the start
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return prerrors.NewToolInstallTimeoutError("golangci-lint", timeout, 0)
	}

	// Start progress tracking for golangci-lint installation
	tracker := progress.New(progress.Options{
		Operation:    "Tool installation",
		Context:      "golangci-lint",
		Timeout:      timeout,
		ProgressFunc: progress.InstallProgressFunc("golangci-lint"),
	})
	tracker.Start(ctx)
	defer tracker.Stop()

	// Use the official installation method with retry logic
	installScript := `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin %s`

	start := time.Now()
	var output []byte

	err := retryWithBackoff(ctx, "installing golangci-lint", func() error {
		cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf(installScript, version)) //nolint:gosec // version comes from trusted config
		cmd.Env = append(os.Environ(), "GO111MODULE=on")

		var cmdErr error
		output, cmdErr = cmd.CombinedOutput()
		return cmdErr
	})

	elapsed := time.Since(start)

	if err != nil {
		// Check for timeout on primary installation
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolInstallTimeoutError("golangci-lint", timeout, elapsed)
		}

		// If primary installation failed with network error after retries, try fallback
		if isNetworkError(err) {
			if _, logErr := fmt.Fprintf(os.Stdout, "%s Primary installation failed with network error, trying fallback method...\n",
				color.YellowString("⚠")); logErr != nil {
				// Ignore write error to stdout
				_ = logErr
			}
		}

		// Fallback to go install method with retry logic
		installPath := fmt.Sprintf("github.com/golangci/golangci-lint/cmd/golangci-lint@%s", version)

		var fallbackOutput []byte
		fallbackStart := time.Now()

		fallbackErr := retryWithBackoff(ctx, "installing golangci-lint (fallback)", func() error {
			fallbackCmd := exec.CommandContext(ctx, "go", "install", installPath) //nolint:gosec // installPath is constructed from trusted tool config
			fallbackCmd.Env = append(os.Environ(), "GO111MODULE=on")

			var cmdErr error
			fallbackOutput, cmdErr = fallbackCmd.CombinedOutput()
			return cmdErr
		})

		fallbackElapsed := time.Since(fallbackStart)
		totalElapsed := time.Since(start)

		if fallbackErr != nil {
			// Check for timeout on fallback
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return prerrors.NewToolInstallTimeoutError("golangci-lint", timeout, totalElapsed)
			}

			// Both methods failed
			if isNetworkError(fallbackErr) {
				attempts, _ := GetRetryConfig()
				return fmt.Errorf("%w for golangci-lint after %d attempts on both methods (network errors): %w\nScript output: %s\nGo install output: %s",
					ErrInstallFailed, attempts, err, output, fallbackOutput)
			}

			return fmt.Errorf("%w for golangci-lint: %w\nScript output: %s\nGo install output: %s",
				ErrInstallFailed, err, output, fallbackOutput)
		}

		// Fallback succeeded, log that we used it
		if _, logErr := fmt.Fprintf(os.Stdout, "%s Installed golangci-lint using fallback method (took %v)\n",
			color.YellowString("⚠"), fallbackElapsed); logErr != nil {
			// Ignore write error to stdout
			_ = logErr
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
