package builder

import (
	"bufio"
	"bytes"
	"fmt"
	"goctx/internal/model"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const AI_PROMPT_HEADER = `You are an AI developer agent. Return ONLY JSON.
For surgical updates to existing files, use the format:
"path/file.go": "<<<<<< SEARCH\n[old lines]\n======\n[new lines]\n>>>>>> REPLACE"

Otherwise, provide the full file content for new files.`

func LoadIgnorePatterns(root string) []string {
	patterns := []string{".git", ".stashes", "node_modules", "goctx", "go.sum", "ctx.json", ".exe", ".bin"}
	f, err := os.Open(filepath.Join(root, ".ctxignore"))
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
	return patterns
}

func isBinary(data []byte) bool {
	return bytes.Contains(data, []byte{0})
}

func BuildSelectiveContext(root string, description string) (model.ProjectOutput, error) {
	absRoot, _ := filepath.Abs(root)
	out := model.ProjectOutput{
		InstructionHeader: AI_PROMPT_HEADER,
		ShortDescription:  description,
		Files:             make(map[string]string),
	}

	ignorePatterns := LoadIgnorePatterns(absRoot)
	dirChan := make(chan string, 10000)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allPaths []string
	totalChars := 0

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

					ignored := false
					for _, p := range ignorePatterns {
						if strings.Contains(relPath, p) {
							ignored = true
							break
						}
					}
					if ignored { continue }

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
							out.Files[relPath] = string(content)
							totalChars += len(content)
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
	out.EstimatedTokens = (totalChars + len(out.ProjectTree)) / 4
	return out, nil
}