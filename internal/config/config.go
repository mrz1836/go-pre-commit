// Package config provides configuration loading for the GoFortress pre-commit system
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

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
		Fmt         bool // GO_PRE_COMMIT_ENABLE_FMT
		Fumpt       bool // GO_PRE_COMMIT_ENABLE_FUMPT
		Goimports   bool // GO_PRE_COMMIT_ENABLE_GOIMPORTS
		Lint        bool // GO_PRE_COMMIT_ENABLE_LINT
		ModTidy     bool // GO_PRE_COMMIT_ENABLE_MOD_TIDY
		Whitespace  bool // GO_PRE_COMMIT_ENABLE_WHITESPACE
		EOF         bool // GO_PRE_COMMIT_ENABLE_EOF
		AIDetection bool // GO_PRE_COMMIT_ENABLE_AI_DETECTION
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
	}

	// Git settings
	Git struct {
		HooksPath       string   // GO_PRE_COMMIT_HOOKS_PATH (default: .git/hooks)
		ExcludePatterns []string // GO_PRE_COMMIT_EXCLUDE_PATTERNS
	}

	// UI settings
	UI struct {
		ColorOutput bool // GO_PRE_COMMIT_COLOR_OUTPUT (default: true)
	}

	// Plugin settings
	Plugins struct {
		Enabled   bool   // GO_PRE_COMMIT_ENABLE_PLUGINS
		Directory string // GO_PRE_COMMIT_PLUGIN_DIR
		Timeout   int    // GO_PRE_COMMIT_PLUGIN_TIMEOUT
	}
}

// Load reads configuration from .github/.env.base and .github/.env.custom
func Load() (*Config, error) {
	// Find .env.base file (required)
	basePath, err := findBaseEnvFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find .env.base: %w", err)
	}

	// Load base environment file
	if err := godotenv.Load(basePath); err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", basePath, err)
	}

	// Load custom environment file if it exists (overrides base)
	customPath := findCustomEnvFile(basePath)
	if customPath != "" {
		if err := godotenv.Overload(customPath); err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", customPath, err)
		}
	}

	cfg := &Config{
		Directory: "", // No longer using directory-based approach
	}

	// Core settings
	cfg.Enabled = getBoolEnv("ENABLE_GO_PRE_COMMIT", false)
	cfg.LogLevel = getStringEnv("GO_PRE_COMMIT_LOG_LEVEL", "info")
	cfg.MaxFileSize = int64(getIntEnv("GO_PRE_COMMIT_MAX_FILE_SIZE_MB", 10)) * 1024 * 1024
	cfg.MaxFilesOpen = getIntEnv("GO_PRE_COMMIT_MAX_FILES_OPEN", 100)
	cfg.Timeout = getIntEnv("GO_PRE_COMMIT_TIMEOUT_SECONDS", 300)

	// Check configurations
	cfg.Checks.Fmt = getBoolEnv("GO_PRE_COMMIT_ENABLE_FMT", true)
	cfg.Checks.Fumpt = getBoolEnv("GO_PRE_COMMIT_ENABLE_FUMPT", true)
	cfg.Checks.Goimports = getBoolEnv("GO_PRE_COMMIT_ENABLE_GOIMPORTS", true)
	cfg.Checks.Lint = getBoolEnv("GO_PRE_COMMIT_ENABLE_LINT", true)
	cfg.Checks.ModTidy = getBoolEnv("GO_PRE_COMMIT_ENABLE_MOD_TIDY", true)
	cfg.Checks.Whitespace = getBoolEnv("GO_PRE_COMMIT_ENABLE_WHITESPACE", true)
	cfg.Checks.EOF = getBoolEnv("GO_PRE_COMMIT_ENABLE_EOF", true)
	cfg.Checks.AIDetection = getBoolEnv("GO_PRE_COMMIT_ENABLE_AI_DETECTION", true)

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

	// Performance settings
	cfg.Performance.ParallelWorkers = getIntEnv("GO_PRE_COMMIT_PARALLEL_WORKERS", 0) // 0 = auto
	cfg.Performance.FailFast = getBoolEnv("GO_PRE_COMMIT_FAIL_FAST", false)

	// Check timeouts
	cfg.CheckTimeouts.Fmt = getIntEnv("GO_PRE_COMMIT_FMT_TIMEOUT", 30)
	cfg.CheckTimeouts.Fumpt = getIntEnv("GO_PRE_COMMIT_FUMPT_TIMEOUT", 30)
	cfg.CheckTimeouts.Goimports = getIntEnv("GO_PRE_COMMIT_GOIMPORTS_TIMEOUT", 30)
	cfg.CheckTimeouts.Lint = getIntEnv("GO_PRE_COMMIT_LINT_TIMEOUT", 60)
	cfg.CheckTimeouts.ModTidy = getIntEnv("GO_PRE_COMMIT_MOD_TIDY_TIMEOUT", 30)
	cfg.CheckTimeouts.Whitespace = getIntEnv("GO_PRE_COMMIT_WHITESPACE_TIMEOUT", 30)
	cfg.CheckTimeouts.EOF = getIntEnv("GO_PRE_COMMIT_EOF_TIMEOUT", 30)
	cfg.CheckTimeouts.AIDetection = getIntEnv("GO_PRE_COMMIT_AI_DETECTION_TIMEOUT", 30)

	// Git settings
	cfg.Git.HooksPath = getStringEnv("GO_PRE_COMMIT_HOOKS_PATH", ".git/hooks")
	excludes := getStringEnv("GO_PRE_COMMIT_EXCLUDE_PATTERNS", "vendor/,node_modules/,.git/")
	if excludes != "" {
		cfg.Git.ExcludePatterns = strings.Split(excludes, ",")
		for i := range cfg.Git.ExcludePatterns {
			cfg.Git.ExcludePatterns[i] = strings.TrimSpace(cfg.Git.ExcludePatterns[i])
		}
	}

	// UI settings
	cfg.UI.ColorOutput = getBoolEnv("GO_PRE_COMMIT_COLOR_OUTPUT", true)

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
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[strings.ToLower(c.LogLevel)] {
		errors = append(errors, "GO_PRE_COMMIT_LOG_LEVEL must be one of: debug, info, warn, error")
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

Example .github/.env.base:
  # Enable the system
  ENABLE_GO_PRE_COMMIT=true

  # Configure checks
  GO_PRE_COMMIT_ENABLE_FMT=true
  GO_PRE_COMMIT_ENABLE_FUMPT=true
  GO_PRE_COMMIT_ENABLE_LINT=true
  GO_PRE_COMMIT_ENABLE_MOD_TIDY=true

  # Set timeouts
  GO_PRE_COMMIT_FUMPT_TIMEOUT=30
  GO_PRE_COMMIT_LINT_TIMEOUT=120

  # Exclude patterns
  GO_PRE_COMMIT_EXCLUDE_PATTERNS="vendor/,node_modules/,.git/,*.tmp,*.log"

Example .github/.env.custom (optional overrides):
  # Override timeout for your project
  GO_PRE_COMMIT_LINT_TIMEOUT=180

  # Disable specific checks
  GO_PRE_COMMIT_ENABLE_AI_DETECTION=false
`
}

// findBaseEnvFile locates the .github/.env.base file
func findBaseEnvFile() (string, error) {
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
	return val
}
