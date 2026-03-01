// Package update provides self-update functionality from GitHub releases.
package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/go-github/v68/github"
)

const (
	// GitHubOwner is the GitHub repository owner.
	GitHubOwner = "nathabonfim59"
	// GitHubRepo is the GitHub repository name.
	GitHubRepo = "hubbiott"
)

// ReleaseInfo contains information about a GitHub release.
type ReleaseInfo struct {
	Version      string
	DownloadURL  string
	Checksum     string
	PublishedAt  time.Time
	ReleaseNotes string
}

// Updater handles self-update operations from GitHub releases.
type Updater struct {
	client *github.Client
}

// NewUpdater creates a new Updater instance.
func NewUpdater() *Updater {
	return &Updater{
		client: github.NewClient(nil),
	}
}

// NewUpdaterWithToken creates a new Updater with a GitHub token for higher rate limits.
func NewUpdaterWithToken(token string) *Updater {
	return &Updater{
		client: github.NewClient(nil).WithAuthToken(token),
	}
}

// CheckForUpdate checks if a newer version is available on GitHub releases.
// Returns the release info if an update is available, or nil if current version is up to date.
func (u *Updater) CheckForUpdate(ctx context.Context, currentVersion string) (*ReleaseInfo, error) {
	release, _, err := u.client.Repositories.GetLatestRelease(ctx, GitHubOwner, GitHubRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.GetTagName(), "v")
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	// If current version is "dev" or unknown, always suggest update
	if currentVersion == "dev" || currentVersion == "unknown" || currentVersion == "" {
		return u.buildReleaseInfo(release, latestVersion)
	}

	// Compare versions (simple string comparison, semantic versioning assumed)
	if latestVersion != currentVersion {
		return u.buildReleaseInfo(release, latestVersion)
	}

	return nil, nil // Already up to date
}

// GetReleaseByVersion fetches a specific release by version tag.
func (u *Updater) GetReleaseByVersion(ctx context.Context, version string) (*ReleaseInfo, error) {
	tag := "v" + strings.TrimPrefix(version, "v")
	release, _, err := u.client.Repositories.GetReleaseByTag(ctx, GitHubOwner, GitHubRepo, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to get release %s: %w", tag, err)
	}

	return u.buildReleaseInfo(release, strings.TrimPrefix(release.GetTagName(), "v"))
}

func (u *Updater) buildReleaseInfo(release *github.RepositoryRelease, version string) (*ReleaseInfo, error) {
	downloadURL, checksum, err := u.findAssetURLs(release)
	if err != nil {
		return nil, err
	}

	return &ReleaseInfo{
		Version:      version,
		DownloadURL:  downloadURL,
		Checksum:     checksum,
		PublishedAt:  release.GetPublishedAt().Time,
		ReleaseNotes: release.GetBody(),
	}, nil
}

// findAssetURLs finds the download URL and checksum for the current platform.
func (u *Updater) findAssetURLs(release *github.RepositoryRelease) (string, string, error) {
	assetName := getAssetName()
	var downloadURL, checksumURL string

	for _, asset := range release.Assets {
		name := asset.GetName()
		if strings.Contains(name, assetName) && strings.HasSuffix(name, ".tar.gz") {
			downloadURL = asset.GetBrowserDownloadURL()
		}
		if strings.Contains(name, "checksums") {
			checksumURL = asset.GetBrowserDownloadURL()
		}
	}

	if downloadURL == "" {
		return "", "", fmt.Errorf("no download asset found for platform %s", assetName)
	}

	// Fetch checksum if available
	var checksum string
	if checksumURL != "" {
		checksums, err := u.fetchChecksums(checksumURL)
		if err == nil {
			checksum = findChecksumForAsset(checksums, filepath.Base(downloadURL))
		}
	}

	return downloadURL, checksum, nil
}

