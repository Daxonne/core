// Package postgres implements schema.ISchemaReader for PostgreSQL databases.
// It uses the pgx driver (via database/sql compatibility layer) — no external
// client installation is required.
package postgres

import (
	"database/sql"
	"fmt"

	// Register the "pgx" driver with database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/daxonne/core/internal/schema"
)

// Reader implements schema.ISchemaReader for PostgreSQL databases.
type Reader struct {
	db *sql.DB
}

// Connect opens a connection to the PostgreSQL database identified by connString
// and verifies reachability with a Ping. The connection string format is:
//
//	postgres://user:password@host:5432/dbname
func (r *Reader) Connect(connString string) error {
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return fmt.Errorf("opening postgres connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("pinging postgres database: %w", err)
	}

	r.db = db
	return nil
}

// ReadSchema queries information_schema to build a complete Schema for the given
// schema name (owner). Pass "public" for the default PostgreSQL schema.
func (r *Reader) ReadSchema(owner string) (*schema.Schema, error) {
	tables, err := r.readTablesAndColumns(owner)
	if err != nil {
		return nil, fmt.Errorf("reading tables and columns: %w", err)
	}

	primaryKeys, err := r.readPrimaryKeys(owner)
	if err != nil {
		return nil, fmt.Errorf("reading primary keys: %w", err)
	}

	foreignKeys, err := r.readForeignKeys(owner)
	if err != nil {
		return nil, fmt.Errorf("reading foreign keys: %w", err)
	}

	for i := range tables {
		pks := primaryKeys[tables[i].Name]
		tables[i].PrimaryKeys = pks

		pkSet := make(map[string]bool, len(pks))
		for _, pk := range pks {
			pkSet[pk] = true
		}
		for j := range tables[i].Columns {
			if pkSet[tables[i].Columns[j].Name] {
				tables[i].Columns[j].IsPrimary = true
			}
		}

		tables[i].ForeignKeys = foreignKeys[tables[i].Name]
	}

	return &schema.Schema{Tables: tables}, nil
}

// Close releases the underlying database connection.
func (r *Reader) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// readTablesAndColumns queries information_schema.columns for all user tables.
func (r *Reader) readTablesAndColumns(owner string) ([]schema.Table, error) {
	const query = `
SELECT
    c.table_name,
    c.column_name,
    c.data_type,
    c.udt_name,
    c.is_nullable,
    c.character_maximum_length,
    c.numeric_precision,
    c.numeric_scale,
    c.ordinal_position
FROM information_schema.columns c
JOIN information_schema.tables t
    ON t.table_schema = c.table_schema
    AND t.table_name  = c.table_name
WHERE c.table_schema = $1
  AND t.table_type   = 'BASE TABLE'
ORDER BY c.table_name, c.ordinal_position`

	rows, err := r.db.Query(query, owner)
	if err != nil {
		return nil, fmt.Errorf("querying information_schema.columns: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*schema.Table)
	var order []string

	for rows.Next() {
		var (
			tableName string
			colName   string
			dataType  string
			udtName   string
			isNull    string
			charLen   sql.NullInt64
			numPrec   sql.NullInt64
			numScale  sql.NullInt64
			ordinal   int
		)
		if err := rows.Scan(&tableName, &colName, &dataType, &udtName, &isNull,
			&charLen, &numPrec, &numScale, &ordinal); err != nil {
			return nil, fmt.Errorf("scanning column row: %w", err)
		}

		if _, exists := tableMap[tableName]; !exists {
			tableMap[tableName] = &schema.Table{Name: tableName}
			order = append(order, tableName)
		}

		var charLenPtr, numPrecPtr, numScalePtr *int
		if charLen.Valid {
			v := int(charLen.Int64)
			charLenPtr = &v
		}
		if numPrec.Valid {
			v := int(numPrec.Int64)
			numPrecPtr = &v
		}
		if numScale.Valid {
			v := int(numScale.Int64)
			numScalePtr = &v
		}

		col := schema.Column{
			Name:      colName,
			Type:      mapPostgresType(dataType, udtName, charLenPtr, numPrecPtr, numScalePtr),
			Nullable:  isNull == "YES",
			Length:    charLenPtr,
			Precision: numPrecPtr,
			Scale:     numScalePtr,
		}
		tableMap[tableName].Columns = append(tableMap[tableName].Columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating column rows: %w", err)
	}

	tables := make([]schema.Table, 0, len(order))
	for _, name := range order {
		tables = append(tables, *tableMap[name])
	}
	return tables, nil
}

// readPrimaryKeys returns a map of table name → primary key column names.
func (r *Reader) readPrimaryKeys(owner string) (map[string][]string, error) {
	const query = `
SELECT kcu.table_name, kcu.column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema    = kcu.table_schema
WHERE tc.constraint_type = 'PRIMARY KEY'
  AND tc.table_schema    = $1
ORDER BY kcu.table_name, kcu.ordinal_position`

	rows, err := r.db.Query(query, owner)
	if err != nil {
		return nil, fmt.Errorf("querying primary keys: %w", err)
	}
	defer rows.Close()

	pks := make(map[string][]string)
	for rows.Next() {
		var tableName, colName string
		if err := rows.Scan(&tableName, &colName); err != nil {
			return nil, fmt.Errorf("scanning primary key row: %w", err)
		}
		pks[tableName] = append(pks[tableName], colName)
	}

	return pks, rows.Err()
}

// readForeignKeys returns a map of table name → foreign key constraints.
func (r *Reader) readForeignKeys(owner string) (map[string][]schema.ForeignKey, error) {
	const query = `
SELECT
    kcu.table_name,
    kcu.column_name,
    ccu.table_name  AS referenced_table,
    ccu.column_name AS referenced_column
FROM information_schema.referential_constraints rc
JOIN information_schema.key_column_usage kcu
    ON rc.constraint_name = kcu.constraint_name
    AND kcu.table_schema  = $1
JOIN information_schema.constraint_column_usage ccu
    ON rc.unique_constraint_name = ccu.constraint_name`

	rows, err := r.db.Query(query, owner)
	if err != nil {
		return nil, fmt.Errorf("querying foreign keys: %w", err)
	}
	defer rows.Close()

	fks := make(map[string][]schema.ForeignKey)
	for rows.Next() {
		var tableName, colName, refTable, refCol string
		if err := rows.Scan(&tableName, &colName, &refTable, &refCol); err != nil {
			return nil, fmt.Errorf("scanning foreign key row: %w", err)
		}
		fks[tableName] = append(fks[tableName], schema.ForeignKey{
			Column:           colName,
			ReferencedTable:  refTable,
			ReferencedColumn: refCol,
		})
	}

	return fks, rows.Err()
}
