// Package config provides configuration loading for the GoFortress pre-commit system
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mrz1836/go-pre-commit/internal/envfile"
	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

// Config holds the configuration for the pre-commit system
type Config struct {
	// Core settings
	Enabled      bool   // ENABLE_GO_PRE_COMMIT
	Directory    string // Directory containing pre-commit tools (derived)
	LogLevel     string // GO_PRE_COMMIT_LOG_LEVEL
	MaxFileSize  int64  // GO_PRE_COMMIT_MAX_FILE_SIZE_MB
	MaxFilesOpen int    // GO_PRE_COMMIT_MAX_FILES_OPEN
	Timeout      int    // GO_PRE_COMMIT_TIMEOUT_SECONDS

	// Check configurations
	Checks struct {
		Fmt              bool // GO_PRE_COMMIT_ENABLE_FMT
		Fumpt            bool // GO_PRE_COMMIT_ENABLE_FUMPT
		Goimports        bool // GO_PRE_COMMIT_ENABLE_GOIMPORTS
		Lint             bool // GO_PRE_COMMIT_ENABLE_LINT
		ModTidy          bool // GO_PRE_COMMIT_ENABLE_MOD_TIDY
		Whitespace       bool // GO_PRE_COMMIT_ENABLE_WHITESPACE
		EOF              bool // GO_PRE_COMMIT_ENABLE_EOF
		AIDetection      bool // GO_PRE_COMMIT_ENABLE_AI_DETECTION
		Gitleaks         bool // GO_PRE_COMMIT_ENABLE_GITLEAKS
		GitleaksAllFiles bool // GO_PRE_COMMIT_GITLEAKS_ALL_FILES
	}

	// Check behaviors
	CheckBehaviors struct {
		FmtAutoStage        bool // GO_PRE_COMMIT_FMT_AUTO_STAGE
		FumptAutoStage      bool // GO_PRE_COMMIT_FUMPT_AUTO_STAGE
		GoimportsAutoStage  bool // GO_PRE_COMMIT_GOIMPORTS_AUTO_STAGE
		WhitespaceAutoStage bool // GO_PRE_COMMIT_WHITESPACE_AUTO_STAGE
		EOFAutoStage        bool // GO_PRE_COMMIT_EOF_AUTO_STAGE
		AIDetectionAutoFix  bool // GO_PRE_COMMIT_AI_DETECTION_AUTO_FIX
	}

	// Tool versions
	ToolVersions struct {
		Fumpt        string // GO_PRE_COMMIT_FUMPT_VERSION
		Goimports    string // GO_PRE_COMMIT_GOIMPORTS_VERSION
		GolangciLint string // GO_PRE_COMMIT_GOLANGCI_LINT_VERSION
		Gitleaks     string // GO_PRE_COMMIT_GITLEAKS_VERSION
	}

	// Performance settings
	Performance struct {
		ParallelWorkers int  // GO_PRE_COMMIT_PARALLEL_WORKERS
		FailFast        bool // GO_PRE_COMMIT_FAIL_FAST
	}

	// Check timeouts (in seconds)
	CheckTimeouts struct {
		Fmt         int // GO_PRE_COMMIT_FMT_TIMEOUT (default: 30)
		Fumpt       int // GO_PRE_COMMIT_FUMPT_TIMEOUT (default: 30)
		Goimports   int // GO_PRE_COMMIT_GOIMPORTS_TIMEOUT (default: 30)
		Lint        int // GO_PRE_COMMIT_LINT_TIMEOUT (default: 60)
		ModTidy     int // GO_PRE_COMMIT_MOD_TIDY_TIMEOUT (default: 30)
		Whitespace  int // GO_PRE_COMMIT_WHITESPACE_TIMEOUT (default: 30)
		EOF         int // GO_PRE_COMMIT_EOF_TIMEOUT (default: 30)
		AIDetection int // GO_PRE_COMMIT_AI_DETECTION_TIMEOUT (default: 30)
		Gitleaks    int // GO_PRE_COMMIT_GITLEAKS_TIMEOUT (default: 60)
	}

	// Git settings
	Git struct {
		HooksPath       string   // GO_PRE_COMMIT_HOOKS_PATH (default: .git/hooks)
		ExcludePatterns []string // GO_PRE_COMMIT_EXCLUDE_PATTERNS
	}

	// Go module settings
	Module struct {
		GoSumFile string // GO_SUM_FILE (default: go.sum) - location of go.sum, used to determine module directory
	}

	// UI settings
	UI struct {
		ColorOutput bool // GO_PRE_COMMIT_COLOR_OUTPUT (default: true)
	}

	// Tool installation settings
	ToolInstallation struct {
		Timeout int // GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT (default: 300)
	}

	// Environment detection
	Environment struct {
		IsCI             bool   // Detected if running in CI
		CIProvider       string // Which CI provider (github, gitlab, jenkins, etc.)
		AutoAdjustTimers bool   // GO_PRE_COMMIT_AUTO_ADJUST_CI_TIMEOUTS (default: true)
	}

	// Plugin settings
	Plugins struct {
		Enabled   bool   // GO_PRE_COMMIT_ENABLE_PLUGINS
		Directory string // GO_PRE_COMMIT_PLUGIN_DIR
		Timeout   int    // GO_PRE_COMMIT_PLUGIN_TIMEOUT
	}
}

