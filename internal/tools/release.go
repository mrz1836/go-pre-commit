package tools

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Tools installed from their GitHub release archives, and the limits a
// release download must stay within.
const (
	golangciLintRepo = "golangci/golangci-lint"
	gitleaksRepo     = "gitleaks/gitleaks"

	maxChecksumsBytes = 1 << 20   // a checksums file is a few KiB
	maxArchiveBytes   = 256 << 20 // release archives are tens of MiB
	maxBinaryBytes    = 512 << 20 // the binary inside, uncompressed
	maxExtractBytes   = 1 << 30   // everything decompressed while looking for it
	maxRedirects      = 10

	schemeHTTPS = "https"
	osWindows   = "windows"
)

// Errors returned by release installs. Each is wrapped in an install error
// that names the tool, the version, and the step that failed.
var (
	// ErrInsecureDownload is returned for a download, or a redirect, that is not over https
	ErrInsecureDownload = errors.New("refusing a download that is not over https")
	// ErrDownloadFailed is returned when a release server answers with an error status
	ErrDownloadFailed = errors.New("download failed")
	// ErrDownloadTooLarge is returned when a download or the binary in it exceeds its size limit
	ErrDownloadTooLarge = errors.New("exceeds the size limit")
	// ErrTooManyRedirects is returned when a download is redirected too many times
	ErrTooManyRedirects = errors.New("too many redirects")
	// ErrChecksumNotListed is returned when the checksums file has no line for the archive's exact name
	ErrChecksumNotListed = errors.New("the checksums file does not list the archive")
	// ErrChecksumAmbiguous is returned when the checksums file lists the archive's exact name more than once
	ErrChecksumAmbiguous = errors.New("the checksums file lists the archive more than once")
	// ErrChecksumMalformed is returned when the archive's line in the checksums file is not a SHA-256
	ErrChecksumMalformed = errors.New("malformed checksum")
	// ErrChecksumMismatch is returned when the archive's SHA-256 differs from the checksums file
	ErrChecksumMismatch = errors.New("checksum mismatch")
	// ErrBinaryNotInArchive is returned when the release archive does not hold the tool's binary
	ErrBinaryNotInArchive = errors.New("binary not found in archive")
	// ErrUnsafeArchive is returned for an archive entry that is a link, escapes the archive, or repeats the binary
	ErrUnsafeArchive = errors.New("unsafe archive entry")
	// ErrInvalidVersion is returned for a version that is not a release tag such as v1.2.3
	ErrInvalidVersion = errors.New("invalid release version")
	// ErrLatestNotResolved is returned when the latest release can't be found
	ErrLatestNotResolved = errors.New("could not resolve the latest release")
)

// releaseVersionRe matches a release version: v1.2.3, 1.2.3, or v1.2.3-rc.1.
var releaseVersionRe = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)

// releaseBaseURL is where release files are downloaded from, and
// releaseTransport carries the requests. They are package variables so tests
// can serve releases from an httptest server instead of the network.
//
//nolint:gochecknoglobals // Injectable seams so tests never reach the network
var (
	releaseBaseURL                     = "https://github.com"
	releaseTransport http.RoundTripper = http.DefaultTransport
)

// releaseAsset is one platform's archive of a tool's GitHub release.
type releaseAsset struct {
	tool      string // tool name, for messages
	version   string // release tag, such as v2.15.0
	repo      string // GitHub owner/name
	archive   string // archive file name in the release
	checksums string // checksums file name in the release
	binary    string // slash-separated path of the binary inside the archive
}

// golangciLintAsset describes golangci-lint's release archive for a platform,
// for example golangci-lint-2.15.0-darwin-arm64.tar.gz holding
// golangci-lint-2.15.0-darwin-arm64/golangci-lint.
func golangciLintAsset(tag, goos, goarch string) releaseAsset {
	num := strings.TrimPrefix(tag, "v")
	arch := goarch
	if goarch == "arm" {
		arch = "armv6" // the build that runs on every 32-bit ARM board
	}
	ext, bin := ".tar.gz", toolGolangciLint
	if goos == osWindows {
		ext, bin = ".zip", toolGolangciLint+".exe"
	}
	dir := fmt.Sprintf("%s-%s-%s-%s", toolGolangciLint, num, goos, arch)
	return releaseAsset{
		tool:      toolGolangciLint,
		version:   tag,
		repo:      golangciLintRepo,
		archive:   dir + ext,
		checksums: fmt.Sprintf("%s-%s-checksums.txt", toolGolangciLint, num),
		binary:    dir + "/" + bin,
	}
}

