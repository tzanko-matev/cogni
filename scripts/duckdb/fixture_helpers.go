package main

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"path/filepath"

	duckdbdriver "github.com/duckdb/duckdb-go/v2"
	"github.com/google/uuid"
)

type duckdbUUID = duckdbdriver.UUID

// newMeasurementAppender creates a DuckDB appender for bulk measurement inserts.
func newMeasurementAppender(conn *sql.Conn) (*duckdbdriver.Appender, error) {
	var appender *duckdbdriver.Appender
	if err := conn.Raw(func(driverConn any) error {
		rawConn, ok := driverConn.(driver.Conn)
		if !ok {
			return fmt.Errorf("duckdb driver connection unavailable (got %T)", driverConn)
		}
		var err error
		appender, err = duckdbdriver.NewAppenderFromConn(rawConn, "", "measurements")
		return err
	}); err != nil {
		return nil, err
	}
	if appender == nil {
		return nil, fmt.Errorf("duckdb appender initialization failed")
	}
	return appender, nil
}

// parseDuckDBUUID converts a UUID string into the duckdb-go UUID wrapper.
func parseDuckDBUUID(value string) (duckdbUUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return duckdbUUID{}, err
	}
	return duckdbUUID(parsed), nil
}

// dirOf returns the parent directory for a file path.
func dirOf(path string) string {
	if path == "" {
		return "."
	}
	if idx := len(path) - 1; idx >= 0 && path[idx] == os.PathSeparator {
		return path
	}
	return filepath.Dir(path)
}

// removeIfExists deletes an existing fixture file so we always start fresh.
func removeIfExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove existing fixture: %w", err)
		}
		return nil
	}
	if os.IsNotExist(err) {
		return nil
	}
	return fmt.Errorf("stat fixture: %w", err)
}

// deterministicID generates a repeatable UUID for fixture rows.
func deterministicID(prefix string, index int) string {
	return uuid.NewSHA1(fixtureNamespace, []byte(fmt.Sprintf("%s-%d", prefix, index))).String()
}

// fixtureNamespace ensures stable UUIDs across fixture runs.
var fixtureNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