// Load reads configuration from modular .github/env/*.env files or legacy .github/.env.base
func Load() (*Config, error) {
	// Try modular mode first (preferred)
	if envDir := findEnvDir(); envDir != "" {
		if err := envfile.LoadDir(envDir, isCI()); err != nil {
			return nil, fmt.Errorf("failed to load modular configuration from %s: %w", envDir, err)
		}
	} else {
		// Fall back to legacy mode
		basePath, err := findBaseEnvFile()
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %w", err)
		}
		if loadErr := envfile.Load(basePath); loadErr != nil {
			return nil, fmt.Errorf("failed to load %s: %w", basePath, loadErr)
		}
		customPath := findCustomEnvFile(basePath)
		if customPath != "" {
			if overloadErr := envfile.Overload(customPath); overloadErr != nil {
				return nil, fmt.Errorf("failed to load %s: %w", customPath, overloadErr)
			}
		}
	}

	cfg := &Config{
		Directory: "", // No longer using directory-based approach
	}

	// Core settings
	cfg.Enabled = getBoolEnv("ENABLE_GO_PRE_COMMIT", true)
	cfg.LogLevel = getStringEnv("GO_PRE_COMMIT_LOG_LEVEL", "info")
	cfg.MaxFileSize = int64(getIntEnv("GO_PRE_COMMIT_MAX_FILE_SIZE_MB", 10)) * 1024 * 1024
	cfg.MaxFilesOpen = getIntEnv("GO_PRE_COMMIT_MAX_FILES_OPEN", 100)
	cfg.Timeout = getIntEnv("GO_PRE_COMMIT_TIMEOUT_SECONDS", 720) // Global timeout in seconds (updated default)

	// Check configurations
	cfg.Checks.Fmt = getBoolEnv("GO_PRE_COMMIT_ENABLE_FMT", true)
	cfg.Checks.Fumpt = getBoolEnv("GO_PRE_COMMIT_ENABLE_FUMPT", true)
	cfg.Checks.Goimports = getBoolEnv("GO_PRE_COMMIT_ENABLE_GOIMPORTS", true)
	cfg.Checks.Lint = getBoolEnv("GO_PRE_COMMIT_ENABLE_LINT", true)
	cfg.Checks.ModTidy = getBoolEnv("GO_PRE_COMMIT_ENABLE_MOD_TIDY", true)
	cfg.Checks.Whitespace = getBoolEnv("GO_PRE_COMMIT_ENABLE_WHITESPACE", true)
	cfg.Checks.EOF = getBoolEnv("GO_PRE_COMMIT_ENABLE_EOF", true)
	cfg.Checks.AIDetection = getBoolEnv("GO_PRE_COMMIT_ENABLE_AI_DETECTION", true)
	cfg.Checks.Gitleaks = getBoolEnv("GO_PRE_COMMIT_ENABLE_GITLEAKS", false)
	cfg.Checks.GitleaksAllFiles = getBoolEnv("GO_PRE_COMMIT_GITLEAKS_ALL_FILES", false)

	// Check behaviors
	cfg.CheckBehaviors.FmtAutoStage = getBoolEnv("GO_PRE_COMMIT_FMT_AUTO_STAGE", true)
	cfg.CheckBehaviors.FumptAutoStage = getBoolEnv("GO_PRE_COMMIT_FUMPT_AUTO_STAGE", true)
	cfg.CheckBehaviors.GoimportsAutoStage = getBoolEnv("GO_PRE_COMMIT_GOIMPORTS_AUTO_STAGE", true)
	cfg.CheckBehaviors.WhitespaceAutoStage = getBoolEnv("GO_PRE_COMMIT_WHITESPACE_AUTO_STAGE", true)
	cfg.CheckBehaviors.EOFAutoStage = getBoolEnv("GO_PRE_COMMIT_EOF_AUTO_STAGE", true)
	cfg.CheckBehaviors.AIDetectionAutoFix = getBoolEnv("GO_PRE_COMMIT_AI_DETECTION_AUTO_FIX", false)

	// Tool versions
	cfg.ToolVersions.Fumpt = getStringEnv("GO_PRE_COMMIT_FUMPT_VERSION", "latest")
	cfg.ToolVersions.Goimports = getStringEnv("GO_PRE_COMMIT_GOIMPORTS_VERSION", "latest")
	cfg.ToolVersions.GolangciLint = getStringEnv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION", "latest")
	cfg.ToolVersions.Gitleaks = getStringEnv("GO_PRE_COMMIT_GITLEAKS_VERSION", "v8.29.0")

	// Performance settings
	cfg.Performance.ParallelWorkers = getIntEnv("GO_PRE_COMMIT_PARALLEL_WORKERS", 0) // 0 = auto
	cfg.Performance.FailFast = getBoolEnv("GO_PRE_COMMIT_FAIL_FAST", false)

	// Check timeouts
	cfg.CheckTimeouts.Fmt = getIntEnv("GO_PRE_COMMIT_FMT_TIMEOUT", 30)
	cfg.CheckTimeouts.Fumpt = getIntEnv("GO_PRE_COMMIT_FUMPT_TIMEOUT", 30)
	cfg.CheckTimeouts.Goimports = getIntEnv("GO_PRE_COMMIT_GOIMPORTS_TIMEOUT", 30)
	cfg.CheckTimeouts.Lint = getIntEnv("GO_PRE_COMMIT_LINT_TIMEOUT", 600)
	cfg.CheckTimeouts.ModTidy = getIntEnv("GO_PRE_COMMIT_MOD_TIDY_TIMEOUT", 60)
	cfg.CheckTimeouts.Whitespace = getIntEnv("GO_PRE_COMMIT_WHITESPACE_TIMEOUT", 30)
	cfg.CheckTimeouts.EOF = getIntEnv("GO_PRE_COMMIT_EOF_TIMEOUT", 30)
	cfg.CheckTimeouts.AIDetection = getIntEnv("GO_PRE_COMMIT_AI_DETECTION_TIMEOUT", 30)
	cfg.CheckTimeouts.Gitleaks = getIntEnv("GO_PRE_COMMIT_GITLEAKS_TIMEOUT", 60)

	// Git settings
	cfg.Git.HooksPath = getStringEnv("GO_PRE_COMMIT_HOOKS_PATH", ".git/hooks")
	excludes := getStringEnv("GO_PRE_COMMIT_EXCLUDE_PATTERNS", "vendor/,node_modules/,.git/")
	if excludes != "" {
		cfg.Git.ExcludePatterns = strings.Split(excludes, ",")
		for i := range cfg.Git.ExcludePatterns {
			cfg.Git.ExcludePatterns[i] = strings.TrimSpace(cfg.Git.ExcludePatterns[i])
		}
	}

	// Go module settings
	cfg.Module.GoSumFile = getStringEnv("GO_SUM_FILE", "go.sum")

	// UI settings
	cfg.UI.ColorOutput = getBoolEnv("GO_PRE_COMMIT_COLOR_OUTPUT", true)

	// Tool installation settings
	cfg.ToolInstallation.Timeout = getIntEnv("GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT", 300)

	// Environment detection
	cfg.Environment.IsCI, cfg.Environment.CIProvider = detectCIEnvironment()
	cfg.Environment.AutoAdjustTimers = getBoolEnv("GO_PRE_COMMIT_AUTO_ADJUST_CI_TIMEOUTS", true)

	// Apply CI-specific timeout adjustments if enabled
	if cfg.Environment.IsCI && cfg.Environment.AutoAdjustTimers {
		applyCITimeoutAdjustments(cfg)
	}

	// Plugin settings
	cfg.Plugins.Enabled = getBoolEnv("GO_PRE_COMMIT_ENABLE_PLUGINS", false)
	cfg.Plugins.Directory = getStringEnv("GO_PRE_COMMIT_PLUGIN_DIR", ".pre-commit-plugins")
	cfg.Plugins.Timeout = getIntEnv("GO_PRE_COMMIT_PLUGIN_TIMEOUT", 60)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration and provides helpful error messages
