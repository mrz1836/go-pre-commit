package tools

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// archiveEntry is one entry of a fixture release archive.
type archiveEntry struct {
	name     string
	body     []byte
	typeflag byte   // tar type; zero means a regular file
	linkname string // target of a symlink or hard link
}

// buildTarGz builds an in-memory .tar.gz archive holding entries, in order.
func buildTarGz(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Mode: 0o755, Typeflag: e.typeflag, Linkname: e.linkname}
		if hdr.Typeflag == 0 {
			hdr.Typeflag = tar.TypeReg
		}
		if hdr.Typeflag == tar.TypeReg {
			hdr.Size = int64(len(e.body))
		}
		require.NoError(t, tw.WriteHeader(hdr))
		if hdr.Typeflag == tar.TypeReg {
			_, err := tw.Write(e.body)
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

// buildZip builds an in-memory .zip archive holding entries, in order. An entry
// with a linkname is stored as a symlink to it.
func buildZip(t *testing.T, entries []archiveEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		hdr := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		body := e.body
		if e.linkname != "" {
			hdr.SetMode(os.ModeSymlink | 0o777)
			body = []byte(e.linkname)
		} else {
			hdr.SetMode(0o755)
		}
		w, err := zw.CreateHeader(hdr)
		require.NoError(t, err)
		_, err = w.Write(body)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

// buildArchive builds the archive format the asset's file name calls for.
func buildArchive(t *testing.T, a releaseAsset, entries []archiveEntry) []byte {
	t.Helper()
	if strings.HasSuffix(a.archive, ".zip") {
		return buildZip(t, entries)
	}
	return buildTarGz(t, entries)
}

// sumLine is one line of a checksums file listing data under name.
func sumLine(name string, data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]) + "  " + name + "\n"
}

// releaseServer serves fake GitHub release assets over TLS and records every
// request, so release installs are tested without reaching the network.
type releaseServer struct {
	mu       sync.Mutex
	assets   map[string][]byte // request path -> body
	status   map[string]int    // request path -> forced status code
	latest   map[string]string // repo -> tag that /releases/latest redirects to
	requests []string
	url      string
	close    func()
}

// newReleaseServer starts a fake release server and points the release
// downloader at it for the rest of the test.
func newReleaseServer(t *testing.T) *releaseServer {
	t.Helper()
	rs := &releaseServer{
		assets: make(map[string][]byte),
		status: make(map[string]int),
		latest: make(map[string]string),
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(rs.serve))
	rs.url = srv.URL
	rs.close = srv.Close
	t.Cleanup(srv.Close)

	origBase, origTransport := releaseBaseURL, releaseTransport
	releaseBaseURL, releaseTransport = srv.URL, srv.Client().Transport
	t.Cleanup(func() { releaseBaseURL, releaseTransport = origBase, origTransport })
	return rs
}

func (rs *releaseServer) serve(w http.ResponseWriter, r *http.Request) {
	rs.mu.Lock()
	rs.requests = append(rs.requests, r.URL.Path)
	body, ok := rs.assets[r.URL.Path]
	code, forced := rs.status[r.URL.Path]
	var latestTag string
	for repo, tag := range rs.latest {
		if r.URL.Path == "/"+repo+"/releases/latest" {
			latestTag = "/" + repo + "/releases/tag/" + tag
		}
	}
	rs.mu.Unlock()

	switch {
	case forced:
		w.WriteHeader(code)
	case latestTag != "":
		http.Redirect(w, r, latestTag, http.StatusFound)
	case ok:
		_, _ = w.Write(body)
	default:
		http.NotFound(w, r)
	}
}

// assetPath is the request path of a file of the asset's release.
func assetPath(a releaseAsset, name string) string {
	return "/" + a.repo + "/releases/download/" + a.version + "/" + name
}

// publish serves archive and checksums as the asset's release files.
func (rs *releaseServer) publish(a releaseAsset, archive []byte, checksums string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.assets[assetPath(a, a.archive)] = archive
	rs.assets[assetPath(a, a.checksums)] = []byte(checksums)
}

