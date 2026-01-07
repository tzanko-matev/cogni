package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogni/internal/spec"
)

// validateAdapters checks adapter entries and returns a map of adapter IDs.
func validateAdapters(cfg *spec.Config, baseDir string, add issueAdder) map[string]struct{} {
	adapterIDs := map[string]struct{}{}
	for i, adapter := range cfg.Adapters {
		fieldPrefix := fmt.Sprintf("adapters[%d]", i)
		id := strings.TrimSpace(adapter.ID)
		if id == "" {
			add(fieldPrefix+".id", "is required")
		} else if _, exists := adapterIDs[id]; exists {
			add("adapters.id", fmt.Sprintf("duplicate id %q", id))
		} else {
			adapterIDs[id] = struct{}{}
		}

		adapterType := strings.TrimSpace(adapter.Type)
		switch adapterType {
		case "cucumber":
			if strings.TrimSpace(adapter.Runner) == "" {
				add(fieldPrefix+".runner", "is required")
			} else if adapter.Runner != "godog" {
				add(fieldPrefix+".runner", fmt.Sprintf("unsupported runner %q", adapter.Runner))
			}
			if strings.TrimSpace(adapter.Formatter) == "" {
				add(fieldPrefix+".formatter", "is required")
			} else if adapter.Formatter != "json" {
				add(fieldPrefix+".formatter", fmt.Sprintf("unsupported formatter %q", adapter.Formatter))
			}
		case "cucumber_manual":
			if strings.TrimSpace(adapter.ExpectationsDir) == "" {
				add(fieldPrefix+".expectations_dir", "is required")
			}
		case "":
			add(fieldPrefix+".type", "is required")
		default:
			add(fieldPrefix+".type", fmt.Sprintf("unsupported type %q", adapter.Type))
		}

		if len(adapter.FeatureRoots) == 0 {
			add(fieldPrefix+".feature_roots", "must include at least one entry")
		}
		for rootIndex, root := range adapter.FeatureRoots {
			root = strings.TrimSpace(root)
			if root == "" {
				add(fmt.Sprintf("%s.feature_roots[%d]", fieldPrefix, rootIndex), "is required")
				continue
			}
			rootPath := root
			if !filepath.IsAbs(rootPath) {
				rootPath = filepath.Join(baseDir, rootPath)
			}
			info, err := os.Stat(rootPath)
			if err != nil {
				add(fmt.Sprintf("%s.feature_roots[%d]", fieldPrefix, rootIndex), fmt.Sprintf("path not found at %q", root))
				continue
			}
			if !info.IsDir() {
				add(fmt.Sprintf("%s.feature_roots[%d]", fieldPrefix, rootIndex), fmt.Sprintf("path %q is not a directory", root))
			}
		}
		if adapterType == "cucumber_manual" && strings.TrimSpace(adapter.ExpectationsDir) != "" {
			dirPath := adapter.ExpectationsDir
			if !filepath.IsAbs(dirPath) {
				dirPath = filepath.Join(baseDir, dirPath)
			}
			info, err := os.Stat(dirPath)
			if err != nil {
				add(fieldPrefix+".expectations_dir", fmt.Sprintf("path not found at %q", adapter.ExpectationsDir))
			} else if !info.IsDir() {
				add(fieldPrefix+".expectations_dir", fmt.Sprintf("path %q is not a directory", adapter.ExpectationsDir))
			}
		}
	}
	return adapterIDs
}