func (c *Config) Validate() error {
	var errors []string

	// Validate timeouts
	if c.Timeout <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_TIMEOUT_SECONDS must be greater than 0")
	}

	if c.ToolInstallation.Timeout <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.Fmt <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_FMT_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.Fumpt <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_FUMPT_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.Lint <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_LINT_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.ModTidy <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_MOD_TIDY_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.Whitespace <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_WHITESPACE_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.EOF <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_EOF_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.AIDetection <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_AI_DETECTION_TIMEOUT must be greater than 0")
	}

	if c.CheckTimeouts.Gitleaks <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_GITLEAKS_TIMEOUT must be greater than 0")
	}

	// Validate file size limits
	if c.MaxFileSize <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_MAX_FILE_SIZE_MB must be greater than 0")
	}

	if c.MaxFilesOpen <= 0 {
		errors = append(errors, "GO_PRE_COMMIT_MAX_FILES_OPEN must be greater than 0")
	}

	// Validate performance settings
	if c.Performance.ParallelWorkers < 0 {
		errors = append(errors, "GO_PRE_COMMIT_PARALLEL_WORKERS must be 0 (auto) or positive")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		errors = append(errors, fmt.Sprintf("GO_PRE_COMMIT_LOG_LEVEL must be one of: trace, debug, info, warn, error (got: '%s')", c.LogLevel))
	}

	// Directory validation no longer needed - using PATH-based binary lookup

	// Validate tool versions
	if c.ToolVersions.Fumpt != "" && c.ToolVersions.Fumpt != "latest" {
		if !isValidVersion(c.ToolVersions.Fumpt) {
			errors = append(errors, "GO_PRE_COMMIT_FUMPT_VERSION must be 'latest' or a valid version (e.g., v0.6.0)")
		}
	}

	if c.ToolVersions.GolangciLint != "" && c.ToolVersions.GolangciLint != "latest" {
		if !isValidVersion(c.ToolVersions.GolangciLint) {
			errors = append(errors, "GO_PRE_COMMIT_GOLANGCI_LINT_VERSION must be 'latest' or a valid version (e.g., v1.55.2)")
		}
	}

	// Validate exclude patterns
	for i, pattern := range c.Git.ExcludePatterns {
		if strings.TrimSpace(pattern) == "" {
			errors = append(errors, fmt.Sprintf("exclude pattern at index %d is empty", i))
		}
	}

	if len(errors) > 0 {
		return &ValidationError{
			Errors: errors,
		}
	}

	return nil
}

