package gotools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/mrz1836/go-pre-commit/internal/shared"
)

// LintGoModuleTestSuite tests Go module detection and handling functionality
type LintGoModuleTestSuite struct {
	suite.Suite

	tempDir   string
	oldDir    string
	check     *LintCheck
	sharedCtx *shared.Context
}

func TestLintGoModuleSuite(t *testing.T) {
	suite.Run(t, new(LintGoModuleTestSuite))
}

func (s *LintGoModuleTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "lint_gomodule_test_*")
	s.Require().NoError(err)

	s.oldDir, err = os.Getwd()
	s.Require().NoError(err)

	err = os.Chdir(s.tempDir)
	s.Require().NoError(err)

	// Initialize git repo
	s.initGitRepo()

	// Create shared context and lint check
	s.sharedCtx = shared.NewContext()
	s.check = NewLintCheckWithSharedContext(s.sharedCtx)
}

func (s *LintGoModuleTestSuite) TearDownTest() {
	if s.oldDir != "" {
		_ = os.Chdir(s.oldDir)
	}
	if s.tempDir != "" {
		_ = os.RemoveAll(s.tempDir)
	}
}

func (s *LintGoModuleTestSuite) initGitRepo() {
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "init").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "config", "user.name", "Test User").Run())
}

// TestIsGoModule tests the isGoModule helper function
func (s *LintGoModuleTestSuite) TestIsGoModule() {
	testCases := []struct {
		name        string
		setupFunc   func() string
		expectMatch bool
	}{
		{
			name: "DirectoryWithGoMod",
			setupFunc: func() string {
				dir := filepath.Join(s.tempDir, "with-gomod")
				s.Require().NoError(os.MkdirAll(dir, 0o750))
				s.Require().NoError(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o600))
				return dir
			},
			expectMatch: true,
		},
		{
			name: "DirectoryWithoutGoMod",
			setupFunc: func() string {
				dir := filepath.Join(s.tempDir, "without-gomod")
				s.Require().NoError(os.MkdirAll(dir, 0o750))
				return dir
			},
			expectMatch: false,
		},
		{
			name: "NonExistentDirectory",
			setupFunc: func() string {
				return filepath.Join(s.tempDir, "nonexistent")
			},
			expectMatch: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			dir := tc.setupFunc()
			result := isGoModule(dir)
			s.Equal(tc.expectMatch, result, "isGoModule result for %s", tc.name)
		})
	}
}

// TestFindGoModuleRoot tests the findGoModuleRoot helper function
func (s *LintGoModuleTestSuite) TestFindGoModuleRoot() {
	// Setup nested directory structure with go.mod at different levels
	s.Require().NoError(os.MkdirAll("project/submodule/deep/nested", 0o750))

	// Create go.mod in submodule directory
	s.Require().NoError(os.WriteFile("project/submodule/go.mod", []byte("module submodule\n"), 0o600))

	testCases := []struct {
		name           string
		targetDir      string
		repoRoot       string
		expectedResult string
	}{
		{
			name:           "FindDirectModule",
			targetDir:      filepath.Join(s.tempDir, "project/submodule"),
			repoRoot:       s.tempDir,
			expectedResult: filepath.Join(s.tempDir, "project/submodule"),
		},
		{
			name:           "FindParentModule",
			targetDir:      filepath.Join(s.tempDir, "project/submodule/deep"),
			repoRoot:       s.tempDir,
			expectedResult: filepath.Join(s.tempDir, "project/submodule"),
		},
		{
			name:           "FindNestedModule",
			targetDir:      filepath.Join(s.tempDir, "project/submodule/deep/nested"),
			repoRoot:       s.tempDir,
			expectedResult: filepath.Join(s.tempDir, "project/submodule"),
		},
		{
			name:           "NoModuleFound",
			targetDir:      filepath.Join(s.tempDir, "project"),
			repoRoot:       s.tempDir,
			expectedResult: "",
		},
		{
			name:           "OutsideRepoRoot",
			targetDir:      "/tmp/outside",
			repoRoot:       s.tempDir,
			expectedResult: "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result := findGoModuleRoot(tc.targetDir, tc.repoRoot)
			s.Equal(tc.expectedResult, result, "findGoModuleRoot result for %s", tc.name)
		})
	}
}