// fail makes the asset's release file name answer with code.
func (rs *releaseServer) fail(a releaseAsset, name string, code int) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.status[assetPath(a, name)] = code
}

// requested reports how many requests the server saw for the path.
func (rs *releaseServer) requested(path string) int {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	n := 0
	for _, p := range rs.requests {
		if p == path {
			n++
		}
	}
	return n
}

// requestCount reports how many requests the server saw in all.
func (rs *releaseServer) requestCount() int {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return len(rs.requests)
}

// testAsset is a release asset of a fictitious tool on the host platform.
func testAsset() releaseAsset {
	bin := "faketool"
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		bin += ".exe"
		ext = ".zip"
	}
	dir := "faketool-1.2.3-" + runtime.GOOS + "-" + runtime.GOARCH
	return releaseAsset{
		tool:      "faketool",
		version:   "v1.2.3",
		repo:      "acme/faketool",
		archive:   dir + ext,
		checksums: "faketool-1.2.3-checksums.txt",
		binary:    dir + "/" + bin,
	}
}

// fakeBinary is the content of the binary in fixture archives.
var fakeBinary = []byte("#!/bin/sh\necho faketool\n") //nolint:gochecknoglobals // shared read-only test fixture

// goodArchive is a well-formed archive of the asset holding fakeBinary.
func goodArchive(t *testing.T, a releaseAsset) []byte {
	t.Helper()
	dir := strings.TrimSuffix(a.binary, "/"+filepath.Base(a.binary))
	return buildArchive(t, a, []archiveEntry{
		{name: dir + "/LICENSE", body: []byte("license")},
		{name: a.binary, body: fakeBinary},
		{name: dir + "/README.md", body: []byte("readme")},
	})
}

// requireStepError checks that err is an install failure naming the tool, the
// version, and the step that failed.
func requireStepError(t *testing.T, err error, a releaseAsset, step string) {
	t.Helper()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInstallFailed)
	assert.Contains(t, err.Error(), a.tool+" "+a.version+": "+step+":")
}

// requireNothingInstalled checks that dir holds no file at all, not even a
// leftover temporary one.
func requireNothingInstalled(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "nothing may be installed after a failed install")
}

func TestInstallReleaseInstallsVerifiedBinary(t *testing.T) {
	a := testAsset()
	rs := newReleaseServer(t)
	archive := goodArchive(t, a)
	rs.publish(a, archive, sumLine(a.archive, archive))
	dest := t.TempDir()

	path, err := installRelease(t.Context(), a, dest)
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(dest, filepath.Base(a.binary)), path)
	got, err := os.ReadFile(path) //nolint:gosec // path is inside the test's temp dir
	require.NoError(t, err)
	assert.Equal(t, fakeBinary, got)
	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(path)
		require.NoError(t, statErr)
		assert.NotZero(t, info.Mode().Perm()&0o100, "the installed binary must be executable")
	}
	entries, err := os.ReadDir(dest)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "only the binary is installed")
}

// TestInstallReleaseIgnoresLongerNames reproduces golangci-lint v2.15.0's
// checksums file, which lists both the archive and "<archive>.sbom.json". Only
// the line for the archive's exact name may be used.
func TestInstallReleaseIgnoresLongerNames(t *testing.T) {
	a := testAsset()
	rs := newReleaseServer(t)
	archive := goodArchive(t, a)
	sbom := []byte(`{"sbom":true}`)
	rs.publish(a, archive, sumLine(a.archive, archive)+sumLine(a.archive+".sbom.json", sbom))
	dest := t.TempDir()

	_, err := installRelease(t.Context(), a, dest)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(dest, filepath.Base(a.binary)))
}

