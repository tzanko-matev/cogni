package reportserver

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

//go:embed assets/*
var embeddedAssets embed.FS

// AssetResolver maps asset file paths to URLs for the report HTML shell.
type AssetResolver struct {
	baseURL string
}

// ManifestEntry captures the subset of a Vite manifest entry we need.
type ManifestEntry struct {
	File    string   `json:"file"`
	CSS     []string `json:"css"`
	IsEntry bool     `json:"isEntry"`
}

// AssetManifest maps entry keys to manifest metadata.
type AssetManifest map[string]ManifestEntry

// ReportAssets captures the JS and CSS files needed for the report shell.
type ReportAssets struct {
	Script string
	Styles []string
}

const reportEntryKey = "index.html"

// newAssetResolver creates a resolver for embedded or externally hosted assets.
func newAssetResolver(baseURL string) AssetResolver {
	return AssetResolver{
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// URL resolves an asset file path to a full URL.
func (r AssetResolver) URL(assetPath string) string {
	trimmed := strings.TrimLeft(assetPath, "/")
	if r.baseURL == "" {
		return "/assets/" + trimmed
	}
	return r.baseURL + "/" + trimmed
}

// embeddedAssetsFS returns the file system rooted at the embedded assets directory.
func embeddedAssetsFS() (fs.FS, error) {
	sub, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		return nil, fmt.Errorf("reportserver: open embedded assets: %w", err)
	}
	return sub, nil
}

// loadEmbeddedManifest loads the JSON manifest from embedded assets.
func loadEmbeddedManifest() (AssetManifest, error) {
	manifestFile, err := embeddedAssets.Open("assets/manifest.json")
	if err != nil {
		return nil, fmt.Errorf("reportserver: read manifest: %w", err)
	}
	defer manifestFile.Close()

	manifestBytes, err := io.ReadAll(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("reportserver: read manifest: %w", err)
	}

	var manifest AssetManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("reportserver: parse manifest: %w", err)
	}
	if len(manifest) == 0 {
		return nil, errors.New("reportserver: manifest is empty")
	}
	return manifest, nil
}

// resolveReportAssets selects the main entry assets for the report shell.
func resolveReportAssets(manifest AssetManifest) (ReportAssets, error) {
	entry, ok := manifest[reportEntryKey]
	if !ok {
		return ReportAssets{}, fmt.Errorf("reportserver: manifest missing %s entry", reportEntryKey)
	}
	if entry.File == "" {
		return ReportAssets{}, errors.New("reportserver: manifest entry missing script file")
	}
	return ReportAssets{
		Script: entry.File,
		Styles: entry.CSS,
	}, nil
}
