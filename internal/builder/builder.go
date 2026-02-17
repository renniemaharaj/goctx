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

const AI_PROMPT_HEADER = `
System instruction header: GoCtx Patch Protocol
1. json schema:
   Use type ProjectOutput struct unmarshallable json object:
   - ShortDescription (string, optional)
   - ProjectTree (string)
   - Files (map[string]string) — 
     * For existing files, use SEARCH/REPLACE format.
     * For new files, send full content.
	 * Use the native code block to present response project outputs with patches that do not fail build or 
2. Patch Rules:
   - SEARCH block must match old lines exactly (including indentation).
   - Include sufficient context lines to avoid collisions (3-5 lines recommended).
   - All changes must be atomic and auto-stashed.
   - Prioritize small patches over monolithic rewrites.
3. Workflow Guidance:
   - Scan clipboard or other inputs for ProjectOutput objects.
   - If patches stop working:
     * Request fresh context from the user.
     * Optionally send raw file content for manual replacement.
   - Explicitly indicate patch mode: "surgical" or "full".
4. Examples:
   - Surgical patch for existing file:
     "path/file.go": "<<<<<< SEARCH\n[old lines]\n======\n[new lines]\n>>>>>>
   - **Option A (JSON)**: Wrap markers in the JSON value. Use \n for newlines.
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
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		// Never ignore configuration files for context, but respect others
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
		// Pass the build command to SmartResolve so it can find broken files
		related := SmartResolve(root, whitelist, cfg.Scripts.Build)
		for _, r := range related {
			// Only add if not explicitly selected (we will process priority later)
			if !filter[r] {
				// We mark related files as 'implicit' in logic, but here we just add to filter for processing
				// Priority logic: Whitelist > Related > Others (if we were doing full walk)
				// Current logic: Selective Context only includes what is in filter.
				filter[r] = true
			}
		}
	}

	absRoot, _ := filepath.Abs(root)
	out := model.ProjectOutput{
		ShortDescription: description,
		Files:            make(map[string]string),
	}

	ignorePatterns := LoadIgnorePatterns(absRoot)
	dirChan := make(chan string, 10000)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allPaths []string
	totalTokens := 0

	for i := 0; i < 8; i++ {
		go func() {
			for dir := range dirChan {
				entries, err := os.ReadDir(dir)
				if err != nil {
					wg.Done()
					continue
				}

				for _, entry := range entries {
					fullPath := filepath.Join(dir, entry.Name())
					relPath, _ := filepath.Rel(absRoot, fullPath)

					if relPath == "." {
						continue
					}

					// Use new ignore logic
					ignored := MatchesIgnore(relPath, ignorePatterns)
					if relPath == ".ctxignore" || relPath == ".gitignore" {
						ignored = false
					}

					if ignored {
						continue
					}

					// If we have a whitelist (from UI or Smart), only include those
					if !entry.IsDir() && whitelist != nil && !filter[relPath] {
						continue
					}

					mu.Lock()
					allPaths = append(allPaths, relPath)
					mu.Unlock()

					if entry.IsDir() {
						wg.Add(1)
						dirChan <- fullPath
					} else {
						content, err := os.ReadFile(fullPath)
						if err == nil && !isBinary(content) {
							mu.Lock()
							// Budget Check
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
	for _, p := range allPaths {
		depth := strings.Count(p, string(os.PathSeparator))
		indent := ""
		if depth > 0 {
			indent = strings.Repeat("  ", depth-1) + "└── "
		} else if p != "." {
			indent = ""
		}
		tree.WriteString(fmt.Sprintf("%s%s\n", indent, filepath.Base(p)))
	}

	out.ProjectTree = tree.String()
	return out, nil
}