// GetModuleDir returns the directory containing go.mod based on GO_SUM_FILE
// Returns empty string if GO_SUM_FILE is "go.sum" (root directory)
// Returns the directory path if GO_SUM_FILE is in a subdirectory (e.g., "lib/go.sum" returns "lib")
func (c *Config) GetModuleDir() string {
	if c.Module.GoSumFile == "" || c.Module.GoSumFile == "go.sum" {
		return ""
	}
	return filepath.Dir(c.Module.GoSumFile)
}

// ValidationError represents configuration validation errors
type ValidationError struct {
	Errors []string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	return fmt.Sprintf("configuration validation failed:\n  - %s", strings.Join(e.Errors, "\n  - "))
}

// isValidVersion checks if a version string is valid
func isValidVersion(version string) bool {
	// Simple validation for semantic versioning
	if strings.HasPrefix(version, "v") && len(version) > 1 {
		// Remove 'v' prefix and check basic format
		version = version[1:]
		parts := strings.Split(version, ".")
		if len(parts) >= 2 {
			return true
		}
	}
	return false
}

// GetConfigHelp returns helpful information about configuration options
func GetConfigHelp() string {
	return `GoFortress Pre-commit System Configuration Help

Environment Variables:

Core Settings:
  ENABLE_GO_PRE_COMMIT=true/false          Enable/disable the pre-commit system
  GO_PRE_COMMIT_LOG_LEVEL=info             Log level (debug, info, warn, error)
  GO_PRE_COMMIT_MAX_FILE_SIZE_MB=10         Maximum file size to process (MB)
  GO_PRE_COMMIT_MAX_FILES_OPEN=100          Maximum files to keep open
  GO_PRE_COMMIT_TIMEOUT_SECONDS=300         Global timeout in seconds
  GO_PRE_COMMIT_TOOL_INSTALL_TIMEOUT=300   Tool installation timeout in seconds
  GO_PRE_COMMIT_AUTO_ADJUST_CI_TIMEOUTS=true   Auto-adjust timeouts for CI environments

Check Configuration:
  GO_PRE_COMMIT_ENABLE_FMT=true             Enable go fmt formatting
  GO_PRE_COMMIT_ENABLE_FUMPT=true           Enable gofumpt formatting
  GO_PRE_COMMIT_ENABLE_LINT=true            Enable golangci-lint
  GO_PRE_COMMIT_ENABLE_MOD_TIDY=true        Enable go mod tidy
  GO_PRE_COMMIT_ENABLE_WHITESPACE=true      Enable whitespace check
  GO_PRE_COMMIT_ENABLE_EOF=true             Enable EOF newline check

Check Behaviors:
  GO_PRE_COMMIT_FMT_AUTO_STAGE=true         Auto-stage files after fmt fixes
  GO_PRE_COMMIT_FUMPT_AUTO_STAGE=true       Auto-stage files after fumpt fixes
  GO_PRE_COMMIT_GOIMPORTS_AUTO_STAGE=true   Auto-stage files after goimports fixes
  GO_PRE_COMMIT_WHITESPACE_AUTO_STAGE=true  Auto-stage files after whitespace fixes
  GO_PRE_COMMIT_EOF_AUTO_STAGE=true         Auto-stage files after EOF fixes

Tool Versions:
  GO_PRE_COMMIT_FUMPT_VERSION=latest        gofumpt version
  GO_PRE_COMMIT_GOLANGCI_LINT_VERSION=latest  golangci-lint version

Performance Settings:
  GO_PRE_COMMIT_PARALLEL_WORKERS=0          Parallel workers (0=auto)
  GO_PRE_COMMIT_FAIL_FAST=false             Stop on first failure

Check Timeouts (seconds):
  GO_PRE_COMMIT_FMT_TIMEOUT=30              go fmt timeout
  GO_PRE_COMMIT_FUMPT_TIMEOUT=30            gofumpt timeout
  GO_PRE_COMMIT_LINT_TIMEOUT=60             golangci-lint timeout
  GO_PRE_COMMIT_MOD_TIDY_TIMEOUT=30         go mod tidy timeout
  GO_PRE_COMMIT_WHITESPACE_TIMEOUT=30       whitespace check timeout
  GO_PRE_COMMIT_EOF_TIMEOUT=30              EOF check timeout

Git Settings:
  GO_PRE_COMMIT_HOOKS_PATH=.git/hooks       Git hooks directory
  GO_PRE_COMMIT_EXCLUDE_PATTERNS="vendor/,node_modules/,.git/"  Exclude patterns

UI Settings:
  GO_PRE_COMMIT_COLOR_OUTPUT=true           Enable colored output

Configuration Methods (auto-detected):

  Modular (preferred): .github/env/*.env
    Files are loaded in lexicographic order (00-core.env, 10-tools.env, 90-project.env).
    Later files override earlier ones (last wins). 99-local.env is skipped in CI (CI=true).

  Legacy: .github/.env.base + optional .github/.env.custom
    Base file is loaded first, custom file overrides base values.

  Detection: If .github/env/ exists with >=1 .env file, modular mode is used.
  Otherwise, falls back to legacy .env.base/.env.custom.

Example .github/env/ (modular):
  00-core.env:
    ENABLE_GO_PRE_COMMIT=true
    GO_PRE_COMMIT_LOG_LEVEL=info

  10-tools.env:
    GO_PRE_COMMIT_FUMPT_VERSION=v0.9.1
    GO_PRE_COMMIT_GOLANGCI_LINT_VERSION=v2.5.0

  90-project.env:
    GO_PRE_COMMIT_LINT_TIMEOUT=120
    GO_PRE_COMMIT_EXCLUDE_PATTERNS="vendor/,node_modules/,.git/"

  99-local.env (git-ignored, skipped in CI):
    GO_PRE_COMMIT_ENABLE_AI_DETECTION=false

Example .github/.env.base (legacy):
  ENABLE_GO_PRE_COMMIT=true
  GO_PRE_COMMIT_ENABLE_FMT=true
  GO_PRE_COMMIT_ENABLE_FUMPT=true
  GO_PRE_COMMIT_LINT_TIMEOUT=120
  GO_PRE_COMMIT_EXCLUDE_PATTERNS="vendor/,node_modules/,.git/,*.tmp,*.log"

Example .github/.env.custom (legacy, optional overrides):
  GO_PRE_COMMIT_LINT_TIMEOUT=180
  GO_PRE_COMMIT_ENABLE_AI_DETECTION=false
`
}

