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

const AI_PROMPT_HEADER = `You are an AI developer agent. Return ONLY JSON:\n{\n  "short_description": "summary",\n  "files": { "path": "content" }\n}`

func LoadIgnorePatterns(root string) []string {
	patterns := []string{".git", ".stashes", "node_modules", "goctx", "go.sum", "ctx.json"}
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
	out := model.ProjectOutput{
		InstructionHeader: AI_PROMPT_HEADER,
		ShortDescription:  description,
		Files:             make(map[string]string),
	}

	ignorePatterns := LoadIgnorePatterns(root)
	dirChan := make(chan string, 10000) // Large buffer for directory paths
	var wg sync.WaitGroup
	var mu sync.Mutex
	var allPaths []string
	totalChars := 0

	// Worker Pool
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
					relPath, _ := filepath.Rel(root, fullPath)

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
						// Non-blocking push to avoid deadlock
						go func(p string) { dirChan <- p }(fullPath)
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
	dirChan <- root

	// This goroutine safely closes the channel once all workers finish
	go func() {
		wg.Wait()
		close(dirChan)
	}()

	// Block until the channel is closed and drained
	for range dirChan {}

	sort.Strings(allPaths)
	var tree strings.Builder
	for _, p := range allPaths {
		depth := strings.Count(p, string(os.PathSeparator))
		indent := ""
		if depth > 0 { indent = strings.Repeat("  ", depth) + "└── " }
		tree.WriteString(fmt.Sprintf("%s%s\n", indent, filepath.Base(p)))
	}

	out.ProjectTree = tree.String()
	out.EstimatedTokens = (totalChars + len(out.ProjectTree)) / 4
	return out, nil
}