// gitleaksAsset describes gitleaks' release archive for a platform, for
// example gitleaks_8.30.1_darwin_arm64.tar.gz holding gitleaks. A platform
// gitleaks does not publish keeps its own name, so it has no checksum and the
// install fails rather than fetching another platform's binary.
func gitleaksAsset(tag, goos, goarch string) releaseAsset {
	num := strings.TrimPrefix(tag, "v")
	arch := goarch
	switch goarch {
	case "amd64":
		arch = "x64"
	case "386":
		arch = "x32"
	case "arm":
		arch = "armv6"
	}
	ext, bin := ".tar.gz", toolGitleaks
	if goos == osWindows {
		ext, bin = ".zip", toolGitleaks+".exe"
	}
	return releaseAsset{
		tool:      toolGitleaks,
		version:   tag,
		repo:      gitleaksRepo,
		archive:   fmt.Sprintf("%s_%s_%s_%s%s", toolGitleaks, num, goos, arch, ext),
		checksums: fmt.Sprintf("%s_%s_checksums.txt", toolGitleaks, num),
		binary:    bin,
	}
}

// url is the download URL of one of the release's files.
func (a releaseAsset) url(name string) string {
	return fmt.Sprintf("%s/%s/releases/download/%s/%s", releaseBaseURL, a.repo, a.version, name)
}

// stepError reports a failed install step of the asset.
func (a releaseAsset) stepError(step string, err error) error {
	return installStepError(a.tool, a.version, step, err)
}

// installStepError reports a failed install step, naming the tool and version.
func installStepError(tool, version, step string, err error) error {
	return fmt.Errorf("%w: %s %s: %s: %w", ErrInstallFailed, tool, version, step, err)
}

// releaseTag turns a configured version into a release tag: v1.2.3 and 1.2.3
// both become v1.2.3. Anything else is refused, so a version can never change
// the download URL's path.
func releaseTag(version string) (string, error) {
	v := strings.TrimSpace(version)
	if !releaseVersionRe.MatchString(v) {
		return "", fmt.Errorf("%w: %q", ErrInvalidVersion, version)
	}
	return "v" + strings.TrimPrefix(v, "v"), nil
}

// resolveReleaseTag returns the release tag to install for a configured
// version, looking up the newest release for "latest" (or no version).
func resolveReleaseTag(ctx context.Context, tool, repo, version string) (string, error) {
	if version != "" && version != toolVersionLatest {
		tag, err := releaseTag(version)
		if err != nil {
			return "", installStepError(tool, version, "check version", err)
		}
		return tag, nil
	}

	var tag string
	err := retryWithBackoff(ctx, "resolving the latest "+tool+" release", func() error {
		var lookupErr error
		tag, lookupErr = latestReleaseTag(ctx, repo)
		return lookupErr
	})
	if err != nil {
		return "", installStepError(tool, toolVersionLatest, "resolve latest release", err)
	}
	return tag, nil
}

// latestReleaseTag finds the tag of a repository's newest release from where
// GitHub redirects <repo>/releases/latest, which needs no API token.
func latestReleaseTag(ctx context.Context, repo string) (string, error) {
	latestURL := fmt.Sprintf("%s/%s/releases/latest", releaseBaseURL, repo)
	if err := requireHTTPS(latestURL); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Transport: releaseTransport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	location, err := resp.Location()
	if err != nil || resp.StatusCode < 300 || resp.StatusCode > 399 {
		return "", fmt.Errorf("%w: %s answered %s", ErrLatestNotResolved, latestURL, resp.Status)
	}
	prefix := "/" + repo + "/releases/tag/"
	if !strings.HasPrefix(location.Path, prefix) {
		return "", fmt.Errorf("%w: %s redirected to %s", ErrLatestNotResolved, latestURL, location.Redacted())
	}
	return releaseTag(strings.TrimPrefix(location.Path, prefix))
}

