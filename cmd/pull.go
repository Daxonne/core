package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/daxonne/core/internal/config"
	"github.com/daxonne/core/internal/plugin"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull the database schema and cache it locally",
	Long: `Pull connects to the database configured in daxonne.yaml, reads the full schema
for the configured owner, and stores the result in .daxonne/schema.json.

Run this command whenever your database schema changes.`,
	RunE: runPull,
}

func runPull(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	color.Cyan("Connecting to %s database...", cfg.Database.Type)

	reader, err := plugin.GetSchemaReader(cfg.Database.Type)
	if err != nil {
		return err
	}

	if err := reader.Connect(cfg.Database.Connection); err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer reader.Close()

	color.Cyan("Reading schema for owner %q...", cfg.Database.Owner)

	s, err := reader.ReadSchema(cfg.Database.Owner)
	if err != nil {
		return fmt.Errorf("reading schema: %w", err)
	}

	if err := os.MkdirAll(".daxonne", 0755); err != nil {
		return fmt.Errorf("creating .daxonne directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling schema: %w", err)
	}

	if err := os.WriteFile(".daxonne/schema.json", data, 0644); err != nil {
		return fmt.Errorf("writing schema cache: %w", err)
	}

	// Compute totals for the summary.
	totalCols := 0
	totalFKs := 0
	for _, t := range s.Tables {
		totalCols += len(t.Columns)
		totalFKs += len(t.ForeignKeys)
	}

	color.Green("\nSchema pulled successfully!")
	fmt.Printf("  Tables      : %d\n", len(s.Tables))
	fmt.Printf("  Columns     : %d\n", totalCols)
	fmt.Printf("  Foreign keys: %d\n", totalFKs)
	fmt.Printf("  Cached at   : .daxonne/schema.json\n")

	return nil
}
