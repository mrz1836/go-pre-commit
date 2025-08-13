// Package plugins provides a plugin system for custom pre-commit hooks
package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	prerrors "github.com/mrz1836/go-pre-commit/internal/errors"
)

// Define plugin errors
var (
	ErrManifestNil     = errors.New("manifest cannot be nil")
	ErrPluginNameEmpty = errors.New("plugin name is required")
	ErrExecutableEmpty = errors.New("plugin executable is required")
)

// PluginManifest defines the structure of a plugin configuration file
type PluginManifest struct {
	// Basic metadata
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
	Author      string `json:"author,omitempty" yaml:"author,omitempty"`
	License     string `json:"license,omitempty" yaml:"license,omitempty"`
	Homepage    string `json:"homepage,omitempty" yaml:"homepage,omitempty"`

	// Execution configuration
	Executable    string            `json:"executable" yaml:"executable"`
	Args          []string          `json:"args,omitempty" yaml:"args,omitempty"`
	FilePatterns  []string          `json:"file_patterns" yaml:"file_patterns"`
	Timeout       string            `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Category      string            `json:"category,omitempty" yaml:"category,omitempty"`
	RequiresFiles bool              `json:"requires_files,omitempty" yaml:"requires_files,omitempty"`
	Environment   map[string]string `json:"environment,omitempty" yaml:"environment,omitempty"`

	// Advanced configuration
	Dependencies []string `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	MinVersion   string   `json:"min_go_pre_commit_version,omitempty" yaml:"min_go_pre_commit_version,omitempty"`
	MaxVersion   string   `json:"max_go_pre_commit_version,omitempty" yaml:"max_go_pre_commit_version,omitempty"`

	// Security settings
	ReadOnly      bool     `json:"read_only,omitempty" yaml:"read_only,omitempty"`
	AllowedPaths  []string `json:"allowed_paths,omitempty" yaml:"allowed_paths,omitempty"`
	MaxMemoryMB   int      `json:"max_memory_mb,omitempty" yaml:"max_memory_mb,omitempty"`
	MaxCPUPercent int      `json:"max_cpu_percent,omitempty" yaml:"max_cpu_percent,omitempty"`
}

// Plugin represents a custom check implemented as an external executable
type Plugin struct {
	manifest  *PluginManifest
	directory string
	timeout   time.Duration
}

// NewPlugin creates a new plugin from a manifest and directory
func NewPlugin(manifest *PluginManifest, directory string) (*Plugin, error) {
	if manifest == nil {
		return nil, ErrManifestNil
	}

	if manifest.Name == "" {
		return nil, ErrPluginNameEmpty
	}

	if manifest.Executable == "" {
		return nil, ErrExecutableEmpty
	}

	// Parse timeout
	timeout := 30 * time.Second // default
	if manifest.Timeout != "" {
		parsedTimeout, err := time.ParseDuration(manifest.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout format: %w", err)
		}
		timeout = parsedTimeout
	}

	return &Plugin{
		manifest:  manifest,
		directory: directory,
		timeout:   timeout,
	}, nil
}

// Name returns the name of the plugin
func (p *Plugin) Name() string {
	return p.manifest.Name
}

// Description returns a brief description of the plugin
func (p *Plugin) Description() string {
	return p.manifest.Description
}

// Metadata returns comprehensive metadata about the plugin
func (p *Plugin) Metadata() interface{} {
	return PluginMetadata{
		Name:              p.manifest.Name,
		Description:       p.manifest.Description,
		FilePatterns:      p.manifest.FilePatterns,
		EstimatedDuration: p.timeout,
		Dependencies:      p.manifest.Dependencies,
		DefaultTimeout:    p.timeout,
		Category:          p.manifest.Category,
		RequiresFiles:     p.manifest.RequiresFiles,
		Version:           p.manifest.Version,
		Author:            p.manifest.Author,
	}
}

