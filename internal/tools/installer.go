// Package tools provides automatic installation and management of required Go tools
package tools

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
	"github.com/mrz1836/go-pre-commit/internal/progress"
)

const (
	toolGolangciLint  = "golangci-lint"
	toolGitleaks      = "gitleaks"
	toolVersionLatest = "latest"

	// EnvAutoInstallTools turns automatic tool installs off when set to false.
	// A missing tool then fails its check with the command that installs it.
	EnvAutoInstallTools = "GO_PRE_COMMIT_AUTO_INSTALL_TOOLS"
)

// runInstallCommand runs an install/download command and returns its combined
// output. It is a package variable so tests can substitute a fake implementation
// that avoids real process execution and network access (go install, curl).
// When env is nil the process inherits the parent environment.
//
//nolint:gochecknoglobals // Injectable seam so tests can avoid real exec/network
var runInstallCommand = func(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // arguments are constructed from trusted tool config
	if env != nil {
		cmd.Env = env
	}
	return cmd.CombinedOutput()
}

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
	// ErrToolNotInstalled is returned when a tool is not installed or not found
	ErrToolNotInstalled = errors.New("tool not installed")
	// ErrToolNeedsNewerGo is returned when a tool cannot be installed because it
	// requires a newer Go toolchain than the one available (and GOTOOLCHAIN is not
	// permitted to upgrade). This is a common, actionable failure when a pinned tool
	// version outpaces the runner's Go version.
	ErrToolNeedsNewerGo = errors.New("tool requires a newer Go toolchain")
	// ErrAutoInstallDisabled is returned for a missing tool when automatic
	// installs are turned off with GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false
	ErrAutoInstallDisabled = errors.New("tool not installed and automatic installs are off")
)

