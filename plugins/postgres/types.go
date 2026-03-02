package postgres

import (
	"strings"

	"github.com/daxonne/core/internal/schema"
)

// mapPostgresType converts a PostgreSQL data type string to the Daxonne InternalType.
// udt is the udt_name from information_schema (e.g. "int4", "varchar").
// charLen, numPrecision, numScale are from information_schema.columns.
func mapPostgresType(dataType, udtName string, charLen, numPrecision, numScale *int) schema.InternalType {
	dt := strings.ToLower(dataType)
	udt := strings.ToLower(udtName)

	// Boolean
	if dt == "boolean" || udt == "bool" {
		return schema.TypeBool
	}

	// UUID
	if dt == "uuid" || udt == "uuid" {
		return schema.TypeUUID
	}

	// Integer types
	switch udt {
	case "int2", "smallint":
		return schema.TypeInt
	case "int4", "integer", "serial":
		return schema.TypeInt
	case "int8", "bigint", "bigserial":
		return schema.TypeLong
	}
	switch dt {
	case "smallint":
		return schema.TypeInt
	case "integer", "serial":
		return schema.TypeInt
	case "bigint", "bigserial":
		return schema.TypeLong
	}

	// Numeric / decimal
	if dt == "numeric" || dt == "decimal" || udt == "numeric" {
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
	}

	// Floating point → decimal (closest match)
	if dt == "real" || dt == "double precision" || udt == "float4" || udt == "float8" {
		return schema.TypeDecimal
	}

	// Date / time
	if dt == "date" || udt == "date" {
		return schema.TypeDate
	}
	if strings.HasPrefix(dt, "timestamp") || strings.HasPrefix(udt, "timestamp") {
		return schema.TypeDateTime
	}
	if dt == "time" || strings.HasPrefix(dt, "time ") {
		return schema.TypeString // no dedicated time type
	}

	// Binary
	if dt == "bytea" || udt == "bytea" {
		return schema.TypeBytes
	}

	// String / text (default)
	return schema.TypeString
}
