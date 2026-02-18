package renderer

import (
	"fmt"
	"goctx/internal/git"
	"goctx/internal/model"
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

func (r *Renderer) RenderSummary(p model.ProjectOutput) {
	*r.isLoading = true
	defer func() { *r.isLoading = false }()

	r.statsBuf.SetText("")
	r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), "=== CONTEXT BUILD SUMMARY ===\n\n", r.GetTag("header"))

	r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), "Status: ", r.GetTag("header"))
	r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), "SUCCESS\n", r.GetTag("added"))

	stats := []struct {
		Label string
		Value interface{}
	}{
		{"Files Included", p.FileCount},
		{"Directories Scanned", p.DirCount},
		{"Estimated Tokens", p.TokenCount},
		{"Total Characters", p.TokenCount * 4},
	}

	for _, s := range stats {
		r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), fmt.Sprintf("%-20s: ", s.Label), r.GetTag("header"))
		r.statsBuf.Insert(r.statsBuf.GetEndIter(), fmt.Sprintf("%v\n", s.Value))
	}

	r.statsBuf.Insert(r.statsBuf.GetEndIter(), "\nPROJECT TREE:\n")
	r.statsBuf.Insert(r.statsBuf.GetEndIter(), p.ProjectTree)
	r.updateStatus(r.statusLabel, fmt.Sprintf("Build Success: %d files / ~%dk tokens", p.FileCount, p.TokenCount/1000))
}
