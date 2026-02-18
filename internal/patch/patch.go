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
	// Edge Case: Empty search string would match everywhere/nowhere meaningfully
	if hunk.Search == "" {
		return fileStr, false
	}

	// 1. High-Integrity Exact Match
	if strings.Contains(fileStr, hunk.Search) {
		return strings.Replace(fileStr, hunk.Search, hunk.Replace, 1), true
	}

	// 2. Resilient Fuzzy Match (Whitespace insensitive per line)
	// Normalize line endings to LF for processing
	normalize := func(s string) string {
		return strings.ReplaceAll(s, "\r\n", "\n")
	}

	fNorm := normalize(fileStr)
	sNorm := normalize(hunk.Search)

	// Helper to split into lines while treating a single trailing newline as a line terminator,
	// not a precursor to a new empty line. This ensures "A\n" matches "A" in line-by-line mode.
	splitLines := func(s string) []string {
		if s == "" {
			return []string{}
		}
		return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	}

	fileLines := splitLines(fNorm)
	searchLines := splitLines(sNorm)

	// Guard against empty search or search longer than file
	if len(searchLines) == 0 || len(searchLines) > len(fileLines) {
		return fileStr, false
	}

	for i := 0; i <= len(fileLines)-len(searchLines); i++ {
		match := true
		for j := 0; j < len(searchLines); j++ {
			// Business Logic: Ignore leading/trailing whitespace variations introduced by LLM formatting
			if strings.TrimSpace(fileLines[i+j]) != strings.TrimSpace(searchLines[j]) {
				match = false
				break
			}
		}

		if match {
			// Construct result: [Lines before] + [New Content] + [Lines after]
			head := fileLines[:i]
			tail := fileLines[i+len(searchLines):]

			var parts []string
			if len(head) > 0 {
				parts = append(parts, strings.Join(head, "\n"))
			}
			parts = append(parts, hunk.Replace)
			if len(tail) > 0 {
				parts = append(parts, strings.Join(tail, "\n"))
			}

			result := strings.Join(parts, "\n")

			// Integrity Check: If original file ended in a newline, preserve that
			// property unless the replacement explicitly handles the end of file.
			if strings.HasSuffix(fNorm, "\n") && !strings.HasSuffix(result, "\n") {
				result += "\n"
			}

			return result, true
		}
	}

	return fileStr, false
}
