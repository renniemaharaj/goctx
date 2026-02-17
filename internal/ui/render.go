package ui

import (
	"fmt"
	"goctx/internal/model"
	"goctx/internal/patch"
	"os"
	"regexp"
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
		content := p.Files[path]
		if !utf8.ValidString(content) {
			continue
		}

		statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("FILE: %s\n", path), getTag("header"))

		oldData, err := os.ReadFile(path)
		var oldStr string
		if err != nil {
			if os.IsNotExist(err) {
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), "(NEW FILE)\n", getTag("header"))
			}
		} else {
			oldStr = string(oldData)
		}

		hunks := patch.ParseHunks(content)
		if len(hunks) > 0 {
			for _, h := range hunks {
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), "--- SURGICAL MODIFICATION ---\n", getTag("header"))

				// Show granular changes inside the block
				diffs := dmp.DiffMain(h.Search, h.Replace, false)
				diffs = dmp.DiffCleanupSemantic(diffs)

				statsBuf.Insert(statsBuf.GetEndIter(), "[CHANGES]:\n")
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

				// Check if the block actually matches what's on disk
				if oldStr != "" && !strings.Contains(oldStr, h.Search) {
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), "\n\nERROR: SEARCH block not found in target file!\n", getTag("deleted"))
				} else if oldStr != "" {
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), "\n\nREADY: Hunk match validated.\n", getTag("added"))
				}
				statsBuf.Insert(statsBuf.GetEndIter(), "\n---\n\n")
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
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag(tag))
				} else {
					statsBuf.Insert(statsBuf.GetEndIter(), d.Text)
				}
			}
			statsBuf.Insert(statsBuf.GetEndIter(), "\n\n")
		}
	}
}

func RenderFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		RenderError(err)
		return
	}

	statsBuf.SetText("")
	content := string(data)
	statsBuf.SetText(content)

	// Simple regex-based syntax highlighting for Go/JSON
	highlight(statsBuf, `\b(func|package|import|type|struct|interface|return|if|else|for|range|go|chan|select|case|default|var|const|map|switch)\b`, "keyword")
	highlight(statsBuf, `//.*`, "comment")
	highlight(statsBuf, `/\*[^*]*\*+([^/*][^*]*\*+)*/`, "comment")
	highlight(statsBuf, `".*?"`, "added") // reuse 'added' green for strings

	updateStatus(statusLabel, "Viewing: "+path)
}

func highlight(buffer *gtk.TextBuffer, pattern string, tag string) {
	re := regexp.MustCompile(pattern)
	text, _ := buffer.GetText(buffer.GetStartIter(), buffer.GetEndIter(), false)
	matches := re.FindAllStringIndex(text, -1)

	for _, m := range matches {
		start := buffer.GetIterAtOffset(m[0])
		end := buffer.GetIterAtOffset(m[1])
		buffer.ApplyTagByName(tag, start, end)
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

	tagK, _ := gtk.TextTagNew("keyword")
	tagK.SetProperty("foreground", "#c586c0")
	tab.Add(tagK)

	tagC, _ := gtk.TextTagNew("comment")
	tagC.SetProperty("foreground", "#6a9955")
	tab.Add(tagC)
}

func getTag(n string) *gtk.TextTag {
	tab, err := statsBuf.GetTagTable()
	if err != nil {
		return nil
	}
	tag, _ := tab.Lookup(n)
	return tag
}

// RenderError displays application or verification failures in the main panel
func RenderError(err error) {
	statsBuf.SetText("")
	statsBuf.InsertWithTag(statsBuf.GetEndIter(), "=== APPLICATION / VERIFICATION FAILURE ===\n\n", getTag("deleted"))

	msg := err.Error()
	// If the error contains build/test output with newlines, it will be preserved here
	statsBuf.Insert(statsBuf.GetEndIter(), msg+"\n")

	// Apply syntax highlighting to the error output to help identify issues
	highlight(statsBuf, `(?i)error:.*`, "deleted")
	highlight(statsBuf, `(?i)failed:.*`, "deleted")
	highlight(statsBuf, `line \d+`, "header")
	highlight(statsBuf, `\./.*\.go:\d+:\d+`, "header")

	updateStatus(statusLabel, "Error details rendered to panel")
}
