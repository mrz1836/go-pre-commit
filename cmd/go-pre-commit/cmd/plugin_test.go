package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mrz1836/go-pre-commit/internal/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadManifestFromDir(t *testing.T) {
	testCases := []struct {
		name          string
		setupFunc     func(t *testing.T) string
		expectedError string
		expectedName  string
		cleanup       bool
	}{
		{
			name: "valid YAML manifest",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				yamlContent := `name: test-plugin
version: "1.0.0"
description: "Test plugin for unit testing"
author: "Test Author"
category: "testing"
file_patterns:
  - "*.test"
  - "*.spec"
executable: "./test-check.sh"
args: []
timeout: "30s"
requires_files: true
dependencies:
  - "bash"
  - "grep"`
				err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(yamlContent), 0o600)
				require.NoError(t, err)
				return tmpDir
			},
			expectedName: "test-plugin",
			cleanup:      true,
		},
		{
			name: "valid JSON manifest",
			setupFunc: func(_ *testing.T) string {
				// Create temp dir with only JSON manifest
				tmpDir := t.TempDir()
				jsonContent := `{
  "name": "json-only-plugin",
  "version": "1.0.0",
  "description": "JSON-only test plugin",
  "author": "Test Author",
  "category": "testing",
  "file_patterns": ["*.json"],
  "command": ["./check.sh"],
  "default_timeout": 30,
  "requires_files": true
}`
				err := os.WriteFile(filepath.Join(tmpDir, "plugin.json"), []byte(jsonContent), 0o600)
				require.NoError(t, err)
				return tmpDir
			},
			expectedName: "json-only-plugin",
			cleanup:      true,
		},
		{
			name: "valid YML extension manifest",
			setupFunc: func(_ *testing.T) string {
				tmpDir := t.TempDir()
				ymlContent := `name: yml-plugin
version: "1.0.0"
description: "YML extension test plugin"
author: "Test Author"
category: "testing"
file_patterns:
  - "*.yml"
command: ["./check.sh"]
default_timeout: 30
requires_files: true`
				err := os.WriteFile(filepath.Join(tmpDir, "plugin.yml"), []byte(ymlContent), 0o600)
				require.NoError(t, err)
				return tmpDir
			},
			expectedName: "yml-plugin",
			cleanup:      true,
		},
		{
			name: "no manifest files",
			setupFunc: func(_ *testing.T) string {
				tmpDir := t.TempDir()
				// Create some other files but no manifest
				err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0o600)
				require.NoError(t, err)
				return tmpDir
			},
			expectedError: "no manifest file found",
			cleanup:       true,
		},
		{
			name: "malformed YAML",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				malformedYAML := `name: malformed-plugin
version: 1.0.0
description: "Plugin with malformed YAML"
invalid_field: [unclosed_bracket
category: "test"`
				err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(malformedYAML), 0o600)
				require.NoError(t, err)
				return tmpDir
			},
			expectedError: "failed to parse plugin.yaml",
			cleanup:       true,
		},
		{
			name: "nonexistent directory",
			setupFunc: func(_ *testing.T) string {
				return "/nonexistent/directory"
			},
			expectedError: "no manifest file found",
		},
		{
			name: "malformed JSON",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				malformedJSON := `{
  "name": "malformed-json-plugin",
  "version": "1.0.0"
  "description": "Missing comma here"
  "category": "testing"
}`
				err := os.WriteFile(filepath.Join(tmpDir, "plugin.json"), []byte(malformedJSON), 0o600)
				require.NoError(t, err)
				return tmpDir
			},
			expectedError: "failed to parse plugin.json",
			cleanup:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.setupFunc(t)

			manifest, err := loadManifestFromDir(dir)

			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				assert.Nil(t, manifest)
			} else {
				require.NoError(t, err)
				require.NotNil(t, manifest)
				assert.Equal(t, tc.expectedName, manifest.Name)
				assert.NotEmpty(t, manifest.Version)
				assert.NotEmpty(t, manifest.Description)
				assert.NotEmpty(t, manifest.Category)
			}
		})
	}
}

