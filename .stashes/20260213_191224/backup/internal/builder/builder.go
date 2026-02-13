package builder

import (
	"goctx/internal/model"
	"os"
	"path/filepath"
	"strings"
)

func BuildSelectiveContext(root string, targets []string) (model.ProjectOutput, error) {
	out := model.ProjectOutput{Files: make(map[string]string)}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.Contains(path, ".git") || strings.Contains(path, ".stashes") { return nil }
		rel, _ := filepath.Rel(root, path)
		if len(targets) == 0 {
			c, _ := os.ReadFile(path)
			out.Files[rel] = string(c)
			return nil
		}
		for _, t := range targets {
			if strings.Contains(rel, t) {
				c, _ := os.ReadFile(path)
			out.Files[rel] = string(c)
			}
		}
		return nil
	})
	return out, nil
}