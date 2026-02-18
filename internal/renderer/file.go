package renderer

import (
	"os"
	"strings"
)

func (r *Renderer) RenderFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		r.RenderError(err)
		return
	}

	*r.isLoading = true
	defer func() { *r.isLoading = false }()

	r.statsBuf.SetText("")
	content := string(data)
	r.statsBuf.SetText(content)

	// Simple regex-based syntax highlighting for Go/JSON/Ignore
	highlight(r.statsBuf, `\b(func|package|import|type|struct|interface|return|if|else|for|range|go|chan|select|case|default|var|const|map|switch)\b`, "keyword")
	highlight(r.statsBuf, `//.*`, "comment")
	highlight(r.statsBuf, `^#.*`, "comment") // Shell/Ignore comments
	highlight(r.statsBuf, `/\*[^*]*\*+([^/*][^*]*\*+)*/`, "comment")
	highlight(r.statsBuf, `".*?"`, "added")

	if strings.HasSuffix(path, ".ctxignore") {
		highlight(r.statsBuf, `^[^#\s]+`, "header") // Highlight ignore patterns
	}

	r.updateStatus(r.statusLabel, "Viewing: "+path)
}
