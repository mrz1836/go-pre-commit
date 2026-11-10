package tools

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallGitleaksFromVerifiedRelease(t *testing.T) {
	resetToolState(t)
	gobin := setupInstallDirs(t)
	failOnInstallCommand(t)

	a := gitleaksAsset("v8.30.1", runtime.GOOS, runtime.GOARCH)
	rs := newReleaseServer(t)
	archive := buildArchive(t, a, []archiveEntry{
		{name: "LICENSE", body: []byte("license")},
		{name: a.binary, body: fakeBinary},
		{name: "README.md", body: []byte("readme")},
	})
	rs.publish(a, archive, sumLine(a.archive, archive))

	require.NoError(t, installGitleaks(t.Context(), "v8.30.1"))
	assert.FileExists(t, filepath.Join(gobin, a.binary))
	assert.Equal(t, 1, rs.requested(assetPath(a, a.archive)))
}

func TestInstallGitleaksAcceptsVersionWithoutPrefix(t *testing.T) {
	resetToolState(t)
	gobin := setupInstallDirs(t)
	failOnInstallCommand(t)

	a := gitleaksAsset("v8.30.1", runtime.GOOS, runtime.GOARCH)
	rs := newReleaseServer(t)
	archive := buildArchive(t, a, []archiveEntry{{name: a.binary, body: fakeBinary}})
	rs.publish(a, archive, sumLine(a.archive, archive))

	require.NoError(t, installGitleaks(t.Context(), "8.30.1"))
	assert.FileExists(t, filepath.Join(gobin, a.binary))
}

func TestInstallGitleaksResolvesLatest(t *testing.T) {
	resetToolState(t)
	gobin := setupInstallDirs(t)
	failOnInstallCommand(t)

	a := gitleaksAsset("v8.31.0", runtime.GOOS, runtime.GOARCH)
	rs := newReleaseServer(t)
	rs.latest[gitleaksRepo] = "v8.31.0"
	archive := buildArchive(t, a, []archiveEntry{{name: a.binary, body: fakeBinary}})
	rs.publish(a, archive, sumLine(a.archive, archive))

	require.NoError(t, installGitleaks(t.Context(), toolVersionLatest))
	assert.FileExists(t, filepath.Join(gobin, a.binary))
}

func TestInstallGitleaksFailures(t *testing.T) {
	a := gitleaksAsset("v8.30.1", runtime.GOOS, runtime.GOARCH)

	tests := []struct {
		name    string
		publish func(t *testing.T, rs *releaseServer)
		step    string
		wantErr error
	}{
		{
			name: "checksum mismatch",
			publish: func(t *testing.T, rs *releaseServer) {
				archive := buildArchive(t, a, []archiveEntry{{name: a.binary, body: fakeBinary}})
				rs.publish(a, archive, sumLine(a.archive, []byte("tampered")))
			},
			step:    "verify checksum",
			wantErr: ErrChecksumMismatch,
		},
		{
			name: "binary missing from archive",
			publish: func(t *testing.T, rs *releaseServer) {
				archive := buildArchive(t, a, []archiveEntry{{name: "README.md", body: []byte("no binary here")}})
				rs.publish(a, archive, sumLine(a.archive, archive))
			},
			step:    "extract binary",
			wantErr: ErrBinaryNotInArchive,
		},
		{
			name: "release not published",
			publish: func(*testing.T, *releaseServer) {
				// Nothing is served, so the checksums file is not found.
			},
			step:    "download checksums",
			wantErr: ErrDownloadFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetToolState(t)
			gobin := setupInstallDirs(t)
			failOnInstallCommand(t)
			rs := newReleaseServer(t)
			tt.publish(t, rs)

			err := installGitleaks(t.Context(), "v8.30.1")
			require.ErrorIs(t, err, tt.wantErr)
			requireStepError(t, err, a, tt.step)
			requireNothingInstalled(t, gobin)
		})
	}

	t.Run("invalid version", func(t *testing.T) {
		resetToolState(t)
		gobin := setupInstallDirs(t)
		failOnInstallCommand(t)
		rs := newReleaseServer(t)

		err := installGitleaks(t.Context(), "v8.30.1/../../other")
		require.ErrorIs(t, err, ErrInvalidVersion)
		require.ErrorIs(t, err, ErrInstallFailed)
		assert.Zero(t, rs.requestCount())
		requireNothingInstalled(t, gobin)
	})
}
