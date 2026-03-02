// Package mysql implements schema.ISchemaReader for MySQL / MariaDB databases.
// It uses the go-sql-driver/mysql driver registered with database/sql.
package mysql

import (
	"database/sql"
	"fmt"

	// Register the "mysql" driver with database/sql.
	_ "github.com/go-sql-driver/mysql"

	"github.com/daxonne/core/internal/schema"
)

// Reader implements schema.ISchemaReader for MySQL databases.
type Reader struct {
	db *sql.DB
}

// Connect opens a connection to the MySQL database identified by connString and
// verifies reachability with a Ping. The connection string format is:
//
//	mysql://user:password@host:3306/dbname
//	user:password@tcp(host:3306)/dbname
func (r *Reader) Connect(connString string) error {
	// go-sql-driver/mysql does not understand the "mysql://" URL scheme;
	// convert it to the DSN format if needed.
	dsn := toMySQLDSN(connString)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("opening mysql connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("pinging mysql database: %w", err)
	}

	r.db = db
	return nil
}

// ReadSchema queries information_schema to build a complete Schema for the given
// database name (owner).
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

// readTablesAndColumns queries information_schema.columns for all base tables
// in the given database (owner).
func (r *Reader) readTablesAndColumns(owner string) ([]schema.Table, error) {
	const query = `
SELECT
    c.TABLE_NAME,
    c.COLUMN_NAME,
    c.COLUMN_TYPE,
    c.IS_NULLABLE,
    c.CHARACTER_MAXIMUM_LENGTH,
    c.NUMERIC_PRECISION,
    c.NUMERIC_SCALE,
    c.ORDINAL_POSITION
FROM information_schema.COLUMNS c
JOIN information_schema.TABLES t
    ON t.TABLE_SCHEMA = c.TABLE_SCHEMA
    AND t.TABLE_NAME  = c.TABLE_NAME
WHERE c.TABLE_SCHEMA = ?
  AND t.TABLE_TYPE   = 'BASE TABLE'
ORDER BY c.TABLE_NAME, c.ORDINAL_POSITION`

	rows, err := r.db.Query(query, owner)
	if err != nil {
		return nil, fmt.Errorf("querying information_schema.COLUMNS: %w", err)
	}
	defer rows.Close()

	tableMap := make(map[string]*schema.Table)
	var order []string

	for rows.Next() {
		var (
			tableName  string
			colName    string
			colType    string
			isNull     string
			charLen    sql.NullInt64
			numPrec    sql.NullInt64
			numScale   sql.NullInt64
			ordinal    int
		)
		if err := rows.Scan(&tableName, &colName, &colType, &isNull,
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
			Type:      mapMySQLType(colType, numPrecPtr, numScalePtr),
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
SELECT TABLE_NAME, COLUMN_NAME
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = ?
  AND INDEX_NAME   = 'PRIMARY'
ORDER BY TABLE_NAME, SEQ_IN_INDEX`

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
    kcu.TABLE_NAME,
    kcu.COLUMN_NAME,
    kcu.REFERENCED_TABLE_NAME,
    kcu.REFERENCED_COLUMN_NAME
FROM information_schema.KEY_COLUMN_USAGE kcu
JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
    ON rc.CONSTRAINT_SCHEMA = kcu.TABLE_SCHEMA
    AND rc.CONSTRAINT_NAME  = kcu.CONSTRAINT_NAME
WHERE kcu.TABLE_SCHEMA           = ?
  AND kcu.REFERENCED_TABLE_NAME IS NOT NULL`

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

// toMySQLDSN converts a mysql:// URL to the go-sql-driver DSN format when needed.
// If the string does not start with "mysql://", it is returned as-is.
func toMySQLDSN(connString string) string {
	const prefix = "mysql://"
	if len(connString) <= len(prefix) || connString[:len(prefix)] != prefix {
		return connString
	}
	// mysql://user:pass@host:port/dbname → user:pass@tcp(host:port)/dbname
	rest := connString[len(prefix):]
	// Find the @ to split user:pass from host:port/db
	atIdx := -1
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i] == '@' {
			atIdx = i
			break
		}
	}
	if atIdx < 0 {
		return rest // malformed, return as-is and let the driver report the error
	}
	userPass := rest[:atIdx]
	hostDB := rest[atIdx+1:]

	// Split host:port from /dbname
	slashIdx := -1
	for i, c := range hostDB {
		if c == '/' {
			slashIdx = i
			break
		}
	}
	if slashIdx < 0 {
		return userPass + "@tcp(" + hostDB + ")/"
	}
	host := hostDB[:slashIdx]
	db := hostDB[slashIdx:] // includes leading /
	return userPass + "@tcp(" + host + ")" + db
}
