package builder

import (
	"goctx/internal/model"
	"os"
	"path/filepath"
	"strings"
)

// BuildSelectiveContext only includes files that match a certain criteria or recent changes
func BuildSelectiveContext(root string, targetFiles []string) (model.ProjectOutput, error) {
	output := model.ProjectOutput{
		Files: make(map[string]string),
	}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.HasPrefix(path, ".") {
			return nil
		}

		rel, _ := filepath.Rel(root, path)
		
		// If targetFiles is empty, we act like a full build, 
		// otherwise we only include what is requested.
		include := len(targetFiles) == 0
		for _, t := range targetFiles {
			if strings.Contains(rel, t) {
				include = true
				break
			}
		}

		if include {
			content, _ := os.ReadFile(path)
			output.Files[rel] = string(content)
		}
		return nil
	})

	return output, nil
}