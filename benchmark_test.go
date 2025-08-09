package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrz1836/go-pre-commit/internal/checks/builtin"
	"github.com/mrz1836/go-pre-commit/internal/config"
	"github.com/mrz1836/go-pre-commit/internal/git"
	"github.com/mrz1836/go-pre-commit/internal/runner"
)

// BenchmarkPreCommitSystem_EndToEnd measures complete pre-commit system performance
func BenchmarkPreCommitSystem_EndToEnd(b *testing.B) {
	tmpDir := setupTestRepo(b)

	// Create realistic commit scenario
	testFiles := createRealisticFiles(b, tmpDir)

	cfg := &config.Config{
		Enabled: true,
		Timeout: 120,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Create runner and execute checks
		r := runner.New(cfg, tmpDir)
		opts := runner.Options{
			Files: testFiles,
		}

		results, err := r.Run(context.Background(), opts)
		duration := time.Since(start)

		if err != nil {
			b.Fatal(err)
		}

		b.Logf("End-to-end iteration %d: %v (files: %d, passed: %d, failed: %d)",
			i, duration, results.TotalFiles, results.Passed, results.Failed)
	}
}

// BenchmarkPreCommitSystem_SmallProject simulates small project performance
func BenchmarkPreCommitSystem_SmallProject(b *testing.B) {
	tmpDir := setupTestRepo(b)

	// Small project: 3-5 files
	testFiles := []string{
		"main.go",
		"config.go",
		"README.md",
	}

	for _, file := range testFiles {
		content := generateSimpleContent(file)
		err := os.WriteFile(filepath.Join(tmpDir, file), []byte(content), 0o600)
		if err != nil {
			b.Fatal(err)
		}
	}

	cfg := &config.Config{
		Enabled: true,
		Timeout: 60,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()

		r := runner.New(cfg, tmpDir)
		opts := runner.Options{Files: testFiles}

		results, err := r.Run(context.Background(), opts)
		duration := time.Since(start)

		if err != nil {
			b.Fatal(err)
		}

		b.Logf("Small project iteration %d: %v (files: %d)",
			i, duration, results.TotalFiles)
	}
}

// BenchmarkPreCommitSystem_LargeProject simulates large project performance
func BenchmarkPreCommitSystem_LargeProject(b *testing.B) {
	tmpDir := setupTestRepo(b)

	// Large project: 25+ files
	testFiles := createLargeProjectFiles(b, tmpDir, 25)

	cfg := &config.Config{
		Enabled: true,
		Timeout: 300,
	}
	cfg.Checks.Whitespace = true
	cfg.Checks.EOF = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()

		r := runner.New(cfg, tmpDir)
		opts := runner.Options{
			Files: testFiles,
		}

		results, err := r.Run(context.Background(), opts)
		duration := time.Since(start)

		if err != nil {
			b.Fatal(err)
		}

		b.Logf("Large project iteration %d: %v (files: %d)",
			i, duration, results.TotalFiles)
	}
}

// BenchmarkPreCommitSystem_CheckCombinations tests different check combinations
func BenchmarkPreCommitSystem_CheckCombinations(b *testing.B) {
	tmpDir := setupTestRepo(b)
	testFiles := createRealisticFiles(b, tmpDir)

	checkCombinations := []struct {
		name   string
		config func(*config.Config)
	}{
		{
			name: "WhitespaceOnly",
			config: func(cfg *config.Config) {
				cfg.Checks.Whitespace = true
			},
		},
		{
			name: "WhitespaceAndEOF",
			config: func(cfg *config.Config) {
				cfg.Checks.Whitespace = true
				cfg.Checks.EOF = true
			},
		},
	}

	for _, combo := range checkCombinations {
		b.Run(combo.name, func(b *testing.B) {
			cfg := &config.Config{
				Enabled: true,
				Timeout: 120,
			}
			combo.config(cfg)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				start := time.Now()

				r := runner.New(cfg, tmpDir)
				opts := runner.Options{Files: testFiles}

				results, err := r.Run(context.Background(), opts)
				duration := time.Since(start)

				if err != nil {
					b.Fatal(err)
				}

				b.Logf("%s iteration %d: %v (checks: %d)",
					combo.name, i, duration, len(results.CheckResults))
			}
		})
	}
}

// BenchmarkPreCommitSystem_HookInstallation measures hook management performance
func BenchmarkPreCommitSystem_HookInstallation(b *testing.B) {
	scenarios := []string{"pre-commit", "pre-push"}

	for _, hookType := range scenarios {
		b.Run(fmt.Sprintf("Install_%s", hookType), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tmpDir := b.TempDir()
				gitDir := filepath.Join(tmpDir, ".git")
				hooksDir := filepath.Join(gitDir, "hooks")

				err := os.MkdirAll(hooksDir, 0o750)
				if err != nil {
					b.Fatal(err)
				}

				installer := git.NewInstaller(tmpDir, "/tmp/go-pre-commit")

				start := time.Now()
				err = installer.InstallHook(hookType, false)
				duration := time.Since(start)

				if err != nil {
					b.Fatal(err)
				}

				b.Logf("Install %s iteration %d: %v", hookType, i, duration)
			}
		})
	}
}

