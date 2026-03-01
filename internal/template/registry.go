// Package template manages the Daxonne template registry and local installation.
package template

// RegistryEntry describes a template available in the Daxonne registry.
type RegistryEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Author      string `json:"author"`
	Version     string `json:"version"`
}

// DefaultRegistry is the built-in list of known templates.
// In a future version this will be fetched from a remote registry.
var DefaultRegistry = []RegistryEntry{
	{
		Name:        "csharp-dapper",
		Description: "Generates C# records + Dapper queries",
		Language:    "csharp",
		Author:      "daxonne",
		Version:     "1.0.0",
	},
	{
		Name:        "typescript-prisma",
		Description: "Generates TypeScript types + Prisma client queries",
		Language:    "typescript",
		Author:      "daxonne",
		Version:     "1.0.0",
	},
	{
		Name:        "java-jpa",
		Description: "Generates Java entities + Spring Data JPA repositories",
		Language:    "java",
		Author:      "daxonne",
		Version:     "1.0.0",
	},
	{
		Name:        "python-sqlalchemy",
		Description: "Generates Python models + SQLAlchemy queries",
		Language:    "python",
		Author:      "daxonne",
		Version:     "1.0.0",
	},
}

// FindByName returns the registry entry for the given template name, if it exists.
func FindByName(name string) (*RegistryEntry, bool) {
	for i := range DefaultRegistry {
		if DefaultRegistry[i].Name == name {
			return &DefaultRegistry[i], true
		}
	}
	return nil, false
}
