package updater

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	githubRepo = "sabizmil/planck"
	binaryName = "planck"
)

// Release represents a GitHub release.
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckResult is the result of checking for an update.
type CheckResult struct {
	CurrentVersion string
	LatestVersion  string
	UpdateAvail    bool
	ArchiveURL     string
	ChecksumsURL   string
}

// Check queries GitHub for the latest release and compares with the current version.
func Check(currentVersion string) (*CheckResult, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)

	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "planck/"+currentVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found for %s", githubRepo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	result := &CheckResult{
		CurrentVersion: current,
		LatestVersion:  latestVersion,
		UpdateAvail:    current != latestVersion && current != "dev",
	}

	// Find the correct archive for this platform
	archiveName := fmt.Sprintf("%s_%s_%s_%s.tar.gz", binaryName, latestVersion, runtime.GOOS, runtime.GOARCH)
	for _, asset := range release.Assets {
		if asset.Name == archiveName {
			result.ArchiveURL = asset.BrowserDownloadURL
		}
		if asset.Name == "checksums.txt" {
			result.ChecksumsURL = asset.BrowserDownloadURL
		}
	}

	if result.UpdateAvail && result.ArchiveURL == "" {
		return nil, fmt.Errorf("no release binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return result, nil
}

// Update downloads and installs the latest version.
func Update(result *CheckResult) error {
	if !result.UpdateAvail {
		return nil
	}

	// Detect if installed via Homebrew
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}
	if strings.Contains(execPath, "Cellar") || strings.Contains(execPath, "homebrew") {
		return fmt.Errorf("planck appears to be installed via Homebrew — use 'brew upgrade planck' instead")
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "planck-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download archive
	archivePath := filepath.Join(tmpDir, "archive.tar.gz")
	if err := downloadFile(result.ArchiveURL, archivePath); err != nil {
		return fmt.Errorf("downloading archive: %w", err)
	}

	// Download and verify checksum
	if result.ChecksumsURL != "" {
		checksumsPath := filepath.Join(tmpDir, "checksums.txt")
		if err := downloadFile(result.ChecksumsURL, checksumsPath); err != nil {
			return fmt.Errorf("downloading checksums: %w", err)
		}
		if err := verifyChecksum(archivePath, checksumsPath); err != nil {
			return fmt.Errorf("checksum verification: %w", err)
		}
	}

	// Extract binary from archive
	binaryPath := filepath.Join(tmpDir, binaryName)
	if err := extractBinary(archivePath, binaryPath); err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	// Replace the current binary atomically
	if err := replaceBinary(execPath, binaryPath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url) //nolint:gosec // URL comes from GitHub API
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func verifyChecksum(archivePath, checksumsPath string) error {
	// Read checksums file
	data, err := os.ReadFile(checksumsPath)
	if err != nil {
		return err
	}

	// Compute actual checksum
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))

	// Find expected checksum
	archiveName := filepath.Base(archivePath)
	// Checksums file from GoReleaser uses the original archive name, not our temp name
	// We need to match by the archive pattern
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && strings.HasSuffix(parts[1], ".tar.gz") {
			// The checksums file has "hash  filename" format
			if actual == parts[0] {
				return nil // checksum matches
			}
		}
	}

	// Try matching by finding any .tar.gz entry that matches our OS/arch
	expected := ""
	target := fmt.Sprintf("%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && strings.HasSuffix(parts[1], target) {
			expected = parts[0]
			break
		}
	}

	if expected == "" {
		return fmt.Errorf("no checksum found for %s in checksums.txt", archiveName)
	}

	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

func extractBinary(archivePath, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if filepath.Base(header.Name) == binaryName && header.Typeflag == tar.TypeReg {
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
			if err != nil {
				return err
			}

			_, copyErr := io.Copy(out, tr)
			out.Close()
			if copyErr != nil {
				return copyErr
			}
			return nil
		}
	}

	return fmt.Errorf("binary %q not found in archive", binaryName)
}

func replaceBinary(currentPath, newPath string) error {
	// Get permissions of current binary
	info, err := os.Stat(currentPath)
	if err != nil {
		return err
	}

	// Set same permissions on new binary
	if err := os.Chmod(newPath, info.Mode()); err != nil {
		return err
	}

	// Atomic replace: rename new binary to current path
	// On Unix, this is atomic if same filesystem. If cross-filesystem,
	// we need to copy then rename.
	dir := filepath.Dir(currentPath)
	tmpPath := filepath.Join(dir, ".planck-update-tmp")

	// Copy new binary to same directory as current (ensures same filesystem)
	src, err := os.Open(newPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("writing to %s: %w (you may need to run with sudo)", dir, err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(tmpPath)
		return err
	}
	dst.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, currentPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}
