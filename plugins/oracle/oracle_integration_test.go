//go:build integration

package oracle

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daxonne/core/internal/config"
	"github.com/daxonne/core/internal/generator"
	"github.com/daxonne/core/internal/schema"
)

// ─── Test configuration ───────────────────────────────────────────────────────

func testDSN() string {
	if v := os.Getenv("DAXONNE_TEST_DSN"); v != "" {
		return v
	}
	return "oracle://autosendmail:autosendmail@localhost:1521/FREEPDB1"
}

func testOwner() string {
	if v := os.Getenv("DAXONNE_TEST_OWNER"); v != "" {
		return v
	}
	return "AUTOSENDMAIL"
}

// ─── Test table names ─────────────────────────────────────────────────────────

const (
	tblCustomers = "DAX_TEST_CUSTOMERS"
	tblOrders    = "DAX_TEST_ORDERS"
	tblItems     = "DAX_TEST_ITEMS"
)

// ─── TestMain — global setup & teardown ───────────────────────────────────────

func TestMain(m *testing.M) {
	db, err := sql.Open("oracle", testDSN())
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: open: %v\n", err)
		os.Exit(1)
	}
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "setup: ping: %v\n", err)
		os.Exit(1)
	}

	if err := setupSchema(db); err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", err)
		db.Close()
		os.Exit(1)
	}

	code := m.Run()

	teardownSchema(db)
	db.Close()
	os.Exit(code)
}

func setupSchema(db *sql.DB) error {
	// Drop in reverse FK order (ignore errors — tables may not exist yet)
	for _, t := range []string{tblItems, tblOrders, tblCustomers} {
		db.Exec("DROP TABLE " + t + " CASCADE CONSTRAINTS PURGE")
	}

	stmts := []string{
		// CUSTOMERS
		`CREATE TABLE DAX_TEST_CUSTOMERS (
			ID         NUMBER(10)    NOT NULL,
			FIRST_NAME VARCHAR2(100) NOT NULL,
			LAST_NAME  VARCHAR2(100) NOT NULL,
			EMAIL      VARCHAR2(255),
			BALANCE    NUMBER(12,2)  DEFAULT 0 NOT NULL,
			JOINED_AT  DATE          DEFAULT SYSDATE NOT NULL,
			CONSTRAINT PK_DAXCUST PRIMARY KEY (ID)
		)`,
		// ORDERS
		`CREATE TABLE DAX_TEST_ORDERS (
			ID          NUMBER(10)   NOT NULL,
			CUSTOMER_ID NUMBER(10)   NOT NULL,
			STATUS      VARCHAR2(20) NOT NULL,
			TOTAL_AMT   NUMBER(12,2) NOT NULL,
			CREATED_AT  TIMESTAMP    DEFAULT SYSTIMESTAMP NOT NULL,
			CONSTRAINT PK_DAXORD     PRIMARY KEY (ID),
			CONSTRAINT FK_DAXORD_CUST FOREIGN KEY (CUSTOMER_ID) REFERENCES DAX_TEST_CUSTOMERS(ID)
		)`,
		// ITEMS
		`CREATE TABLE DAX_TEST_ITEMS (
			ID         NUMBER(10)   NOT NULL,
			ORDER_ID   NUMBER(10)   NOT NULL,
			SKU        VARCHAR2(50) NOT NULL,
			QUANTITY   NUMBER(9,0)  NOT NULL,
			UNIT_PRICE NUMBER(12,2) NOT NULL,
			ATTACHMENT BLOB,
			CONSTRAINT PK_DAXITEM     PRIMARY KEY (ID),
			CONSTRAINT FK_DAXITEM_ORD FOREIGN KEY (ORDER_ID) REFERENCES DAX_TEST_ORDERS(ID)
		)`,
		// Seed — customers
		`INSERT INTO DAX_TEST_CUSTOMERS (ID,FIRST_NAME,LAST_NAME,EMAIL,BALANCE,JOINED_AT) VALUES (1,'Alice','Martin','alice@example.com',1500.00,SYSDATE)`,
		`INSERT INTO DAX_TEST_CUSTOMERS (ID,FIRST_NAME,LAST_NAME,EMAIL,BALANCE,JOINED_AT) VALUES (2,'Bob','Dupont',NULL,0.00,SYSDATE)`,
		// Seed — orders
		`INSERT INTO DAX_TEST_ORDERS (ID,CUSTOMER_ID,STATUS,TOTAL_AMT,CREATED_AT) VALUES (1,1,'SHIPPED',299.99,SYSTIMESTAMP)`,
		`INSERT INTO DAX_TEST_ORDERS (ID,CUSTOMER_ID,STATUS,TOTAL_AMT,CREATED_AT) VALUES (2,2,'PENDING',150.00,SYSTIMESTAMP)`,
		// Seed — items
		`INSERT INTO DAX_TEST_ITEMS (ID,ORDER_ID,SKU,QUANTITY,UNIT_PRICE) VALUES (1,1,'PROD-001',3,99.99)`,
		`INSERT INTO DAX_TEST_ITEMS (ID,ORDER_ID,SKU,QUANTITY,UNIT_PRICE) VALUES (2,1,'PROD-002',1,0.02)`,
		`INSERT INTO DAX_TEST_ITEMS (ID,ORDER_ID,SKU,QUANTITY,UNIT_PRICE) VALUES (3,2,'PROD-003',2,75.00)`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("executing %q: %w", s[:min(60, len(s))], err)
		}
	}
	return nil
}

