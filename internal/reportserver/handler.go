package reportserver

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

const indexHTMLTemplate = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Cogni Report</title>
    <link rel="stylesheet" href="%s" />
  </head>
  <body>
    <div id="app">
      <h1>Cogni Report</h1>
      <p>Report assets and interactive charts will load here.</p>
    </div>
    <script type="module" src="%s"></script>
  </body>
</html>`

// NewHandler builds the HTTP handler for serving the report UI and DuckDB file.
func NewHandler(cfg Config) (http.Handler, error) {
	if cfg.DBPath == "" {
		return nil, errors.New("reportserver: db path is required")
	}

	manifest, err := loadEmbeddedManifest()
	if err != nil {
		return nil, err
	}
	resolver := newAssetResolver(cfg.AssetsBaseURL, manifest)

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex(resolver))
	mux.Handle("/data/db.duckdb", serveDatabase(cfg.DBPath))
	if cfg.AssetsBaseURL == "" {
		assetsFS, err := embeddedAssetsFS()
		if err != nil {
			return nil, err
		}
		mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsFS))))
	}
	return mux, nil
}

// serveIndex builds a handler that writes the HTML shell with resolved assets.
func serveIndex(resolver AssetResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		cssURL, err := resolver.URL("app.css")
		if err != nil {
			http.Error(w, "missing css asset", http.StatusInternalServerError)
			return
		}
		jsURL, err := resolver.URL("app.js")
		if err != nil {
			http.Error(w, "missing js asset", http.StatusInternalServerError)
			return
		}
		html := fmt.Sprintf(indexHTMLTemplate, cssURL, jsURL)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, html)
	}
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
