// Package cmd contains all Cobra CLI commands for the Daxonne tool.
package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "daxonne",
	Short: "Language-agnostic code generation from database schemas",
	Long: `Daxonne reads your database schema and generates code (DTOs, queries, repositories)
using a template-based, plugin-driven architecture.

Get started:
  daxonne init       Configure your project and create daxonne.yaml
  daxonne pull       Fetch your database schema and cache it locally
  daxonne add        Install a code generation template
  daxonne generate   Generate code from the cached schema`,
}

// Execute runs the root command. Called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(generateCmd)

	// Disable the auto-generated completion command to keep the CLI clean.
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
