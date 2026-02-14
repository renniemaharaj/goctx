package builder

import (
	"goctx/internal/model"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type fileResult struct {
	path    string
	content string
}

func BuildSelectiveContext(root string, targets []string) (model.ProjectOutput, error) {
	out := model.ProjectOutput{Files: make(map[string]string)}
	fileChan := make(chan string, 100)
	resultChan := make(chan fileResult, 100)
	var wg sync.WaitGroup

	// 1. Start Workers (Parallel Reading)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				c, err := os.ReadFile(path)
				if err == nil {
					rel, _ := filepath.Rel(root, path)
					resultChan <- fileResult{rel, string(c)}
				}
			}
		}()
	}

	// 2. Start Collector
	done := make(chan bool)
	go func() {
		for res := range resultChan {
			out.Files[res.path] = res.content
		}
		done <- true
	}()

	// 3. Walk and Pipe paths
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil { return nil }
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == ".stashes" || name == ".vscode" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		matched := len(targets) == 0
		for _, t := range targets {
			if strings.Contains(rel, t) { matched = true; break }
		}

		if matched { fileChan <- path }
		return nil
	})

	close(fileChan)
	wg.Wait()
	close(resultChan)
	<-done

	return out, err
}