func TestInstallReleaseRefusesBadChecksums(t *testing.T) {
	a := testAsset()
	other := []byte("some other file")

	tests := []struct {
		name      string
		checksums func(archive []byte) string
		step      string
		wantErr   error
	}{
		{
			name: "archive listed twice with different sums",
			checksums: func(archive []byte) string {
				return sumLine(a.archive, other) + sumLine(a.archive, archive)
			},
			step:    "read checksums",
			wantErr: ErrChecksumAmbiguous,
		},
		{
			name: "archive listed twice with the same sum",
			checksums: func(archive []byte) string {
				return sumLine(a.archive, archive) + sumLine(a.archive, archive)
			},
			step:    "read checksums",
			wantErr: ErrChecksumAmbiguous,
		},
		{
			name: "only a longer name that contains the archive's",
			checksums: func(archive []byte) string {
				return sumLine(a.archive+".sbom.json", archive)
			},
			step:    "read checksums",
			wantErr: ErrChecksumNotListed,
		},
		{
			name: "archive not listed",
			checksums: func([]byte) string {
				return sumLine("faketool-1.2.3-plan9-mips.tar.gz", other)
			},
			step:    "read checksums",
			wantErr: ErrChecksumNotListed,
		},
		{
			name: "malformed sum for the archive",
			checksums: func([]byte) string {
				return "not-a-sha256  " + a.archive + "\n"
			},
			step:    "read checksums",
			wantErr: ErrChecksumMalformed,
		},
		{
			name: "sum does not match the archive",
			checksums: func([]byte) string {
				return sumLine(a.archive, other)
			},
			step:    "verify checksum",
			wantErr: ErrChecksumMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := newReleaseServer(t)
			archive := goodArchive(t, a)
			rs.publish(a, archive, tt.checksums(archive))
			dest := t.TempDir()

			_, err := installRelease(t.Context(), a, dest)
			require.ErrorIs(t, err, tt.wantErr)
			requireStepError(t, err, a, tt.step)
			requireNothingInstalled(t, dest)
		})
	}
}

func TestInstallReleaseSkipsArchiveWithoutChecksum(t *testing.T) {
	// The archive is not even downloaded when its checksum can't be found.
	a := testAsset()
	rs := newReleaseServer(t)
	archive := goodArchive(t, a)
	rs.publish(a, archive, sumLine(a.archive, archive)+sumLine(a.archive, archive))

	_, err := installRelease(t.Context(), a, t.TempDir())
	require.ErrorIs(t, err, ErrChecksumAmbiguous)
	assert.Zero(t, rs.requested(assetPath(a, a.archive)))
}

func TestInstallReleaseDownloadFailures(t *testing.T) {
	a := testAsset()

	origAttempts, origDelay := GetRetryConfig()
	t.Cleanup(func() { SetRetryConfig(origAttempts, origDelay) })
	SetRetryConfig(3, time.Millisecond)

	tests := []struct {
		name     string
		setup    func(rs *releaseServer)
		step     string
		attempts int // requests expected for the failing file
		failing  string
		wantErr  error
	}{
		{
			name:     "checksums not found",
			setup:    func(rs *releaseServer) { rs.fail(a, a.checksums, http.StatusNotFound) },
			step:     "download checksums",
			attempts: 1,
			failing:  a.checksums,
			wantErr:  ErrDownloadFailed,
		},
		{
			name:     "archive not found",
			setup:    func(rs *releaseServer) { rs.fail(a, a.archive, http.StatusNotFound) },
			step:     "download archive",
			attempts: 1,
			failing:  a.archive,
			wantErr:  ErrDownloadFailed,
		},
		{
			name:     "archive unavailable is retried",
			setup:    func(rs *releaseServer) { rs.fail(a, a.archive, http.StatusServiceUnavailable) },
			step:     "download archive",
			attempts: 3,
			failing:  a.archive,
			wantErr:  ErrDownloadFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := newReleaseServer(t)
			archive := goodArchive(t, a)
			rs.publish(a, archive, sumLine(a.archive, archive))
			tt.setup(rs)
			dest := t.TempDir()

			_, err := installRelease(t.Context(), a, dest)
			require.ErrorIs(t, err, tt.wantErr)
			requireStepError(t, err, a, tt.step)
			assert.Equal(t, tt.attempts, rs.requested(assetPath(a, tt.failing)))
			requireNothingInstalled(t, dest)
		})
	}

	t.Run("server unreachable", func(t *testing.T) {
		rs := newReleaseServer(t)
		rs.close()
		dest := t.TempDir()

		_, err := installRelease(t.Context(), a, dest)
		requireStepError(t, err, a, "download checksums")
		requireNothingInstalled(t, dest)
	})
}

