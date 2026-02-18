package renderer

import (
	"fmt"
	"goctx/internal/model"
	"goctx/internal/patch"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func (r *Renderer) RenderDiff(p model.ProjectOutput, title string) {
	*r.isLoading = true
	defer func() { *r.isLoading = false }()

	r.statsBuf.SetText("")
	r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), fmt.Sprintf("=== %s ===\n\n", strings.ToUpper(title)), r.GetTag("header"))

	// if p.ProjectTree != "" {
	// 	r.statsBuf.Insert(r.statsBuf.GetEndIter(), "PROJECT STRUCTURE:\n")
	// 	r.statsBuf.Insert(r.statsBuf.GetEndIter(), p.ProjectTree+"\n")
	// 	r.statsBuf.Insert(r.statsBuf.GetEndIter(), "---\n\n")
	// }

	dmp := diffmatchpatch.New()
	var keys []string
	for k := range p.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, path := range keys {
		if i >= 20 {
			continue
		}
		content := p.Files[path]
		if !utf8.ValidString(content) {
			continue
		}

		r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), fmt.Sprintf("FILE: %s\n", path), r.GetTag("header"))

		oldData, err := os.ReadFile(path)
		var oldStr string
		if err != nil {
			if os.IsNotExist(err) {
				r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), "(NEW FILE)\n", r.GetTag("header"))
			}
		} else {
			oldStr = string(oldData)
		}

		hunks := patch.ParseHunks(content)
		if len(hunks) > 0 {
			for _, h := range hunks {
				r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), "--- SURGICAL MODIFICATION ---\n", r.GetTag("header"))

				// Show granular changes inside the block
				diffs := dmp.DiffMain(h.Search, h.Replace, false)
				diffs = dmp.DiffCleanupSemantic(diffs)

				r.statsBuf.Insert(r.statsBuf.GetEndIter(), "[CHANGES]:\n")
				for _, d := range diffs {
					tag := ""
					switch d.Type {
					case diffmatchpatch.DiffInsert:
						tag = "added"
					case diffmatchpatch.DiffDelete:
						tag = "deleted"
					}
					if tag != "" {
						r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), d.Text, r.GetTag(tag))
					} else {
						r.statsBuf.Insert(r.statsBuf.GetEndIter(), d.Text)
					}
				}

				// Check if the block actually matches what's on disk
				if oldStr != "" && !strings.Contains(oldStr, h.Search) {
					r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), "\n\nERROR: SEARCH block not found in target file!\n", r.GetTag("deleted"))
				} else if oldStr != "" {
					r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), "\n\nREADY: Hunk match validated.\n", r.GetTag("added"))
				}
				r.statsBuf.Insert(r.statsBuf.GetEndIter(), "\n---\n\n")
			}
		} else {
			// standard diff for full files
			diffs := dmp.DiffMain(oldStr, content, false)
			diffs = dmp.DiffCleanupSemantic(diffs)
			for _, d := range diffs {
				tag := ""
				switch d.Type {
				case diffmatchpatch.DiffInsert:
					tag = "added"
				case diffmatchpatch.DiffDelete:
					tag = "deleted"
				}
				if tag != "" {
					r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), d.Text, r.GetTag(tag))
				} else {
					r.statsBuf.Insert(r.statsBuf.GetEndIter(), d.Text)
				}
			}
			r.statsBuf.Insert(r.statsBuf.GetEndIter(), "\n\n")
		}
	}
}