// findBaseEnvFile locates the .github/.env.base file
func findBaseEnvFile() (string, error) {
	// Check for test-specific config directory override (used by integration tests)
	if testConfigDir := os.Getenv("GO_PRE_COMMIT_TEST_CONFIG_DIR"); testConfigDir != "" {
		envPath := filepath.Join(testConfigDir, ".github", ".env.base")
		if _, err := os.Stat(envPath); err == nil { // #nosec G703 - path constructed from config dir env var
			return envPath, nil
		}
		// If test config dir is set but file doesn't exist, don't walk up
		return "", prerrors.ErrEnvFileNotFound
	}

	// First, check if we're already in the right place
	if _, err := os.Stat(".github/.env.base"); err == nil {
		return ".github/.env.base", nil
	}

	// Walk up the directory tree looking for .github/.env.base
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		envPath := filepath.Join(cwd, ".github", ".env.base")
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			// Reached root
			break
		}
		cwd = parent
	}

	return "", prerrors.ErrEnvFileNotFound
}

// findCustomEnvFile locates the .github/.env.custom file in the same directory as the base file
func findCustomEnvFile(basePath string) string {
	baseDir := filepath.Dir(basePath)
	customPath := filepath.Join(baseDir, ".env.custom")
	if _, err := os.Stat(customPath); err == nil {
		return customPath
	}
	return ""
}

