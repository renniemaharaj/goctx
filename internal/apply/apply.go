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

type ProgressFunc func(phase, desc string, logLine string)

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

func ApplyPatch(root string, input model.ProjectOutput, onProgress ProgressFunc) error {
	if len(input.Files) == 0 {
		return fmt.Errorf("no files to apply")
	}

	if onProgress != nil {
		onProgress("Applying", "Modifying workspace files...", "")
	}

	for path, content := range input.Files {
		if !safePath(root, path) {
			continue
		}

		os.MkdirAll(filepath.Join(root, filepath.Dir(path)), 0755)

		var applyErr error
		if strings.Contains(content, "<<<<<< SEARCH") && strings.Contains(content, ">>>>>> REPLACE") {
			hunks := patch.ParseHunks(content)
			if len(hunks) > 0 {
				applyErr = applySurgicalEdit(filepath.Join(root, path), hunks)
			} else {
				applyErr = fmt.Errorf("surgical markers found but failed to parse hunks in %s", path)
			}
		} else {
			applyErr = os.WriteFile(filepath.Join(root, path), []byte(content), 0644)
		}

		if applyErr != nil {
			return fmt.Errorf("PATCH_ERROR: Critical failure at %s: %w", path, applyErr)
		}
	}

	cfg, _ := config.Load(root)

	if cfg.Scripts.Build != "" {
		if onProgress != nil {
			onProgress("Building", fmt.Sprintf("Running: %s", cfg.Scripts.Build), "")
		}
		if out, err := runner.Run(root, cfg.Scripts.Build, func(line string) {
			if onProgress != nil {
				onProgress("Building", "", line)
			}
		}); err != nil {
			stash.Push(root, fmt.Sprintf("Auto-stash: Build Failed - %s", input.ShortDescription))
			return fmt.Errorf("BUILD_FAILURE: Verification failed for '%s'\n\nOutput:\n%s", cfg.Scripts.Build, string(out))
		}
	}

	if cfg.Scripts.Test != "" {
		if onProgress != nil {
			onProgress("Testing", fmt.Sprintf("Running: %s", cfg.Scripts.Test), "")
		}
		if out, err := runner.Run(root, cfg.Scripts.Test, func(line string) {
			if onProgress != nil {
				onProgress("Testing", "", line)
			}
		}); err != nil {
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