// BenchmarkPreCommitSystem_GitOperations measures git-related performance
func BenchmarkPreCommitSystem_GitOperations(b *testing.B) {
	tmpDir := setupTestRepo(b)
	repo := git.NewRepository(tmpDir)

	gitOps := []struct {
		name string
		op   func() error
	}{
		{
			name: "GetAllFiles",
			op: func() error {
				_, err := repo.GetAllFiles()
				return err
			},
		},
		{
			name: "GetStagedFiles",
			op: func() error {
				_, err := repo.GetStagedFiles()
				return err
			},
		},
	}

	for _, gitOp := range gitOps {
		b.Run(gitOp.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				start := time.Now()
				err := gitOp.op()
				duration := time.Since(start)

				if err != nil {
					b.Fatal(err)
				}

				b.Logf("%s iteration %d: %v", gitOp.name, i, duration)
			}
		})
	}
}

// BenchmarkPreCommitSystem_IndividualChecks measures individual check performance
func BenchmarkPreCommitSystem_IndividualChecks(b *testing.B) {
	tmpDir := b.TempDir()

	// Create test files
	testFiles := []string{"file1.txt", "file2.go", "README.md"}

	for _, file := range testFiles {
		content := generateSimpleContent(file)
		// Add whitespace issues
		content += "   \n\t  \n"

		err := os.WriteFile(filepath.Join(tmpDir, file), []byte(content), 0o600)
		if err != nil {
			b.Fatal(err)
		}
	}

	checks := []struct {
		name  string
		check interface {
			Run(ctx context.Context, files []string) error
			FilterFiles(files []string) []string
		}
		files []string
	}{
		{
			name:  "WhitespaceCheck",
			check: &builtin.WhitespaceCheck{},
			files: testFiles,
		},
		{
			name:  "EOFCheck",
			check: &builtin.EOFCheck{},
			files: testFiles,
		},
	}

	for _, checkBench := range checks {
		b.Run(checkBench.name, func(b *testing.B) {
			// Convert relative paths to absolute
			absFiles := make([]string, len(checkBench.files))
			for i, file := range checkBench.files {
				absFiles[i] = filepath.Join(tmpDir, file)
			}

			filteredFiles := checkBench.check.FilterFiles(absFiles)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				start := time.Now()
				err := checkBench.check.Run(context.Background(), filteredFiles)
				duration := time.Since(start)

				if err != nil {
					b.Fatal(err)
				}

				b.Logf("%s iteration %d: %v (files: %d)",
					checkBench.name, i, duration, len(filteredFiles))
			}
		})
	}
}

// Helper functions

func setupTestRepo(b *testing.B) string {
	tmpDir := b.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")

	err := os.MkdirAll(gitDir, 0o750)
	if err != nil {
		b.Fatal(err)
	}

	return tmpDir
}

func createRealisticFiles(b *testing.B, tmpDir string) []string {
	files := []string{
		"main.go", "config.go", "handler.go", "service.go",
		"README.md", "go.mod", "Dockerfile", "config.yaml",
	}

	for _, file := range files {
		content := generateSimpleContent(file)
		err := os.WriteFile(filepath.Join(tmpDir, file), []byte(content), 0o600)
		if err != nil {
			b.Fatal(err)
		}
	}

	return files
}

func createLargeProjectFiles(b *testing.B, tmpDir string, count int) []string {
	var files []string

	for i := 0; i < count; i++ {
		// Create directory structure
		dir := fmt.Sprintf("pkg/module%d", i%10)
		err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o750)
		if err != nil {
			b.Fatal(err)
		}

		// Create Go file
		goFile := filepath.Join(dir, fmt.Sprintf("service%d.go", i))
		files = append(files, goFile)

		content := generateSimpleContent(goFile)
		err = os.WriteFile(filepath.Join(tmpDir, goFile), []byte(content), 0o600)
		if err != nil {
			b.Fatal(err)
		}

		// Add some non-Go files periodically
		if i%10 == 0 {
			mdFile := filepath.Join(dir, "README.md")
			files = append(files, mdFile)

			content := generateSimpleContent(mdFile)
			err = os.WriteFile(filepath.Join(tmpDir, mdFile), []byte(content), 0o600)
			if err != nil {
				b.Fatal(err)
			}
		}
	}

	return files
}

func generateSimpleContent(filename string) string {
	ext := filepath.Ext(filename)

	switch ext {
	case ".go":
		return `package main

import (
	"fmt"
	"context"
	"time"
)

// Service provides functionality for the application
type Service struct {
	name string
}

// New creates a new service instance
func New(name string) *Service {
	return &Service{name: name}
}

// Run starts the service
func (s *Service) Run(ctx context.Context) error {
	fmt.Printf("Running service: %s\n", s.name)
	return nil
}`
	case ".md":
		return fmt.Sprintf(`# %s

This is a documentation file.

## Overview

This component provides functionality for the application.

## Usage

Basic usage example:

	go run main.go

## Configuration

Set environment variables as needed.
`, filename)
	case ".yaml", ".yml":
		return `# Configuration
app:
  name: go-pre-commit
  version: 1.0.0

server:
  port: 8080
  timeout: 30s

logging:
  level: info
  format: json`
	default:
		return fmt.Sprintf("# %s\n\nThis is a test file for benchmarking.\n\nContent line 1\nContent line 2\nContent line 3\n", filename)
	}
}
