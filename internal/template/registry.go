// Package template manages the Daxonne template registry and local installation.
package template

import (
	"encoding/json"
	"net/http"
	"time"
)

const (
	// remoteRegistryURL is the raw URL of the registry.json file in the templates repo.
	remoteRegistryURL = "https://raw.githubusercontent.com/Daxonne/templates/main/registry.json"
)

// RegistryEntry describes a template available in the Daxonne registry.
type RegistryEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Author      string `json:"author"`
	Version     string `json:"version"`
}

// fallbackRegistry is the built-in list used when the remote registry is unavailable.
var fallbackRegistry = []RegistryEntry{
	{
		Name:        "csharp-dapper",
		Description: "C# records + Dapper CRUD repositories",
		Language:    "csharp",
		Author:      "daxonne",
		Version:     "1.0.0",
	},
	{
		Name:        "typescript-prisma",
		Description: "TypeScript types + Prisma client queries",
		Language:    "typescript",
		Author:      "daxonne",
		Version:     "1.0.0",
	},
	{
		Name:        "java-jpa",
		Description: "Java entities + Spring Data JPA repositories",
		Language:    "java",
		Author:      "daxonne",
		Version:     "1.0.0",
	},
	{
		Name:        "python-sqlalchemy",
		Description: "Python SQLAlchemy models + async repositories",
		Language:    "python",
		Author:      "daxonne",
		Version:     "1.0.0",
	},
}

// LoadRegistry fetches the template registry from the Daxonne templates repository.
// If the remote fetch fails (e.g. no network), it returns the built-in fallback list.
func LoadRegistry() []RegistryEntry {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(remoteRegistryURL)
	if err != nil {
		return fallbackRegistry
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackRegistry
	}

	var entries []RegistryEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return fallbackRegistry
	}

	if len(entries) == 0 {
		return fallbackRegistry
	}

	return entries
}

// FindByName looks up a template by name in the registry.
// It fetches the remote registry first, falling back to the built-in list.
func FindByName(name string) (*RegistryEntry, bool) {
	for _, e := range LoadRegistry() {
		if e.Name == name {
			ec := e
			return &ec, true
		}
	}
	return nil, false
}