// getAssetName returns the asset name pattern for the current platform.
func getAssetName() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map architecture names to match goreleaser output
	if arch == "amd64" {
		arch = "x86_64"
	}

	return fmt.Sprintf("%s_%s", strings.Title(os), arch)
}

// fetchChecksums downloads and parses the checksums file.
func (u *Updater) fetchChecksums(url string) (map[string]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download checksums: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read checksums: %w", err)
	}

	checksums := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			checksums[parts[1]] = parts[0]
		}
	}

	return checksums, nil
}

// findChecksumForAsset finds the checksum for a specific asset name.
func findChecksumForAsset(checksums map[string]string, assetName string) string {
	// Try exact match first
	if checksum, ok := checksums[assetName]; ok {
		return checksum
	}

	// Try without archive extension
	baseName := strings.TrimSuffix(assetName, ".tar.gz")
	for name, checksum := range checksums {
		if strings.HasPrefix(name, baseName) {
			return checksum
		}
	}

	return ""
}

// DownloadBinary downloads the binary from the given URL to a temporary file.
func (u *Updater) DownloadBinary(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download binary: status %d", resp.StatusCode)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "hubbiott-update-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Copy the download to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write download: %w", err)
	}

	return tmpFile.Name(), nil
}

// VerifyChecksum verifies the SHA256 checksum of a file.
func (u *Updater) VerifyChecksum(filePath, expectedChecksum string) error {
	if expectedChecksum == "" {
		return nil // No checksum to verify
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// ExtractBinary extracts the binary from the tar.gz archive.
func (u *Updater) ExtractBinary(archivePath string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "hubbiott-extract-")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Extract using Go's archive/tar with compress/gzip for portability
	extractedPath := filepath.Join(tmpDir, "hubbiott")

	if err := u.extractTarGz(archivePath, tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to extract archive: %w", err)
	}

	return extractedPath, nil
}

// extractTarGz extracts a tar.gz archive to the destination directory.
func (u *Updater) extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		// Construct the destination path
		destPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			outFile.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}
			if err := os.Symlink(header.Linkname, destPath); err != nil {
				return fmt.Errorf("failed to create symlink: %w", err)
			}
		}
	}

	return nil
}

// ReplaceBinary atomically replaces the current binary with the new one.
func (u *Updater) ReplaceBinary(newBinaryPath string) error {
	// Get the current executable path
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current binary path: %w", err)
	}

	// Get the real path (resolve symlinks)
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("failed to resolve binary path: %w", err)
	}

	// Make the new binary executable
	if err := os.Chmod(newBinaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make new binary executable: %w", err)
	}

	// Create backup of current binary
	backupPath := currentBinary + ".backup"
	if err := copyFile(currentBinary, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Rename new binary to current binary location
	if err := os.Rename(newBinaryPath, currentBinary); err != nil {
		// Try to restore backup on failure
		os.Rename(backupPath, currentBinary)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Remove backup on success
	os.Remove(backupPath)

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// Update performs the full update process.
func (u *Updater) Update(ctx context.Context, currentVersion string) (*ReleaseInfo, error) {
	// Check for update
	release, err := u.CheckForUpdate(ctx, currentVersion)
	if err != nil {
		return nil, fmt.Errorf("check for update failed: %w", err)
	}

	if release == nil {
		return nil, nil // Already up to date
	}

	// Download the new binary
	archivePath, err := u.DownloadBinary(ctx, release.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(archivePath)

	// Verify checksum if available
	if release.Checksum != "" {
		if err := u.VerifyChecksum(archivePath, release.Checksum); err != nil {
			return nil, fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// Extract the binary
	newBinaryPath, err := u.ExtractBinary(archivePath)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}
	defer os.RemoveAll(filepath.Dir(newBinaryPath))

	// Replace the current binary
	if err := u.ReplaceBinary(newBinaryPath); err != nil {
		return nil, fmt.Errorf("binary replacement failed: %w", err)
	}

	return release, nil
}
