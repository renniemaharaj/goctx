package builder

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SmartResolve analyzes the selected files and finds related dependencies (LSP-style)
func SmartResolve(root string, selectedFiles []string) []string {
	absRoot, _ := filepath.Abs(root)
	relatedMap := make(map[string]bool)

	// 1. Identify imports from selected .go files
	fset := token.NewFileSet()
	var imports []string

	for _, file := range selectedFiles {
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

	if len(imports) == 0 {
		return nil
	}

	// 2. Resolve imports to directories using `go list`
	// This acts as a robust "Find Definition" for packages.
	type GoList struct {
		Dir string
	}

	args := append([]string{"list", "-json"}, imports...)
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Parse JSON stream from go list
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var pkg GoList
		if err := dec.Decode(&pkg); err == nil {
			if pkg.Dir != "" && strings.HasPrefix(pkg.Dir, absRoot) {
				// It's an internal project dependency
				// Add all .go files in that directory
				entries, _ := os.ReadDir(pkg.Dir)
				for _, e := range entries {
					if !e.IsDir() && filepath.Ext(e.Name()) == ".go" {
						fullPath := filepath.Join(pkg.Dir, e.Name())
						relPath, _ := filepath.Rel(absRoot, fullPath)
						relatedMap[relPath] = true
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
