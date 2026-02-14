package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/builder"
	"goctx/internal/model"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const AI_PROMPT_WRAPPER = `SYSTEM: You are a GoCtx AI agent. You have access to the project state below.\nTo apply changes, output a SINGLE JSON code block.\n`

var (
	activeContext  model.ProjectOutput
	lastClipboard  string
	statsBuf       *gtk.TextBuffer
	stashList      *gtk.ListBox
	pendingList    *gtk.ListBox
	pendingPatches []model.ProjectOutput
	selectedStash  model.ProjectOutput
	win            *gtk.Window
	statusLabel    *gtk.Label
	btnApplyPatch  *gtk.Button
	btnApplyStash  *gtk.Button
)

func Run() {
	gtk.Init(nil)
	win, _ = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Manager")
	win.SetDefaultSize(1400, 950)
	win.Connect("destroy", gtk.MainQuit)

	vmain, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	hmain, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	leftBar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 15)
	leftBar.SetMarginStart(15)
	leftBar.SetMarginEnd(15)
	leftBar.SetMarginTop(15)
	leftBar.SetSizeRequest(320, -1)

	btnBuild := newBtn("CURRENT CONTEXT")
	btnCopy := newBtn("COPY CONTEXT")
	btnApplyPatch = newBtn("APPLY SELECTED PATCH")
	btnApplyStash = newBtn("APPLY SELECTED STASH")
	btnApplyPatch.SetSensitive(false)
	btnApplyStash.SetSensitive(false)

	leftBar.PackStart(btnBuild, false, false, 0)
	leftBar.PackStart(btnCopy, false, false, 0)
	leftBar.PackStart(btnApplyPatch, false, false, 0)
	leftBar.PackStart(btnApplyStash, false, false, 0)

	label(leftBar, "PENDING PATCHES")
	pendingList, _ = gtk.ListBoxNew()
	swPending, _ := gtk.ScrolledWindowNew(nil, nil)
	swPending.SetSizeRequest(-1, 200)
	swPending.Add(pendingList)
	leftBar.PackStart(swPending, false, false, 0)

	label(leftBar, "STASHES")
	stashList, _ = gtk.ListBoxNew()
	swStash, _ := gtk.ScrolledWindowNew(nil, nil)
	swStash.Add(stashList)
	leftBar.PackStart(swStash, true, true, 0)

	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	rightStack.SetMarginStart(20)
	rightStack.SetMarginEnd(20)
	rightStack.SetMarginTop(15)

	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ := gtk.TextViewNew()
	statsView.SetMonospace(true)
	statsView.SetEditable(false)
	statsBuf, _ = statsView.GetBuffer()
	statsScroll.Add(statsView)
	rightStack.PackStart(statsScroll, true, true, 0)

	setupTags(statsBuf)
	statusLabel, _ = gtk.LabelNew("Ready")

	hmain.PackStart(leftBar, false, false, 0)
	hmain.PackStart(rightStack, true, true, 0)
	vmain.PackStart(hmain, true, true, 0)
	vmain.PackStart(statusLabel, false, false, 5)

	btnBuild.Connect("clicked", func() {
		updateStatus("Building context...")
		go func() {
			root, _ := filepath.Abs(".")
			out, err := builder.BuildSelectiveContext(root, "Manual Build")
			glib.IdleAdd(func() {
				if err != nil {
					updateStatus("Error: " + err.Error())
					return
				}
				activeContext = out
				renderDiff(activeContext, "Current Workspace")
				updateStatus("Context built successfully")
			})
		}()
	})

	btnCopy.Connect("clicked", func() {
		fullPrompt := AI_PROMPT_WRAPPER + string(mustMarshal(activeContext))
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(fullPrompt)
		updateStatus("Prompt copied to clipboard")
	})

	pendingList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		idx := row.GetIndex()
		renderDiff(pendingPatches[idx], "Pending Patch")
		btnApplyPatch.SetSensitive(true)
	})

	go func() {
		for {
			time.Sleep(1 * time.Second)
			glib.IdleAdd(func() {
				clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
				text, _ := clip.WaitForText()
				if text != "" && text != lastClipboard {
					lastClipboard = text
					processClipboard(text)
				}
			})
		}
	}()

	refreshStashes(stashList)
	win.Add(vmain)
	win.ShowAll()
	gtk.Main()
}

func renderDiff(p model.ProjectOutput, title string) {
	statsBuf.SetText("")
	iter := statsBuf.GetStartIter()

	statsBuf.InsertWithTag(iter, fmt.Sprintf("=== %s ===\n\n", strings.ToUpper(title)), getTag("header"))

	if p.ProjectTree != "" {
		statsBuf.Insert(statsBuf.GetEndIter(), "STRUCTURE:\n"+p.ProjectTree+"\n---\n\n")
	}

	dmp := diffmatchpatch.New()
	keys := make([]string, 0, len(p.Files))
	for k := range p.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, path := range keys {
		statsBuf.InsertWithTag(statsBuf.GetEndIter(), "FILE: "+path+"\n", getTag("header"))
		newContent := p.Files[path]

		// Try to find the file locally to show a real diff
		old, _ := os.ReadFile(path)
		oldStr := string(old)

		if oldStr == newContent || oldStr == "" {
			statsBuf.Insert(statsBuf.GetEndIter(), newContent+"\n\n")
		} else {
			diffs := dmp.DiffMain(oldStr, newContent, false)
			for _, d := range diffs {
				switch d.Type {
				case diffmatchpatch.DiffInsert:
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag("added"))
				case diffmatchpatch.DiffDelete:
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag("deleted"))
				default:
					statsBuf.Insert(statsBuf.GetEndIter(), d.Text)
				}
			}
			statsBuf.Insert(statsBuf.GetEndIter(), "\n\n")
		}
	}
}

func processClipboard(text string) {
	re := regexp.MustCompile(`(?s)\{.*\"files\".*\}`)
	match := re.FindString(text)
	if match != "" {
		var patch model.ProjectOutput
		if err := json.Unmarshal([]byte(match), &patch); err == nil {
			pendingPatches = append(pendingPatches, patch)
			row, _ := gtk.ListBoxRowNew()
			lbl, _ := gtk.LabelNew(fmt.Sprintf("Patch %d (%d files)", len(pendingPatches), len(patch.Files)))
			row.Add(lbl)
			pendingList.Add(row)
			pendingList.ShowAll()
		}
	}
}

func setupTags(buffer *gtk.TextBuffer) {
	tab, _ := buffer.GetTagTable()
	tagA, _ := gtk.TextTagNew("added")
	tagA.SetProperty("background", "#1e3a1e")
	tab.Add(tagA)
	tagD, _ := gtk.TextTagNew("deleted")
	tagD.SetProperty("background", "#4b1818")
	tab.Add(tagD)
	tagH, _ := gtk.TextTagNew("header")
	tagH.SetProperty("foreground", "#569cd6")
	tagH.SetProperty("weight", 700)
	tab.Add(tagH)
}

func getTag(n string) *gtk.TextTag {
	tab, _ := statsBuf.GetTagTable()
	tag, _ := tab.Lookup(n)
	return tag
}

func updateStatus(m string) {
	glib.IdleAdd(func() { statusLabel.SetText("[" + time.Now().Format("15:04:05") + "] " + m) })
}
func refreshStashes(list *gtk.ListBox) { /* simplified for brevity */ }
func mustMarshal(v interface{}) []byte { b, _ := json.MarshalIndent(v, "", "  "); return b }
func newBtn(l string) *gtk.Button      { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	box.PackStart(l, false, false, 0)
}
