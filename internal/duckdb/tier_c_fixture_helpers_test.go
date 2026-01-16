//go:build duckdbtierc

package duckdb_test

import (
	"database/sql"
	"database/sql/driver"
	"fmt"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/google/uuid"
)

// fixtureNamespace ensures deterministic UUID generation.
var fixtureNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

// deterministicID generates a repeatable UUID from a prefix and index.
func deterministicID(prefix string, index int) string {
	return uuid.NewSHA1(fixtureNamespace, []byte(fmt.Sprintf("%s-%d", prefix, index))).String()
}

// parseDuckDBUUID converts a UUID string into the duckdb-go UUID wrapper.
func parseDuckDBUUID(value string) (duckdb.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return duckdb.UUID{}, err
	}
	return duckdb.UUID(parsed), nil
}

// newMeasurementAppender creates a DuckDB appender for bulk measurement inserts.
func newMeasurementAppender(conn *sql.Conn) (*duckdb.Appender, error) {
	var appender *duckdb.Appender
	if err := conn.Raw(func(driverConn any) error {
		rawConn, ok := driverConn.(driver.Conn)
		if !ok {
			return fmt.Errorf("duckdb driver connection unavailable (got %T)", driverConn)
		}
		var err error
		appender, err = duckdb.NewAppenderFromConn(rawConn, "", "measurements")
		return err
	}); err != nil {
		return nil, err
	}
	if appender == nil {
		return nil, fmt.Errorf("duckdb appender initialization failed")
	}
	return appender, nil
}
