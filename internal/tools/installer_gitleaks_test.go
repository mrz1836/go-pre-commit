package tools

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errMissingCurlTarget is returned by the curl fake when no -o target is present.
var errMissingCurlTarget = errors.New("no -o target in curl args")

// makeTarGz builds an in-memory .tar.gz archive from the given name->content map.
func makeTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o750,
			Size: int64(len(content)),
		}))
		_, err := tw.Write(content)
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

// fakeCurlWritingArchive installs a runInstallCommand fake that, for the curl
// download, writes the supplied archive bytes to the -o target and records the
// requested download URL.
func fakeCurlWritingArchive(t *testing.T, archive []byte, capturedURL *string) {
	t.Helper()
	fakeInstall(t, func(_ context.Context, _ []string, name string, args ...string) ([]byte, error) {
		if name != "curl" {
			return nil, nil
		}
		var outPath string
		for i, a := range args {
			if a == "-o" && i+1 < len(args) {
				outPath = args[i+1]
			}
		}
		if len(args) > 0 && capturedURL != nil {
			*capturedURL = args[len(args)-1]
		}
		if outPath == "" {
			return nil, errMissingCurlTarget
		}
		return nil, os.WriteFile(outPath, archive, 0o600) //nolint:gosec // outPath is a temp path constructed by the code under test
	})
}

func TestInstallGitleaks_DownloadURLMatrix(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "gitleaks_8.29.0_linux_x64.tar.gz"},
		{"linux", "arm64", "gitleaks_8.29.0_linux_arm64.tar.gz"},
		{"darwin", "arm64", "gitleaks_8.29.0_darwin_arm64.tar.gz"},
		{"windows", "amd64", "gitleaks_8.29.0_windows_x64.zip"},
		{"linux", "ppc64", "gitleaks_8.29.0_linux_x64.tar.gz"}, // unknown arch -> x64 fallback
	}
	for _, tc := range cases {
		t.Run(tc.goos+"_"+tc.goarch, func(t *testing.T) {
			t.Setenv("GOOS", tc.goos)
			t.Setenv("GOARCH", tc.goarch)

			var url string
			// Return a non-network error after capturing the URL so the flow stops
			// before extraction; we only assert URL construction here.
			fakeInstall(t, func(_ context.Context, _ []string, name string, args ...string) ([]byte, error) {
				if name == "curl" && len(args) > 0 {
					url = args[len(args)-1]
				}
				return []byte("boom"), errSimulatedInstall
			})

			err := installGitleaks(context.Background(), "v8.29.0")
			require.Error(t, err)
			assert.Contains(t, url, tc.want)
		})
	}
}

func TestInstallGitleaks_ExtractionPaths(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses unix tar extraction")
	}

	// Provide a clean PATH that has system tools (tar) plus an isolated GOBIN so
	// the final LookPath verification resolves to our freshly-installed copy.
	setup := func(t *testing.T) string {
		t.Helper()
		gobin := t.TempDir()
		t.Setenv("GOOS", "linux") // skip uname lookups
		t.Setenv("GOARCH", "amd64")
		t.Setenv("GOBIN", gobin)
		t.Setenv("PATH", gobin+":/usr/bin:/bin")
		return gobin
	}

	t.Run("successful install", func(t *testing.T) {
		gobin := setup(t)
		archive := makeTarGz(t, map[string][]byte{"gitleaks": []byte("#!/bin/sh\nexit 0\n")})
		fakeCurlWritingArchive(t, archive, nil)

		require.NoError(t, installGitleaks(context.Background(), "v8.29.0"))
		assert.FileExists(t, filepath.Join(gobin, "gitleaks"))
	})

	t.Run("binary missing from archive", func(t *testing.T) {
		setup(t)
		archive := makeTarGz(t, map[string][]byte{"README.md": []byte("no binary here")})
		fakeCurlWritingArchive(t, archive, nil)

		err := installGitleaks(context.Background(), "v8.29.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "binary not found in archive")
	})

	t.Run("corrupt archive fails extraction", func(t *testing.T) {
		setup(t)
		fakeCurlWritingArchive(t, []byte("not a real tar.gz"), nil)

		err := installGitleaks(context.Background(), "v8.29.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to extract")
	})
}