func TestInstallReleaseRefusesInsecureDownloads(t *testing.T) {
	a := testAsset()

	t.Run("plain http base URL", func(t *testing.T) {
		newReleaseServer(t)
		releaseBaseURL = "http://127.0.0.1:1"
		dest := t.TempDir()

		_, err := installRelease(t.Context(), a, dest)
		require.ErrorIs(t, err, ErrInsecureDownload)
		requireStepError(t, err, a, "download checksums")
		requireNothingInstalled(t, dest)
	})

	t.Run("redirect to plain http", func(t *testing.T) {
		var archive []byte
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, a.checksums) {
				_, _ = w.Write([]byte(sumLine(a.archive, archive)))
				return
			}
			http.Redirect(w, r, "http://127.0.0.1:1/archive", http.StatusFound)
		}))
		t.Cleanup(srv.Close)
		archive = goodArchive(t, a)
		origBase, origTransport := releaseBaseURL, releaseTransport
		t.Cleanup(func() { releaseBaseURL, releaseTransport = origBase, origTransport })
		releaseBaseURL, releaseTransport = srv.URL, srv.Client().Transport
		dest := t.TempDir()

		_, err := installRelease(t.Context(), a, dest)
		require.ErrorIs(t, err, ErrInsecureDownload)
		requireStepError(t, err, a, "download archive")
		requireNothingInstalled(t, dest)
	})
}

func TestFetchRefusesOversizedDownloads(t *testing.T) {
	rs := newReleaseServer(t)
	rs.assets["/big"] = bytes.Repeat([]byte("x"), 100)

	var buf bytes.Buffer
	err := fetch(t.Context(), rs.url+"/big", 10, &buf)
	require.ErrorIs(t, err, ErrDownloadTooLarge)

	buf.Reset()
	require.NoError(t, fetch(t.Context(), rs.url+"/big", 100, &buf))
	assert.Equal(t, 100, buf.Len())
}

func TestInstallReleaseRefusesUnsafeArchives(t *testing.T) {
	a := testAsset()
	dir := strings.TrimSuffix(a.binary, "/"+filepath.Base(a.binary))
	zipOnly := strings.HasSuffix(a.archive, ".zip")

	tests := []struct {
		name    string
		entries []archiveEntry
		tarOnly bool
		wantErr error
	}{
		{
			name:    "binary missing",
			entries: []archiveEntry{{name: dir + "/README.md", body: []byte("readme")}},
			wantErr: ErrBinaryNotInArchive,
		},
		{
			name:    "binary in another folder",
			entries: []archiveEntry{{name: "elsewhere/" + filepath.Base(a.binary), body: fakeBinary}},
			wantErr: ErrBinaryNotInArchive,
		},
		{
			name:    "binary is a symlink",
			entries: []archiveEntry{{name: a.binary, typeflag: tar.TypeSymlink, linkname: "/bin/sh"}},
			wantErr: ErrUnsafeArchive,
		},
		{
			name: "binary is a hard link",
			entries: []archiveEntry{
				{name: dir + "/README.md", body: []byte("readme")},
				{name: a.binary, typeflag: tar.TypeLink, linkname: dir + "/README.md"},
			},
			tarOnly: true,
			wantErr: ErrUnsafeArchive,
		},
		{
			name: "binary listed twice",
			entries: []archiveEntry{
				{name: a.binary, body: fakeBinary},
				{name: a.binary, body: []byte("second copy")},
			},
			wantErr: ErrUnsafeArchive,
		},
		{
			name: "entry escapes the archive",
			entries: []archiveEntry{
				{name: "../../evil", body: []byte("evil")},
				{name: a.binary, body: fakeBinary},
			},
			wantErr: ErrUnsafeArchive,
		},
		{
			name: "absolute entry",
			entries: []archiveEntry{
				{name: "/tmp/evil", body: []byte("evil")},
				{name: a.binary, body: fakeBinary},
			},
			wantErr: ErrUnsafeArchive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tarOnly && zipOnly {
				t.Skip("zip archives have no hard links")
			}
			rs := newReleaseServer(t)
			archive := buildArchive(t, a, tt.entries)
			rs.publish(a, archive, sumLine(a.archive, archive))
			dest := t.TempDir()

			_, err := installRelease(t.Context(), a, dest)
			require.ErrorIs(t, err, tt.wantErr)
			requireStepError(t, err, a, "extract binary")
			requireNothingInstalled(t, dest)
		})
	}

	t.Run("corrupt archive", func(t *testing.T) {
		rs := newReleaseServer(t)
		archive := []byte("not an archive")
		rs.publish(a, archive, sumLine(a.archive, archive))
		dest := t.TempDir()

		_, err := installRelease(t.Context(), a, dest)
		requireStepError(t, err, a, "extract binary")
		requireNothingInstalled(t, dest)
	})
}

