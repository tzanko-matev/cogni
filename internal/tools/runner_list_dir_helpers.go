package tools

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// listDirRelPath returns a root-relative path for error reporting.
func listDirRelPath(rootRel, normalized string) string {
	if normalized == "" {
		return rootRel
	}
	return filepath.Join(rootRel, filepath.FromSlash(normalized))
}

// classifyListDirEntry determines suffix and traversal eligibility for an entry.
func classifyListDirEntry(info os.FileInfo) (suffix string, isDir bool, isSymlink bool) {
	mode := info.Mode()
	if mode&os.ModeSymlink != 0 {
		return "@", false, true
	}
	if info.IsDir() {
		return "/", true, false
	}
	if mode.IsRegular() {
		return "", false, false
	}
	return "?", false, false
}

// joinNormalizedPath joins a parent path and entry name using slash separators.
func joinNormalizedPath(parent, name string) string {
	if parent == "" {
		return path.Join(name)
	}
	return path.Join(parent, name)
}

// formatListDirOutput builds the list_dir output with pagination and sorting.
func formatListDirOutput(abs string, entries []listDirEntry, offset, limit int) (string, error) {
	total := len(entries)
	if total == 0 {
		if offset > 1 {
			return "", fmt.Errorf("offset exceeds directory entry count")
		}
		return fmt.Sprintf("Absolute path: %s", abs), nil
	}
	if offset > total {
		return "", fmt.Errorf("offset exceeds directory entry count")
	}
	start := offset - 1
	end := start + limit
	if end > total {
		end = total
	}
	selected := append([]listDirEntry(nil), entries[start:end]...)
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].normalized < selected[j].normalized
	})

	var builder strings.Builder
	builder.WriteString("Absolute path: ")
	builder.WriteString(abs)
	if len(selected) > 0 {
		builder.WriteString("\n")
	}
	for i, entry := range selected {
		builder.WriteString(formatListDirEntry(entry))
		if i < len(selected)-1 || total > end {
			builder.WriteString("\n")
		}
	}
	if total > end {
		builder.WriteString(fmt.Sprintf("More than %d entries found.", limit))
	}
	return builder.String(), nil
}

// formatListDirEntry applies indentation to a formatted entry.
func formatListDirEntry(entry listDirEntry) string {
	indent := strings.Repeat(" ", (entry.depth-1)*2)
	return indent + entry.displayName
}

// truncateListDirName limits entry display names to a safe byte boundary.
func truncateListDirName(name string) string {
	if len(name) <= listDirMaxNameBytes {
		return name
	}
	truncated := name[:listDirMaxNameBytes]
	for len(truncated) > 0 && !utf8.ValidString(truncated) {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}
