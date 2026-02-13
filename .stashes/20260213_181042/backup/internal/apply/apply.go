package apply

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"goctx/internal/model"
	"goctx/internal/stash"
)

func ApplyPatch(root string, input model.ProjectOutput) error {
	if len(input.Files) == 0 {
		return errors.New("no files to apply")
	}

	stashID, err := stash.CreateStash(root, input)
	if err != nil {
		return err
	}

	for path, content := range input.Files {
		if !safePath(root, path) {
			continue
		}

		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(content), 0644)
	}

	return stash.MarkApplied(root, stashID)
}

func safePath(root, path string) bool {
	abs, _ := filepath.Abs(path)
	rootAbs, _ := filepath.Abs(root)
	return strings.HasPrefix(abs, rootAbs)
}
