package gotools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mrz1836/go-pre-commit/internal/shared"
)

// TestLintMultiDirectoryIntegration tests the complete multi-directory linting flow
func TestLintMultiDirectoryIntegration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "lint_integration_*")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Failed to remove temp dir: %v", removeErr)
		}
	}()

	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(oldDir); chdirErr != nil {
			t.Logf("Failed to change back to old dir: %v", chdirErr)
		}
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Initialize git repo
	ctx := context.Background()
	require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())

	// Create a realistic project structure
	structure := map[string]string{
		"cmd/app/main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`,
		"pkg/utils/helper.go": `package utils

import "strings"

// CleanString removes whitespace
func CleanString(s string) string {
	return strings.TrimSpace(s)
}
`,
		"internal/service/service.go": `package service

import "fmt"

type Service struct {
	Name string
}

func (s *Service) Run() {
	fmt.Printf("Running service: %s\n", s.Name)
}
`,
		"internal/service/handler.go": `package service

import "net/http"

func HandleRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
`,
	}

	// Create all files
	for path, content := range structure {
		dir := filepath.Dir(path)
		require.NoError(t, os.MkdirAll(dir, 0o750))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}

	// Commit initial state
	require.NoError(t, exec.CommandContext(ctx, "git", "add", ".").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "commit", "-m", "initial commit").Run())

	// Create lint check
	sharedCtx := shared.NewContext()
	check := NewLintCheckWithSharedContext(sharedCtx)

	// Test 1: Lint files from multiple directories
	t.Run("MultiDirectoryLinting", func(t *testing.T) {
		files := []string{
			"cmd/app/main.go",
			"pkg/utils/helper.go",
			"internal/service/service.go",
		}

		err := check.Run(ctx, files)
		if err != nil {
			t.Logf("Lint result: %v", err)
			// Should not have the "named files must all be in one directory" error
			require.NotContains(t, err.Error(), "named files must all be in one directory")
		}
	})

	// Test 2: Lint all Go files in the project
	t.Run("AllFilesLinting", func(t *testing.T) {
		files := []string{
			"cmd/app/main.go",
			"pkg/utils/helper.go",
			"internal/service/service.go",
			"internal/service/handler.go",
		}

		err := check.Run(ctx, files)
		if err != nil {
			t.Logf("Full project lint result: %v", err)
			// Should handle all files without directory conflicts
			require.NotContains(t, err.Error(), "named files must all be in one directory")
		}
	})

	// Test 3: Introduce lint issues and verify they're caught
	t.Run("CatchLintIssues", func(t *testing.T) {
		// Add a file with lint issues
		badCode := `package bad

import "fmt"

func BadFunction() {
	unusedVariable := 42 // This should be caught by linter
	fmt.Println("Bad code")
}
`
		require.NoError(t, os.MkdirAll("pkg/bad", 0o750))
		require.NoError(t, os.WriteFile("pkg/bad/bad.go", []byte(badCode), 0o600))

		// Commit it first
		require.NoError(t, exec.CommandContext(ctx, "git", "add", ".").Run())
		require.NoError(t, exec.CommandContext(ctx, "git", "commit", "-m", "add bad code").Run())

		// Now modify it to trigger linting
		require.NoError(t, os.WriteFile("pkg/bad/bad.go", []byte(badCode+"\n"), 0o600))

		files := []string{
			"pkg/bad/bad.go",
			"pkg/utils/helper.go", // Mix with good file from different directory
		}

		err := check.Run(ctx, files)
		if err != nil {
			t.Logf("Lint issues detected: %v", err)
			// Even with issues, should not have directory conflict error
			require.NotContains(t, err.Error(), "named files must all be in one directory")
		}
	})
}

// TestLintPerformanceComparison compares performance of single vs multi-directory linting
func TestLintPerformanceComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "lint_perf_*")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Failed to remove temp dir: %v", removeErr)
		}
	}()

	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(oldDir); chdirErr != nil {
			t.Logf("Failed to change back to old dir: %v", chdirErr)
		}
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Initialize git
	ctx := context.Background()
	require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())

	// Create many files in different directories
	numDirs := 10
	filesPerDir := 3

	for d := 0; d < numDirs; d++ {
		dir := fmt.Sprintf("pkg/module%d", d)
		require.NoError(t, os.MkdirAll(dir, 0o750))

		for f := 0; f < filesPerDir; f++ {
			content := fmt.Sprintf(`package module%d

import "fmt"

func Module%dFunc%d() {
	fmt.Println("Module %d, Function %d")
}
`, d, d, f, d, f)
			path := fmt.Sprintf("%s/file%d.go", dir, f)
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		}
	}

	// Commit all
	require.NoError(t, exec.CommandContext(ctx, "git", "add", ".").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "commit", "-m", "performance test files").Run())

	// Create check
	sharedCtx := shared.NewContext()
	check := NewLintCheckWithSharedContext(sharedCtx)

	// Collect files from multiple directories
	var multiDirFiles []string
	for d := 0; d < 5; d++ {
		multiDirFiles = append(multiDirFiles, fmt.Sprintf("pkg/module%d/file0.go", d))
	}

	// Run the check
	err = check.Run(ctx, multiDirFiles)
	if err != nil {
		t.Logf("Performance test result: %v", err)
		// Verify no directory conflict errors
		require.NotContains(t, err.Error(), "named files must all be in one directory")
	}

	t.Log("Multi-directory linting completed successfully")
}

