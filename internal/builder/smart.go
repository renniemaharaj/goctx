package builder

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SmartResolve analyzes the selected files, finds related dependencies (LSP-style),
// and includes files with build errors. It filters stable/stale dependencies.
func SmartResolve(root string, selectedFiles []string, buildCmd string) []string {
	absRoot, _ := filepath.Abs(root)
	relatedMap := make(map[string]bool)
	brokenMap := make(map[string]bool)

	// 1. Run Build Command to find broken files
	// We execute the configured build script blindly and parse stderr for filenames
	if buildCmd != "" {
		parts := strings.Fields(buildCmd)
		if len(parts) > 0 {
			cmd := exec.Command(parts[0], parts[1:]...)
			cmd.Dir = root
			// Ignore error (err will be non-nil on build failure), we want the output
			out, _ := cmd.CombinedOutput()

			// Regex for Go errors: file.go:line:col:
			re := regexp.MustCompile(`(?m)(?:^|\s)([^:\s]+\.go):\d+:\d+:`)
			matches := re.FindAllStringSubmatch(string(out), -1)
			for _, m := range matches {
				if len(m) > 1 {
					fpath := m[1]
					// Normalize relative paths (e.g. "./main.go" -> "main.go")
					if strings.HasPrefix(fpath, "./") {
						fpath = fpath[2:]
					}
					// Ensure it's inside root before adding
					if _, err := os.Stat(filepath.Join(root, fpath)); err == nil {
						brokenMap[fpath] = true
						relatedMap[fpath] = true // Force include broken files
					}
				}
			}
		}
	}

	// 2. Identify imports from (Selected Files + Broken Files)
	// We want to verify dependencies of broken files too, not just the manually selected ones.
	scanSet := make(map[string]bool)
	for _, f := range selectedFiles {
		scanSet[f] = true
	}
	for f := range brokenMap {
		scanSet[f] = true
	}

	fset := token.NewFileSet()
	var imports []string

	for file := range scanSet {
		if filepath.Ext(file) != ".go" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			continue
		}

		astFile, err := parser.ParseFile(fset, "", content, parser.ImportsOnly)
		if err != nil {
			continue
		}

		for _, i := range astFile.Imports {
			// Strip quotes
			path := strings.Trim(i.Path.Value, "\"")
			imports = append(imports, path)
		}
	}

	// If no imports found and no broken files found, return empty
	if len(imports) == 0 && len(relatedMap) == 0 {
		return nil
	}

	// 3. Resolve imports to directories using `go list`
	// This acts as a robust "Find Definition" for packages.
	if len(imports) > 0 {
		type GoList struct {
			Dir string
		}

		args := append([]string{"list", "-json"}, imports...)
		cmd := exec.Command("go", args...)
		cmd.Dir = root
		out, err := cmd.Output()

		if err == nil {
			dec := json.NewDecoder(bytes.NewReader(out))
			for dec.More() {
				var pkg GoList
				if err := dec.Decode(&pkg); err == nil {
					if pkg.Dir != "" && strings.HasPrefix(pkg.Dir, absRoot) {
						// It's an internal project dependency.
						// Add all .go files in that directory, subject to staleness check.
						entries, _ := os.ReadDir(pkg.Dir)
						for _, e := range entries {
							if !e.IsDir() && filepath.Ext(e.Name()) == ".go" {
								fullPath := filepath.Join(pkg.Dir, e.Name())
								relPath, _ := filepath.Rel(absRoot, fullPath)

								// STALENESS CHECK:
								// Only include implicit dependencies if they are:
								// a) Broken (already in map)
								// b) Modified recently (e.g. last 5 days)
								// This keeps context focused on active work and reduces token usage.

								if brokenMap[relPath] {
									continue // Already added
								}

								info, err := e.Info()
								// 120 hours = 5 days
								if err == nil && time.Since(info.ModTime()) < 120*time.Hour {
									relatedMap[relPath] = true
								}
							}
						}
					}
				}
			}
		}
	}

	var result []string
	for k := range relatedMap {
		result = append(result, k)
	}
	return result
}
