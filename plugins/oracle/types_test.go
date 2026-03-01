package oracle

import (
	"testing"

	"github.com/daxonne/core/internal/schema"
)

func intPtr(v int) *int { return &v }

func TestMapOracleType(t *testing.T) {
	tests := []struct {
		name       string
		oracleType string
		precision  *int
		scale      *int
		want       schema.InternalType
	}{
		// ── String types ────────────────────────────────────────────────────────
		{"varchar2", "VARCHAR2", nil, nil, schema.TypeString},
		{"varchar2_lower", "varchar2", nil, nil, schema.TypeString},
		{"varchar", "VARCHAR", nil, nil, schema.TypeString},
		{"char", "CHAR", nil, nil, schema.TypeString},
		{"nvarchar2", "NVARCHAR2", nil, nil, schema.TypeString},
		{"nchar", "NCHAR", nil, nil, schema.TypeString},
		{"clob", "CLOB", nil, nil, schema.TypeString},
		{"nclob", "NCLOB", nil, nil, schema.TypeString},
		{"long", "LONG", nil, nil, schema.TypeString},

		// ── NUMBER — no precision → long ────────────────────────────────────────
		{"number_no_prec", "NUMBER", nil, nil, schema.TypeLong},
		{"number_zero_prec", "NUMBER", intPtr(0), nil, schema.TypeLong},

		// ── NUMBER — precision ≤ 9, scale 0 → int ───────────────────────────────
		{"number_p9_s0", "NUMBER", intPtr(9), intPtr(0), schema.TypeInt},
		{"number_p5_snil", "NUMBER", intPtr(5), nil, schema.TypeInt},

		// ── NUMBER — precision > 9, scale 0 → long ──────────────────────────────
		{"number_p18_s0", "NUMBER", intPtr(18), intPtr(0), schema.TypeLong},
		{"number_p10_snil", "NUMBER", intPtr(10), nil, schema.TypeLong},

		// ── NUMBER — scale > 0 → decimal ────────────────────────────────────────
		{"number_p12_s2", "NUMBER", intPtr(12), intPtr(2), schema.TypeDecimal},
		{"number_p5_s3", "NUMBER", intPtr(5), intPtr(3), schema.TypeDecimal},

		// ── Integer types ────────────────────────────────────────────────────────
		{"integer", "INTEGER", nil, nil, schema.TypeInt},
		{"smallint", "SMALLINT", nil, nil, schema.TypeInt},
		{"int", "INT", nil, nil, schema.TypeInt},

		// ── Float types ──────────────────────────────────────────────────────────
		{"float", "FLOAT", nil, nil, schema.TypeDecimal},
		{"binary_float", "BINARY_FLOAT", nil, nil, schema.TypeDecimal},
		{"binary_double", "BINARY_DOUBLE", nil, nil, schema.TypeDecimal},

		// ── Date / time types ────────────────────────────────────────────────────
		{"date", "DATE", nil, nil, schema.TypeDate},
		{"timestamp", "TIMESTAMP", nil, nil, schema.TypeDateTime},
		{"timestamp_prec", "TIMESTAMP(6)", nil, nil, schema.TypeDateTime},
		{"timestamp_tz", "TIMESTAMP WITH TIME ZONE", nil, nil, schema.TypeDateTime},
		{"timestamp_ltz", "TIMESTAMP WITH LOCAL TIME ZONE", nil, nil, schema.TypeDateTime},

		// ── Binary types ─────────────────────────────────────────────────────────
		{"blob", "BLOB", nil, nil, schema.TypeBytes},
		{"raw", "RAW", nil, nil, schema.TypeBytes},
		{"long_raw", "LONG RAW", nil, nil, schema.TypeBytes},

		// ── Unknown fallback → string ─────────────────────────────────────────────
		{"xmltype", "XMLTYPE", nil, nil, schema.TypeString},
		{"unknown", "SOME_CUSTOM_TYPE", nil, nil, schema.TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapOracleType(tt.oracleType, tt.precision, tt.scale)
			if got != tt.want {
				t.Errorf("mapOracleType(%q, %v, %v) = %q, want %q",
					tt.oracleType, tt.precision, tt.scale, got, tt.want)
			}
		})
	}
}
