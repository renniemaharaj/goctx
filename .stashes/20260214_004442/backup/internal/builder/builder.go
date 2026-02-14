package builder

import (
	"bufio"
	"fmt"
	"goctx/internal/model"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const AI_PROMPT_HEADER = `You are an AI developer agent. Return ONLY JSON:
{
  "short_description": "summary",
  "files": { "path": "content" }
}`

func LoadIgnorePatterns(root string) []string {
	patterns := []string{".git", ".stashes", "node_modules", "goctx", "go.sum"}
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

func BuildSelectiveContext(root string, targets []string) (model.ProjectOutput, error) {
	out := model.ProjectOutput{
		InstructionHeader: AI_PROMPT_HEADER,
		ShortDescription:  "Workspace snapshot",
		Files:             make(map[string]string),
	}
	ignorePatterns := LoadIgnorePatterns(root)
	fileChan := make(chan string, 100)
	resultChan := make(chan struct{p string; c string}, 100)
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				c, err := os.ReadFile(path)
				if err == nil {
					rel, _ := filepath.Rel(root, path)
					resultChan <- struct{p string; c string}{rel, string(c)}
				}
			}
		}()
	}

	done := make(chan bool)
	var totalChars int
	go func() {
		for res := range resultChan {
			out.Files[res.p] = res.c
			totalChars += len(res.c)
		}
		done <- true
	}()

	var tree strings.Builder
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || path == root { return nil }
		rel, _ := filepath.Rel(root, path)
		for _, p := range ignorePatterns {
			if strings.Contains(rel, p) {
				if info.IsDir() { return filepath.SkipDir }
				return nil
			}
		}
		depth := strings.Count(rel, string(os.PathSeparator))
		indent := ""
		if depth > 0 { indent = strings.Repeat("  ", depth) + "└── " }
		tree.WriteString(fmt.Sprintf("%s%s\n", indent, info.Name()))
		if !info.IsDir() { fileChan <- path }
		return nil
	})

	close(fileChan); wg.Wait(); close(resultChan); <-done
	out.ProjectTree = tree.String()
	out.EstimatedTokens = (totalChars + len(out.ProjectTree)) / 4
	return out, err
}