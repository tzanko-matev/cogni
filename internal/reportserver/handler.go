package reportserver

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const indexHTMLTemplate = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Cogni Report</title>
    %s
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
	reportAssets, err := resolveReportAssets(manifest)
	if err != nil {
		return nil, err
	}
	resolver := newAssetResolver(cfg.AssetsBaseURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveIndex(resolver, reportAssets))
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
func serveIndex(resolver AssetResolver, assets ReportAssets) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		cssLinks := buildCSSLinks(resolver, assets.Styles)
		jsURL := resolver.URL(assets.Script)
		html := fmt.Sprintf(indexHTMLTemplate, cssLinks, jsURL)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, html)
	}
}

// buildCSSLinks renders link tags for the provided stylesheet assets.
func buildCSSLinks(resolver AssetResolver, styles []string) string {
	if len(styles) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, style := range styles {
		fmt.Fprintf(&builder, "<link rel=\"stylesheet\" href=\"%s\" />\n", resolver.URL(style))
	}
	return builder.String()
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
