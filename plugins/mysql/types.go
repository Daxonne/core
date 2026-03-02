package mysql

import (
	"strings"

	"github.com/daxonne/core/internal/schema"
)

// mapMySQLType converts a MySQL column type to the Daxonne InternalType.
// columnType is the full type string from information_schema (e.g. "int(11)", "varchar(255)").
// numPrecision and numScale are from information_schema.columns NUMERIC_PRECISION / NUMERIC_SCALE.
func mapMySQLType(columnType string, numPrecision, numScale *int) schema.InternalType {
	ct := strings.ToLower(columnType)

	// TINYINT(1) is the MySQL canonical boolean representation
	if ct == "tinyint(1)" {
		return schema.TypeBool
	}

	// Strip display width / length suffix for matching (e.g. "int(11)" → "int")
	base := ct
	if idx := strings.Index(ct, "("); idx != -1 {
		base = ct[:idx]
	}
	base = strings.TrimSpace(base)
	// Remove "unsigned" / "zerofill" suffixes
	base = strings.Fields(base)[0]

	switch base {
	case "bool", "boolean":
		return schema.TypeBool

	case "tinyint", "smallint", "mediumint", "int", "integer", "year":
		return schema.TypeInt

	case "bigint":
		return schema.TypeLong

	case "decimal", "numeric":
		if numScale != nil && *numScale > 0 {
			return schema.TypeDecimal
		}
		if numPrecision != nil {
			if *numPrecision <= 9 {
				return schema.TypeInt
			}
			if *numPrecision <= 18 {
				return schema.TypeLong
			}
		}
		return schema.TypeDecimal

	case "float", "double", "real":
		return schema.TypeDecimal

	case "date":
		return schema.TypeDate

	case "datetime", "timestamp":
		return schema.TypeDateTime

	case "time":
		return schema.TypeString // no dedicated time type

	case "binary", "varbinary", "tinyblob", "blob", "mediumblob", "longblob":
		return schema.TypeBytes

	case "char", "varchar", "tinytext", "text", "mediumtext", "longtext",
		"enum", "set", "json":
		return schema.TypeString
	}

	return schema.TypeString
}
