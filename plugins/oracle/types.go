// Package oracle provides the Oracle database plugin for Daxonne.
package oracle

import (
	"strings"

	"github.com/daxonne/core/internal/schema"
)

// mapOracleType translates an Oracle DATA_TYPE string into a Daxonne InternalType.
// The precision and scale pointers come from ALL_TAB_COLUMNS and are used to
// distinguish between integer, decimal, and long NUMBER variants.
func mapOracleType(oracleType string, precision, scale *int) schema.InternalType {
	normalized := strings.ToUpper(strings.TrimSpace(oracleType))

	// TIMESTAMP and its variants (e.g. TIMESTAMP WITH TIME ZONE) are handled first
	// because they share a common prefix.
	if strings.HasPrefix(normalized, "TIMESTAMP") {
		return schema.TypeDateTime
	}

	switch normalized {
	case "VARCHAR2", "VARCHAR", "CHAR", "NVARCHAR2", "NCHAR", "CLOB", "NCLOB", "LONG":
		return schema.TypeString

	case "NUMBER":
		if precision != nil && *precision > 0 {
			if scale != nil && *scale > 0 {
				return schema.TypeDecimal
			}
			if *precision <= 9 {
				return schema.TypeInt
			}
			return schema.TypeLong
		}
		return schema.TypeLong

	case "INTEGER", "SMALLINT", "INT":
		return schema.TypeInt

	case "FLOAT", "BINARY_FLOAT", "BINARY_DOUBLE":
		return schema.TypeDecimal

	case "DATE":
		return schema.TypeDate

	case "BLOB", "RAW", "LONG RAW":
		return schema.TypeBytes
	}

	// Default: treat unknown types as strings to avoid generation failures.
	return schema.TypeString
}
