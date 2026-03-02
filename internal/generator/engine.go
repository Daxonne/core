package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aymerick/raymond"
	"github.com/daxonne/core/internal/config"
	"github.com/daxonne/core/internal/schema"
)

// TemplateManifest is the parsed representation of a template's template.json file.
type TemplateManifest struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Language    string         `json:"language"`
	Description string         `json:"description"`
	Author      string         `json:"author"`
	Files       []TemplateFile `json:"files"`
}

// TemplateFile describes a single Handlebars template and its output pattern.
type TemplateFile struct {
	Template string `json:"template"` // filename of the .hbs file
	Output   string `json:"output"`   // Handlebars expression for the output filename
	Per      string `json:"per"`      // "table" or "schema"
}

var registerOnce sync.Once

// Engine drives Handlebars-based code generation using installed templates.
type Engine struct{}

// NewEngine creates an Engine and registers all built-in Handlebars helpers (once).
func NewEngine() *Engine {
	registerOnce.Do(registerHelpers)
	return &Engine{}
}

// GenerateFromTemplates generates code for every template listed in cfg.Templates.
// Templates are loaded from .daxonne/templates/<name>/.
func (e *Engine) GenerateFromTemplates(s *schema.Schema, cfg *config.Config) ([]GeneratedFile, error) {
	var generated []GeneratedFile

	for _, name := range cfg.Templates {
		templateDir := filepath.Join(".daxonne", "templates", name)
		files, err := e.processTemplate(s, templateDir)
		if err != nil {
			return nil, fmt.Errorf("template %q: %w", name, err)
		}
		generated = append(generated, files...)
	}

	return generated, nil
}

func (e *Engine) processTemplate(s *schema.Schema, templateDir string) ([]GeneratedFile, error) {
	manifestPath := filepath.Join(templateDir, "template.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading template.json: %w", err)
	}

	var manifest TemplateManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("parsing template.json: %w", err)
	}

	var generated []GeneratedFile

	for _, fileDef := range manifest.Files {
		hbsPath := filepath.Join(templateDir, fileDef.Template)
		hbsContent, err := os.ReadFile(hbsPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", fileDef.Template, err)
		}

		tpl, err := raymond.Parse(string(hbsContent))
		if err != nil {
			return nil, fmt.Errorf("parsing handlebars %s: %w", fileDef.Template, err)
		}

		switch fileDef.Per {
		case "table":
			for _, table := range s.Tables {
				data := tableToTemplateData(table)

				content, err := tpl.Exec(data)
				if err != nil {
					return nil, fmt.Errorf("rendering %s for table %s: %w", fileDef.Template, table.Name, err)
				}

				outName, err := raymond.Render(fileDef.Output, data)
				if err != nil {
					return nil, fmt.Errorf("rendering output name for table %s: %w", table.Name, err)
				}

				generated = append(generated, GeneratedFile{Path: outName, Content: content})
			}
		case "schema":
			data := schemaToTemplateData(s)
			content, err := tpl.Exec(data)
			if err != nil {
				return nil, fmt.Errorf("rendering %s: %w", fileDef.Template, err)
			}
			outName, err := raymond.Render(fileDef.Output, data)
			if err != nil {
				return nil, fmt.Errorf("rendering output name: %w", err)
			}
			generated = append(generated, GeneratedFile{Path: outName, Content: content})
		}
	}

	return generated, nil
}

// tableToTemplateData converts a schema.Table into a map suitable for Handlebars templates.
// All keys use camelCase to match the template conventions ({{name}}, {{columns}}, etc.).
func tableToTemplateData(t schema.Table) map[string]interface{} {
	columns := make([]interface{}, len(t.Columns))
	for i, col := range t.Columns {
		columns[i] = map[string]interface{}{
			"name":      col.Name,
			"type":      string(col.Type),
			"nullable":  col.Nullable,
			"isPrimary": col.IsPrimary,
		}
	}

	fks := make([]interface{}, len(t.ForeignKeys))
	for i, fk := range t.ForeignKeys {
		fks[i] = map[string]interface{}{
			"column":           fk.Column,
			"referencedTable":  fk.ReferencedTable,
			"referencedColumn": fk.ReferencedColumn,
		}
	}

	return map[string]interface{}{
		"name":        t.Name,
		"columns":     columns,
		"primaryKeys": t.PrimaryKeys,
		"foreignKeys": fks,
	}
}

