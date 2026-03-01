package oracle

import (
	"database/sql"
	"fmt"

	// Register the "oracle" driver with database/sql.
	_ "github.com/sijms/go-ora/v2"

	"github.com/daxonne/core/internal/schema"
)

// Reader implements schema.ISchemaReader for Oracle databases.
// It uses the pure-Go go-ora driver, so no Oracle Client installation is required.
type Reader struct {
	db *sql.DB
}

// Connect opens a connection to the Oracle database identified by connString and
// verifies reachability with a Ping. The connection string format is:
//
//	oracle://user:password@host:port/service_name
func (r *Reader) Connect(connString string) error {
	db, err := sql.Open("oracle", connString)
	if err != nil {
		return fmt.Errorf("opening oracle connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("pinging oracle database: %w", err)
	}

	r.db = db
	return nil
}

// ReadSchema queries ALL_TABLES, ALL_TAB_COLUMNS, ALL_CONSTRAINTS, and
// ALL_CONS_COLUMNS to build a complete Schema for the given owner.
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

	// Enrich tables with primary key and foreign key metadata.
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

// readTablesAndColumns fetches all tables and their columns for the given owner.
func (r *Reader) readTablesAndColumns(owner string) ([]schema.Table, error) {
	const query = `
SELECT
    t.TABLE_NAME,
    c.COLUMN_NAME,
    c.DATA_TYPE,
    c.NULLABLE,
    c.DATA_LENGTH,
    c.DATA_PRECISION,
    c.DATA_SCALE,
    c.COLUMN_ID
FROM ALL_TABLES t
JOIN ALL_TAB_COLUMNS c
    ON t.TABLE_NAME = c.TABLE_NAME
    AND t.OWNER = c.OWNER
WHERE t.OWNER = :owner
ORDER BY t.TABLE_NAME, c.COLUMN_ID`

	rows, err := r.db.Query(query, sql.Named("owner", owner))
	if err != nil {
		return nil, fmt.Errorf("querying ALL_TAB_COLUMNS: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*schema.Table)
	var order []string // preserve table discovery order

	for rows.Next() {
		var (
			tableName  string
			colName    string
			dataType   string
			nullable   string
			dataLength sql.NullInt64
			precision  sql.NullInt64
			scale      sql.NullInt64
			colID      int
		)

		if err := rows.Scan(&tableName, &colName, &dataType, &nullable,
			&dataLength, &precision, &scale, &colID); err != nil {
			return nil, fmt.Errorf("scanning column row: %w", err)
		}

		if _, exists := tableMap[tableName]; !exists {
			tableMap[tableName] = &schema.Table{Name: tableName}
			order = append(order, tableName)
		}

		var precPtr, scalePtr, lenPtr *int
		if precision.Valid {
			v := int(precision.Int64)
			precPtr = &v
		}
		if scale.Valid {
			v := int(scale.Int64)
			scalePtr = &v
		}
		if dataLength.Valid {
			v := int(dataLength.Int64)
			lenPtr = &v
		}

		col := schema.Column{
			Name:      colName,
			Type:      mapOracleType(dataType, precPtr, scalePtr),
			Nullable:  nullable == "Y",
			Length:    lenPtr,
			Precision: precPtr,
			Scale:     scalePtr,
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
SELECT cols.TABLE_NAME, cols.COLUMN_NAME
FROM ALL_CONSTRAINTS cons
JOIN ALL_CONS_COLUMNS cols
    ON cons.CONSTRAINT_NAME = cols.CONSTRAINT_NAME
    AND cons.OWNER = cols.OWNER
WHERE cons.CONSTRAINT_TYPE = 'P'
AND cons.OWNER = :owner`

	rows, err := r.db.Query(query, sql.Named("owner", owner))
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
    a.TABLE_NAME,
    a.COLUMN_NAME,
    c_pk.TABLE_NAME  AS REFERENCED_TABLE,
    b.COLUMN_NAME    AS REFERENCED_COLUMN
FROM ALL_CONS_COLUMNS a
JOIN ALL_CONSTRAINTS c
    ON a.OWNER = c.OWNER AND a.CONSTRAINT_NAME = c.CONSTRAINT_NAME
JOIN ALL_CONSTRAINTS c_pk
    ON c.R_OWNER = c_pk.OWNER AND c.R_CONSTRAINT_NAME = c_pk.CONSTRAINT_NAME
JOIN ALL_CONS_COLUMNS b
    ON c_pk.OWNER = b.OWNER AND c_pk.CONSTRAINT_NAME = b.CONSTRAINT_NAME
WHERE c.CONSTRAINT_TYPE = 'R'
AND a.OWNER = :owner`

	rows, err := r.db.Query(query, sql.Named("owner", owner))
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