func TestExtractBinaryCapsSize(t *testing.T) {
	for _, archiveName := range []string{"tool.tar.gz", "tool.zip"} {
		t.Run(archiveName, func(t *testing.T) {
			entries := []archiveEntry{{name: "tool", body: bytes.Repeat([]byte("x"), 64)}}
			var data []byte
			if strings.HasSuffix(archiveName, ".zip") {
				data = buildZip(t, entries)
			} else {
				data = buildTarGz(t, entries)
			}
			archivePath := filepath.Join(t.TempDir(), "archive")
			require.NoError(t, os.WriteFile(archivePath, data, 0o600))

			var out bytes.Buffer
			err := extractBinary(archivePath, archiveName, "tool", 16, &out)
			require.ErrorIs(t, err, ErrDownloadTooLarge)

			out.Reset()
			require.NoError(t, extractBinary(archivePath, archiveName, "tool", 64, &out))
			assert.Equal(t, 64, out.Len())
		})
	}
}

func TestExtractBinaryZipSymlink(t *testing.T) {
	data := buildZip(t, []archiveEntry{{name: "tool", linkname: "/bin/sh"}})
	archivePath := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.WriteFile(archivePath, data, 0o600))

	err := extractBinary(archivePath, "tool.zip", "tool", maxBinaryBytes, io.Discard)
	require.ErrorIs(t, err, ErrUnsafeArchive)
}

