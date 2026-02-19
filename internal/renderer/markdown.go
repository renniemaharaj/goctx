package renderer

import "regexp"

func (r *Renderer) RenderMarkdown(text string) {
	*r.isLoading = true
	defer func() { *r.isLoading = false }()

	r.statsBuf.SetText(text)

	// Highlight Code Blocks
	reCode := regexp.MustCompile("(?s)```[a-zA-Z]*\\n(.*?)\\n```")
	matches := reCode.FindAllStringSubmatchIndex(text, -1)

	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		// m[0], m[1] is the whole block including backticks
		start := r.statsBuf.GetIterAtOffset(m[0])
		end := r.statsBuf.GetIterAtOffset(m[1])
		r.statsBuf.ApplyTagByName("added", start, end)

		// Interior content (the actual code)
		cStart := r.statsBuf.GetIterAtOffset(m[2])
		cEnd := r.statsBuf.GetIterAtOffset(m[3])
		r.statsBuf.ApplyTagByName("keyword", cStart, cEnd)
	}

	// Highlight inline code
	highlight(r.statsBuf, "`[^`]+`", "header")
}