// findEnvDir locates .github/env/ directory with modular env files.
// Walks up directory tree (same strategy as findBaseEnvFile).
// Returns path if found with >=1 .env file, or empty string.
func findEnvDir() string {
	// Check for test-specific config directory override
	if testConfigDir := os.Getenv("GO_PRE_COMMIT_TEST_CONFIG_DIR"); testConfigDir != "" {
		envDir := filepath.Join(testConfigDir, ".github", "env")
		if hasEnvFiles(envDir) {
			return envDir
		}
		return ""
	}

	// Check .github/env relative to cwd
	if hasEnvFiles(".github/env") {
		return ".github/env"
	}

	// Walk up directory tree
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		envDir := filepath.Join(cwd, ".github", "env")
		if hasEnvFiles(envDir) {
			return envDir
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}

	return ""
}

// hasEnvFiles checks if dirPath exists, is a directory, and contains >=1 *.env file.
func hasEnvFiles(dirPath string) bool {
	info, err := os.Stat(dirPath) // #nosec G703 - path is validated by caller
	if err != nil || !info.IsDir() {
		return false
	}
	matches, err := filepath.Glob(filepath.Join(dirPath, "*.env"))
	if err != nil {
		return false
	}
	return len(matches) > 0
}

// isCI returns true if CI environment variable equals "true"
func isCI() bool {
	return os.Getenv("CI") == "true"
}