func TestIsDirectory(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected bool
		setup    func(t *testing.T) string
	}{
		{
			name: "existing directory",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			expected: true,
		},
		{
			name: "existing file",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(filePath, []byte("test"), 0o600)
				require.NoError(t, err)
				return filePath
			},
			expected: false,
		},
		{
			name:     "nonexistent path",
			path:     "/nonexistent/path",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.path
			if tc.setup != nil {
				path = tc.setup(t)
			}

			result := isDirectory(path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCopyDir(t *testing.T) {
	testCases := []struct {
		name          string
		setupFunc     func(t *testing.T) (string, string)
		expectedError string
		validateFunc  func(t *testing.T, dst string)
	}{
		{
			name: "successful copy",
			setupFunc: func(t *testing.T) (string, string) {
				srcDir := t.TempDir()
				dstDir := t.TempDir()

				// Create source structure
				err := os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o600)
				require.NoError(t, err)

				subDir := filepath.Join(srcDir, "subdir")
				err = os.Mkdir(subDir, 0o750)
				require.NoError(t, err)

				err = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0o600)
				require.NoError(t, err)

				return srcDir, filepath.Join(dstDir, "target")
			},
			validateFunc: func(t *testing.T, dst string) {
				// Check files were copied
				content1, err := os.ReadFile(filepath.Join(dst, "file1.txt")) //nolint:gosec // Test file path
				require.NoError(t, err)
				assert.Equal(t, "content1", string(content1))

				content2, err := os.ReadFile(filepath.Join(dst, "subdir", "file2.txt")) //nolint:gosec // Test file path
				require.NoError(t, err)
				assert.Equal(t, "content2", string(content2))

				// Check directory structure
				info, err := os.Stat(filepath.Join(dst, "subdir"))
				require.NoError(t, err)
				assert.True(t, info.IsDir())
			},
		},
		{
			name: "nonexistent source",
			setupFunc: func(t *testing.T) (string, string) {
				dstDir := t.TempDir()
				return "/nonexistent/source", filepath.Join(dstDir, "target")
			},
			expectedError: "no such file or directory",
		},
		{
			name: "copy to readonly destination parent",
			setupFunc: func(t *testing.T) (string, string) {
				srcDir := t.TempDir()
				readOnlyDir := t.TempDir()

				// Create source file
				err := os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("content"), 0o600)
				require.NoError(t, err)

				// Make destination parent read-only
				err = os.Chmod(readOnlyDir, 0o400)
				require.NoError(t, err)

				// Cleanup function to restore permissions for cleanup
				t.Cleanup(func() {
					_ = os.Chmod(readOnlyDir, 0o750) //nolint:gosec // Test cleanup
				})

				return srcDir, filepath.Join(readOnlyDir, "target")
			},
			expectedError: "permission denied",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src, dst := tc.setupFunc(t)

			err := copyDir(src, dst)

			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
				if tc.validateFunc != nil {
					tc.validateFunc(t, dst)
				}
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	testCases := []struct {
		name          string
		setupFunc     func(t *testing.T) (string, string)
		expectedError string
		validateFunc  func(t *testing.T, src, dst string)
	}{
		{
			name: "successful file copy",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				src := filepath.Join(tmpDir, "source.txt")
				dst := filepath.Join(tmpDir, "dest.txt")

				err := os.WriteFile(src, []byte("test content"), 0o600)
				require.NoError(t, err)

				return src, dst
			},
			validateFunc: func(t *testing.T, src, dst string) {
				// Check content
				content, err := os.ReadFile(dst) //nolint:gosec // Test file path
				require.NoError(t, err)
				assert.Equal(t, "test content", string(content))

				// Check permissions are preserved
				srcInfo, err := os.Stat(src)
				require.NoError(t, err)
				dstInfo, err := os.Stat(dst)
				require.NoError(t, err)
				assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())
			},
		},
		{
			name: "nonexistent source file",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "nonexistent.txt"), filepath.Join(tmpDir, "dest.txt")
			},
			expectedError: "no such file or directory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src, dst := tc.setupFunc(t)

			err := copyFile(src, dst)

			if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				require.NoError(t, err)
				if tc.validateFunc != nil {
					tc.validateFunc(t, src, dst)
				}
			}
		})
	}
}

