// Package config handles reading and writing the daxonne.yaml project configuration.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config is the root configuration structure for a Daxonne project.
type Config struct {
	Database  DatabaseConfig `yaml:"database"  mapstructure:"database"`
	Output    OutputConfig   `yaml:"output"    mapstructure:"output"`
	Templates []string       `yaml:"templates" mapstructure:"templates"`
}

// DatabaseConfig holds the database connection parameters.
type DatabaseConfig struct {
	Type       string `yaml:"type"       mapstructure:"type"`
	Connection string `yaml:"connection" mapstructure:"connection"`
	Owner      string `yaml:"owner"      mapstructure:"owner"`
}

// OutputConfig defines where generated files are written.
type OutputConfig struct {
	Path string `yaml:"path" mapstructure:"path"`
}

// Load reads daxonne.yaml from the current working directory and returns the parsed Config.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("daxonne")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading daxonne.yaml: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing daxonne.yaml: %w", err)
	}

	return &cfg, nil
}

// Save writes the given Config to daxonne.yaml in the current working directory.
// The file is written in plain YAML; any existing file is overwritten.
func Save(cfg *Config) error {
	var sb strings.Builder

	sb.WriteString("database:\n")
	sb.WriteString(fmt.Sprintf("  type: %s\n", cfg.Database.Type))
	sb.WriteString(fmt.Sprintf("  connection: %q\n", cfg.Database.Connection))
	sb.WriteString(fmt.Sprintf("  owner: %q\n", cfg.Database.Owner))
	sb.WriteString("\n")
	sb.WriteString("output:\n")
	sb.WriteString(fmt.Sprintf("  path: %q\n", cfg.Output.Path))
	sb.WriteString("\n")
	sb.WriteString("templates:\n")

	if len(cfg.Templates) == 0 {
		sb.WriteString("  []\n")
	} else {
		for _, t := range cfg.Templates {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}

	if err := os.WriteFile("daxonne.yaml", []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("writing daxonne.yaml: %w", err)
	}

	return nil
}
