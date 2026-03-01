// Package generator provides the code-generation interfaces and Handlebars engine for Daxonne.
package generator

import (
	"github.com/daxonne/core/internal/config"
	"github.com/daxonne/core/internal/schema"
)

// ICodeGenerator is the interface every Daxonne code generator must implement.
// A generator takes a parsed schema and a project config, and produces a set of
// files ready to be written to disk.
type ICodeGenerator interface {
	// Name returns the human-readable name of this generator.
	Name() string

	// Generate produces code files from the given schema and configuration.
	Generate(s *schema.Schema, cfg *config.Config) ([]GeneratedFile, error)
}

// GeneratedFile represents a single file produced by the code generator.
type GeneratedFile struct {
	// Path is the relative path (including filename) where the file should be written.
	Path string

	// Content is the full text content of the generated file.
	Content string
}