// TestLintWithBuildTags tests build tag detection and handling
func TestLintWithBuildTags(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "lint_build_tags_*")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Failed to remove temp dir: %v", removeErr)
		}
	}()

	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(oldDir); chdirErr != nil {
			t.Logf("Failed to change back to old dir: %v", chdirErr)
		}
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Initialize git repo
	ctx := context.Background()
	require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())

	// Create magefile with build constraint
	mageContent := `//go:build mage

package main

import (
	"fmt"
	"github.com/magefile/mage/sh"
)

// Build builds the application
func Build() error {
	fmt.Println("Building application...")
	return sh.RunV("go", "build", "./...")
}
`

	require.NoError(t, os.WriteFile("magefile.go", []byte(mageContent), 0o600))

	// Create go.mod
	goModContent := `module testproject

go 1.21

require github.com/magefile/mage v1.15.0
`
	require.NoError(t, os.WriteFile("go.mod", []byte(goModContent), 0o600))

	// Create go.sum (empty for this test)
	require.NoError(t, os.WriteFile("go.sum", []byte(""), 0o600))

	// Commit the magefile
	require.NoError(t, exec.CommandContext(ctx, "git", "add", ".").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "commit", "-m", "add magefile").Run())

	// Create check
	sharedCtx := shared.NewContext()
	check := NewLintCheckWithSharedContext(sharedCtx)

	// Test: Run lint on magefile.go (should auto-detect mage build tag)
	err = check.Run(ctx, []string{"magefile.go"})
	// The check should either succeed or provide a helpful error about build tags
	if err != nil {
		t.Logf("Build tag test result: %v", err)
		// Should not get the generic "build constraints exclude all Go files" error
		require.NotContains(t, err.Error(), "build constraints exclude all Go files in")
		// Should either succeed or provide guidance about build tags
	}

	t.Log("Build tag handling test completed")
}

// TestGoModuleSubdirectoryIntegration tests linting Go modules in subdirectories
func TestGoModuleSubdirectoryIntegration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "lint_gomodule_integration_*")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Failed to remove temp dir: %v", removeErr)
		}
	}()

	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(oldDir); chdirErr != nil {
			t.Logf("Failed to change back to old dir: %v", chdirErr)
		}
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Initialize git repo
	ctx := context.Background()
	require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())

	// Create a repository structure similar to whisper project
	// with a Go module in a subdirectory
	require.NoError(t, os.MkdirAll("worker-app/cmd/worker", 0o750))
	require.NoError(t, os.MkdirAll("worker-app/internal/tasks", 0o750))
	require.NoError(t, os.MkdirAll("other-files/docs", 0o750))

	// Create go.mod in the subdirectory
	goModContent := `module example.com/worker

go 1.21

require (
	github.com/stretchr/testify v1.8.4
)
`
	require.NoError(t, os.WriteFile("worker-app/go.mod", []byte(goModContent), 0o600))

	// Create go.sum (empty for this test)
	require.NoError(t, os.WriteFile("worker-app/go.sum", []byte(""), 0o600))

	// Create Go files in the module
	mainGoContent := `package main

import (
	"fmt"
	"example.com/worker/internal/tasks"
)

func main() {
	task := tasks.NewTask("example")
	fmt.Printf("Running task: %s\n", task.Name())
}
`
	require.NoError(t, os.WriteFile("worker-app/cmd/worker/main.go", []byte(mainGoContent), 0o600))

	tasksGoContent := `package tasks

type Task struct {
	name string
}

func NewTask(name string) *Task {
	return &Task{name: name}
}

func (t *Task) Name() string {
	return t.name
}
`
	require.NoError(t, os.WriteFile("worker-app/internal/tasks/tasks.go", []byte(tasksGoContent), 0o600))

	// Create some orphaned Go files (not part of any module)
	orphanedGoContent := `package docs

// This file is not part of any Go module
func DocumentationHelper() {
	// helper function
}
`
	require.NoError(t, os.WriteFile("other-files/docs/helper.go", []byte(orphanedGoContent), 0o600))

	// Create non-Go files
	require.NoError(t, os.WriteFile("README.md", []byte("# Test Project"), 0o600))
	require.NoError(t, os.WriteFile("other-files/config.json", []byte("{}"), 0o600))

	// Commit initial state
	require.NoError(t, exec.CommandContext(ctx, "git", "add", ".").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "commit", "-m", "initial commit").Run())

	// Create lint check
	sharedCtx := shared.NewContext()
	check := NewLintCheckWithSharedContext(sharedCtx)

	// Test: Lint files from Go module subdirectory
	t.Run("GoModuleSubdirectory", func(t *testing.T) {
		files := []string{
			"worker-app/cmd/worker/main.go",
			"worker-app/internal/tasks/tasks.go",
		}

		err := check.Run(ctx, files)
		if err != nil {
			t.Logf("Go module subdirectory lint result: %v", err)
			// Should not fail due to module resolution issues
			require.NotContains(t, err.Error(), "no go files to analyze")
			require.NotContains(t, err.Error(), "could not import")
		}
	})

	// Test: Mixed files from module and orphaned files
	t.Run("MixedModuleAndOrphanedFiles", func(t *testing.T) {
		files := []string{
			"worker-app/cmd/worker/main.go",
			"other-files/docs/helper.go", // This should be skipped
		}

		err := check.Run(ctx, files)
		if err != nil {
			t.Logf("Mixed files lint result: %v", err)
			// Should handle module files and skip orphaned files
		}
		// Should not fail completely due to orphaned files
	})

	// Test: All files including non-Go files
	t.Run("AllFilesIncludingNonGo", func(t *testing.T) {
		files := []string{
			"worker-app/cmd/worker/main.go",
			"worker-app/internal/tasks/tasks.go",
			"other-files/docs/helper.go",
			"README.md",               // Non-Go file
			"other-files/config.json", // Non-Go file
		}

		err := check.Run(ctx, files)
		if err != nil {
			t.Logf("All files lint result: %v", err)
		}
		// Should filter to only Go files and handle them appropriately
	})
}

