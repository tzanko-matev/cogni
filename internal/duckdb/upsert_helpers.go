package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// nullableString converts an optional string pointer into a SQL argument.
func nullableString(value *string) interface{} {
	if value == nil {
		return nil
	}
	if *value == "" {
		return nil
	}
	return *value
}

// mapExpression builds a map constructor expression for SQL literals.
func mapExpression(dims map[string]string) string {
	if dims == nil {
		return "NULL"
	}
	if len(dims) == 0 {
		return "map([], [])"
	}
	keys := make([]string, 0, len(dims))
	for k := range dims {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	keyLiterals := make([]string, 0, len(keys))
	valLiterals := make([]string, 0, len(keys))
	for _, k := range keys {
		keyLiterals = append(keyLiterals, quoteLiteral(k))
		valLiterals = append(valLiterals, quoteLiteral(dims[k]))
	}
	return fmt.Sprintf("map([%s], [%s])", strings.Join(keyLiterals, ", "), strings.Join(valLiterals, ", "))
}

// quoteLiteral escapes a string for SQL literal use.
func quoteLiteral(value string) string {
	escaped := strings.ReplaceAll(value, "'", "''")
	return "'" + escaped + "'"
}

// lookupID fetches a single ID column value for a row keyed by keyColumn.
func lookupID(ctx context.Context, db *sql.DB, table, idColumn, keyColumn, key string) (string, error) {
	query := fmt.Sprintf("SELECT CAST(%s AS VARCHAR) FROM %s WHERE %s = ?", idColumn, table, keyColumn)
	var id string
	if err := db.QueryRowContext(ctx, query, key).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}