func TestReleaseAssetNames(t *testing.T) {
	tests := []struct {
		name          string
		asset         releaseAsset
		wantArchive   string
		wantChecksums string
		wantBinary    string
	}{
		{
			name:          "golangci-lint darwin arm64",
			asset:         golangciLintAsset("v2.15.0", "darwin", "arm64"),
			wantArchive:   "golangci-lint-2.15.0-darwin-arm64.tar.gz",
			wantChecksums: "golangci-lint-2.15.0-checksums.txt",
			wantBinary:    "golangci-lint-2.15.0-darwin-arm64/golangci-lint",
		},
		{
			name:          "golangci-lint linux amd64",
			asset:         golangciLintAsset("v2.15.0", "linux", "amd64"),
			wantArchive:   "golangci-lint-2.15.0-linux-amd64.tar.gz",
			wantChecksums: "golangci-lint-2.15.0-checksums.txt",
			wantBinary:    "golangci-lint-2.15.0-linux-amd64/golangci-lint",
		},
		{
			name:          "golangci-lint linux arm",
			asset:         golangciLintAsset("v2.15.0", "linux", "arm"),
			wantArchive:   "golangci-lint-2.15.0-linux-armv6.tar.gz",
			wantChecksums: "golangci-lint-2.15.0-checksums.txt",
			wantBinary:    "golangci-lint-2.15.0-linux-armv6/golangci-lint",
		},
		{
			name:          "golangci-lint windows amd64",
			asset:         golangciLintAsset("v2.15.0", "windows", "amd64"),
			wantArchive:   "golangci-lint-2.15.0-windows-amd64.zip",
			wantChecksums: "golangci-lint-2.15.0-checksums.txt",
			wantBinary:    "golangci-lint-2.15.0-windows-amd64/golangci-lint.exe",
		},
		{
			name:          "gitleaks darwin arm64",
			asset:         gitleaksAsset("v8.30.1", "darwin", "arm64"),
			wantArchive:   "gitleaks_8.30.1_darwin_arm64.tar.gz",
			wantChecksums: "gitleaks_8.30.1_checksums.txt",
			wantBinary:    "gitleaks",
		},
		{
			name:          "gitleaks darwin amd64",
			asset:         gitleaksAsset("v8.30.1", "darwin", "amd64"),
			wantArchive:   "gitleaks_8.30.1_darwin_x64.tar.gz",
			wantChecksums: "gitleaks_8.30.1_checksums.txt",
			wantBinary:    "gitleaks",
		},
		{
			name:          "gitleaks linux 386",
			asset:         gitleaksAsset("v8.30.1", "linux", "386"),
			wantArchive:   "gitleaks_8.30.1_linux_x32.tar.gz",
			wantChecksums: "gitleaks_8.30.1_checksums.txt",
			wantBinary:    "gitleaks",
		},
		{
			name:          "gitleaks linux arm",
			asset:         gitleaksAsset("v8.30.1", "linux", "arm"),
			wantArchive:   "gitleaks_8.30.1_linux_armv6.tar.gz",
			wantChecksums: "gitleaks_8.30.1_checksums.txt",
			wantBinary:    "gitleaks",
		},
		{
			name:          "gitleaks windows amd64",
			asset:         gitleaksAsset("v8.30.1", "windows", "amd64"),
			wantArchive:   "gitleaks_8.30.1_windows_x64.zip",
			wantChecksums: "gitleaks_8.30.1_checksums.txt",
			wantBinary:    "gitleaks.exe",
		},
		{
			// No x64 guess for a platform gitleaks doesn't publish: the name has
			// no checksum, so the install fails clearly instead of installing a
			// binary that can't run here.
			name:          "gitleaks unpublished platform keeps its own name",
			asset:         gitleaksAsset("v8.30.1", "linux", "ppc64"),
			wantArchive:   "gitleaks_8.30.1_linux_ppc64.tar.gz",
			wantChecksums: "gitleaks_8.30.1_checksums.txt",
			wantBinary:    "gitleaks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantArchive, tt.asset.archive)
			assert.Equal(t, tt.wantChecksums, tt.asset.checksums)
			assert.Equal(t, tt.wantBinary, tt.asset.binary)
		})
	}
}

