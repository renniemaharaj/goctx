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
<-System instruction header: GoCtx Patch Protocol->
1. Patching Protocol:
   - SEARCH block must match old lines exactly (including indentation, literal char for char).
   - Include sufficient context lines to avoid collisions (3-3 lines recommended).
   - All changes must Prioritize small patches over monolithic rewrites.
   	- eg:
   		Surgical patch for existing file:
   		**(Hunks Search and Replace Native Dialect)**: Use hunks native search and replace code blocks for clipboard transfer.
		{backticks}
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
		{backticks}
   - **(Full File Replacement)**: For new files or complete rewrites, provide the full file content with a clear header.
     eg:
		{backticks}
	 	"internal/pkg/newfile.go":
	 	package main
	 	func main (){}
	 	{backticks}
	- **(Full File Deletion)**: For file deletions, specify the file with an empty content block.
	 	eg:
		{backticks}
	 	"internal/pkg/oldfile.go":
	 	{backticks}
	- Always ensure you output code blocks for hunks and search and replace or full file replacement are properly formatted in the chat's ui code block for clipboard transfer. 
	- {backticks} placeholder to denote where code fence back ticks would be placed if they didn't the prompt's formatting.
	- Ensure you correctly match searches with the minimal necessary to avoid failures and ensure the search exactly matches the current state of the file.
	<-System instruction footer: GoCtx Patch Protocol->
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
