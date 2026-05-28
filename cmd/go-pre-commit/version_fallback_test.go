package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetVersionWithFallback_LdflagsBranches exercises the ldflags-set branch of
// each fallback function by mutating the package-level build variables. The
// debug.ReadBuildInfo branches remain environment-dependent and are covered by
// the existing tests that call the functions with default values.
func TestGetVersionWithFallback_LdflagsBranches(t *testing.T) {
	orig := version
	t.Cleanup(func() { version = orig })

	t.Run("explicit version is returned", func(t *testing.T) {
		version = "v9.9.9"
		assert.Equal(t, "v9.9.9", getVersionWithFallback())
	})

	t.Run("template placeholder falls through", func(t *testing.T) {
		version = "{{.Version}}"
		got := getVersionWithFallback()
		assert.NotEqual(t, "{{.Version}}", got)
		assert.False(t, isTemplateString(got))
	})

	t.Run("empty falls through", func(t *testing.T) {
		version = ""
		assert.NotEmpty(t, getVersionWithFallback())
	})

	t.Run("dev falls through", func(t *testing.T) {
		version = "dev"
		assert.NotEmpty(t, getVersionWithFallback())
	})
}

func TestGetCommitWithFallback_LdflagsBranches(t *testing.T) {
	orig := commit
	t.Cleanup(func() { commit = orig })

	t.Run("explicit commit is returned", func(t *testing.T) {
		commit = "abcdef1234"
		assert.Equal(t, "abcdef1234", getCommitWithFallback())
	})

	t.Run("template placeholder falls through", func(t *testing.T) {
		commit = "{{.Commit}}"
		got := getCommitWithFallback()
		assert.NotEqual(t, "{{.Commit}}", got)
		assert.False(t, isTemplateString(got))
	})

	t.Run("none falls through", func(t *testing.T) {
		commit = "none"
		assert.NotEmpty(t, getCommitWithFallback())
	})
}

func TestGetBuildDateWithFallback_LdflagsBranches(t *testing.T) {
	orig := buildDate
	t.Cleanup(func() { buildDate = orig })

	t.Run("explicit build date is returned", func(t *testing.T) {
		buildDate = "2025-05-28_10:00:00_UTC"
		assert.Equal(t, "2025-05-28_10:00:00_UTC", getBuildDateWithFallback())
	})

	t.Run("template placeholder falls through", func(t *testing.T) {
		buildDate = "{{.Date}}"
		got := getBuildDateWithFallback()
		assert.NotEqual(t, "{{.Date}}", got)
		assert.False(t, isTemplateString(got))
	})

	t.Run("unknown falls through", func(t *testing.T) {
		buildDate = "unknown"
		assert.NotEmpty(t, getBuildDateWithFallback())
	})
}
