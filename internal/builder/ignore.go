package builder

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// LoadIgnorePatterns reads .gitignore and .ctxignore to build a list of patterns
func LoadIgnorePatterns(root string) []string {
	// Default strict ignores
	patterns := []string{".git", ".stashes", ".bin", ".exe"}

	read := func(fname string) {
		f, err := os.Open(filepath.Join(root, fname))
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" && !strings.HasPrefix(line, "#") {
					patterns = append(patterns, line)
				}
			}
		}
	}

	read(".gitignore")
	read(".ctxignore")
	return patterns
}

// MatchesIgnore checks if a relative path matches any ignore pattern
func MatchesIgnore(relPath string, patterns []string) bool {
	pathParts := strings.Split(relPath, string(os.PathSeparator))

	for _, p := range patterns {
		// Directory match (e.g. "node_modules/")
		if strings.HasSuffix(p, "/") {
			clean := strings.TrimSuffix(p, "/")
			for _, part := range pathParts {
				if part == clean {
					return true
				}
			}
		}

		// Exact match or Glob match on filename
		matched, _ := filepath.Match(p, filepath.Base(relPath))
		if matched {
			return true
		}

		// Path prefix match
		if strings.HasPrefix(relPath, p) {
			return true
		}
	}
	return false
}
