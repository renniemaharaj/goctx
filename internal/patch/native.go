package patch

import (
	"fmt"
	"goctx/internal/model"
	"regexp"
	"strings"
)

// ParseNative detects and parses a custom "Native Dialect" for patches.
// It extracts file blocks defined by "filename": header followed by standard SEARCH/REPLACE hunks.
func ParseNative(text string) (model.ProjectOutput, bool) {
	// Regex matches lines starting with a quoted string and a colon, e.g. "path/file.go":
	re := regexp.MustCompile(`(?m)^"([^"]+)":`)

	matches := re.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return model.ProjectOutput{}, false
	}

	out := model.ProjectOutput{
		Files: make(map[string]string),
	}
	var filesFound []string

	for i, match := range matches {
		// Group 1 is the filename (indices 2 and 3)
		filename := text[match[2]:match[3]]

		// Content starts after the current match (colon)
		start := match[1]
		end := len(text)

		// Content ends at the start of the next file match
		if i < len(matches)-1 {
			end = matches[i+1][0]
		}

		// Extract the block - this is the content between this file header and the next header (or EOF)
		var content string
		if start < end {
			content = text[start:end]
		}
		trimmedContent := strings.TrimSpace(content)

		// Validate: Accept if any of these conditions are met:
		// 1. Has SEARCH/REPLACE markers (surgical edit)
		// 2. Is empty/whitespace-only (file deletion - moving to trash)
		// 3. Has non-marker content (new file or complete file replacement)
		hasSurgicalMarkers := strings.Contains(content, "<<<<<< SEARCH") && strings.Contains(content, "======")
		isEmpty := trimmedContent == ""
		hasContent := !isEmpty

		if hasSurgicalMarkers || isEmpty || hasContent {
			out.Files[filename] = trimmedContent
			filesFound = append(filesFound, filename)
		}
	}

	if len(out.Files) == 0 {
		return model.ProjectOutput{}, false
	}

	if len(filesFound) == 1 {
		out.ShortDescription = fmt.Sprintf("Update to %s", filesFound[0])
	} else {
		out.ShortDescription = fmt.Sprintf("Updates to %d files", len(filesFound))
	}

	return out, true
}