func schemaToTemplateData(s *schema.Schema) map[string]interface{} {
	tables := make([]interface{}, len(s.Tables))
	for i, t := range s.Tables {
		tables[i] = tableToTemplateData(t)
	}
	return map[string]interface{}{"tables": tables}
}

// ─── Handlebars helpers ───────────────────────────────────────────────────────

// csharpTypeMap maps Daxonne internal types to their C# equivalents.
var csharpTypeMap = map[string]string{
	"string":   "string",
	"int":      "int",
	"long":     "long",
	"decimal":  "decimal",
	"bool":     "bool",
	"date":     "DateOnly",
	"datetime": "DateTime",
	"bytes":    "byte[]",
	"uuid":     "Guid",
}

// typescriptTypeMap maps Daxonne internal types to their TypeScript equivalents.
var typescriptTypeMap = map[string]string{
	"string":   "string",
	"int":      "number",
	"long":     "number",
	"decimal":  "number",
	"bool":     "boolean",
	"date":     "Date",
	"datetime": "Date",
	"bytes":    "Uint8Array",
	"uuid":     "string",
}

// javaTypeMap maps Daxonne internal types to their Java equivalents.
var javaTypeMap = map[string]string{
	"string":   "String",
	"int":      "Integer",
	"long":     "Long",
	"decimal":  "BigDecimal",
	"bool":     "Boolean",
	"date":     "LocalDate",
	"datetime": "LocalDateTime",
	"bytes":    "byte[]",
	"uuid":     "UUID",
}

// pythonTypeMap maps Daxonne internal types to their Python equivalents.
var pythonTypeMap = map[string]string{
	"string":   "str",
	"int":      "int",
	"long":     "int",
	"decimal":  "Decimal",
	"bool":     "bool",
	"date":     "date",
	"datetime": "datetime",
	"bytes":    "bytes",
	"uuid":     "UUID",
}

// sqlalchemyTypeMap maps Daxonne internal types to their SQLAlchemy column types.
var sqlalchemyTypeMap = map[string]string{
	"string":   "String",
	"int":      "Integer",
	"long":     "BigInteger",
	"decimal":  "Numeric",
	"bool":     "Boolean",
	"date":     "Date",
	"datetime": "DateTime",
	"bytes":    "LargeBinary",
	"uuid":     "Uuid",
}

// toPascalCase converts UPPER_SNAKE_CASE, snake_case, or kebab-case to PascalCase.
// Example: "USER_ACCOUNT" → "UserAccount", "first-name" → "FirstName".
func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, "")
}

// toSnakeCase converts UPPER_SNAKE_CASE or PascalCase to lower_snake_case.
// Example: "USER_ACCOUNT" → "user_account", "UserAccount" → "user_account".
func toSnakeCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "-", "_"))
}

