package template

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	// TemplatesDir is where installed templates live inside the project work directory.
	TemplatesDir = ".daxonne/templates"

	// SourceDir is the local directory used as the simulated remote template source.
	// In production this would be replaced by a network download.
	SourceDir = "templates-src"
)

// Install copies a template from SourceDir/<name> into TemplatesDir/<name>.
// It returns an error if the source directory does not exist.
func Install(name string) error {
	srcDir := filepath.Join(SourceDir, name)
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return fmt.Errorf("template source %q not found (expected at %s)", name, srcDir)
	}

	destDir := filepath.Join(TemplatesDir, name)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	return copyDir(srcDir, destDir)
}

// IsInstalled reports whether the template has already been installed
// (i.e. whether .daxonne/templates/<name>/template.json exists).
func IsInstalled(name string) bool {
	_, err := os.Stat(filepath.Join(TemplatesDir, name, "template.json"))
	return err == nil
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
