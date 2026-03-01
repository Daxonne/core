package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/daxonne/core/internal/config"
	"github.com/daxonne/core/internal/generator"
	"github.com/daxonne/core/internal/schema"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate code from the cached schema",
	Long: `Generate loads the schema from .daxonne/schema.json and applies every template
listed in daxonne.yaml, writing the produced files to the configured output directory.

Run 'daxonne pull' first to populate the schema cache.`,
	RunE: runGenerate,
}

func runGenerate(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Templates) == 0 {
		color.Yellow("No templates configured. Run: daxonne add <template-name>")
		return nil
	}

	raw, err := os.ReadFile(".daxonne/schema.json")
	if err != nil {
		return fmt.Errorf("reading schema cache — run 'daxonne pull' first: %w", err)
	}

	var s schema.Schema
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("parsing schema cache: %w", err)
	}

	color.Cyan("Generating code for %d table(s) using template(s): %v\n",
		len(s.Tables), cfg.Templates)

	engine := generator.NewEngine()

	files, err := engine.GenerateFromTemplates(&s, cfg)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	if err := os.MkdirAll(cfg.Output.Path, 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	for _, f := range files {
		dest := filepath.Join(cfg.Output.Path, f.Path)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", f.Path, err)
		}
		if err := os.WriteFile(dest, []byte(f.Content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", f.Path, err)
		}
		color.Green("  + %s", dest)
	}

	color.Green("\nDone! %d file(s) written to %s", len(files), cfg.Output.Path)
	return nil
}
