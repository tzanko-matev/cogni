package reportserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewHandlerServesHTML ensures the root path returns the report HTML shell.
func TestNewHandlerServesHTML(t *testing.T) {
	dbPath := writeTempDB(t, "duckdb")
	handler, err := NewHandler(Config{DBPath: dbPath})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, "app.js") {
		t.Fatalf("expected app.js reference in HTML")
	}
	if !strings.Contains(body, "app.css") {
		t.Fatalf("expected app.css reference in HTML")
	}
}

// TestNewHandlerServesDatabase ensures the DuckDB endpoint returns the file content.
func TestNewHandlerServesDatabase(t *testing.T) {
	dbPath := writeTempDB(t, "duckdb")
	handler, err := NewHandler(Config{DBPath: dbPath})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/data/db.duckdb", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if got := resp.Body.String(); got != "duckdb" {
		t.Fatalf("unexpected db payload: %s", got)
	}
}

// TestNewHandlerUsesAssetsBaseURL verifies HTML assets use the configured base URL.
func TestNewHandlerUsesAssetsBaseURL(t *testing.T) {
	dbPath := writeTempDB(t, "duckdb")
	handler, err := NewHandler(Config{
		DBPath:        dbPath,
		AssetsBaseURL: "https://cdn.example.com/assets",
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, "https://cdn.example.com/assets/app.js") {
		t.Fatalf("expected base url in js asset")
	}
	if !strings.Contains(body, "https://cdn.example.com/assets/app.css") {
		t.Fatalf("expected base url in css asset")
	}
}

// writeTempDB writes a fake DuckDB file for handler tests.
func writeTempDB(t *testing.T, contents string) string {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "report.duckdb")
	if err := os.WriteFile(dbPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write temp db: %v", err)
	}
	return dbPath
}
