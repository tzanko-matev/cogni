package duckdb

import (
	"database/sql"
	_ "embed"
	"errors"
)

// schemaDDL holds the DuckDB schema definition.
//
//go:embed schema.sql
var schemaDDL string

// SchemaDDL returns the schema DDL used for initializing DuckDB databases.
func SchemaDDL() string {
	return schemaDDL
}

// EnsureSchema applies the schema DDL to the provided database connection.
func EnsureSchema(db *sql.DB) error {
	if db == nil {
		return errors.New("duckdb: db is nil")
	}
	_, err := db.Exec(schemaDDL)
	return err
}
