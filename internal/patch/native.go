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

		// Extract the block
		if start >= end {
			continue
		}
		content := text[start:end]

		// Validate: Must contain at least one SEARCH marker to be considered a patch
		if strings.Contains(content, "<<<<<< SEARCH") && strings.Contains(content, "======") {
			// Trim leading whitespace (like space after colon) but preserve internal indentation
			out.Files[filename] = strings.TrimSpace(content)
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
