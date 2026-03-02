package template

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// TemplatesDir is where installed templates live inside the project work directory.
	TemplatesDir = ".daxonne/templates"

	// remoteRepoAPI is the GitHub Contents API base URL for the templates repository.
	remoteRepoAPI = "https://api.github.com/repos/Daxonne/templates/contents"

	// httpTimeout is the maximum time allowed for a single HTTP request.
	httpTimeout = 15 * time.Second
)

// githubFileEntry is the shape of one element returned by the GitHub Contents API.
type githubFileEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "file" or "dir"
	DownloadURL string `json:"download_url"`
}

// Install downloads a template from the Daxonne templates repository on GitHub
// and writes it into TemplatesDir/<name>. If the GitHub download fails (e.g. no
// network), it falls back to copying from a local "templates-src/<name>" directory.
func Install(name string) error {
	destDir := filepath.Join(TemplatesDir, name)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	if err := downloadFromGitHub(name, destDir); err == nil {
		return nil
	}

	// Fallback: copy from local templates-src/ (dev/offline use)
	srcDir := filepath.Join("templates-src", name)
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return fmt.Errorf("template %q not found remotely and no local fallback at %s", name, srcDir)
	}
	return copyDir(srcDir, destDir)
}

// IsInstalled reports whether the template has already been installed
// (i.e. whether .daxonne/templates/<name>/template.json exists).
func IsInstalled(name string) bool {
	_, err := os.Stat(filepath.Join(TemplatesDir, name, "template.json"))
	return err == nil
}

// downloadFromGitHub fetches all files in the remote templates/<name>/ directory
// and writes them to destDir.
func downloadFromGitHub(name, destDir string) error {
	client := &http.Client{Timeout: httpTimeout}

	apiURL := remoteRepoAPI + "/" + name
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("listing template directory: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned %d for %s", resp.StatusCode, apiURL)
	}

	var entries []githubFileEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return fmt.Errorf("decoding GitHub API response: %w", err)
	}

	for _, entry := range entries {
		if entry.Type != "file" || entry.DownloadURL == "" {
			continue
		}
		if err := downloadFile(client, entry.DownloadURL, filepath.Join(destDir, entry.Name)); err != nil {
			return fmt.Errorf("downloading %s: %w", entry.Name, err)
		}
	}

	return nil
}

// downloadFile fetches rawURL and writes the body to dstPath.
func downloadFile(client *http.Client, rawURL, dstPath string) error {
	resp, err := client.Get(rawURL)
	if err != nil {
		return fmt.Errorf("GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned %d", rawURL, resp.StatusCode)
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dstPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("writing %s: %w", dstPath, err)
	}

	return nil
}

// copyDir recursively copies the contents of src into dst.
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", src, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", dstPath, err)
			}
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file from src to dst, creating or truncating dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copying %s → %s: %w", src, dst, err)
	}

	return nil
}
