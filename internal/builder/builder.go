package builder

import (
	"bytes"
	"fmt"
	"goctx/internal/config"
	"goctx/internal/model"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const MaxTreeDepth = 5

var systemIgnores = map[string]bool{
	"proc":       true,
	"sys":        true,
	"dev":        true,
	"run":        true,
	"boot":       true,
	"lost+found": true,
	"snap":       true,
	"var/lib":    true,
}

const AI_PROMPT_HEADER = `
System instruction header: GoCtx Patch Protocol
1. Patch Rules:
   - SEARCH block must match old lines exactly (including indentation).
   - Include sufficient context lines to avoid collisions (3-3 lines recommended).
   - All changes must be atomic and sensible
   - Prioritize small patches over monolithic rewrites.
3. Workflow Guidance:
   - Scan clipboard or other inputs for ProjectOutput objects.
   - If patches stop working:
     * Request fresh context from the user.
	 * Adjust search to match more reliably
     * Optionally send raw file content for manual replacement.
   - Explicitly indicate patch mode: "surgical" or "full".
4. Examples:
   - Surgical patch for existing file:
   - **Option B (Native Dialect)**: Use the raw format for clipboard transfer. This is preferred to avoid JSON escaping issues.
	 // text
     "internal/pkg/file.go":
     <<<<<< SEARCH
     func Old() {
         return
     }
     ======
     func New() {
         return "updated"
     }
     >>>>>> REPLACE
	Ensure you correctly match searches
   - Full file creation:
     "path/new_file.go": "[full file content]"
Please wrap your output patches in your native code editor or code block for user to copy
`

func isBinary(data []byte) bool {
	return bytes.Contains(data, []byte{0})
}

func GetFileList(root string) ([]string, error) {
	ignorePatterns := LoadIgnorePatterns(root)
	var files []string
	const safetyCap = 10000
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if len(files) >= safetyCap {
			return filepath.SkipAll
		}
		rel, _ := filepath.Rel(root, path)
		if rel == ".ctxignore" || rel == ".gitignore" {
			files = append(files, rel)
			return nil
		}
		if MatchesIgnore(rel, ignorePatterns) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}

func BuildSelectiveContext(root string, description string, whitelist []string, tokenLimit int, smartMode bool) (model.ProjectOutput, error) {
	filter := make(map[string]bool)
	for _, f := range whitelist {
		filter[f] = true
	}

	// Smart Mode: LSP-like resolution of dependencies
	if smartMode {
		cfg, _ := config.Load(root)
		related := SmartResolve(root, whitelist, cfg.Scripts.Build)
		for _, r := range related {
			if !filter[r] {
				filter[r] = true
			}
		}
	}

	absRoot, _ := filepath.Abs(root)
	out := model.ProjectOutput{
		ShortDescription: description,
		Files:            make(map[string]string),
	}

	var dirCount int
	ignorePatterns := LoadIgnorePatterns(absRoot)
	dirChan := make(chan string, 10000)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allPaths []string
	hardMaxEntry := 100
	totalTokens := 0

	// Concurrent Walker
	for i := 0; i < 8; i++ {
		go func() {
			for dir := range dirChan {
				entries, err := os.ReadDir(dir)
				if err != nil {
					wg.Done()
					continue
				}
				for entryI, entry := range entries {
					if entryI > hardMaxEntry {
						continue
					}
					name := entry.Name()
					fullPath := filepath.Join(dir, name)
					relPath, _ := filepath.Rel(absRoot, fullPath)

					// Hard-coded system bypass
					if systemIgnores[relPath] || systemIgnores[name] {
						continue
					}

					// Calculate depth: count separators in the relative path
					currentDepth := 0
					if relPath != "." {
						currentDepth = strings.Count(relPath, string(os.PathSeparator))
					}

					// Hardening: Skip directories deeper than MaxTreeDepth
					if entry.IsDir() && currentDepth >= MaxTreeDepth {
						continue
					}

					// Ignore logic
					ignored := MatchesIgnore(relPath, ignorePatterns)
					if relPath == ".ctxignore" || relPath == ".gitignore" {
						ignored = false
					}

					if ignored {
						continue
					}

					// Whitelist filtering
					if !entry.IsDir() && whitelist != nil && !filter[relPath] {
						continue
					}

					mu.Lock()
					allPaths = append(allPaths, relPath)
					mu.Unlock()

					if entry.IsDir() {
						mu.Lock()
						dirCount++
						mu.Unlock()
						wg.Add(1)
						dirChan <- fullPath
					} else {
						content, err := os.ReadFile(fullPath)
						if err == nil && !isBinary(content) {
							mu.Lock()
							// Budget Check (approx 4 chars per token)
							estTok := len(content) / 4
							if totalTokens+estTok < tokenLimit {
								out.Files[relPath] = string(content)
								totalTokens += estTok
							}
							mu.Unlock()
						}
					}
				}
				wg.Done()
			}
		}()
	}

	wg.Add(1)
	dirChan <- absRoot
	wg.Wait()
	close(dirChan)

	sort.Strings(allPaths)
	var tree strings.Builder
	lastSkipped := ""

	for _, p := range allPaths {
		depth := strings.Count(p, string(os.PathSeparator))

		if depth > MaxTreeDepth {
			parent := filepath.Dir(p)
			if lastSkipped != parent {
				indent := strings.Repeat("  ", MaxTreeDepth) + "└── ..."
				tree.WriteString(fmt.Sprintf("%s\n", indent))
				lastSkipped = parent
			}
			continue
		}

		indent := ""
		if depth > 0 {
			indent = strings.Repeat("  ", depth) + "└── "
		}
		tree.WriteString(fmt.Sprintf("%s%s\n", indent, filepath.Base(p)))
	}

	out.ProjectTree = tree.String()
	out.FileCount = len(out.Files)
	out.TokenCount = totalTokens
	out.DirCount = dirCount
	return out, nil
}
