// Package schema defines the universal internal database schema model used
// across all Daxonne database plugins and generators.
package schema

// Schema represents the full schema of a database owner/namespace.
type Schema struct {
	Tables []Table `json:"tables"`
}

// Table represents a single database table.
type Table struct {
	Name        string       `json:"name"`
	Columns     []Column     `json:"columns"`
	PrimaryKeys []string     `json:"primaryKeys"`
	ForeignKeys []ForeignKey `json:"foreignKeys"`
}

// Column represents a single column within a table.
type Column struct {
	Name      string       `json:"name"`
	Type      InternalType `json:"type"`
	Nullable  bool         `json:"nullable"`
	IsPrimary bool         `json:"isPrimary"`
	Length    *int         `json:"length,omitempty"`
	Precision *int         `json:"precision,omitempty"`
	Scale     *int         `json:"scale,omitempty"`
}

// InternalType is the normalized column type used across all Daxonne plugins.
// Every database plugin maps its native types to one of these constants.
type InternalType string

const (
	TypeString   InternalType = "string"
	TypeInt      InternalType = "int"
	TypeLong     InternalType = "long"
	TypeDecimal  InternalType = "decimal"
	TypeBool     InternalType = "bool"
	TypeDate     InternalType = "date"
	TypeDateTime InternalType = "datetime"
	TypeBytes    InternalType = "bytes"
	TypeUUID     InternalType = "uuid"
)

// ForeignKey describes a foreign key constraint between two tables.
type ForeignKey struct {
	Column           string `json:"column"`
	ReferencedTable  string `json:"referencedTable"`
	ReferencedColumn string `json:"referencedColumn"`
}
