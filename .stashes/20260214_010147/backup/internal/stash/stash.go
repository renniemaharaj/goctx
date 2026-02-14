package stash

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"goctx/internal/model"
)

func CreateStash(root string, patch model.ProjectOutput) (string, error) {
	id := time.Now().Format("20060102_150405")
	base := filepath.Join(root, ".stashes", id)

	os.MkdirAll(filepath.Join(base, "backup"), 0755)

	for path := range patch.Files {
		if _, err := os.Stat(path); err == nil {
			data, _ := os.ReadFile(path)
			bpath := filepath.Join(base, "backup", path)
			os.MkdirAll(filepath.Dir(bpath), 0755)
			os.WriteFile(bpath, data, 0644)
		}
	}

	patchData, _ := json.MarshalIndent(patch, "", "  ")
	os.WriteFile(filepath.Join(base, "patch.json"), patchData, 0644)

	return id, nil
}

func MarkApplied(root, id string) error {
	meta := map[string]string{
		"applied":   "true",
		"timestamp": time.Now().String(),
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	return os.WriteFile(filepath.Join(root, ".stashes", id, "meta.json"), data, 0644)
}
