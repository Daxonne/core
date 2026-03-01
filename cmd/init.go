package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/daxonne/core/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Daxonne project",
	Long: `Initialize a new Daxonne project by answering a few questions.

A daxonne.yaml configuration file will be created in the current directory.
You can edit it manually at any time.`,
	RunE: runInit,
}

func runInit(_ *cobra.Command, _ []string) error {
	color.Cyan("Welcome to Daxonne — let's configure your project.\n")

	scanner := bufio.NewScanner(os.Stdin)

	dbType := promptScanner(scanner, "Database type (oracle) [oracle]: ")
	if dbType == "" {
		dbType = "oracle"
	}

	connStr := promptScanner(scanner, "Connection string (e.g. oracle://user:pass@host:1521/svc): ")
	if connStr == "" {
		return fmt.Errorf("connection string is required")
	}

	owner := promptScanner(scanner, "Schema owner / namespace: ")
	if owner == "" {
		return fmt.Errorf("schema owner is required")
	}

	outputPath := promptScanner(scanner, "Output directory [./generated]: ")
	if outputPath == "" {
		outputPath = "./generated"
	}

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Type:       dbType,
			Connection: connStr,
			Owner:      strings.ToUpper(owner),
		},
		Output: config.OutputConfig{
			Path: outputPath,
		},
		Templates: []string{},
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	color.Green("\ndaxonne.yaml created successfully.\n")
	color.Yellow("Next steps:")
	fmt.Println("  daxonne pull             — fetch your database schema")
	fmt.Println("  daxonne add csharp-dapper — install a code template")
	fmt.Println("  daxonne generate          — generate code")

	return nil
}

// promptScanner prints question to stdout and returns the trimmed line read from scanner.
func promptScanner(scanner *bufio.Scanner, question string) string {
	fmt.Print(question)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}