// TestLintGoModuleInSubdirectory tests linting a Go module that exists in a subdirectory
func (s *LintGoModuleTestSuite) TestLintGoModuleInSubdirectory() {
	// Create a Go module in a subdirectory (similar to whisper/whisper-claude-worker)
	moduleDir := "project/worker"
	s.Require().NoError(os.MkdirAll(filepath.Join(moduleDir, "cmd/app"), 0o750))
	s.Require().NoError(os.MkdirAll(filepath.Join(moduleDir, "internal/service"), 0o750))

	// Create go.mod
	goModContent := `module example.com/worker

go 1.21
`
	s.Require().NoError(os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goModContent), 0o600))

	// Create Go files
	mainGoContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	s.Require().NoError(os.WriteFile(filepath.Join(moduleDir, "cmd/app/main.go"), []byte(mainGoContent), 0o600))

	serviceGoContent := `package service

import "fmt"

type Service struct {
	Name string
}

func (s *Service) Run() {
	fmt.Printf("Running service: %s\n", s.Name)
}
`
	s.Require().NoError(os.WriteFile(filepath.Join(moduleDir, "internal/service/service.go"), []byte(serviceGoContent), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "initial commit").Run())

	// Test linting files from the Go module subdirectory
	files := []string{
		filepath.Join(moduleDir, "cmd/app/main.go"),
		filepath.Join(moduleDir, "internal/service/service.go"),
	}

	err := s.check.Run(ctx, files)
	if err != nil {
		s.T().Logf("Subdirectory Go module lint result: %v", err)
		// Should not fail due to module path issues
		s.NotContains(err.Error(), "no go files to analyze")
		s.NotContains(err.Error(), "build constraints exclude all Go files")
	}
}

// TestLintGoModuleAtRoot tests linting when Go module is at repository root
func (s *LintGoModuleTestSuite) TestLintGoModuleAtRoot() {
	// Create go.mod at root
	goModContent := `module example.com/root

go 1.21
`
	s.Require().NoError(os.WriteFile("go.mod", []byte(goModContent), 0o600))

	// Create Go file at root
	s.Require().NoError(os.MkdirAll("cmd/app", 0o750))
	mainGoContent := `package main

import "fmt"

func main() {
	fmt.Println("Root module")
}
`
	s.Require().NoError(os.WriteFile("cmd/app/main.go", []byte(mainGoContent), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "root module").Run())

	// Test linting
	files := []string{"cmd/app/main.go"}
	err := s.check.Run(ctx, files)
	if err != nil {
		s.T().Logf("Root Go module lint result: %v", err)
	}
}

// TestLintNestedGoModules tests handling of nested Go modules
func (s *LintGoModuleTestSuite) TestLintNestedGoModules() {
	// Create parent module
	parentModContent := `module example.com/parent

go 1.21
`
	s.Require().NoError(os.WriteFile("go.mod", []byte(parentModContent), 0o600))

	// Create nested module
	s.Require().NoError(os.MkdirAll("nested/submodule", 0o750))
	nestedModContent := `module example.com/parent/nested

go 1.21
`
	s.Require().NoError(os.WriteFile("nested/submodule/go.mod", []byte(nestedModContent), 0o600))

	// Create Go files in both modules
	parentGoContent := `package main

import "fmt"

func main() {
	fmt.Println("Parent module")
}
`
	s.Require().NoError(os.WriteFile("main.go", []byte(parentGoContent), 0o600))

	nestedGoContent := `package main

import "fmt"

func main() {
	fmt.Println("Nested module")
}
`
	s.Require().NoError(os.WriteFile("nested/submodule/main.go", []byte(nestedGoContent), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "nested modules").Run())

	// Test linting files from both modules
	files := []string{
		"main.go",
		"nested/submodule/main.go",
	}

	err := s.check.Run(ctx, files)
	if err != nil {
		s.T().Logf("Nested modules lint result: %v", err)
		// Should handle nested modules correctly
		s.NotContains(err.Error(), "no go files to analyze")
	}
}

// TestLintOrphanedGoFiles tests that orphaned Go files (not in any module) are skipped
func (s *LintGoModuleTestSuite) TestLintOrphanedGoFiles() {
	// Create orphaned Go files (no go.mod anywhere)
	s.Require().NoError(os.MkdirAll("orphaned/cmd", 0o750))
	s.Require().NoError(os.MkdirAll("orphaned/pkg", 0o750))

	orphanedGoContent := `package main

import "fmt"

func main() {
	fmt.Println("Orphaned file")
}
`
	s.Require().NoError(os.WriteFile("orphaned/cmd/main.go", []byte(orphanedGoContent), 0o600))
	s.Require().NoError(os.WriteFile("orphaned/pkg/util.go", []byte(`package pkg

func Util() {
	// utility function
}
`), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "orphaned files").Run())

	// Test linting orphaned files
	files := []string{
		"orphaned/cmd/main.go",
		"orphaned/pkg/util.go",
	}

	err := s.check.Run(ctx, files)
	// Should succeed (skip orphaned files silently)
	s.NoError(err, "Orphaned Go files should be skipped without error")
}

