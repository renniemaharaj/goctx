package patch

import (
	"strings"
)

type Hunk struct {
	Search  string
	Replace string
}

func ParseHunks(content string) []Hunk {
	if !strings.Contains(content, "<<<<<< SEARCH") || !strings.Contains(content, "======") {
		return nil
	}

	var hunks []Hunk
	parts := strings.Split(content, ">>>>>> REPLACE")

	for _, p := range parts {
		sIdx := strings.Index(p, "<<<<<< SEARCH")
		dIdx := strings.Index(p, "======")

		if sIdx != -1 && dIdx != -1 && dIdx > (sIdx+13) {
			// Trim only the single leading/trailing newline caused by the markers
			search := strings.TrimPrefix(p[sIdx+13:dIdx], "\n")
			search = strings.TrimSuffix(search, "\n")

			replace := strings.TrimPrefix(p[dIdx+6:], "\n")
			replace = strings.TrimSuffix(replace, "\n")

			hunks = append(hunks, Hunk{
				Search:  search,
				Replace: replace,
			})
		}
	}
	return hunks
}

func ApplyHunk(fileStr string, hunk Hunk) (string, bool) {
	if strings.Contains(fileStr, hunk.Search) {
		return strings.Replace(fileStr, hunk.Search, hunk.Replace, 1), true
	}

	fileLines := strings.Split(fileStr, "\n")
	searchLines := strings.Split(hunk.Search, "\n")

	for i := 0; i <= len(fileLines)-len(searchLines); i++ {
		match := true
		for j := 0; j < len(searchLines); j++ {
			if strings.TrimSpace(fileLines[i+j]) != strings.TrimSpace(searchLines[j]) {
				match = false
				break
			}
		}
		if match {
			head := append([]string{}, fileLines[:i]...)
			tail := fileLines[i+len(searchLines):]
			return strings.Join(append(append(head, hunk.Replace), tail...), "\n"), true
		}
	}
	return fileStr, false
}
