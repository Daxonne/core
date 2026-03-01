package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aymerick/raymond"
	"github.com/daxonne/core/internal/config"
	"github.com/daxonne/core/internal/schema"
)

// ─── toPascalCase ─────────────────────────────────────────────────────────────

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"USER_ACCOUNT", "UserAccount"},
		{"user_account", "UserAccount"},
		{"FIRST_NAME", "FirstName"},
		{"order_id", "OrderId"},
		{"ID", "Id"},
		{"id", "Id"},
		{"DAX_TEST_CUSTOMERS", "DaxTestCustomers"},
		{"TOTAL_AMT", "TotalAmt"},
		{"kebab-case", "KebabCase"},
		{"with spaces", "WithSpaces"},
		{"", ""},
		{"SINGLE", "Single"},
		{"already_mixed_CASE", "AlreadyMixedCase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ─── Handlebars helpers ───────────────────────────────────────────────────────

// helperEngine ensures helpers are registered before each test group.
func helperEngine() *Engine { return NewEngine() }

func renderHelper(t *testing.T, tmpl string, data interface{}) string {
	t.Helper()
	out, err := raymond.Render(tmpl, data)
	if err != nil {
		t.Fatalf("raymond.Render(%q): %v", tmpl, err)
	}
	return strings.TrimSpace(out)
}

func TestHelper_PascalCase(t *testing.T) {
	helperEngine()
	tests := []struct{ input, want string }{
		{"USER_ACCOUNT", "UserAccount"},
		{"order_id", "OrderId"},
		{"ID", "Id"},
	}
	for _, tt := range tests {
		got := renderHelper(t, `{{PascalCase name}}`, map[string]interface{}{"name": tt.input})
		if got != tt.want {
			t.Errorf("PascalCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHelper_CamelCase(t *testing.T) {
	helperEngine()
	tests := []struct{ input, want string }{
		{"USER_ACCOUNT", "userAccount"},
		{"FIRST_NAME", "firstName"},
		{"ID", "id"},
	}
	for _, tt := range tests {
		got := renderHelper(t, `{{CamelCase name}}`, map[string]interface{}{"name": tt.input})
		if got != tt.want {
			t.Errorf("CamelCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHelper_CSharpType(t *testing.T) {
	helperEngine()
	tests := []struct {
		internalType string
		want         string
	}{
		{"string", "string"},
		{"int", "int"},
		{"long", "long"},
		{"decimal", "decimal"},
		{"bool", "bool"},
		{"date", "DateOnly"},
		{"datetime", "DateTime"},
		{"bytes", "byte[]"},
		{"uuid", "Guid"},
		{"unknown", "object"}, // fallback
	}
	for _, tt := range tests {
		got := renderHelper(t, `{{CSharpType type}}`, map[string]interface{}{"type": tt.internalType})
		if got != tt.want {
			t.Errorf("CSharpType(%q) = %q, want %q", tt.internalType, got, tt.want)
		}
	}
}

func TestHelper_JoinColumns(t *testing.T) {
	helperEngine()
	cols := []interface{}{
		map[string]interface{}{"name": "ID",         "type": "int"},
		map[string]interface{}{"name": "FIRST_NAME", "type": "string"},
		map[string]interface{}{"name": "BALANCE",    "type": "decimal"},
	}
	got := renderHelper(t, `{{JoinColumns columns}}`, map[string]interface{}{"columns": cols})
	want := "ID, FIRST_NAME, BALANCE"
	if got != want {
		t.Errorf("JoinColumns = %q, want %q", got, want)
	}
}

func TestHelper_JoinColumns_Empty(t *testing.T) {
	helperEngine()
	got := renderHelper(t, `{{JoinColumns columns}}`, map[string]interface{}{"columns": []interface{}{}})
	if got != "" {
		t.Errorf("JoinColumns(empty) = %q, want empty string", got)
	}
}

func TestHelper_JoinParams(t *testing.T) {
	helperEngine()
	cols := []interface{}{
		map[string]interface{}{"name": "ID",         "type": "int"},
		map[string]interface{}{"name": "FIRST_NAME", "type": "string"},
		map[string]interface{}{"name": "BALANCE",    "type": "decimal"},
	}
	got := renderHelper(t, `{{JoinParams columns}}`, map[string]interface{}{"columns": cols})
	want := ":Id, :FirstName, :Balance"
	if got != want {
		t.Errorf("JoinParams = %q, want %q", got, want)
	}
}

func TestHelper_PrimaryKeyColumn(t *testing.T) {
	helperEngine()
	cols := []interface{}{
		map[string]interface{}{"name": "ID",    "type": "int",    "isPrimary": true},
		map[string]interface{}{"name": "EMAIL", "type": "string", "isPrimary": false},
	}
	got := renderHelper(t, `{{PrimaryKeyColumn columns}}`, map[string]interface{}{"columns": cols})
	if got != "ID" {
		t.Errorf("PrimaryKeyColumn = %q, want ID", got)
	}
}

func TestHelper_PrimaryKeyColumn_NoPK_Fallback(t *testing.T) {
	helperEngine()
	cols := []interface{}{
		map[string]interface{}{"name": "NAME", "type": "string", "isPrimary": false},
	}
	got := renderHelper(t, `{{PrimaryKeyColumn columns}}`, map[string]interface{}{"columns": cols})
	if got != "id" {
		t.Errorf("PrimaryKeyColumn(no PK) = %q, want fallback \"id\"", got)
	}
}

func TestHelper_PrimaryKeyType(t *testing.T) {
	helperEngine()
	tests := []struct {
		pkType string
		want   string
	}{
		{"int", "int"},
		{"long", "long"},
		{"string", "string"},
		{"uuid", "Guid"},
	}
	for _, tt := range tests {
		cols := []interface{}{
			map[string]interface{}{"name": "ID", "type": tt.pkType, "isPrimary": true},
		}
		got := renderHelper(t, `{{PrimaryKeyType columns}}`, map[string]interface{}{"columns": cols})
		if got != tt.want {
			t.Errorf("PrimaryKeyType(%q) = %q, want %q", tt.pkType, got, tt.want)
		}
	}
}

// ─── tableToTemplateData ──────────────────────────────────────────────────────

func TestTableToTemplateData(t *testing.T) {
	tbl := schema.Table{
		Name: "CUSTOMERS",
		Columns: []schema.Column{
			{Name: "ID",    Type: schema.TypeInt,    IsPrimary: true, Nullable: false},
			{Name: "EMAIL", Type: schema.TypeString, IsPrimary: false, Nullable: true},
		},
		PrimaryKeys: []string{"ID"},
		ForeignKeys: []schema.ForeignKey{
			{Column: "DEPT_ID", ReferencedTable: "DEPARTMENTS", ReferencedColumn: "ID"},
		},
	}

	data := tableToTemplateData(tbl)

	if data["name"] != "CUSTOMERS" {
		t.Errorf("name = %v, want CUSTOMERS", data["name"])
	}

	cols, ok := data["columns"].([]interface{})
	if !ok {
		t.Fatalf("columns is not []interface{}: %T", data["columns"])
	}
	if len(cols) != 2 {
		t.Fatalf("columns len = %d, want 2", len(cols))
	}

	col0, _ := cols[0].(map[string]interface{})
	if col0["name"] != "ID" {
		t.Errorf("col[0].name = %v, want ID", col0["name"])
	}
	if col0["type"] != "int" {
		t.Errorf("col[0].type = %v, want int", col0["type"])
	}
	if col0["isPrimary"] != true {
		t.Errorf("col[0].isPrimary = %v, want true", col0["isPrimary"])
	}

	col1, _ := cols[1].(map[string]interface{})
	if col1["nullable"] != true {
		t.Errorf("col[1].nullable = %v, want true", col1["nullable"])
	}

	pks, _ := data["primaryKeys"].([]string)
	if len(pks) != 1 || pks[0] != "ID" {
		t.Errorf("primaryKeys = %v, want [ID]", pks)
	}
}

// ─── GenerateFromTemplates end-to-end ────────────────────────────────────────

func setupTemplateDir(t *testing.T, name string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, ".daxonne", "templates", name)
	if err := os.MkdirAll(tmplDir, 0755); err != nil {
		t.Fatal(err)
	}
	for fname, content := range files {
		if err := os.WriteFile(filepath.Join(tmplDir, fname), []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", fname, err)
		}
	}
	return dir
}

func TestGenerateFromTemplates_DTO(t *testing.T) {
	dir := setupTemplateDir(t, "test-tmpl", map[string]string{
		"template.json": `{
  "name": "test-tmpl", "version": "1.0.0", "language": "csharp",
  "description": "", "author": "",
  "files": [{"template": "dto.hbs", "output": "{{PascalCase name}}Dto.cs", "per": "table"}]
}`,
		"dto.hbs": `public record {{PascalCase name}}Dto(
{{#each columns}}    {{CSharpType type}} {{PascalCase name}}{{#unless @last}},{{/unless}}
{{/each}});`,
	})

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	s := &schema.Schema{
		Tables: []schema.Table{
			{
				Name: "CUSTOMERS",
				Columns: []schema.Column{
					{Name: "ID",    Type: schema.TypeInt,     IsPrimary: true,  Nullable: false},
					{Name: "EMAIL", Type: schema.TypeString,  IsPrimary: false, Nullable: true},
					{Name: "SCORE", Type: schema.TypeDecimal, IsPrimary: false, Nullable: false},
				},
				PrimaryKeys: []string{"ID"},
			},
		},
	}

	cfg := &config.Config{
		Output:    config.OutputConfig{Path: "./out"},
		Templates: []string{"test-tmpl"},
	}

	eng := NewEngine()
	files, err := eng.GenerateFromTemplates(s, cfg)
	if err != nil {
		t.Fatalf("GenerateFromTemplates: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	f := files[0]
	if f.Path != "CustomersDto.cs" {
		t.Errorf("path = %q, want CustomersDto.cs", f.Path)
	}

	assertContains := func(sub string) {
		t.Helper()
		if !strings.Contains(f.Content, sub) {
			t.Errorf("content missing %q\nGot:\n%s", sub, f.Content)
		}
	}

	assertContains("public record CustomersDto(")
	assertContains("int Id")
	assertContains("string Email")
	assertContains("decimal Score")
}

func TestGenerateFromTemplates_MultipleTemplates(t *testing.T) {
	dir := t.TempDir()

	// Template A — DTOs
	aDir := filepath.Join(dir, ".daxonne", "templates", "tmpl-a")
	os.MkdirAll(aDir, 0755)
	os.WriteFile(filepath.Join(aDir, "template.json"), []byte(`{
  "name":"tmpl-a","version":"1.0.0","language":"csharp","description":"","author":"",
  "files":[{"template":"dto.hbs","output":"{{PascalCase name}}Dto.cs","per":"table"}]
}`), 0644)
	os.WriteFile(filepath.Join(aDir, "dto.hbs"), []byte(`// DTO: {{name}}`), 0644)

	// Template B — repos
	bDir := filepath.Join(dir, ".daxonne", "templates", "tmpl-b")
	os.MkdirAll(bDir, 0755)
	os.WriteFile(filepath.Join(bDir, "template.json"), []byte(`{
  "name":"tmpl-b","version":"1.0.0","language":"csharp","description":"","author":"",
  "files":[{"template":"repo.hbs","output":"{{PascalCase name}}Repo.cs","per":"table"}]
}`), 0644)
	os.WriteFile(filepath.Join(bDir, "repo.hbs"), []byte(`// Repo: {{name}}`), 0644)

	orig, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(orig) })

	s := &schema.Schema{
		Tables: []schema.Table{
			{Name: "ORDERS", Columns: []schema.Column{{Name: "ID", Type: schema.TypeInt, IsPrimary: true}}},
			{Name: "ITEMS",  Columns: []schema.Column{{Name: "ID", Type: schema.TypeInt, IsPrimary: true}}},
		},
	}
	cfg := &config.Config{
		Output:    config.OutputConfig{Path: "./out"},
		Templates: []string{"tmpl-a", "tmpl-b"},
	}

	eng := NewEngine()
	files, err := eng.GenerateFromTemplates(s, cfg)
	if err != nil {
		t.Fatalf("GenerateFromTemplates: %v", err)
	}

	// 2 tables × 2 templates = 4 files
	if len(files) != 4 {
		t.Errorf("expected 4 files, got %d: %v", len(files), fileNames(files))
	}

	paths := map[string]bool{}
	for _, f := range files {
		paths[f.Path] = true
	}
	for _, want := range []string{"OrdersDto.cs", "ItemsDto.cs", "OrdersRepo.cs", "ItemsRepo.cs"} {
		if !paths[want] {
			t.Errorf("missing file %q; got %v", want, fileNames(files))
		}
	}
}

func TestGenerateFromTemplates_MissingTemplate(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(orig) })

	s := &schema.Schema{Tables: []schema.Table{{Name: "X"}}}
	cfg := &config.Config{
		Output:    config.OutputConfig{Path: "./out"},
		Templates: []string{"nonexistent"},
	}

	eng := NewEngine()
	_, err := eng.GenerateFromTemplates(s, cfg)
	if err == nil {
		t.Error("expected error for missing template, got nil")
	}
}

func TestGenerateFromTemplates_SchemaPerFile(t *testing.T) {
	dir := setupTemplateDir(t, "schema-tmpl", map[string]string{
		"template.json": `{
  "name":"schema-tmpl","version":"1.0.0","language":"csharp","description":"","author":"",
  "files":[{"template":"summary.hbs","output":"Summary.txt","per":"schema"}]
}`,
		"summary.hbs": `Tables: {{tables.length}}`,
	})

	orig, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(orig) })

	s := &schema.Schema{
		Tables: []schema.Table{
			{Name: "T1"}, {Name: "T2"}, {Name: "T3"},
		},
	}
	cfg := &config.Config{Output: config.OutputConfig{Path: "./out"}, Templates: []string{"schema-tmpl"}}

	eng := NewEngine()
	files, err := eng.GenerateFromTemplates(s, cfg)
	if err != nil {
		t.Fatalf("GenerateFromTemplates: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "Summary.txt" {
		t.Errorf("path = %q, want Summary.txt", files[0].Path)
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func fileNames(files []GeneratedFile) []string {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = f.Path
	}
	return names
}
