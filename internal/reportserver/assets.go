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

// AssetResolver maps logical asset names to URLs for the report HTML shell.
type AssetResolver struct {
	baseURL  string
	manifest map[string]string
}

// newAssetResolver creates a resolver for embedded or externally hosted assets.
func newAssetResolver(baseURL string, manifest map[string]string) AssetResolver {
	return AssetResolver{
		baseURL:  strings.TrimRight(baseURL, "/"),
		manifest: manifest,
	}
}

// URL resolves a logical asset name to a URL using the manifest and base URL.
func (r AssetResolver) URL(logicalName string) (string, error) {
	filename, ok := r.manifest[logicalName]
	if !ok {
		return "", fmt.Errorf("reportserver: asset not found: %s", logicalName)
	}
	if r.baseURL == "" {
		return "/assets/" + filename, nil
	}
	return r.baseURL + "/" + filename, nil
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
func loadEmbeddedManifest() (map[string]string, error) {
	manifestFile, err := embeddedAssets.Open("assets/manifest.json")
	if err != nil {
		return nil, fmt.Errorf("reportserver: read manifest: %w", err)
	}
	defer manifestFile.Close()

	manifestBytes, err := io.ReadAll(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("reportserver: read manifest: %w", err)
	}

	var manifest map[string]string
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("reportserver: parse manifest: %w", err)
	}
	if len(manifest) == 0 {
		return nil, errors.New("reportserver: manifest is empty")
	}
	return manifest, nil
}