// Helper functions for environment variable parsing
func getBoolEnv(key string, defaultValue bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return b
}

// detectCIEnvironment detects if we're running in a CI environment and which provider
func detectCIEnvironment() (bool, string) {
	// Common CI environment variables and their providers
	ciEnvs := map[string]string{
		"GITHUB_ACTIONS":        "github-actions",
		"GITLAB_CI":             "gitlab",
		"JENKINS_URL":           "jenkins",
		"BUILDKITE":             "buildkite",
		"CIRCLECI":              "circleci",
		"TRAVIS":                "travis",
		"APPVEYOR":              "appveyor",
		"AZURE_HTTP_USER_AGENT": "azure-devops",
		"TEAMCITY_VERSION":      "teamcity",
		"DRONE":                 "drone",
		"SEMAPHORE":             "semaphore",
		"CODEBUILD_BUILD_ID":    "aws-codebuild",
	}

	for envVar, provider := range ciEnvs {
		if os.Getenv(envVar) != "" {
			return true, provider
		}
	}

	// Generic CI detection
	if os.Getenv("CI") != "" {
		return true, "unknown"
	}

	return false, ""
}

// applyCITimeoutAdjustments adjusts timeouts for CI environments where network and disk I/O may be slower
func applyCITimeoutAdjustments(cfg *Config) {
	// Increase tool installation timeout in CI (network downloads can be slow)
	if cfg.ToolInstallation.Timeout == 300 { // Only adjust if using default
		cfg.ToolInstallation.Timeout = 600 // 10 minutes for CI
	}

	// Increase global timeout if using default (updated for new 720s default)
	if cfg.Timeout == 720 {
		cfg.Timeout = 1440 // 24 minutes for CI (2x)
	}

	// Increase lint timeout as it's often the slowest check (updated for new 600s default)
	if cfg.CheckTimeouts.Lint == 600 { // Only adjust if using default
		cfg.CheckTimeouts.Lint = 1800 // 30 minutes for lint in CI (3x)
	}

	// Slightly increase other check timeouts
	adjustTimeout := func(current *int, defaultVal, newVal int) {
		if *current == defaultVal {
			*current = newVal
		}
	}

	adjustTimeout(&cfg.CheckTimeouts.Fmt, 30, 60)
	adjustTimeout(&cfg.CheckTimeouts.Fumpt, 30, 60)
	adjustTimeout(&cfg.CheckTimeouts.Goimports, 30, 60)
	adjustTimeout(&cfg.CheckTimeouts.ModTidy, 60, 180)
	adjustTimeout(&cfg.CheckTimeouts.Whitespace, 30, 45)
	adjustTimeout(&cfg.CheckTimeouts.EOF, 30, 45)
	adjustTimeout(&cfg.CheckTimeouts.AIDetection, 30, 60)
}

func getIntEnv(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	// Validate that the value is within 32-bit signed integer range
	// This prevents extremely large values that could cause issues
	if i < -2147483648 || i > 2147483647 {
		return defaultValue
	}
	return i
}

func getStringEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return stripComments(val)
}

// stripComments removes inline comments from environment variable values
func stripComments(value string) string {
	if idx := strings.Index(value, "#"); idx != -1 {
		// Only strip if the # appears to be a comment (preceded by whitespace or at start)
		if idx == 0 || strings.TrimSpace(value[:idx]) != value[:idx] {
			value = value[:idx]
			return strings.TrimSpace(value)
		}
	}
	// No comment found or # is part of the value, return as-is
	return value
}