func registerHelpers() {
	// PascalCase — {{PascalCase name}} → "UserAccount"
	raymond.RegisterHelper("PascalCase", func(str string) string {
		return toPascalCase(str)
	})

	// CamelCase — {{CamelCase name}} → "userAccount"
	raymond.RegisterHelper("CamelCase", func(str string) string {
		pc := toPascalCase(str)
		if len(pc) == 0 {
			return pc
		}
		return strings.ToLower(pc[:1]) + pc[1:]
	})

	// CSharpType — {{CSharpType type}} → "string", "int", "DateOnly", …
	raymond.RegisterHelper("CSharpType", func(t string) string {
		if v, ok := csharpTypeMap[t]; ok {
			return v
		}
		return "object"
	})

	// JoinColumns — {{JoinColumns columns}} → "COL1, COL2, COL3"
	raymond.RegisterHelper("JoinColumns", func(cols interface{}) string {
		return joinColumnField(cols, func(name string) string { return name })
	})

	// JoinParams — {{JoinParams columns}} → ":Col1, :Col2, :Col3"
	raymond.RegisterHelper("JoinParams", func(cols interface{}) string {
		return joinColumnField(cols, func(name string) string { return ":" + toPascalCase(name) })
	})

	// PrimaryKeyColumn — {{PrimaryKeyColumn columns}} → first PK column name
	raymond.RegisterHelper("PrimaryKeyColumn", func(cols interface{}) string {
		if v := firstPKField(cols, "name"); v != "" {
			return v
		}
		return "id"
	})

	// PrimaryKeyType — {{PrimaryKeyType columns}} → C# type of the first PK column
	raymond.RegisterHelper("PrimaryKeyType", func(cols interface{}) string {
		if t := firstPKField(cols, "type"); t != "" {
			if v, ok := csharpTypeMap[t]; ok {
				return v
			}
		}
		return "int"
	})

	// SnakeCase — {{SnakeCase name}} → "user_account"
	raymond.RegisterHelper("SnakeCase", func(str string) string {
		return toSnakeCase(str)
	})

	// TypeScriptType — {{TypeScriptType type}} → "string", "number", "boolean", …
	raymond.RegisterHelper("TypeScriptType", func(t string) string {
		if v, ok := typescriptTypeMap[t]; ok {
			return v
		}
		return "unknown"
	})

	// JavaType — {{JavaType type}} → "String", "Integer", "BigDecimal", …
	raymond.RegisterHelper("JavaType", func(t string) string {
		if v, ok := javaTypeMap[t]; ok {
			return v
		}
		return "Object"
	})

	// PythonType — {{PythonType type}} → "str", "int", "Decimal", …
	raymond.RegisterHelper("PythonType", func(t string) string {
		if v, ok := pythonTypeMap[t]; ok {
			return v
		}
		return "Any"
	})

	// SQLAlchemyType — {{SQLAlchemyType type}} → "String", "Integer", "Numeric", …
	raymond.RegisterHelper("SQLAlchemyType", func(t string) string {
		if v, ok := sqlalchemyTypeMap[t]; ok {
			return v
		}
		return "String"
	})

	// PrimaryKeyCamelCase — {{PrimaryKeyCamelCase columns}} → camelCase name of first PK column
	raymond.RegisterHelper("PrimaryKeyCamelCase", func(cols interface{}) string {
		if name := firstPKField(cols, "name"); name != "" {
			pc := toPascalCase(name)
			if len(pc) == 0 {
				return pc
			}
			return strings.ToLower(pc[:1]) + pc[1:]
		}
		return "id"
	})

	// TypeScriptPrimaryKeyType — {{TypeScriptPrimaryKeyType columns}} → TypeScript type of first PK
	raymond.RegisterHelper("TypeScriptPrimaryKeyType", func(cols interface{}) string {
		if t := firstPKField(cols, "type"); t != "" {
			if v, ok := typescriptTypeMap[t]; ok {
				return v
			}
		}
		return "number"
	})

	// JavaPrimaryKeyType — {{JavaPrimaryKeyType columns}} → Java type of first PK
	raymond.RegisterHelper("JavaPrimaryKeyType", func(cols interface{}) string {
		if t := firstPKField(cols, "type"); t != "" {
			if v, ok := javaTypeMap[t]; ok {
				return v
			}
		}
		return "Long"
	})

	// PythonPrimaryKeyType — {{PythonPrimaryKeyType columns}} → Python type of first PK
	raymond.RegisterHelper("PythonPrimaryKeyType", func(cols interface{}) string {
		if t := firstPKField(cols, "type"); t != "" {
			if v, ok := pythonTypeMap[t]; ok {
				return v
			}
		}
		return "int"
	})
}

// joinColumnField iterates a []interface{} of column maps and applies transform to each "name".
func joinColumnField(cols interface{}, transform func(string) string) string {
	var parts []string
	if list, ok := cols.([]interface{}); ok {
		for _, c := range list {
			if m, ok := c.(map[string]interface{}); ok {
				if name, ok := m["name"].(string); ok {
					parts = append(parts, transform(name))
				}
			}
		}
	}
	return strings.Join(parts, ", ")
}

// firstPKField returns the value of field for the first column where isPrimary == true.
func firstPKField(cols interface{}, field string) string {
	if list, ok := cols.([]interface{}); ok {
		for _, c := range list {
			if m, ok := c.(map[string]interface{}); ok {
				if isPK, _ := m["isPrimary"].(bool); isPK {
					if v, ok := m[field].(string); ok {
						return v
					}
				}
			}
		}
	}
	return ""
}