func teardownSchema(db *sql.DB) {
	for _, t := range []string{tblItems, tblOrders, tblCustomers} {
		db.Exec("DROP TABLE " + t + " CASCADE CONSTRAINTS PURGE")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func mustReadSchema(t *testing.T) *schema.Schema {
	t.Helper()
	r := &Reader{}
	if err := r.Connect(testDSN()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { r.Close() })

	s, err := r.ReadSchema(testOwner())
	if err != nil {
		t.Fatalf("ReadSchema: %v", err)
	}
	return s
}

func findTable(tables []schema.Table, name string) *schema.Table {
	for i := range tables {
		if tables[i].Name == name {
			return &tables[i]
		}
	}
	return nil
}

func findColumn(cols []schema.Column, name string) *schema.Column {
	for i := range cols {
		if cols[i].Name == name {
			return &cols[i]
		}
	}
	return nil
}

func findFK(fks []schema.ForeignKey, col string) *schema.ForeignKey {
	for i := range fks {
		if fks[i].Column == col {
			return &fks[i]
		}
	}
	return nil
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestReader_Connect(t *testing.T) {
	t.Run("valid_dsn", func(t *testing.T) {
		r := &Reader{}
		if err := r.Connect(testDSN()); err != nil {
			t.Fatalf("expected successful connection, got: %v", err)
		}
		r.Close()
	})

	t.Run("invalid_dsn", func(t *testing.T) {
		r := &Reader{}
		err := r.Connect("oracle://bad:creds@localhost:9999/NOPE")
		if err == nil {
			r.Close()
			t.Fatal("expected error for invalid DSN, got nil")
		}
	})
}

func TestReader_Close(t *testing.T) {
	r := &Reader{}
	// Close without Connect should not panic or error
	if err := r.Close(); err != nil {
		t.Errorf("Close on unconnected reader: %v", err)
	}
}

func TestReader_ReadSchema_Tables(t *testing.T) {
	s := mustReadSchema(t)

	if len(s.Tables) == 0 {
		t.Fatal("expected at least one table, got 0")
	}

	// All three test tables must be present
	for _, name := range []string{tblCustomers, tblOrders, tblItems} {
		if tbl := findTable(s.Tables, name); tbl == nil {
			t.Errorf("table %q not found in schema", name)
		}
	}
}

func TestReader_ReadSchema_Customers_Columns(t *testing.T) {
	s := mustReadSchema(t)

	tbl := findTable(s.Tables, tblCustomers)
	if tbl == nil {
		t.Fatalf("table %q not found", tblCustomers)
	}

	tests := []struct {
		col      string
		wantType schema.InternalType
		nullable bool
	}{
		{"ID", schema.TypeLong, false},       // NUMBER(10): precision 10 > 9 → long
		{"FIRST_NAME", schema.TypeString, false},
		{"LAST_NAME", schema.TypeString, false},
		{"EMAIL", schema.TypeString, true},   // nullable
		{"BALANCE", schema.TypeDecimal, false},
		{"JOINED_AT", schema.TypeDate, false},
	}

	for _, tt := range tests {
		t.Run(tt.col, func(t *testing.T) {
			col := findColumn(tbl.Columns, tt.col)
			if col == nil {
				t.Fatalf("column %q not found", tt.col)
			}
			if col.Type != tt.wantType {
				t.Errorf("column %q: type = %q, want %q", tt.col, col.Type, tt.wantType)
			}
			if col.Nullable != tt.nullable {
				t.Errorf("column %q: nullable = %v, want %v", tt.col, col.Nullable, tt.nullable)
			}
		})
	}
}

func TestReader_ReadSchema_Orders_Columns(t *testing.T) {
	s := mustReadSchema(t)

	tbl := findTable(s.Tables, tblOrders)
	if tbl == nil {
		t.Fatalf("table %q not found", tblOrders)
	}

	tests := []struct {
		col      string
		wantType schema.InternalType
	}{
		{"ID", schema.TypeLong},          // NUMBER(10): precision 10 > 9 → long
		{"CUSTOMER_ID", schema.TypeLong}, // same
		{"STATUS", schema.TypeString},
		{"TOTAL_AMT", schema.TypeDecimal},
		{"CREATED_AT", schema.TypeDateTime},
	}

	for _, tt := range tests {
		t.Run(tt.col, func(t *testing.T) {
			col := findColumn(tbl.Columns, tt.col)
			if col == nil {
				t.Fatalf("column %q not found", tt.col)
			}
			if col.Type != tt.wantType {
				t.Errorf("column %q: type = %q, want %q", tt.col, col.Type, tt.wantType)
			}
		})
	}
}

func TestReader_ReadSchema_Items_Columns(t *testing.T) {
	s := mustReadSchema(t)

	tbl := findTable(s.Tables, tblItems)
	if tbl == nil {
		t.Fatalf("table %q not found", tblItems)
	}

	col := findColumn(tbl.Columns, "ATTACHMENT")
	if col == nil {
		t.Fatal("column ATTACHMENT not found")
	}
	if col.Type != schema.TypeBytes {
		t.Errorf("ATTACHMENT type = %q, want %q", col.Type, schema.TypeBytes)
	}
	if !col.Nullable {
		t.Error("ATTACHMENT should be nullable")
	}

	qty := findColumn(tbl.Columns, "QUANTITY")
	if qty == nil {
		t.Fatal("column QUANTITY not found")
	}
	if qty.Type != schema.TypeInt {
		t.Errorf("QUANTITY type = %q, want %q", qty.Type, schema.TypeInt)
	}
}

func TestReader_ReadSchema_PrimaryKeys(t *testing.T) {
	s := mustReadSchema(t)

	tests := []struct {
		table  string
		pkCols []string
	}{
		{tblCustomers, []string{"ID"}},
		{tblOrders, []string{"ID"}},
		{tblItems, []string{"ID"}},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			tbl := findTable(s.Tables, tt.table)
			if tbl == nil {
				t.Fatalf("table %q not found", tt.table)
			}

			if len(tbl.PrimaryKeys) != len(tt.pkCols) {
				t.Fatalf("PrimaryKeys = %v, want %v", tbl.PrimaryKeys, tt.pkCols)
			}
			for _, pk := range tt.pkCols {
				found := false
				for _, tpk := range tbl.PrimaryKeys {
					if tpk == pk {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("PK %q not found in %v", pk, tbl.PrimaryKeys)
				}
			}

			// Verify IsPrimary flag on the column
			col := findColumn(tbl.Columns, "ID")
			if col == nil {
				t.Fatal("column ID not found")
			}
			if !col.IsPrimary {
				t.Error("ID.IsPrimary should be true")
			}
		})
	}
}

func TestReader_ReadSchema_ForeignKeys(t *testing.T) {
	s := mustReadSchema(t)

	t.Run("orders_customer_fk", func(t *testing.T) {
		tbl := findTable(s.Tables, tblOrders)
		if tbl == nil {
			t.Fatalf("table %q not found", tblOrders)
		}
		fk := findFK(tbl.ForeignKeys, "CUSTOMER_ID")
		if fk == nil {
			t.Fatalf("FK on CUSTOMER_ID not found; got %+v", tbl.ForeignKeys)
		}
		if fk.ReferencedTable != tblCustomers {
			t.Errorf("FK references %q, want %q", fk.ReferencedTable, tblCustomers)
		}
		if fk.ReferencedColumn != "ID" {
			t.Errorf("FK references column %q, want ID", fk.ReferencedColumn)
		}
	})

	t.Run("items_order_fk", func(t *testing.T) {
		tbl := findTable(s.Tables, tblItems)
		if tbl == nil {
			t.Fatalf("table %q not found", tblItems)
		}
		fk := findFK(tbl.ForeignKeys, "ORDER_ID")
		if fk == nil {
			t.Fatalf("FK on ORDER_ID not found; got %+v", tbl.ForeignKeys)
		}
		if fk.ReferencedTable != tblOrders {
			t.Errorf("FK references %q, want %q", fk.ReferencedTable, tblOrders)
		}
	})

	t.Run("customers_no_fk", func(t *testing.T) {
		tbl := findTable(s.Tables, tblCustomers)
		if tbl == nil {
			t.Fatalf("table %q not found", tblCustomers)
		}
		if len(tbl.ForeignKeys) != 0 {
			t.Errorf("CUSTOMERS should have 0 FKs, got %d: %+v", len(tbl.ForeignKeys), tbl.ForeignKeys)
		}
	})
}

func TestReader_ReadSchema_ColumnOrder(t *testing.T) {
	// Oracle guarantees COLUMN_ID ordering; verify the first column is always ID.
	s := mustReadSchema(t)
	for _, tblName := range []string{tblCustomers, tblOrders, tblItems} {
		tbl := findTable(s.Tables, tblName)
		if tbl == nil {
			continue
		}
		if len(tbl.Columns) == 0 {
			t.Errorf("%s: no columns", tblName)
			continue
		}
		if tbl.Columns[0].Name != "ID" {
			t.Errorf("%s: first column = %q, want ID", tblName, tbl.Columns[0].Name)
		}
	}
}

// TestReader_FullPipeline is an end-to-end test: Oracle schema → Handlebars engine → generated C# files.
func TestReader_FullPipeline(t *testing.T) {
	s := mustReadSchema(t)

	// Filter schema to our test tables only, to keep assertions stable.
	var testTables []schema.Table
	for _, tbl := range s.Tables {
		if strings.HasPrefix(tbl.Name, "DAX_TEST_") {
			testTables = append(testTables, tbl)
		}
	}
	if len(testTables) == 0 {
		t.Fatal("no DAX_TEST_ tables found in schema")
	}
	filtered := &schema.Schema{Tables: testTables}

	// Create a temporary project directory with a minimal csharp-dapper template.
	dir := t.TempDir()
	templateDir := filepath.Join(dir, ".daxonne", "templates", "csharp-dapper")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatal(err)
	}

	templateJSON := `{
  "name": "csharp-dapper",
  "version": "1.0.0",
  "language": "csharp",
  "description": "test",
  "author": "test",
  "files": [
    {"template": "dto.hbs",        "output": "{{PascalCase name}}Dto.cs",        "per": "table"},
    {"template": "repository.hbs", "output": "{{PascalCase name}}Repository.cs", "per": "table"}
  ]
}`

	dtoHBS := `// Auto-generated
public record {{PascalCase name}}Dto(
{{#each columns}}
    {{CSharpType type}} {{PascalCase name}}{{#unless @last}},{{/unless}}
{{/each}}
);`

	repoHBS := `// Auto-generated
public class {{PascalCase name}}Repository {
    public async Task<IEnumerable<{{PascalCase name}}Dto>> GetAllAsync() {
        const string sql = @"SELECT {{JoinColumns columns}} FROM {{name}}";
        return await _db.QueryAsync<{{PascalCase name}}Dto>(sql);
    }
    public async Task<{{PascalCase name}}Dto?> GetByIdAsync({{PrimaryKeyType columns}} id) {
        const string sql = @"SELECT {{JoinColumns columns}} FROM {{name}} WHERE {{PrimaryKeyColumn columns}} = :id";
        return await _db.QueryFirstOrDefaultAsync<{{PascalCase name}}Dto>(sql, new { id });
    }
}`

	writeFile := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(templateDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeFile("template.json", templateJSON)
	writeFile("dto.hbs", dtoHBS)
	writeFile("repository.hbs", repoHBS)

	// Chdir to the temp directory so the engine can find .daxonne/templates/.
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	cfg := &config.Config{
		Output:    config.OutputConfig{Path: "./out"},
		Templates: []string{"csharp-dapper"},
	}

	eng := generator.NewEngine()
	files, err := eng.GenerateFromTemplates(filtered, cfg)
	if err != nil {
		t.Fatalf("GenerateFromTemplates: %v", err)
	}

	// Expect 2 files per table (dto + repository).
	wantFiles := len(testTables) * 2
	if len(files) != wantFiles {
		t.Errorf("generated %d files, want %d", len(files), wantFiles)
	}

	// Check that CUSTOMERS table produced expected file names and content.
	var customerDto, customerRepo *generator.GeneratedFile
	for i := range files {
		switch files[i].Path {
		case "DaxTestCustomersDto.cs":
			customerDto = &files[i]
		case "DaxTestCustomersRepository.cs":
			customerRepo = &files[i]
		}
	}

	if customerDto == nil {
		t.Error("DaxTestCustomersDto.cs not generated")
	} else {
		assertContains(t, "DTO content", customerDto.Content, "public record DaxTestCustomersDto")
		// ID is NUMBER(10) → precision 10 > 9 → TypeLong → C# long
		assertContains(t, "DTO content", customerDto.Content, "long Id")
		assertContains(t, "DTO content", customerDto.Content, "string FirstName")
		assertContains(t, "DTO content", customerDto.Content, "decimal Balance")
		assertContains(t, "DTO content", customerDto.Content, "DateOnly JoinedAt")
	}

	if customerRepo == nil {
		t.Error("DaxTestCustomersRepository.cs not generated")
	} else {
		assertContains(t, "Repo content", customerRepo.Content, "public class DaxTestCustomersRepository")
		assertContains(t, "Repo content", customerRepo.Content, "SELECT")
		assertContains(t, "Repo content", customerRepo.Content, "FROM DAX_TEST_CUSTOMERS")
		assertContains(t, "Repo content", customerRepo.Content, "WHERE ID = :id")
	}
}

func assertContains(t *testing.T, label, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("%s: expected to contain %q\nGot:\n%s", label, substr, content)
	}
}
