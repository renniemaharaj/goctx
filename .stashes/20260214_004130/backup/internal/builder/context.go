package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goctx/internal/model"
)

func defaultConfig() model.Config {
	return model.Config{
		Ignore:     []string{".git", ".stashes"},
		Extensions: []string{".go", ".json", ".md"},
	}
}

func BuildContext(root string) (model.ProjectOutput, error) {
	cfg := defaultConfig()

	output := model.ProjectOutput{
		Files: make(map[string]string),
	}

	var tree strings.Builder
	totalChars := 0

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		for _, ig := range cfg.Ignore {
			if strings.Contains(path, ig) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		depth := strings.Count(path, string(os.PathSeparator))
		indent := ""
		if depth > 0 {
			indent = strings.Repeat("  ", depth) + "└── "
		}
		tree.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name()))

		if !d.IsDir() && allowed(path, cfg.Extensions) {
			content, err := os.ReadFile(path)
			if err == nil {
				str := string(content)
				output.Files[path] = str
				totalChars += len(str)
			}
		}
		return nil
	})

	if err != nil {
		return output, err
	}

	output.ProjectTree = tree.String()
	output.EstimatedTokens = (totalChars + len(output.ProjectTree)) / 4

	return output, nil
}

func allowed(path string, exts []string) bool {
	ext := filepath.Ext(path)
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}