func TestDisplayPluginDetails(t *testing.T) {
	// Create a test plugin
	manifest := &plugins.PluginManifest{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "Test plugin description",
		Author:      "Test Author",
		Category:    "testing",
		FilePatterns: []string{
			"*.test",
			"*.spec",
		},
		Executable:    "./test-check.sh",
		Args:          []string{},
		Timeout:       "30s",
		RequiresFiles: true,
		Dependencies:  []string{"bash", "grep"},
	}

	plugin, err := plugins.NewPlugin(manifest, "/test/path")
	require.NoError(t, err)

	// Test that function runs without panic
	// Since displayPluginDetails writes to stdout, we can't easily test output
	// but we can ensure it doesn't panic or error
	require.NotPanics(t, func() {
		displayPluginDetails(plugin)
	})
}

func TestPluginManifestFormats(t *testing.T) {
	// Test that the same plugin data can be loaded from both YAML and JSON
	testData := map[string]interface{}{
		"name":           "format-test-plugin",
		"version":        "1.0.0",
		"description":    "Plugin to test format compatibility",
		"author":         "Format Test Author",
		"category":       "testing",
		"file_patterns":  []string{"*.fmt"},
		"executable":     "./fmt-check.sh",
		"args":           []string{},
		"timeout":        "60s",
		"requires_files": false,
		"dependencies":   []string{"fmt", "test"},
	}

	tmpDir := t.TempDir()

	// Create YAML version
	yamlData, err := yaml.Marshal(testData)
	require.NoError(t, err)
	yamlPath := filepath.Join(tmpDir, "yaml-version", "plugin.yaml")
	err = os.MkdirAll(filepath.Dir(yamlPath), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(yamlPath, yamlData, 0o600)
	require.NoError(t, err)

	// Create JSON version
	jsonData, err := json.Marshal(testData)
	require.NoError(t, err)
	jsonPath := filepath.Join(tmpDir, "json-version", "plugin.json")
	err = os.MkdirAll(filepath.Dir(jsonPath), 0o750)
	require.NoError(t, err)
	err = os.WriteFile(jsonPath, jsonData, 0o600)
	require.NoError(t, err)

	// Load both and compare
	yamlManifest, err := loadManifestFromDir(filepath.Dir(yamlPath))
	require.NoError(t, err)

	jsonManifest, err := loadManifestFromDir(filepath.Dir(jsonPath))
	require.NoError(t, err)

	// Compare all fields
	assert.Equal(t, yamlManifest.Name, jsonManifest.Name)
	assert.Equal(t, yamlManifest.Version, jsonManifest.Version)
	assert.Equal(t, yamlManifest.Description, jsonManifest.Description)
	assert.Equal(t, yamlManifest.Author, jsonManifest.Author)
	assert.Equal(t, yamlManifest.Category, jsonManifest.Category)
	assert.Equal(t, yamlManifest.FilePatterns, jsonManifest.FilePatterns)
	assert.Equal(t, yamlManifest.Executable, jsonManifest.Executable)
	assert.Equal(t, yamlManifest.Args, jsonManifest.Args)
	assert.Equal(t, yamlManifest.Timeout, jsonManifest.Timeout)
	assert.Equal(t, yamlManifest.RequiresFiles, jsonManifest.RequiresFiles)
	assert.Equal(t, yamlManifest.Dependencies, jsonManifest.Dependencies)
}

func TestPluginManifestPrecedence(t *testing.T) {
	// Test that YAML takes precedence over JSON when both exist
	tmpDir := t.TempDir()

	// Create YAML manifest
	yamlContent := `name: yaml-precedence-plugin
version: "1.0.0"
description: "YAML version should be loaded"
category: "yaml"
file_patterns: ["*.yaml"]
executable: "./yaml-check.sh"
timeout: "30s"
requires_files: true`

	err := os.WriteFile(filepath.Join(tmpDir, "plugin.yaml"), []byte(yamlContent), 0o600)
	require.NoError(t, err)

	// Create JSON manifest with different content
	jsonContent := `{
  "name": "json-precedence-plugin",
  "version": "2.0.0",
  "description": "JSON version should NOT be loaded",
  "category": "json",
  "file_patterns": ["*.json"],
  "executable": "./json-check.sh",
  "timeout": "60s",
  "requires_files": false
}`

	err = os.WriteFile(filepath.Join(tmpDir, "plugin.json"), []byte(jsonContent), 0o600)
	require.NoError(t, err)

	// Load manifest - should prefer YAML
	manifest, err := loadManifestFromDir(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "yaml-precedence-plugin", manifest.Name)
	assert.Equal(t, "YAML version should be loaded", manifest.Description)
	assert.Equal(t, "yaml", manifest.Category)
}
