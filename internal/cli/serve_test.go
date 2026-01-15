package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"cogni/internal/reportserver"
)

// TestServeCommandRequiresDBPath verifies serve fails when no DB argument is provided.
func TestServeCommandRequiresDBPath(t *testing.T) {
	cmd := findCommand("serve")
	if cmd == nil {
		t.Fatalf("serve command not found")
	}
	var stdout, stderr bytes.Buffer
	exitCode := cmd.Run([]string{}, &stdout, &stderr)
	if exitCode != ExitUsage {
		t.Fatalf("expected usage exit, got %d", exitCode)
	}
}

// TestServeCommandPassesConfig ensures serve forwards parsed config to the server layer.
func TestServeCommandPassesConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "report.duckdb")
	if err := os.WriteFile(dbPath, []byte("duckdb"), 0o644); err != nil {
		t.Fatalf("write temp db: %v", err)
	}

	var gotConfig reportserver.Config
	origServe := serveReport
	serveReport = func(_ context.Context, cfg reportserver.Config) error {
		gotConfig = cfg
		return nil
	}
	t.Cleanup(func() { serveReport = origServe })

	cmd := findCommand("serve")
	if cmd == nil {
		t.Fatalf("serve command not found")
	}
	var stdout, stderr bytes.Buffer
	exitCode := cmd.Run([]string{
		"--addr", "127.0.0.1:5050",
		"--assets-base-url", "https://assets.example.com",
		dbPath,
	}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("expected exit ok, got %d: %s", exitCode, stderr.String())
	}
	if gotConfig.Addr != "127.0.0.1:5050" {
		t.Fatalf("unexpected addr: %s", gotConfig.Addr)
	}
	if gotConfig.AssetsBaseURL != "https://assets.example.com" {
		t.Fatalf("unexpected assets base url: %s", gotConfig.AssetsBaseURL)
	}
	if gotConfig.DBPath != dbPath {
		t.Fatalf("unexpected db path: %s", gotConfig.DBPath)
	}
}
