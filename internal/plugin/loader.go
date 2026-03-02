// Package plugin provides the factory that maps a database type string to its
// ISchemaReader implementation. Adding support for a new database requires
// registering it here.
package plugin

import (
	"fmt"

	"github.com/daxonne/core/internal/schema"
	"github.com/daxonne/core/plugins/mysql"
	"github.com/daxonne/core/plugins/oracle"
	"github.com/daxonne/core/plugins/postgres"
)

// GetSchemaReader returns the ISchemaReader implementation for the given database type.
// Returns an error if the type is not recognised.
func GetSchemaReader(dbType string) (schema.ISchemaReader, error) {
	switch dbType {
	case "oracle":
		return &oracle.Reader{}, nil
	case "postgres", "postgresql":
		return &postgres.Reader{}, nil
	case "mysql", "mariadb":
		return &mysql.Reader{}, nil
	default:
		return nil, fmt.Errorf("unsupported database type %q (supported: oracle, postgres, mysql)", dbType)
	}
}
