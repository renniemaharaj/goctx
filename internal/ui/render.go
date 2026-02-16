package ui

import (
	"fmt"
	"goctx/internal/model"
	"goctx/internal/patch"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gotk3/gotk3/gtk"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func renderDiff(p model.ProjectOutput, title string) {
	statsBuf.SetText("")
	statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("=== %s ===\n\n", strings.ToUpper(title)), getTag("header"))

	if p.ProjectTree != "" {
		statsBuf.Insert(statsBuf.GetEndIter(), "PROJECT STRUCTURE:\n")
		statsBuf.Insert(statsBuf.GetEndIter(), p.ProjectTree+"\n")
		statsBuf.Insert(statsBuf.GetEndIter(), "---\n\n")
	}

	dmp := diffmatchpatch.New()
	var keys []string
	for k := range p.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, path := range keys {
		newContent := p.Files[path]
		if !utf8.ValidString(newContent) {
			continue
		}

		statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("FILE: %s\n", path), getTag("header"))

		old, err := os.ReadFile(path)
		var oldStr string
		if err != nil && os.IsNotExist(err) {
			statsBuf.InsertWithTag(statsBuf.GetEndIter(), "(NEW FILE)\n", getTag("header"))
		} else {
			oldStr = string(old)
		}

		hunks := patch.ParseHunks(newContent)
		if len(hunks) > 0 {
			for _, h := range hunks {
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), "--- SURGICAL MODIFICATION ---\n", getTag("header"))

				// Diff the Search vs Replace blocks to show granular changes
				blockDiffs := dmp.DiffMain(h.Search, h.Replace, false)
				blockDiffs = dmp.DiffCleanupSemantic(blockDiffs)

				statsBuf.Insert(statsBuf.GetEndIter(), "[CHANGES]:\n")
				for _, d := range blockDiffs {
					tag := ""
					switch d.Type {
					case diffmatchpatch.DiffInsert:
						tag = "added"
					case diffmatchpatch.DiffDelete:
						tag = "deleted"
					}
					if tag != "" {
						statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag(tag))
					} else {
						statsBuf.Insert(statsBuf.GetEndIter(), d.Text)
					}
				}
				statsBuf.Insert(statsBuf.GetEndIter(), "\n")

				_, ok := patch.ApplyHunk(oldStr, h)
				if !ok {
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), "\nERROR: Match not found!\n", getTag("header"))
				} else {
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), "\nREADY: Hunk match validated.\n", getTag("added"))
				}
				statsBuf.Insert(statsBuf.GetEndIter(), "\n---\n\n")
			}
		} else {
			diffs := dmp.DiffMain(oldStr, newContent, false)
			for _, d := range diffs {
				tag := ""
				switch d.Type {
				case diffmatchpatch.DiffInsert:
					tag = "added"
				case diffmatchpatch.DiffDelete:
					tag = "deleted"
				}
				if tag != "" {
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag(tag))
				} else {
					statsBuf.Insert(statsBuf.GetEndIter(), d.Text)
				}
			}
			statsBuf.Insert(statsBuf.GetEndIter(), "\n\n")
		}
	}
}

func setupTags(buffer *gtk.TextBuffer) {
	tab, _ := buffer.GetTagTable()
	tagA, _ := gtk.TextTagNew("added")
	tagA.SetProperty("background", "#1e3a1e")
	tagA.SetProperty("foreground", "#afffbc")
	tab.Add(tagA)
	tagD, _ := gtk.TextTagNew("deleted")
	tagD.SetProperty("background", "#4b1818")
	tagD.SetProperty("foreground", "#ffa1a1")
	tab.Add(tagD)
	tagH, _ := gtk.TextTagNew("header")
	tagH.SetProperty("weight", 700)
	tagH.SetProperty("foreground", "#569cd6")
	tab.Add(tagH)
}

func getTag(n string) *gtk.TextTag {
	tab, err := statsBuf.GetTagTable()
	if err != nil {
		return nil
	}
	tag, _ := tab.Lookup(n)
	return tag
}