func TestReleaseTag(t *testing.T) {
	tests := []struct {
		version string
		want    string
		wantErr bool
	}{
		{version: "v2.15.0", want: "v2.15.0"},
		{version: "2.15.0", want: "v2.15.0"},
		{version: "v8.30.1-rc.1", want: "v8.30.1-rc.1"},
		{version: "v2", wantErr: true},
		{version: "latest", wantErr: true},
		{version: "v1.2.3/../../evil", wantErr: true},
		{version: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := releaseTag(tt.version)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidVersion)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLatestReleaseTag(t *testing.T) {
	t.Run("follows the latest redirect", func(t *testing.T) {
		rs := newReleaseServer(t)
		rs.latest["acme/faketool"] = "v1.4.2"

		tag, err := latestReleaseTag(t.Context(), "acme/faketool")
		require.NoError(t, err)
		assert.Equal(t, "v1.4.2", tag)
	})

	t.Run("no redirect", func(t *testing.T) {
		newReleaseServer(t)

		_, err := latestReleaseTag(t.Context(), "acme/faketool")
		require.ErrorIs(t, err, ErrLatestNotResolved)
	})

	t.Run("redirect to an invalid tag", func(t *testing.T) {
		rs := newReleaseServer(t)
		rs.latest["acme/faketool"] = "nightly"

		_, err := latestReleaseTag(t.Context(), "acme/faketool")
		require.ErrorIs(t, err, ErrInvalidVersion)
	})
}

// setupInstallDirs points GOBIN and PATH at fresh temp dirs, so installs land
// in the test's own folder and the binary under test is found there only.
func setupInstallDirs(t *testing.T) string {
	t.Helper()
	gobin := t.TempDir()
	t.Setenv("GOBIN", gobin)
	t.Setenv("GOPATH", t.TempDir())
	t.Setenv("PATH", gobin)
	return gobin
}

// failOnInstallCommand fails the test if any install command (go install,
// curl, sh) is run.
func failOnInstallCommand(t *testing.T) {
	t.Helper()
	fakeInstall(t, func(_ context.Context, _ []string, name string, args ...string) ([]byte, error) {
		t.Errorf("unexpected install command: %s %s", name, strings.Join(args, " "))
		return nil, errSimulatedInstall
	})
}

func TestInstallGolangciLintFromVerifiedRelease(t *testing.T) {
	resetToolState(t)
	gobin := setupInstallDirs(t)
	failOnInstallCommand(t) // no install.sh, no curl | sh, no go install

	a := golangciLintAsset("v2.15.0", runtime.GOOS, runtime.GOARCH)
	rs := newReleaseServer(t)
	archive := goodArchive(t, a)
	sbom := []byte(`{"sbom":true}`)
	rs.publish(a, archive, sumLine(a.archive, archive)+sumLine(a.archive+".sbom.json", sbom))

	require.NoError(t, installGolangciLint(t.Context(), "v2.15.0"))
	assert.FileExists(t, filepath.Join(gobin, filepath.Base(a.binary)))
}

func TestInstallGolangciLintFallsBackToGoInstall(t *testing.T) {
	origAttempts, origDelay := GetRetryConfig()
	t.Cleanup(func() { SetRetryConfig(origAttempts, origDelay) })
	SetRetryConfig(1, time.Millisecond)

	tests := []struct {
		name        string
		version     string
		fallbackErr error
		wantTarget  string
	}{
		{
			name:       "v2 uses the v2 module path",
			version:    "v2.15.0",
			wantTarget: "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0",
		},
		{
			name:       "v1 uses the v1 module path",
			version:    "v1.64.8",
			wantTarget: "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8",
		},
		{
			name:        "both methods fail",
			version:     "v2.15.0",
			fallbackErr: errSimulatedInstall,
			wantTarget:  "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetToolState(t)
			setupInstallDirs(t)

			a := golangciLintAsset(tt.version, runtime.GOOS, runtime.GOARCH)
			rs := newReleaseServer(t)
			archive := goodArchive(t, a)
			rs.publish(a, archive, sumLine(a.archive, []byte("tampered")))

			var commands []string
			fakeInstall(t, func(_ context.Context, _ []string, name string, args ...string) ([]byte, error) {
				commands = append(commands, name+" "+strings.Join(args, " "))
				return []byte("go install output"), tt.fallbackErr
			})

			err := installGolangciLint(t.Context(), tt.version)
			assert.Equal(t, []string{"go install " + tt.wantTarget}, commands)
			if tt.fallbackErr == nil {
				require.NoError(t, err)
				return
			}
			// Both failures are reported, each with its tool, version, and step.
			require.ErrorIs(t, err, ErrChecksumMismatch)
			requireStepError(t, err, a, "verify checksum")
			assert.Contains(t, err.Error(), "golangci-lint v2.15.0: go install fallback:")
			assert.Contains(t, err.Error(), "go install output")
		})
	}
}

func TestInstallToolVerifiesInstalledBinaryIsFound(t *testing.T) {
	// A release install that succeeds but leaves the binary off PATH is not a
	// silent success.
	resetToolState(t)
	setupInstallDirs(t)
	t.Setenv("PATH", t.TempDir())
	failOnInstallCommand(t)

	a := gitleaksAsset("v8.30.1", runtime.GOOS, runtime.GOARCH)
	rs := newReleaseServer(t)
	archive := buildArchive(t, a, []archiveEntry{{name: a.binary, body: fakeBinary}})
	rs.publish(a, archive, sumLine(a.archive, archive))

	err := InstallTool(t.Context(), &Tool{Name: toolGitleaks, Version: "v8.30.1", Binary: toolGitleaks})
	require.ErrorIs(t, err, ErrToolNotInPath)
}

func TestEnsureInstalledAutoInstallDisabled(t *testing.T) {
	resetToolState(t)
	gobin := setupInstallDirs(t)
	t.Setenv(EnvAutoInstallTools, "false")
	t.Setenv("GO_PRE_COMMIT_GOLANGCI_LINT_VERSION", "v2.15.0")
	t.Setenv("GO_PRE_COMMIT_GITLEAKS_VERSION", "v8.30.1")
	t.Setenv("GO_PRE_COMMIT_FUMPT_VERSION", "v0.9.2")
	failOnInstallCommand(t)
	rs := newReleaseServer(t)

	tests := []struct {
		tool        string
		wantCommand string
	}{
		{tool: toolGolangciLint, wantCommand: "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0"},
		{tool: toolGitleaks, wantCommand: "go install github.com/zricethezav/gitleaks/v8@v8.30.1"},
		{tool: "gofumpt", wantCommand: "go install mvdan.cc/gofumpt@v0.9.2"},
		{tool: "goimports", wantCommand: "go install golang.org/x/tools/cmd/goimports@latest"},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			err := EnsureInstalled(t.Context(), tt.tool)
			require.ErrorIs(t, err, ErrAutoInstallDisabled)
			assert.Contains(t, err.Error(), tt.tool)
			assert.Contains(t, err.Error(), EnvAutoInstallTools+"=false")
			assert.Contains(t, err.Error(), tt.wantCommand)
			assert.Equal(t, tt.wantCommand, InstallCommand(tt.tool))
		})
	}

	t.Run("install all tools", func(t *testing.T) {
		err := InstallAllTools(t.Context())
		require.ErrorIs(t, err, ErrInstallFailed)
		assert.Contains(t, err.Error(), EnvAutoInstallTools+"=false")
	})

	assert.Zero(t, rs.requestCount(), "no release may be downloaded")
	requireNothingInstalled(t, gobin)
}

