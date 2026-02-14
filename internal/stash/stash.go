package stash

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"goctx/internal/model"
)

type StashRefs struct {
	CurrentID string   `json:"current_id"`
	History   []string `json:"history"`
}

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

	updateRefs(root, id)
	return id, nil
}

func updateRefs(root, id string) {
	refs := loadRefs(root)
	refs.CurrentID = id
	found := false
	for _, h := range refs.History {
		if h == id {
			found = true
			break
		}
	}
	if !found {
		refs.History = append(refs.History, id)
		sort.Strings(refs.History)
	}
	saveRefs(root, refs)
}

func loadRefs(root string) StashRefs {
	var r StashRefs
	b, _ := os.ReadFile(filepath.Join(root, ".stashes", "refs.json"))
	json.Unmarshal(b, &r)
	return r
}

func saveRefs(root string, r StashRefs) {
	b, _ := json.MarshalIndent(r, "", "  ")
	os.WriteFile(filepath.Join(root, ".stashes", "refs.json"), b, 0644)
}

func MarkApplied(root, id string) error {
	meta := map[string]string{"applied": "true", "timestamp": time.Now().String()}
	data, _ := json.MarshalIndent(meta, "", "  ")
	updateRefs(root, id)
	return os.WriteFile(filepath.Join(root, ".stashes", id, "meta.json"), data, 0644)
}

func GetActiveID(root string) string {
	return loadRefs(root).CurrentID
}

func DeleteStash(root, id string) error {
	base := filepath.Join(root, ".stashes", id)
	return os.RemoveAll(base)
}
