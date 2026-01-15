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
	assets := expectedAssets(t, "")
	if !strings.Contains(body, assets.ScriptURL) {
		t.Fatalf("expected script url %s in HTML", assets.ScriptURL)
	}
	for _, styleURL := range assets.StyleURLs {
		if !strings.Contains(body, styleURL) {
			t.Fatalf("expected style url %s in HTML", styleURL)
		}
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
	baseURL := "https://cdn.example.com/assets"
	handler, err := NewHandler(Config{
		DBPath:        dbPath,
		AssetsBaseURL: baseURL,
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
	assets := expectedAssets(t, baseURL)
	if !strings.Contains(body, assets.ScriptURL) {
		t.Fatalf("expected base url in js asset")
	}
	for _, styleURL := range assets.StyleURLs {
		if !strings.Contains(body, styleURL) {
			t.Fatalf("expected base url in css asset")
		}
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

// expectedAssets resolves asset URLs using the embedded manifest.
func expectedAssets(t *testing.T, baseURL string) resolvedAssets {
	t.Helper()
	manifest, err := loadEmbeddedManifest()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	reportAssets, err := resolveReportAssets(manifest)
	if err != nil {
		t.Fatalf("resolve assets: %v", err)
	}
	resolver := newAssetResolver(baseURL)
	styleURLs := make([]string, 0, len(reportAssets.Styles))
	for _, style := range reportAssets.Styles {
		styleURLs = append(styleURLs, resolver.URL(style))
	}
	return resolvedAssets{
		ScriptURL: resolver.URL(reportAssets.Script),
		StyleURLs: styleURLs,
	}
}

// resolvedAssets holds resolved URLs for HTML assertions.
type resolvedAssets struct {
	ScriptURL string
	StyleURLs []string
}
