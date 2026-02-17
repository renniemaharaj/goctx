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
	tabsRoot, _ := filepath.Abs(root)
	relatedMap := make(map[string]bool)
	brokenMap := make(map[string]bool)

	// 1. Run Build Command to find broken files
	if buildCmd != "" {
		parts := strings.Fields(buildCmd)
		if len(parts) > 0 {
			cmd := exec.Command(parts[0], parts[1:]...)
			cmd.Dir = root
			out, _ := cmd.CombinedOutput()

			re := regexp.MustCompile(`(?m)(?:^|\s)([^:\s]+\.go):\d+:\d+:`)
			matches := re.FindAllStringSubmatch(string(out), -1)
			for _, m := range matches {
				if len(m) > 1 {
					fpath := m[1]
					// Fix: S1017 - Use strings.TrimPrefix
					fpath = strings.TrimPrefix(fpath, "./")
					if _, err := os.Stat(filepath.Join(root, fpath)); err == nil {
						brokenMap[fpath] = true
						relatedMap[fpath] = true
					}
				}
			}
		}
	}

	// 2. Identify imports from (Selected Files + Broken Files)
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
			path := strings.Trim(i.Path.Value, "\"")
			imports = append(imports, path)
		}
	}

	if len(imports) == 0 && len(relatedMap) == 0 {
		return nil
	}

	// 3. Resolve imports
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
					if pkg.Dir != "" && strings.HasPrefix(pkg.Dir, tabsRoot) {
						entries, _ := os.ReadDir(pkg.Dir)
						for _, e := range entries {
							if !e.IsDir() && filepath.Ext(e.Name()) == ".go" {
								fullPath := filepath.Join(pkg.Dir, e.Name())
								relPath, _ := filepath.Rel(tabsRoot, fullPath)

								if brokenMap[relPath] {
									continue
								}

								info, err := e.Info()
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
