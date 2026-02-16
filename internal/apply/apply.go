package apply

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"goctx/internal/model"
	"goctx/internal/patch"
)

func ApplyHunksToString(original string, hunks []patch.Hunk) (string, error) {
	result := original
	for _, h := range hunks {
		newStr, ok := patch.ApplyHunk(result, h)
		if !ok {
			return original, fmt.Errorf("hunk match failed")
		}
		result = newStr
	}
	return result, nil
}

func ApplyPatch(root string, input model.ProjectOutput) error {
	if len(input.Files) == 0 {
		return fmt.Errorf("no files to apply")
	}

	// _, err := stash.CreateStash(root, input)
	// if err != nil {
	// 	return err
	// }

	for path, content := range input.Files {
		if !safePath(root, path) {
			continue
		}

		os.MkdirAll(filepath.Dir(path), 0755)

		var applyErr error
		// Only attempt surgical apply if the content contains the specific markers
		if strings.Contains(content, "<<<<<< SEARCH") && strings.Contains(content, ">>>>>> REPLACE") {
			hunks := patch.ParseHunks(content)
			if len(hunks) > 0 {
				applyErr = applySurgicalEdit(path, hunks)
			} else {
				applyErr = fmt.Errorf("surgical markers found but failed to parse hunks in %s", path)
			}
		} else {
			// Standard full file overwrite
			applyErr = os.WriteFile(path, []byte(content), 0644)
		}

		if applyErr != nil {
			// High-integrity rollback: restore from the stash created at the start of this operation
			_ = exec.Command("git", "stash", "pop", "--index").Run()
			return fmt.Errorf("critical failure at %s; workspace rolled back: %w", path, applyErr)
		}
	}

	return nil
}
func applySurgicalEdit(path string, hunks []patch.Hunk) error {
	originalData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}

	newStr, err := ApplyHunksToString(string(originalData), hunks)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(newStr), 0644)
}

func safePath(root, path string) bool {
	abs, _ := filepath.Abs(path)
	rootAbs, _ := filepath.Abs(root)
	return strings.HasPrefix(abs, rootAbs)
}
