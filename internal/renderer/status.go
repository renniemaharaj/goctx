package renderer

import (
	"fmt"
	"goctx/internal/git"
)

func (r *Renderer) RenderGitStatus(root string) {

	files, err := git.GetStatusFiles(root)
	if err != nil {
		r.RenderError(err)
		return
	}

	*r.isLoading = true
	defer func() { *r.isLoading = false }()

	r.statsBuf.SetText("")
	r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), "=== WORKSPACE MODIFICATIONS ===\n\n", r.GetTag("header"))

	if len(files) == 0 {
		r.statsBuf.Insert(r.statsBuf.GetEndIter(), "No changes detected.\n")
		return
	}

	for _, f := range files {
		r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), "  [Modified] ", r.GetTag("added"))
		r.statsBuf.Insert(r.statsBuf.GetEndIter(), f+"\n")
	}

	r.statsBuf.Insert(r.statsBuf.GetEndIter(), fmt.Sprintf("\nTotal modified: %d\n", len(files)))
}