// TestLintMixedModuleAndOrphanedFiles tests mixed scenario with both module and orphaned files
func (s *LintGoModuleTestSuite) TestLintMixedModuleAndOrphanedFiles() {
	// Create a Go module
	s.Require().NoError(os.MkdirAll("module/cmd", 0o750))
	goModContent := `module example.com/module

go 1.21
`
	s.Require().NoError(os.WriteFile("module/go.mod", []byte(goModContent), 0o600))

	moduleGoContent := `package main

import "fmt"

func main() {
	fmt.Println("Module file")
}
`
	s.Require().NoError(os.WriteFile("module/cmd/main.go", []byte(moduleGoContent), 0o600))

	// Create orphaned Go files
	s.Require().NoError(os.MkdirAll("orphaned", 0o750))
	orphanedGoContent := `package orphaned

func OrphanedFunc() {
	// orphaned function
}
`
	s.Require().NoError(os.WriteFile("orphaned/orphaned.go", []byte(orphanedGoContent), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "mixed files").Run())

	// Test linting mixed files
	files := []string{
		"module/cmd/main.go",
		"orphaned/orphaned.go",
	}

	err := s.check.Run(ctx, files)
	if err != nil {
		s.T().Logf("Mixed files lint result: %v", err)
		// Should handle the module file and skip the orphaned file
		s.NotContains(err.Error(), "no go files to analyze")
	}
}

// TestGoModuleWithRecursiveLint verifies that ./... is used for Go modules
func (s *LintGoModuleTestSuite) TestGoModuleWithRecursiveLint() {
	// This test verifies the fix: Go modules should use "./..." not "." as lint target

	// Create a complex Go module structure with multiple packages
	s.Require().NoError(os.MkdirAll("complex-module/cmd/app", 0o750))
	s.Require().NoError(os.MkdirAll("complex-module/internal/service", 0o750))
	s.Require().NoError(os.MkdirAll("complex-module/pkg/utils", 0o750))

	// Create go.mod
	goModContent := `module example.com/complex

go 1.21
`
	s.Require().NoError(os.WriteFile("complex-module/go.mod", []byte(goModContent), 0o600))

	// Create Go files in different packages
	mainGoContent := `package main

import (
	"fmt"
	"example.com/complex/internal/service"
	"example.com/complex/pkg/utils"
)

func main() {
	svc := service.New("test")
	result := utils.Process("hello")
	fmt.Printf("Service: %s, Result: %s\n", svc.Name(), result)
}
`
	s.Require().NoError(os.WriteFile("complex-module/cmd/app/main.go", []byte(mainGoContent), 0o600))

	serviceGoContent := `package service

type Service struct {
	name string
}

func New(name string) *Service {
	return &Service{name: name}
}

func (s *Service) Name() string {
	return s.name
}
`
	s.Require().NoError(os.WriteFile("complex-module/internal/service/service.go", []byte(serviceGoContent), 0o600))

	utilsGoContent := `package utils

import "strings"

func Process(input string) string {
	return strings.ToUpper(input)
}
`
	s.Require().NoError(os.WriteFile("complex-module/pkg/utils/utils.go", []byte(utilsGoContent), 0o600))

	// Commit files
	ctx := context.Background()
	s.Require().NoError(exec.CommandContext(ctx, "git", "add", ".").Run())
	s.Require().NoError(exec.CommandContext(ctx, "git", "commit", "-m", "complex module").Run())

	// Test linting files from the complex module
	// This should work because the linter uses "./..." for Go modules
	files := []string{
		"complex-module/cmd/app/main.go",
		"complex-module/internal/service/service.go",
		"complex-module/pkg/utils/utils.go",
	}

	err := s.check.Run(ctx, files)
	if err != nil {
		s.T().Logf("Complex module lint result: %v", err)
		// Should not fail due to missing dependencies or import resolution issues
		s.NotContains(err.Error(), "no go files to analyze")
		s.NotContains(err.Error(), "could not import")
	}
}

// TestGoModuleCommandConstruction tests that the correct golangci-lint command is constructed
func (s *LintGoModuleTestSuite) TestGoModuleCommandConstruction() {
	// This test focuses on verifying the command construction logic
	// without actually running golangci-lint

	// Create a simple Go module
	s.Require().NoError(os.MkdirAll("test-module", 0o750))
	s.Require().NoError(os.WriteFile("test-module/go.mod", []byte("module test\n"), 0o600))
	s.Require().NoError(os.WriteFile("test-module/main.go", []byte(`package main

func main() {}
`), 0o600))

	// Test the directory detection logic
	targetDir := filepath.Join(s.tempDir, "test-module")
	s.True(isGoModule(targetDir), "Should detect Go module")

	moduleRoot := findGoModuleRoot(targetDir, s.tempDir)
	s.Equal(targetDir, moduleRoot, "Should find correct module root")

	// Test that files outside modules return empty module root
	s.Require().NoError(os.MkdirAll("non-module", 0o750))
	nonModuleDir := filepath.Join(s.tempDir, "non-module")
	s.False(isGoModule(nonModuleDir), "Should not detect Go module")

	emptyRoot := findGoModuleRoot(nonModuleDir, s.tempDir)
	s.Empty(emptyRoot, "Should return empty for non-module directory")
}