// installRelease installs the asset's binary into destDir and returns its
// path. It downloads the release's checksums file and takes the one line for
// the archive's exact name, downloads the archive and checks its SHA-256
// against that line, and only then extracts the binary, a regular file, into
// destDir. Nothing is written to destDir unless every step succeeds.
func installRelease(ctx context.Context, a releaseAsset, destDir string) (string, error) {
	var sums bytes.Buffer
	err := retryWithBackoff(ctx, "downloading "+a.checksums, func() error {
		sums.Reset()
		return fetch(ctx, a.url(a.checksums), maxChecksumsBytes, &sums)
	})
	if err != nil {
		return "", a.stepError("download checksums", err)
	}
	want, err := checksumFor(sums.Bytes(), a.archive)
	if err != nil {
		return "", a.stepError("read checksums", err)
	}

	tmpDir, err := os.MkdirTemp("", "go-pre-commit-"+a.tool+"-*")
	if err != nil {
		return "", a.stepError("download archive", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	archivePath := filepath.Join(tmpDir, "archive")

	got, err := fetchFile(ctx, a.url(a.archive), archivePath, maxArchiveBytes)
	if err != nil {
		return "", a.stepError("download archive", err)
	}
	if !bytes.Equal(got, want) {
		return "", a.stepError("verify checksum", fmt.Errorf("%w: %s has SHA-256 %x, but %s lists %x",
			ErrChecksumMismatch, a.archive, got, a.checksums, want))
	}

	return installBinary(destDir, path.Base(a.binary), func(w io.Writer) error {
		if extractErr := extractBinary(archivePath, a.archive, a.binary, maxBinaryBytes, w); extractErr != nil {
			return a.stepError("extract binary", extractErr)
		}
		return nil
	}, func(installErr error) error {
		return a.stepError("install binary", installErr)
	})
}

// checksumFor returns the SHA-256 that a checksums file (sha256sum format)
// lists for archive. The name must match a line's whole file name, and exactly
// one line may list it: a file such as golangci-lint's lists both an archive
// and "<archive>.sbom.json", which a substring match would confuse.
func checksumFor(sums []byte, archive string) ([]byte, error) {
	var want []byte
	matches := 0
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != archive {
			continue
		}
		matches++
		sum, err := hex.DecodeString(fields[0])
		if err != nil || len(sum) != sha256.Size {
			return nil, fmt.Errorf("%w for %s: %q", ErrChecksumMalformed, archive, fields[0])
		}
		want = sum
	}
	switch matches {
	case 0:
		return nil, fmt.Errorf("%w: %s", ErrChecksumNotListed, archive)
	case 1:
		return want, nil
	default:
		return nil, fmt.Errorf("%w: %s is listed %d times", ErrChecksumAmbiguous, archive, matches)
	}
}

// requireHTTPS refuses a URL that is not https.
func requireHTTPS(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != schemeHTTPS {
		return fmt.Errorf("%w: %s", ErrInsecureDownload, u.Redacted())
	}
	return nil
}

// fetch downloads rawURL into w over https, following only https redirects,
// and fails when the server answers with an error status or sends more than
// limit bytes.
func fetch(ctx context.Context, rawURL string, limit int64, w io.Writer) error {
	if err := requireHTTPS(rawURL); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	client := &http.Client{
		Transport: releaseTransport,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("%w: %s", ErrTooManyRedirects, rawURL)
			}
			if next.URL.Scheme != schemeHTTPS {
				return fmt.Errorf("%w: redirected to %s", ErrInsecureDownload, next.URL.Redacted())
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// The status text ("503 Service Unavailable") lets retryWithBackoff
		// retry server-side failures, as it does network errors.
		return fmt.Errorf("%w: %s answered %s", ErrDownloadFailed, rawURL, resp.Status)
	}
	if resp.ContentLength > limit {
		return fmt.Errorf("%w: %s is %d bytes, over %d", ErrDownloadTooLarge, rawURL, resp.ContentLength, limit)
	}
	n, err := io.Copy(w, io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return err
	}
	if n > limit {
		return fmt.Errorf("%w: %s is over %d bytes", ErrDownloadTooLarge, rawURL, limit)
	}
	return nil
}

// fetchFile downloads rawURL into the file at dest, retrying network errors,
// and returns the SHA-256 of what it wrote.
func fetchFile(ctx context.Context, rawURL, dest string, limit int64) ([]byte, error) {
	var sum []byte
	err := retryWithBackoff(ctx, "downloading "+path.Base(rawURL), func() error {
		f, err := os.Create(dest) //nolint:gosec // dest is a fixed name in a fresh temp dir
		if err != nil {
			return err
		}
		h := sha256.New()
		fetchErr := fetch(ctx, rawURL, limit, io.MultiWriter(f, h))
		if closeErr := f.Close(); fetchErr == nil {
			fetchErr = closeErr
		}
		if fetchErr != nil {
			return fetchErr
		}
		sum = h.Sum(nil)
		return nil
	})
	return sum, err
}

// installBinary writes a binary named name into destDir. fill writes the
// binary into a temporary file in destDir, which is made executable and then
// renamed over any older copy, so a failed install leaves no partial file and
// never replaces a working binary. fill's error is returned as is; other
// failures are passed through wrap.
func installBinary(destDir, name string, fill func(w io.Writer) error, wrap func(error) error) (string, error) {
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return "", wrap(err)
	}
	tmp, err := os.CreateTemp(destDir, "."+name+".*.tmp")
	if err != nil {
		return "", wrap(err)
	}
	tmpPath := tmp.Name()
	installed := false
	defer func() {
		if !installed {
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if err = fill(tmp); err != nil {
		return "", err
	}
	if err = tmp.Close(); err != nil {
		return "", wrap(err)
	}
	if err = os.Chmod(tmpPath, 0o755); err != nil { //nolint:gosec // a tool binary must be executable, as go install makes it
		return "", wrap(err)
	}
	dest := filepath.Join(destDir, name)
	if err = os.Rename(tmpPath, dest); err != nil {
		return "", wrap(err)
	}
	installed = true
	return dest, nil
}

// extractBinary copies the archive entry named entry (a slash-separated path)
// from the .tar.gz or .zip archive at archivePath into w. The entry must be a
// regular file of at most limit bytes and appear exactly once, and no entry
// may be absolute or climb out of the archive; nothing else is extracted.
func extractBinary(archivePath, archiveName, entry string, limit int64, w io.Writer) error {
	if strings.HasSuffix(archiveName, ".zip") {
		return extractFromZip(archivePath, entry, limit, w)
	}
	return extractFromTarGz(archivePath, entry, limit, w)
}

// entryName returns an archive entry's cleaned name, refusing one that is
// absolute or escapes the archive.
func entryName(name string) (string, error) {
	if strings.Contains(name, `\`) || !filepath.IsLocal(filepath.FromSlash(name)) {
		return "", fmt.Errorf("%w: %q", ErrUnsafeArchive, name)
	}
	return path.Clean(name), nil
}

func extractFromTarGz(archivePath, entry string, limit int64, w io.Writer) error {
	f, err := os.Open(archivePath) //nolint:gosec // archivePath is a fixed name in a fresh temp dir
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(io.LimitReader(gz, maxExtractBytes))
	found := false
	for {
		hdr, nextErr := tr.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nextErr
		}
		name, nameErr := entryName(hdr.Name)
		if nameErr != nil {
			return nameErr
		}
		if name != entry {
			continue
		}
		if found {
			return fmt.Errorf("%w: %s appears more than once", ErrUnsafeArchive, entry)
		}
		found = true
		// Only a regular file: not a symlink, and not a hard link, whose
		// header carries a regular file mode. (Next reports the legacy
		// regular-file type as TypeReg.)
		if hdr.Typeflag != tar.TypeReg {
			return fmt.Errorf("%w: %s is not a regular file", ErrUnsafeArchive, entry)
		}
		if err = copyEntry(w, tr, hdr.Size, limit, entry); err != nil {
			return err
		}
	}
	if !found {
		return fmt.Errorf("%w: %s", ErrBinaryNotInArchive, entry)
	}
	return nil
}

func extractFromZip(archivePath, entry string, limit int64, w io.Writer) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = zr.Close() }()

	var match *zip.File
	for _, zf := range zr.File {
		name, nameErr := entryName(zf.Name)
		if nameErr != nil {
			return nameErr
		}
		if name != entry {
			continue
		}
		if match != nil {
			return fmt.Errorf("%w: %s appears more than once", ErrUnsafeArchive, entry)
		}
		match = zf
	}
	if match == nil {
		return fmt.Errorf("%w: %s", ErrBinaryNotInArchive, entry)
	}
	if !match.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is not a regular file", ErrUnsafeArchive, entry)
	}
	if match.UncompressedSize64 > uint64(limit) { //nolint:gosec // limit is a positive constant
		return fmt.Errorf("%w: %s is %d bytes, over %d", ErrDownloadTooLarge, entry, match.UncompressedSize64, limit)
	}
	rc, err := match.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	return copyEntry(w, rc, int64(match.UncompressedSize64), limit, entry) //nolint:gosec // bounded by limit above
}

// copyEntry copies an archive entry of the given size into w, refusing one
// over limit bytes. Reading to the end lets the archive reader check the
// entry's own size (and, for zip, its CRC-32).
func copyEntry(w io.Writer, r io.Reader, size, limit int64, entry string) error {
	if size > limit {
		return fmt.Errorf("%w: %s is %d bytes, over %d", ErrDownloadTooLarge, entry, size, limit)
	}
	n, err := io.Copy(w, io.LimitReader(r, limit+1))
	if err != nil {
		return err
	}
	if n > limit {
		return fmt.Errorf("%w: %s is over %d bytes", ErrDownloadTooLarge, entry, limit)
	}
	return nil
}
