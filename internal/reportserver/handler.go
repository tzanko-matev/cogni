package reportserver

import (
	"errors"
	"io"
	"net/http"
)

const indexHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Cogni Report</title>
  </head>
  <body>
    <h1>Cogni Report</h1>
    <p>Report assets and interactive charts will load here.</p>
  </body>
</html>`

// NewHandler builds the HTTP handler for serving the report UI and DuckDB file.
func NewHandler(cfg Config) (http.Handler, error) {
	if cfg.DBPath == "" {
		return nil, errors.New("reportserver: db path is required")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex)
	mux.Handle("/data/db.duckdb", serveDatabase(cfg.DBPath))
	return mux, nil
}

// serveIndex writes the base HTML shell for the report UI.
func serveIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, indexHTML)
}

// serveDatabase serves the DuckDB file from disk for browser-side processing.
func serveDatabase(dbPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, dbPath)
	})
}
