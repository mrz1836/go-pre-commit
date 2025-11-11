// Package golangci provides utilities for reading and parsing golangci-lint configuration files.
package golangci

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the relevant portions of golangci-lint configuration.
type Config struct {
	Formatters struct {
		Settings struct {
			Gofumpt struct {
				ModulePath string `json:"module-path" yaml:"module-path"`
			} `json:"gofumpt" yaml:"gofumpt"`
		} `json:"settings" yaml:"settings"`
	} `json:"formatters" yaml:"formatters"`
}

var (
	// ErrNoConfigFound is returned when no golangci-lint config file is found.
	ErrNoConfigFound = errors.New("no golangci-lint config file found")
	// ErrNoModulePath is returned when module-path is not set in the config.
	ErrNoModulePath = errors.New("module-path not set in golangci-lint config")
	// ErrGoModNotFound is returned when go.mod is not found at the specified path.
	ErrGoModNotFound = errors.New("go.mod not found")
	// ErrModuleDirectiveNotFound is returned when module directive is not found in go.mod.
	ErrModuleDirectiveNotFound = errors.New("module directive not found in go.mod")
)

// ReadGofumptModulePath attempts to read the gofumpt module-path setting from golangci-lint configuration.
// It tries the following in order:
//  1. .golangci.json
//  2. .golangci.yml
//  3. .golangci.yaml
//  4. Fallback to parsing go.mod
//
// Returns the module path if found, or an error if not.
func ReadGofumptModulePath(repoRoot string) (string, error) {
	// Try JSON config first
	if modulePath, err := tryReadConfig(filepath.Join(repoRoot, ".golangci.json"), parseJSONConfig); err == nil {
		return modulePath, nil
	}

	// Try YAML configs
	for _, filename := range []string{".golangci.yml", ".golangci.yaml"} {
		if modulePath, err := tryReadConfig(filepath.Join(repoRoot, filename), parseYAMLConfig); err == nil {
			return modulePath, nil
		}
	}

	// Fallback: try to parse go.mod
	return parseGoMod(filepath.Join(repoRoot, "go.mod"))
}

// tryReadConfig attempts to read and parse a config file using the provided parser.
func tryReadConfig(path string, parser func(string) (string, error)) (string, error) {
	if !fileExists(path) {
		return "", ErrNoConfigFound
	}
	return parser(path)
}

// parseJSONConfig reads a golangci-lint JSON config file and extracts the module-path.
func parseJSONConfig(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- Path is validated by caller
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse JSON config: %w", err)
	}

	modulePath := config.Formatters.Settings.Gofumpt.ModulePath
	if modulePath == "" {
		return "", ErrNoModulePath
	}

	return modulePath, nil
}

// parseYAMLConfig reads a golangci-lint YAML config file and extracts the module-path.
func parseYAMLConfig(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- Path is validated by caller
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse YAML config: %w", err)
	}

	modulePath := config.Formatters.Settings.Gofumpt.ModulePath
	if modulePath == "" {
		return "", ErrNoModulePath
	}

	return modulePath, nil
}

// parseGoMod reads the module path from a go.mod file as a fallback.
// The module path is specified on the first line as "module <path>".
func parseGoMod(path string) (string, error) {
	if !fileExists(path) {
		return "", fmt.Errorf("%w at %s", ErrGoModNotFound, path)
	}

	data, err := os.ReadFile(path) // #nosec G304 -- Path is validated by caller
	if err != nil {
		return "", fmt.Errorf("failed to read go.mod: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			modulePath := strings.TrimPrefix(line, "module ")
			modulePath = strings.TrimSpace(modulePath)
			if modulePath != "" {
				return modulePath, nil
			}
		}
	}

	return "", ErrModuleDirectiveNotFound
}

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
