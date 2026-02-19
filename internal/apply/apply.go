package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"goctx/internal/config"
	"goctx/internal/model"
	"goctx/internal/patch"
	"goctx/internal/runner"
	"goctx/internal/stash"
)

type ProgressFunc func(phase, desc string, logLine string)

const trashDir = ".trash"

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

// isFileDeletion detects if content represents a file deletion (empty or whitespace only)
func isFileDeletion(content string) bool {
	return strings.TrimSpace(content) == ""
}

// moveToTrash moves a file to the .trash directory instead of permanently deleting it
func moveToTrash(root, filePath string) error {
	targetPath := filepath.Join(root, filePath)

	// Check if file exists before attempting to move
	_, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, nothing to trash
			return nil
		}
		return fmt.Errorf("could not stat file: %w", err)
	}

	// Create trash directory if it doesn't exist
	trashPath := filepath.Join(root, trashDir)
	if err := os.MkdirAll(trashPath, 0755); err != nil {
		return fmt.Errorf("could not create trash directory: %w", err)
	}

	// Generate unique trash filename to avoid collisions
	baseFileName := filepath.Base(filePath)
	timestamp := time.Now().Format("20060102-150405")
	trashFileName := fmt.Sprintf("%s_%s", timestamp, baseFileName)
	trashedPath := filepath.Join(trashPath, trashFileName)

	// Move file to trash
	if err := os.Rename(targetPath, trashedPath); err != nil {
		return fmt.Errorf("could not move file to trash: %w", err)
	}

	return nil
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

		// Check if this is a deletion (empty/whitespace-only content)
		if isFileDeletion(content) {
			if onProgress != nil {
				onProgress("", fmt.Sprintf("Trashing: %s", path), "")
			}
			if err := moveToTrash(root, path); err != nil {
				return fmt.Errorf("TRASH_ERROR: Could not trash file %s: %w", path, err)
			}
			continue
		}

		targetPath := filepath.Join(root, path)
		os.MkdirAll(filepath.Dir(targetPath), 0755)

		var applyErr error
		if strings.Contains(content, "<<<<<< SEARCH") && strings.Contains(content, ">>>>>> REPLACE") {
			hunks := patch.ParseHunks(content)
			if len(hunks) > 0 {
				applyErr = applySurgicalEdit(targetPath, hunks)
			} else {
				applyErr = fmt.Errorf("surgical markers found but failed to parse hunks in %s", path)
			}
		} else {
			applyErr = os.WriteFile(targetPath, []byte(content), 0644)
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
