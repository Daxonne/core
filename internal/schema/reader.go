package schema

// ISchemaReader is the interface that every Daxonne database plugin must implement.
// It decouples the schema-reading logic from any specific DBMS, allowing Daxonne
// to support multiple databases through a uniform contract.
type ISchemaReader interface {
	// Connect establishes a connection using the provided connection string.
	// The format of connString is driver-specific.
	Connect(connString string) error

	// ReadSchema reads the full table schema for the given database owner or namespace.
	ReadSchema(owner string) (*Schema, error)

	// Close releases the underlying database connection and any held resources.
	Close() error
}
