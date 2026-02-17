package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goctx/internal/config"
	"goctx/internal/model"
	"goctx/internal/patch"
	"goctx/internal/runner"
	"goctx/internal/stash"
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
			return fmt.Errorf("PATCH_ERROR: Critical failure at %s: %w", path, applyErr)
		}
	}

	// Verification Phase: Run Build & Test scripts if configured
	cfg, _ := config.Load(root)

	if cfg.Scripts.Build != "" {
		if out, err := runner.Run(root, cfg.Scripts.Build); err != nil {
			stash.Push(root, fmt.Sprintf("Auto-stash: Build Failed - %s", input.ShortDescription))
			return fmt.Errorf("BUILD_FAILURE: Verification failed for '%s'\n\nOutput:\n%s", cfg.Scripts.Build, string(out))
		}
	}

	if cfg.Scripts.Test != "" {
		if out, err := runner.Run(root, cfg.Scripts.Test); err != nil {
			stash.Push(root, fmt.Sprintf("Auto-stash: Tests Failed - %s", input.ShortDescription))
			return fmt.Errorf("TEST_FAILURE: Verification failed for '%s'\n\nOutput:\n%s", cfg.Scripts.Test, string(out))
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
