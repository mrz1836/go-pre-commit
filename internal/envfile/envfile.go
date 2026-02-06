// Package envfile provides utilities for loading environment variables from .env files
package envfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Sentinel errors for LoadDir
var (
	// ErrNotDirectory is returned when the provided path is not a directory
	ErrNotDirectory = errors.New("path is not a directory")

	// ErrNoEnvFiles is returned when no .env files are found in the directory
	ErrNoEnvFiles = errors.New("no .env files found")
)

// Load reads environment variables from a file and sets them in the current environment.
// It does not override existing environment variables.
func Load(filename string) error {
	return loadFile(filename, false)
}

// Overload reads environment variables from a file and sets them in the current environment.
// It overrides existing environment variables.
func Overload(filename string) error {
	return loadFile(filename, true)
}

// LoadDir loads all *.env files from dirPath in lexicographic sort order.
// Each file overrides variables set by previous files (last wins).
// If skipLocal is true, 99-local.env is skipped (CI environments).
func LoadDir(dirPath string, skipLocal bool) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("env directory not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s", ErrNotDirectory, dirPath)
	}

	matches, err := filepath.Glob(filepath.Join(dirPath, "*.env"))
	if err != nil {
		return fmt.Errorf("failed to glob env files in %s: %w", dirPath, err)
	}

	sort.Strings(matches)

	loaded := 0
	for _, envFile := range matches {
		if skipLocal && filepath.Base(envFile) == "99-local.env" {
			continue
		}
		if err := Overload(envFile); err != nil {
			return fmt.Errorf("failed to load %s: %w", envFile, err)
		}
		loaded++
	}

	if loaded == 0 {
		return fmt.Errorf("%w in %s", ErrNoEnvFiles, dirPath)
	}

	return nil
}

// loadFile reads a .env file and sets environment variables
// If overload is true, existing environment variables are overwritten
func loadFile(filename string, overload bool) error {
	// #nosec G304 - filename is provided by the caller, intentional file read
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Parse the file content
	envMap := parse(string(data))

	// Set environment variables
	for key, value := range envMap {
		// Skip if variable already exists and we're not overloading
		if !overload && os.Getenv(key) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}

	return nil
}

// parse parses .env file content into a map of key-value pairs
// It handles:
// - Empty lines
// - Comment lines starting with #
// - Inline comments (e.g., KEY=value # comment)
// - Quoted values (e.g., KEY="value with spaces")
// - Unquoted values
// - Malformed lines are skipped (tolerant parsing)
func parse(content string) map[string]string {
	envMap := make(map[string]string)

	// Process line by line using manual splitting to avoid bufio.Scanner overhead
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip comment lines
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Find the first '=' to split key and value
		eqIndex := strings.Index(line, "=")
		if eqIndex == -1 {
			// No '=' found - skip this line (tolerant parsing like godotenv)
			continue
		}

		// Extract key (trim whitespace)
		key := strings.TrimSpace(line[:eqIndex])
		if key == "" {
			// Empty key - skip this line
			continue
		}

		// Extract value (everything after '=')
		value := ""
		if eqIndex+1 < len(line) {
			value = line[eqIndex+1:]
		}

		// Process the value
		value = processValue(value)

		// Store in map
		envMap[key] = value

		// Avoid unused variable warning in case we add line number error reporting later
		_ = lineNum
	}

	return envMap
}

// processValue processes a value string by:
// - Stripping inline comments
// - Handling quoted values
// - Trimming whitespace
func processValue(value string) string {
	// Trim all leading whitespace (handles all Unicode whitespace)
	value = strings.TrimLeftFunc(value, isWhitespace)

	// Handle quoted values
	if len(value) >= 2 {
		// Check for double quotes
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			// Remove surrounding quotes
			return value[1 : len(value)-1]
		}
		// Check for single quotes
		if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
			// Remove surrounding quotes
			return value[1 : len(value)-1]
		}
	}

	// For unquoted values, strip inline comments
	// Only strip if # is preceded by at least one space or tab
	if idx := strings.Index(value, " #"); idx != -1 {
		value = value[:idx]
	} else if idx := strings.Index(value, "\t#"); idx != -1 {
		value = value[:idx]
	}

	// Trim all trailing whitespace (handles all Unicode whitespace)
	value = strings.TrimRightFunc(value, isWhitespace)

	return value
}

// isWhitespace checks if a rune is a whitespace character
// This includes all ASCII and Unicode whitespace
func isWhitespace(r rune) bool {
	// Check ASCII whitespace
	if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f' {
		return true
	}
	// Check Unicode whitespace categories
	// This includes: U+0085 (NEL), U+00A0 (NBSP), U+1680, U+2000-U+200A, U+2028, U+2029, U+202F, U+205F, U+3000
	return r == 0x85 || r == 0xA0 || r == 0x1680 ||
		(r >= 0x2000 && r <= 0x200A) || r == 0x2028 || r == 0x2029 ||
		r == 0x202F || r == 0x205F || r == 0x3000
}
