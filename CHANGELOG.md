# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

---

## [0.1.0] — 2026-03-01

### Added

#### Core CLI
- `daxonne init` — interactive project initialisation, generates `daxonne.yaml`
- `daxonne pull` — reads the database schema and caches it to `.daxonne/schema.json`
- `daxonne add <template>` — installs a template from the registry into `.daxonne/templates/`
- `daxonne add --list` — lists all templates available in the registry
- `daxonne generate` — renders all installed templates against the cached schema

#### Schema layer
- Universal internal schema model: `Schema`, `Table`, `Column`, `ForeignKey`
- Normalised `InternalType` enum: `string`, `int`, `long`, `decimal`, `bool`, `date`, `datetime`, `bytes`, `uuid`
- `ISchemaReader` interface decoupling schema reading from any specific DBMS

#### Oracle plugin (`plugins/oracle`)
- Pure-Go Oracle driver via `github.com/sijms/go-ora/v2` — no Oracle Client required
- Reads tables, columns, primary keys and foreign keys from `ALL_TABLES`, `ALL_TAB_COLUMNS`, `ALL_CONSTRAINTS`, `ALL_CONS_COLUMNS`
- Oracle → InternalType mapping covering: `VARCHAR2`, `CHAR`, `NVARCHAR2`, `CLOB`, `NUMBER`, `INTEGER`, `FLOAT`, `DATE`, `TIMESTAMP*`, `BLOB`, `RAW`
- `NUMBER(p,0)` precision-aware mapping: p ≤ 9 → `int`, 10–18 → `long`, scale > 0 → `decimal`

#### Handlebars code generation engine
- Template engine based on `github.com/aymerick/raymond`
- `per: "table"` mode — one output file per table
- `per: "schema"` mode — one output file for the whole schema
- Output filenames are themselves Handlebars expressions (e.g. `{{PascalCase name}}Dto.cs`)
- Built-in helpers: `PascalCase`, `CamelCase`, `CSharpType`, `JoinColumns`, `JoinParams`, `PrimaryKeyColumn`, `PrimaryKeyType`

#### Template — `csharp-dapper` (v1.0.0)
- `{{PascalCase name}}Dto.cs` — C# `record` with correctly typed properties
- `{{PascalCase name}}Repository.cs` — Dapper repository with `GetAllAsync`, `GetByIdAsync`, `InsertAsync`, `DeleteAsync`

#### Template registry
- Hardcoded registry with four entries: `csharp-dapper`, `typescript-prisma`, `java-jpa`, `python-sqlalchemy`
- `daxonne add` auto-updates `daxonne.yaml` with the newly installed template name

#### Tests
- 33 unit tests for Oracle type mapping (`types_test.go`)
- 15 unit tests for the Handlebars engine helpers and pipeline (`engine_test.go`)
- Full Oracle integration test suite (`oracle_integration_test.go`, build tag `integration`):
  - `TestMain` creates and tears down `DAX_TEST_CUSTOMERS`, `DAX_TEST_ORDERS`, `DAX_TEST_ITEMS`
  - Covers: connection, schema discovery, column types, nullability, primary keys, foreign keys, column ordering, end-to-end generation

#### CI — GitHub Actions
- **Build & Unit Tests** job: `go mod verify`, `go vet`, `go build`, `go test -race` on every push and PR
- **Integration Tests** job: Oracle Free service container (`gvenzl/oracle-free`), full integration suite with race detection; runs only after the build job succeeds

[Unreleased]: https://github.com/Daxonne/core/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Daxonne/core/releases/tag/v0.1.0