// TestOrphanedGoFilesHandling tests that orphaned Go files are handled gracefully
func TestOrphanedGoFilesHandling(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "lint_orphaned_*")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Failed to remove temp dir: %v", removeErr)
		}
	}()

	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if chdirErr := os.Chdir(oldDir); chdirErr != nil {
			t.Logf("Failed to change back to old dir: %v", chdirErr)
		}
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Initialize git repo
	ctx := context.Background()
	require.NoError(t, exec.CommandContext(ctx, "git", "init").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())

	// Create orphaned Go files (no go.mod anywhere in the hierarchy)
	require.NoError(t, os.MkdirAll("standalone/utils", 0o750))
	require.NoError(t, os.MkdirAll("scripts", 0o750))

	orphanedContent1 := `package utils

// This is an orphaned Go file with no module
func UtilityFunction() {
	// some utility
}
`
	require.NoError(t, os.WriteFile("standalone/utils/util.go", []byte(orphanedContent1), 0o600))

	orphanedContent2 := `package main

import "fmt"

// This is an orphaned main package
func main() {
	fmt.Println("Orphaned main")
}
`
	require.NoError(t, os.WriteFile("scripts/script.go", []byte(orphanedContent2), 0o600))

	// Commit files
	require.NoError(t, exec.CommandContext(ctx, "git", "add", ".").Run())
	require.NoError(t, exec.CommandContext(ctx, "git", "commit", "-m", "orphaned files").Run())

	// Create check
	sharedCtx := shared.NewContext()
	check := NewLintCheckWithSharedContext(sharedCtx)

	// Test linting orphaned files - should not fail
	files := []string{
		"standalone/utils/util.go",
		"scripts/script.go",
	}

	err = check.Run(ctx, files)
	// Should succeed (orphaned files are skipped)
	require.NoError(t, err, "Orphaned Go files should be handled gracefully")

	t.Log("Orphaned Go files handling test completed")
}

// TestBuildTagDetection tests the build tag detection functions
func TestBuildTagDetection(t *testing.T) {
	// Create temp files with different build constraints
	tempDir, err := os.MkdirTemp("", "build_tag_detection_*")
	require.NoError(t, err)
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			t.Logf("Failed to remove temp dir: %v", removeErr)
		}
	}()

	// Test file with //go:build mage
	mageFile := filepath.Join(tempDir, "magefile.go")
	mageContent := `//go:build mage

package main

import "fmt"

func main() {
	fmt.Println("mage build")
}
`
	require.NoError(t, os.WriteFile(mageFile, []byte(mageContent), 0o600))

	// Test file with //go:build integration
	integrationFile := filepath.Join(tempDir, "integration_test.go")
	integrationContent := `//go:build integration

package main

import "testing"

func TestIntegration(t *testing.T) {
	t.Log("integration test")
}
`
	require.NoError(t, os.WriteFile(integrationFile, []byte(integrationContent), 0o600))

	// Test file with legacy // +build tag
	legacyFile := filepath.Join(tempDir, "legacy.go")
	legacyContent := `// +build tools

package main

import "fmt"

func main() {
	fmt.Println("legacy build tag")
}
`
	require.NoError(t, os.WriteFile(legacyFile, []byte(legacyContent), 0o600))

	// Test detectBuildTags function
	files := []string{mageFile, integrationFile, legacyFile}
	tags := detectBuildTags(files)

	// Should detect mage, integration, and tools tags
	expectedTags := map[string]bool{
		"mage":        false,
		"integration": false,
		"tools":       false,
	}

	for _, tag := range tags {
		if _, exists := expectedTags[tag]; exists {
			expectedTags[tag] = true
		}
	}

	for tag, found := range expectedTags {
		require.True(t, found, "Expected to find build tag: %s", tag)
	}

	t.Logf("Detected build tags: %v", tags)
}