// Run executes the plugin on the given files
func (p *Plugin) Run(ctx context.Context, files []string) error {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// Prepare executable path
	execPath := p.manifest.Executable
	if !filepath.IsAbs(execPath) {
		execPath = filepath.Join(p.directory, execPath)
	}

	// Check if executable exists
	if _, err := os.Stat(execPath); err != nil {
		return prerrors.NewToolNotFoundError(
			p.manifest.Name,
			fmt.Sprintf("Plugin executable not found: %s", execPath),
		)
	}

	// Build command arguments
	args := append([]string{}, p.manifest.Args...)

	// Create command
	cmd := exec.CommandContext(ctx, execPath, args...) //nolint:gosec // Plugin execution
	cmd.Dir = p.directory

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range p.manifest.Environment {
		// Expand environment variables in value
		expandedValue := os.ExpandEnv(value)
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, expandedValue))
	}

	// Create plugin request
	request := PluginRequest{
		Files:   files,
		Command: "check",
		Config:  p.manifest.Environment,
	}

	// Marshal request to JSON
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal plugin request: %w", err)
	}

	// Set up stdin with the request
	cmd.Stdin = bytes.NewReader(requestJSON)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the plugin
	if err := cmd.Run(); err != nil {
		// Check if it's a context timeout
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return prerrors.NewToolExecutionError(
				p.manifest.Name,
				stderr.String(),
				fmt.Sprintf("Plugin timed out after %v", p.timeout),
			)
		}

		// Try to parse plugin response for structured error
		var response PluginResponse
		if jsonErr := json.Unmarshal(stdout.Bytes(), &response); jsonErr == nil && response.Error != "" {
			return prerrors.NewToolExecutionError(
				p.manifest.Name,
				response.Error,
				response.Suggestion,
			)
		}

		// Generic error
		return prerrors.NewToolExecutionError(
			p.manifest.Name,
			stderr.String(),
			fmt.Sprintf("Plugin failed with exit code: %v", err),
		)
	}

	// Parse successful response
	var response PluginResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		// If the plugin doesn't use JSON protocol, treat empty output as success
		if len(stdout.Bytes()) == 0 {
			return nil
		}
		// Non-JSON output is treated as an error message
		if strings.TrimSpace(stdout.String()) != "" {
			return prerrors.NewToolExecutionError(
				p.manifest.Name,
				stdout.String(),
				"Plugin output was not in expected JSON format",
			)
		}
	}

	// Check if plugin reported an error
	if !response.Success {
		return prerrors.NewToolExecutionError(
			p.manifest.Name,
			response.Error,
			response.Suggestion,
		)
	}

	return nil
}

// FilterFiles filters the list of files to only those this plugin should process
func (p *Plugin) FilterFiles(files []string) []string {
	if len(p.manifest.FilePatterns) == 0 {
		return files // No filtering if no patterns specified
	}

	var filtered []string
	for _, file := range files {
		for _, pattern := range p.manifest.FilePatterns {
			// Simple pattern matching (could be enhanced with glob support)
			if strings.HasPrefix(pattern, "*.") {
				// Extension matching
				ext := pattern[1:] // Remove the *
				if strings.HasSuffix(file, ext) {
					filtered = append(filtered, file)
					break
				}
			} else if matched, _ := filepath.Match(pattern, file); matched {
				// Glob pattern matching
				filtered = append(filtered, file)
				break
			}
		}
	}
	return filtered
}

// PluginMetadata contains metadata about a plugin
type PluginMetadata struct {
	Name              string
	Description       string
	FilePatterns      []string
	EstimatedDuration time.Duration
	Dependencies      []string
	DefaultTimeout    time.Duration
	Category          string
	RequiresFiles     bool
	Version           string
	Author            string
}

// PluginRequest is the JSON structure sent to plugins via stdin
type PluginRequest struct {
	Command string            `json:"command"`
	Files   []string          `json:"files"`
	Config  map[string]string `json:"config,omitempty"`
}

// PluginResponse is the JSON structure expected from plugins via stdout
type PluginResponse struct {
	Success    bool     `json:"success"`
	Error      string   `json:"error,omitempty"`
	Suggestion string   `json:"suggestion,omitempty"`
	Modified   []string `json:"modified,omitempty"`
	Output     string   `json:"output,omitempty"`
}