// goRequiresVersionRe extracts the required Go version from a failed `go install`
// output, e.g. "go: mvdan.cc/gofumpt@v0.12.0 requires go >= 1.26.0 (running go
// 1.25.1; GOTOOLCHAIN=local)". Matches both "requires go >= X" and "requires go X".
var goRequiresVersionRe = regexp.MustCompile(`(?i)requires go\s*>?=?\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)

// runGoVersionCommand reports the active Go toolchain version (the `go` on PATH
// that will run `go install`), e.g. "go1.25.3". It is a package variable so tests
// can substitute a fake implementation that avoids executing the real `go` binary.
//
//nolint:gochecknoglobals // Injectable seam so tests can stub Go version detection.
var runGoVersionCommand = func() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "go", "env", "GOVERSION").Output()
	return string(out), err
}

// goMajorMinorRe matches a leading "MAJOR.MINOR" once any "go" prefix is trimmed,
// tolerating trailing ".patch" or ".x" (e.g. "1.25.3", "1.26.x", "1.26").
var goMajorMinorRe = regexp.MustCompile(`^([0-9]+)\.([0-9]+)`)

// parseGoMajorMinor extracts the major and minor components from a Go version
// string. It accepts "go1.25.3", "1.26.x", "1.26" and similar. ok is false when
// the string cannot be parsed.
func parseGoMajorMinor(s string) (major, minor int, ok bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "go")
	m := goMajorMinorRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	// Regex guarantees both groups are non-empty digit strings.
	major, _ = strconv.Atoi(m[1])
	minor, _ = strconv.Atoi(m[2])
	return major, minor, true
}

// detectGoVersion returns the major/minor version of the active Go toolchain.
// ok is false when it cannot be determined (e.g. `go` is not on PATH).
func detectGoVersion() (major, minor int, ok bool) {
	out, err := runGoVersionCommand()
	if err != nil {
		return 0, 0, false
	}
	return parseGoMajorMinor(out)
}

// selectToolVersion chooses between a baseline pin (compatible with older Go) and
// a newer "latest" pin that requires a newer Go toolchain, based on the active Go
// version. It mirrors the GoFortress setup-benchstat selection so the same synced
// env files work across repos on different Go versions without forcing every repo
// to upgrade Go in lockstep with a tool bump:
//
//   - latest == ""                    -> baseline (dual-pin disabled; current behavior)
//   - active Go >= minGo              -> latest
//   - active Go < minGo               -> baseline
//   - active Go undetectable          -> baseline (conservative: the baseline pin is
//     chosen to install on the widest range of Go versions)
//
// minGo defaults to 1.26 when empty or unparseable, matching the benchstat default.
func selectToolVersion(baseline, latest, minGo string) string {
	if latest == "" {
		return baseline
	}

	minMajor, minMinor, ok := parseGoMajorMinor(minGo)
	if !ok {
		minMajor, minMinor = 1, 26
	}

	major, minor, ok := detectGoVersion()
	if !ok {
		return baseline
	}

	if major > minMajor || (major == minMajor && minor >= minMinor) {
		return latest
	}
	return baseline
}

// newToolchainMismatchError inspects failed `go install` output and, when it
// indicates the tool requires a newer Go toolchain than is available, returns a
// clear, actionable error wrapping ErrToolNeedsNewerGo. It returns nil when the
// output is not a toolchain-version mismatch, so callers fall through to their
// generic error handling.
func newToolchainMismatchError(tool *Tool, output []byte) error {
	out := string(output)
	m := goRequiresVersionRe.FindStringSubmatch(out)
	if m == nil {
		return nil
	}

	required := m[1]
	current := "unknown"
	if major, minor, ok := detectGoVersion(); ok {
		current = fmt.Sprintf("%d.%d", major, minor)
	}

	envKey := toolEnvKey(tool.Name)
	return fmt.Errorf("%w: %s@%s requires Go >= %s but this environment has Go %s (GOTOOLCHAIN=%s). "+
		"Upgrade Go, or pin a compatible %s version via GO_PRE_COMMIT_%s_VERSION "+
		"(dual-version pinning: GO_PRE_COMMIT_%s_VERSION_LATEST / _LATEST_MIN_GO)\nOutput: %s",
		ErrToolNeedsNewerGo, tool.Name, tool.Version, required, current, gotoolchainSetting(),
		tool.Name, envKey, envKey, out)
}

// gotoolchainSetting reports the effective GOTOOLCHAIN value for diagnostics,
// defaulting to "auto" (Go's default) when unset.
func gotoolchainSetting() string {
	if v := os.Getenv("GOTOOLCHAIN"); v != "" {
		return v
	}
	return "auto"
}

// toolEnvKey maps a tool name to the uppercase token used in its env-var pins,
// e.g. "gofumpt" -> "FUMPT", "golangci-lint" -> "GOLANGCI_LINT".
func toolEnvKey(name string) string {
	switch name {
	case "gofumpt":
		return "FUMPT"
	case toolGolangciLint:
		return "GOLANGCI_LINT"
	default:
		return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	}
}

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

// newToolRegistry returns the default registry of tools used by go-pre-commit.
// Versions left empty are loaded from the environment by LoadVersionsFromEnv.
func newToolRegistry() map[string]*Tool {
	return map[string]*Tool{
		toolGolangciLint: {
			Name:       toolGolangciLint,
			ImportPath: "github.com/golangci/golangci-lint/cmd/golangci-lint",
			Version:    "", // Will be loaded from env
			Binary:     toolGolangciLint,
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
			Version:    toolVersionLatest,
			Binary:     "goimports",
		},
		toolGitleaks: {
			Name: toolGitleaks,
			// The module keeps its original path; go install of the
			// github.com/gitleaks/gitleaks/v8 path fails.
			ImportPath: "github.com/zricethezav/gitleaks/v8",
			Version:    "", // Will be loaded from env
			Binary:     toolGitleaks,
		},
	}
}

// Common tools used by go-pre-commit
// These globals are necessary for maintaining tool registry and installation state
//
//nolint:gochecknoglobals // Tool registry requires package-level state for singleton pattern
var (
	toolsMu sync.RWMutex
	tools   = newToolRegistry()

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
		if t, ok := tools[toolGolangciLint]; ok {
			t.Version = v
		}
	}

	// gofumpt supports dual-version pinning so a single set of synced env files
	// works across repos on different Go versions. GO_PRE_COMMIT_FUMPT_VERSION is
	// the baseline (installs on older Go); GO_PRE_COMMIT_FUMPT_VERSION_LATEST is a
	// newer build used only when the active Go is >= _LATEST_MIN_GO. When _LATEST is
	// unset this collapses to the original single-version behavior.
	if t, ok := tools["gofumpt"]; ok {
		if v := selectToolVersion(
			os.Getenv("GO_PRE_COMMIT_FUMPT_VERSION"),
			os.Getenv("GO_PRE_COMMIT_FUMPT_VERSION_LATEST"),
			os.Getenv("GO_PRE_COMMIT_FUMPT_VERSION_LATEST_MIN_GO"),
		); v != "" {
			t.Version = v
		}
	}

	if v := os.Getenv("GO_PRE_COMMIT_GOIMPORTS_VERSION"); v != "" {
		if t, ok := tools["goimports"]; ok {
			t.Version = v
		}
	}

	if v := os.Getenv("GO_PRE_COMMIT_GITLEAKS_VERSION"); v != "" {
		if t, ok := tools["gitleaks"]; ok {
			t.Version = v
		}
	}

	// Set defaults if not specified
	if t, ok := tools[toolGolangciLint]; ok && t.Version == "" {
		t.Version = "v2.4.0"
	}
	if t, ok := tools["gofumpt"]; ok && t.Version == "" {
		t.Version = "v0.8.0"
	}
	if t, ok := tools["gitleaks"]; ok && t.Version == "" {
		t.Version = "v8.29.0"
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
	if !exists {
		toolsMu.RUnlock()
		return fmt.Errorf("%w: %s", ErrUnknownTool, toolName)
	}
	// Copy tool struct to avoid race with concurrent LoadVersionsFromEnv calls
	// that may write to tool.Version while InstallTool reads it
	toolCopy := &Tool{
		Name:       tool.Name,
		ImportPath: tool.ImportPath,
		Version:    tool.Version,
		Binary:     tool.Binary,
	}
	toolsMu.RUnlock()

	// Install the tool
	return InstallTool(ctx, toolCopy)
}

// AutoInstallEnabled reports whether missing tools may be installed
// automatically. GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false (or 0) turns installs
// off; unset, or a value that is not a boolean, keeps the default (on), as
// with every other boolean setting.
func AutoInstallEnabled() bool {
	value := strings.TrimSpace(os.Getenv(EnvAutoInstallTools))
	if value == "" {
		return true
	}
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return true
	}
	return enabled
}

// InstallCommand returns the command that installs the configured version of
// a tool by hand, such as "go install mvdan.cc/gofumpt@v0.8.0", or "" for an
// unknown tool. Checks show it when a tool is missing and was not installed.
func InstallCommand(toolName string) string {
	LoadVersionsFromEnv()

	toolsMu.RLock()
	defer toolsMu.RUnlock()
	tool, ok := tools[toolName]
	if !ok {
		return ""
	}
	return manualInstallCommand(tool)
}

// manualInstallCommand is the go install command for the tool's version.
// The Go module proxy and checksum database verify what it downloads.
func manualInstallCommand(tool *Tool) string {
	version := tool.Version
	if version == "" {
		version = toolVersionLatest
	}
	pkg := tool.ImportPath
	if tool.Name == toolGolangciLint {
		pkg = golangciLintPackage(version)
	}
	return "go install " + pkg + "@" + version
}

// golangciLintPackage is the package go install needs for a golangci-lint
// version. golangci-lint moved to the github.com/golangci/golangci-lint/v2
// module at v2, so v2 and later (and latest) use that module's path.
func golangciLintPackage(version string) string {
	major := 2 // latest
	if version != toolVersionLatest {
		if m, _, ok := parseGoMajorMinor(strings.TrimPrefix(version, "v")); ok {
			major = m
		}
	}
	if major < 2 {
		return "github.com/golangci/golangci-lint/cmd/golangci-lint"
	}
	return fmt.Sprintf("github.com/golangci/golangci-lint/v%d/cmd/golangci-lint", major)
}

// InstallTool installs a specific tool. golangci-lint and gitleaks are
// installed from their GitHub release archives, verified against the
// release's checksums file; other tools with go install. When automatic
// installs are off (GO_PRE_COMMIT_AUTO_INSTALL_TOOLS=false) a missing tool is
// never installed: InstallTool returns ErrAutoInstallDisabled with the
// command that installs it.
func InstallTool(ctx context.Context, tool *Tool) error {
	installMu.Lock()
	defer installMu.Unlock()

	// Double-check if installed (in case of concurrent calls)
	if _, err := exec.LookPath(tool.Binary); err == nil {
		installedTools[tool.Name] = true
		return nil
	}

	if !AutoInstallEnabled() {
		return fmt.Errorf("%w: %s is not on PATH, and %s=false, so it was not installed. Install it with: %s",
			ErrAutoInstallDisabled, tool.Name, EnvAutoInstallTools, manualInstallCommand(tool))
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
	if tool.Version != "" && tool.Version != toolVersionLatest {
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

	// golangci-lint and gitleaks are installed from verified release archives
	switch tool.Name {
	case toolGolangciLint:
		if err := installGolangciLint(installCtx, tool.Version); err != nil {
			return err
		}
		return finishInstall(tool)
	case toolGitleaks:
		if err := installGitleaks(installCtx, tool.Version); err != nil {
			return err
		}
		return finishInstall(tool)
	}

	// Standard go install for other tools with retry logic
	var output []byte
	start := time.Now()

	err := retryWithBackoff(installCtx, fmt.Sprintf("installing %s", tool.Name), func() error {
		var cmdErr error
		output, cmdErr = runInstallCommand(installCtx, append(os.Environ(), "GO111MODULE=on"), "go", "install", installPath)
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

		// Detect the common, actionable case where the pinned tool version requires
		// a newer Go toolchain than is available. Surface a clear message instead of
		// the opaque "exit status 1" so users know to upgrade Go or pin an older
		// version (rather than blindly retrying `go install ...@latest`).
		if mismatchErr := newToolchainMismatchError(tool, output); mismatchErr != nil {
			return mismatchErr
		}

		return fmt.Errorf("%w for %s: %w\nOutput: %s", ErrInstallFailed, tool.Name, err, output)
	}

	return finishInstall(tool)
}

// finishInstall checks that a tool just installed is found on PATH, records
// it as installed, and reports the success. Callers hold installMu.
func finishInstall(tool *Tool) error {
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

// installGolangciLint installs golangci-lint from its verified release
// archive. If that fails at any step, it falls back to go install of the same
// version, which the Go checksum database verifies; when both fail, the error
// reports both.
func installGolangciLint(ctx context.Context, version string) error {
	timeout := GetInstallTimeout()
	start := time.Now()

	tag, err := resolveReleaseTag(ctx, toolGolangciLint, golangciLintRepo, version)
	if err == nil {
		_, err = installRelease(ctx, golangciLintAsset(tag, runtime.GOOS, runtime.GOARCH), GetGoBin())
	}
	if err == nil {
		return nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return prerrors.NewToolInstallTimeoutError(toolGolangciLint, timeout, time.Since(start))
	}

	// Fall back to go install of the same version
	fallbackVersion := version
	if tag != "" {
		fallbackVersion = tag
	}
	installTarget := golangciLintPackage(fallbackVersion) + "@" + fallbackVersion
	if _, logErr := fmt.Fprintf(os.Stdout, "%s %v\n%s Trying go install %s instead...\n",
		color.YellowString("⚠"), err, color.YellowString("→"), installTarget); logErr != nil {
		// Ignore write error to stdout
		_ = logErr
	}

	var output []byte
	fallbackStart := time.Now()
	fallbackErr := retryWithBackoff(ctx, "installing golangci-lint (fallback)", func() error {
		var cmdErr error
		output, cmdErr = runInstallCommand(ctx, append(os.Environ(), "GO111MODULE=on"), "go", "install", installTarget)
		return cmdErr
	})
	if fallbackErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolInstallTimeoutError(toolGolangciLint, timeout, time.Since(start))
		}
		return fmt.Errorf("%w; %w\nOutput: %s", err,
			fmt.Errorf("%s %s: go install fallback: %w", toolGolangciLint, fallbackVersion, fallbackErr), output)
	}

	if _, logErr := fmt.Fprintf(os.Stdout, "%s Installed golangci-lint with go install (took %v)\n",
		color.YellowString("⚠"), time.Since(fallbackStart)); logErr != nil {
		// Ignore write error to stdout
		_ = logErr
	}
	return nil
}

// installGitleaks installs gitleaks from its verified release archive.
func installGitleaks(ctx context.Context, version string) error {
	timeout := GetInstallTimeout()
	start := time.Now()

	tag, err := resolveReleaseTag(ctx, toolGitleaks, gitleaksRepo, version)
	if err == nil {
		_, err = installRelease(ctx, gitleaksAsset(tag, runtime.GOOS, runtime.GOARCH), GetGoBin())
	}
	if err != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return prerrors.NewToolInstallTimeoutError(toolGitleaks, timeout, time.Since(start))
	}
	return err
}

// GetToolPath returns the full path to a tool binary
func GetToolPath(toolName string) (string, error) {
	toolsMu.RLock()
	tool, exists := tools[toolName]
	toolsMu.RUnlock()

	if !exists {
		return "", fmt.Errorf("%w: %s", ErrUnknownTool, toolName)
	}

	path, err := exec.LookPath(tool.Binary)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrToolNotInstalled, tool.Name, err)
	}

	return path, nil
}

// InstallAllTools installs all required tools
func InstallAllTools(ctx context.Context) error {
	LoadVersionsFromEnv()

	// Snapshot tool names under read lock to avoid race with map modifications
	toolsMu.RLock()
	toolNames := slices.Collect(maps.Keys(tools))
	toolsMu.RUnlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(toolNames))

	for _, name := range toolNames {
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

	errors := make([]string, 0, len(toolNames))
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
