package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/daxonne/core/internal/config"
	tmpl "github.com/daxonne/core/internal/template"
)

var listFlag bool

var addCmd = &cobra.Command{
	Use:   "add <template-name>",
	Short: "Install a code generation template",
	Long: `Install a template from the Daxonne template registry.

Templates are downloaded from github.com/Daxonne/templates into .daxonne/templates/<name>/.
The template is automatically added to the templates list in daxonne.yaml.
Set GITHUB_TOKEN to avoid rate limiting on the GitHub API.

List available templates with:
  daxonne add --list`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().BoolVarP(&listFlag, "list", "l", false, "List all available templates")
}

func runAdd(_ *cobra.Command, args []string) error {
	if listFlag || len(args) == 0 {
		printTemplateRegistry()
		return nil
	}

	name := args[0]

	entry, ok := tmpl.FindByName(name)
	if !ok {
		color.Red("Template %q not found in the registry.\n", name)
		color.Yellow("Available templates:")
		printTemplateRegistry()
		return fmt.Errorf("unknown template: %s", name)
	}

	color.Cyan("Installing %s v%s (%s)...", entry.Name, entry.Version, entry.Description)

	if tmpl.IsInstalled(name) {
		color.Yellow("Template %q already installed — reinstalling.", name)
	}

	if err := tmpl.Install(name); err != nil {
		return fmt.Errorf("installing template: %w", err)
	}

	color.Green("Template %q installed successfully.", name)

	// Attempt to update daxonne.yaml; non-fatal if not found.
	cfg, err := config.Load()
	if err == nil {
		for _, t := range cfg.Templates {
			if t == name {
				return nil
			}
		}
		cfg.Templates = append(cfg.Templates, name)
		if saveErr := config.Save(cfg); saveErr != nil {
			color.Yellow("Warning: could not update daxonne.yaml: %v", saveErr)
		} else {
			color.Green("Added %q to daxonne.yaml.", name)
		}
	}

	return nil
}

func printTemplateRegistry() {
	fmt.Printf("\n%-25s %-10s %-14s %s\n", "NAME", "VERSION", "LANGUAGE", "DESCRIPTION")
	fmt.Printf("%-25s %-10s %-14s %s\n", "────────────────────────", "───────", "────────────", "───────────────────────────────────────")
	for _, e := range tmpl.LoadRegistry() {
		fmt.Printf("%-25s %-10s %-14s %s\n", e.Name, e.Version, e.Language, e.Description)
	}
	fmt.Println()
}
