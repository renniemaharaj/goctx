package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goctx/internal/model"
	"goctx/internal/stash"
)

func ApplyPatch(root string, input model.ProjectOutput) error {
	if len(input.Files) == 0 {
		return fmt.Errorf("no files to apply")
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

		// DIFFERENTIATION LOGIC:
		// If the content contains the surgical markers, we process it as a patch.
		// Otherwise, we treat it as a full file overwrite.
		if strings.Contains(content, "<<<<<< SEARCH") && strings.Contains(content, "======") {
			err = applySurgicalEdit(path, content)
		} else {
			err = os.WriteFile(path, []byte(content), 0644)
		}

		if err != nil {
			return fmt.Errorf("failed at %s: %w", path, err)
		}
	}

	return stash.MarkApplied(root, stashID)
}

func applySurgicalEdit(path, patchContent string) error {
	originalData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file: %w", err)
	}
	fileStr := string(originalData)

	// MULTI-HUNK SUPPORT:
	// We split the patchContent into individual hunks and apply them sequentially.
	hunks := strings.Split(patchContent, ">>>>>> REPLACE")
	for _, hunk := range hunks {
		hunk = strings.TrimSpace(hunk)
		if hunk == "" {
			continue
		}

		searchMarker := "<<<<<< SEARCH\n"
		dividerMarker := "\n======\n"

		sStart := strings.Index(hunk, searchMarker)
		div := strings.Index(hunk, dividerMarker)

		if sStart == -1 || div == -1 {
			continue // Skip malformed hunks or trailing text
		}

		searchBlock := strings.Trim(hunk[sStart+len(searchMarker):div], "\n\r")
		replaceBlock := strings.Trim(hunk[div+len(dividerMarker):], "\n\r")

		// 1. Try EXACT MATCH
		if strings.Contains(fileStr, searchBlock) {
			fileStr = strings.Replace(fileStr, searchBlock, replaceBlock, 1)
			continue
		}

		// 2. FUZZY MATCH (Indentation-Agnostic)
		fileLines := strings.Split(fileStr, "\n")
		searchLines := strings.Split(strings.TrimSpace(searchBlock), "\n")
		foundIdx := -1

		for i := 0; i <= len(fileLines)-len(searchLines); i++ {
			match := true
			for j := 0; j < len(searchLines); j++ {
				if strings.TrimSpace(fileLines[i+j]) != strings.TrimSpace(searchLines[j]) {
					match = false
					break
				}
			}
			if match {
				foundIdx = i
				break
			}
		}

		if foundIdx != -1 {
			head := append([]string{}, fileLines[:foundIdx]...)
			tail := fileLines[foundIdx+len(searchLines):]
			fileStr = strings.Join(append(append(head, replaceBlock), tail...), "\n")
		} else {
			return fmt.Errorf("hunk match failed: block not found in %s", path)
		}
	}

	return os.WriteFile(path, []byte(fileStr), 0644)
}

func safePath(root, path string) bool {
	abs, _ := filepath.Abs(path)
	rootAbs, _ := filepath.Abs(root)
	return strings.HasPrefix(abs, rootAbs)
}
