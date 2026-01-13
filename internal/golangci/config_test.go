package golangci

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadGofumptModulePath_JSON(t *testing.T) {
	// Create temp directory with test config
	tmpDir := t.TempDir()

	// Test JSON config
	jsonConfig := `{
		"formatters": {
			"settings": {
				"gofumpt": {
					"module-path": "example.com/myproject"
				}
			}
		}
	}`

	configPath := filepath.Join(tmpDir, ".golangci.json")
	// #nosec G306 -- Test file, 0644 is acceptable
	if err := os.WriteFile(configPath, []byte(jsonConfig), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	modulePath, err := ReadGofumptModulePath(tmpDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "example.com/myproject"
	if modulePath != expected {
		t.Errorf("Expected module path %q, got %q", expected, modulePath)
	}
}

func TestReadGofumptModulePath_YAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Test YAML config
	yamlConfig := `formatters:
  settings:
    gofumpt:
      module-path: example.com/yamlproject
`

	configPath := filepath.Join(tmpDir, ".golangci.yml")
	// #nosec G306 -- Test file, 0644 is acceptable
	if err := os.WriteFile(configPath, []byte(yamlConfig), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	modulePath, err := ReadGofumptModulePath(tmpDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "example.com/yamlproject"
	if modulePath != expected {
		t.Errorf("Expected module path %q, got %q", expected, modulePath)
	}
}

func TestReadGofumptModulePath_FallbackToGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod without golangci-lint config
	goModContent := `module github.com/example/fallback

go 1.21
`

	goModPath := filepath.Join(tmpDir, "go.mod")
	// #nosec G306 -- Test file, 0644 is acceptable
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	modulePath, err := ReadGofumptModulePath(tmpDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "github.com/example/fallback"
	if modulePath != expected {
		t.Errorf("Expected module path %q, got %q", expected, modulePath)
	}
}

func TestReadGofumptModulePath_NoConfigFound(t *testing.T) {
	tmpDir := t.TempDir()

	// No config files, no go.mod
	_, err := ReadGofumptModulePath(tmpDir)
	if err == nil {
		t.Fatal("Expected error when no config found, got nil")
	}
}

func TestReadGofumptModulePath_EmptyModulePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Config with empty module-path
	jsonConfig := `{
		"formatters": {
			"settings": {
				"gofumpt": {
					"module-path": ""
				}
			}
		}
	}`

	configPath := filepath.Join(tmpDir, ".golangci.json")
	// #nosec G306 -- Test file, 0644 is acceptable
	if err := os.WriteFile(configPath, []byte(jsonConfig), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Should fallback to go.mod
	goModContent := `module github.com/example/empty

go 1.21
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	// #nosec G306 -- Test file, 0644 is acceptable
	if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	modulePath, err := ReadGofumptModulePath(tmpDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	expected := "github.com/example/empty"
	if modulePath != expected {
		t.Errorf("Expected fallback to go.mod, got %q", modulePath)
	}
}

func TestParseGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		goModContent string
		expected     string
		expectError  bool
	}{
		{
			name: "simple module",
			goModContent: `module example.com/simple

go 1.21
`,
			expected:    "example.com/simple",
			expectError: false,
		},
		{
			name: "module with comments",
			goModContent: `// This is a comment
module github.com/user/project

go 1.21
`,
			expected:    "github.com/user/project",
			expectError: false,
		},
		{
			name: "no module directive",
			goModContent: `go 1.21

require (
	example.com/dep v1.0.0
)
`,
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goModPath := filepath.Join(tmpDir, "go.mod")
			// #nosec G306 -- Test file, 0644 is acceptable
			if err := os.WriteFile(goModPath, []byte(tt.goModContent), 0o644); err != nil {
				t.Fatalf("Failed to write go.mod: %v", err)
			}

			modulePath, err := parseGoMod(goModPath)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if modulePath != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, modulePath)
			}

			// Clean up for next iteration
			if err := os.Remove(goModPath); err != nil {
				t.Logf("Warning: failed to remove test file: %v", err)
			}
		})
	}
}

func TestReadGofumptModulePath_PreferenceOrder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create all config types
	jsonConfig := `{
		"formatters": {
			"settings": {
				"gofumpt": {
					"module-path": "json.example.com/project"
				}
			}
		}
	}`

	yamlConfig := `formatters:
  settings:
    gofumpt:
      module-path: yaml.example.com/project
`

	goModContent := `module gomod.example.com/project

go 1.21
`

	// Write all files
	// #nosec G306 -- Test file, 0644 is acceptable
	if err := os.WriteFile(filepath.Join(tmpDir, ".golangci.json"), []byte(jsonConfig), 0o644); err != nil {
		t.Fatalf("Failed to write JSON config: %v", err)
	}
	// #nosec G306 -- Test file, 0644 is acceptable
	if err := os.WriteFile(filepath.Join(tmpDir, ".golangci.yml"), []byte(yamlConfig), 0o644); err != nil {
		t.Fatalf("Failed to write YAML config: %v", err)
	}
	// #nosec G306 -- Test file, 0644 is acceptable
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0o644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	modulePath, err := ReadGofumptModulePath(tmpDir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should prefer .golangci.json over .yml over go.mod
	expected := "json.example.com/project"
	if modulePath != expected {
		t.Errorf("Expected JSON config to take precedence, got %q instead of %q", modulePath, expected)
	}
}

// TestParseJSONConfig_InvalidJSON tests error handling for malformed JSON
func TestParseJSONConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Write invalid JSON
	invalidJSON := `{
		"formatters": {
			"settings": {
				"gofumpt": {
					"module-path": "test
				}
			}
		}
	}` // Missing closing quote

	configPath := filepath.Join(tmpDir, ".golangci.json")
	// #nosec G306 -- Test file, 0644 is acceptable
	if err := os.WriteFile(configPath, []byte(invalidJSON), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := parseJSONConfig(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse JSON config") {
		t.Errorf("Expected JSON parse error, got: %v", err)
	}
}

// TestParseJSONConfig_ReadError tests error handling when file cannot be read
func TestParseJSONConfig_ReadError(t *testing.T) {
	// Try to read a non-existent file
	_, err := parseJSONConfig("/nonexistent/path/.golangci.json")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("Expected read error, got: %v", err)
	}
}

// TestParseYAMLConfig_InvalidYAML tests error handling for malformed YAML
func TestParseYAMLConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Write invalid YAML (tabs are not allowed in YAML indentation)
	invalidYAML := `formatters:
	settings:
		gofumpt:
			module-path: test
`

	configPath := filepath.Join(tmpDir, ".golangci.yml")
	// #nosec G306 -- Test file, 0644 is acceptable
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := parseYAMLConfig(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse YAML config") {
		t.Errorf("Expected YAML parse error, got: %v", err)
	}
}

// TestParseYAMLConfig_ReadError tests error handling when file cannot be read
func TestParseYAMLConfig_ReadError(t *testing.T) {
	// Try to read a non-existent file
	_, err := parseYAMLConfig("/nonexistent/path/.golangci.yml")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("Expected read error, got: %v", err)
	}
}

// TestParseGoMod_ReadError tests error handling when go.mod cannot be read
func TestParseGoMod_ReadError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory with the go.mod name to cause read error
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.Mkdir(goModPath, 0o750); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	_, err := parseGoMod(goModPath)
	if err == nil {
		t.Fatal("Expected error when reading directory as file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read go.mod") {
		t.Errorf("Expected read error, got: %v", err)
	}
}