func TestEnsureInstalledAutoInstallDisabledKeepsInstalledTools(t *testing.T) {
	// The opt-out only stops installs: a tool already on PATH is used as is.
	resetToolState(t)
	gobin := setupInstallDirs(t)
	t.Setenv(EnvAutoInstallTools, "false")
	failOnInstallCommand(t)

	name := toolGitleaks
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	require.NoError(t, os.WriteFile(filepath.Join(gobin, name), fakeBinary, 0o755)) //nolint:gosec // test binary must be executable

	require.NoError(t, EnsureInstalled(t.Context(), toolGitleaks))
}

func TestAutoInstallEnabled(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "", want: true},
		{value: "true", want: true},
		{value: "1", want: true},
		{value: "false", want: false},
		{value: "FALSE", want: false},
		{value: "0", want: false},
		{value: " false ", want: false},
		// Like every other boolean setting, an unreadable value keeps the default.
		{value: "maybe", want: true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.value), func(t *testing.T) {
			t.Setenv(EnvAutoInstallTools, tt.value)
			assert.Equal(t, tt.want, AutoInstallEnabled())
		})
	}
}

func TestInstallCommand(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		tool string
		want string
	}{
		{
			name: "golangci-lint v2",
			env:  map[string]string{"GO_PRE_COMMIT_GOLANGCI_LINT_VERSION": "v2.15.0"},
			tool: toolGolangciLint,
			want: "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0",
		},
		{
			name: "golangci-lint v1",
			env:  map[string]string{"GO_PRE_COMMIT_GOLANGCI_LINT_VERSION": "v1.64.8"},
			tool: toolGolangciLint,
			want: "go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8",
		},
		{
			name: "golangci-lint latest",
			env:  map[string]string{"GO_PRE_COMMIT_GOLANGCI_LINT_VERSION": toolVersionLatest},
			tool: toolGolangciLint,
			want: "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest",
		},
		{
			name: "gitleaks",
			env:  map[string]string{"GO_PRE_COMMIT_GITLEAKS_VERSION": "v8.30.1"},
			tool: toolGitleaks,
			want: "go install github.com/zricethezav/gitleaks/v8@v8.30.1",
		},
		{
			name: "gofumpt default",
			tool: "gofumpt",
			want: "go install mvdan.cc/gofumpt@v0.8.0",
		},
		{
			name: "unknown tool",
			tool: "no-such-tool",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetToolState(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			assert.Equal(t, tt.want, InstallCommand(tt.tool))
		})
	}
}
