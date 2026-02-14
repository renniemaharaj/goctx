package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const AI_PROMPT_WRAPPER = `SYSTEM: You are a GoCtx AI agent. You have access to the project state below.\nTo apply changes, output a SINGLE JSON code block. The local orchestrator will scan the clipboard, detect the JSON, and prompt the user to integrate it.\n\nFORMAT:\n\u0060\u0060\u0060json\n{\n  "short_description": "Refactor types",\n  "files": { "path/file.go": "full content..." }\n}\n\u0060\u0060\u0060\n\nPROJECT DATA:\n`

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
	ebPending, _ := gtk.EventBoxNew()
	swPending, _ := gtk.ScrolledWindowNew(nil, nil)
	swPending.SetSizeRequest(-1, 200)
	swPending.Add(pendingList)
	ebPending.Add(swPending)
	ebPending.Connect("button-press-event", func() {
		clearAllSelections()
	})
	leftBar.PackStart(ebPending, false, false, 0)

	label(leftBar, "STASHES")
	stashList, _ = gtk.ListBoxNew()
	ebStash, _ := gtk.EventBoxNew()
	swStash, _ := gtk.ScrolledWindowNew(nil, nil)
	swStash.Add(stashList)
	ebStash.Add(swStash)
	ebStash.Connect("button-press-event", func() {
		clearAllSelections()
	})
	leftBar.PackStart(ebStash, true, true, 0)

	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	rightStack.SetMarginStart(20)
	rightStack.SetMarginEnd(20)
	rightStack.SetMarginTop(15)

	label(rightStack, "CONTEXT TOOL GUI (GOCTX)")
	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ := gtk.TextViewNew()
	statsView.SetMonospace(true)
	statsView.SetEditable(false)
	statsView.SetLeftMargin(25)
	statsView.SetTopMargin(25)
	statsBuf, _ = statsView.GetBuffer()
	statsScroll.Add(statsView)
	rightStack.PackStart(statsScroll, true, true, 0)

	setupTags(statsBuf)

	statusPanel, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	statusLabel, _ = gtk.LabelNew("Ready")
	statusPanel.PackStart(statusLabel, false, false, 10)

	hmain.PackStart(leftBar, false, false, 0)
	hmain.PackStart(rightStack, true, true, 0)
	vmain.PackStart(hmain, true, true, 0)
	vmain.PackStart(statusPanel, false, false, 5)

	btnBuild.Connect("clicked", func() {
		go func() {
			out, err := builder.BuildSelectiveContext(".", "Manual Build")
			if err == nil {
				activeContext = out
				glib.IdleAdd(func() {
					renderDiff(activeContext, "Current Workspace State")
					updateStatus("Context built")
				})
			}
		}()
	})

	btnCopy.Connect("clicked", func() {
		fullPrompt := AI_PROMPT_WRAPPER + string(mustMarshal(activeContext))
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(fullPrompt)
		updateStatus("System Prompt + Context copied")
	})

	pendingList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil { return }
		stashList.UnselectAll()
		idx := row.GetIndex()
		renderDiff(pendingPatches[idx], "Pending Patch Preview")
		btnApplyPatch.SetSensitive(true)
		btnApplyStash.SetSensitive(false)
	})

	stashList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil { return }
		pendingList.UnselectAll()
		lblWidget, _ := row.GetChild()
		lbl, _ := lblWidget.(*gtk.Label)
		txt, _ := lbl.GetText()
		data, err := os.ReadFile(filepath.Join(".stashes", txt, "patch.json"))
		if err == nil && json.Unmarshal(data, &selectedStash) == nil {
			renderDiff(selectedStash, "Stash: "+txt)
			btnApplyStash.SetSensitive(true)
			btnApplyPatch.SetSensitive(false)
		}
	})

	btnApplyStash.Connect("clicked", func() {
		if confirmAction("Apply selected stash?") {
			apply.ApplyPatch(".", selectedStash)
			updateStatus("Stash applied")
			refreshStashes(stashList)
		}
	})

	btnApplyPatch.Connect("clicked", func() {
		if confirmAction("Apply selected patch?") {
			row := pendingList.GetSelectedRow()
			if row != nil {
				apply.ApplyPatch(".", pendingPatches[row.GetIndex()])
				updateStatus("Patch applied")
				refreshStashes(stashList)
			}
		}
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

func clearAllSelections() {
	pendingList.UnselectAll()
	stashList.UnselectAll()
	resetView()
}

func resetView() {
	btnApplyPatch.SetSensitive(false)
	btnApplyStash.SetSensitive(false)
	statsBuf.SetText("")
	updateStatus("Selection cleared")
}

func refreshStashes(list *gtk.ListBox) {
	glib.IdleAdd(func() bool {
		list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
		os.MkdirAll(".stashes", 0755)
		filepath.Walk(".stashes", func(path string, info os.FileInfo, err error) error {
			if err == nil && info.IsDir() && path != ".stashes" && filepath.Dir(path) == ".stashes" {
				row, _ := gtk.ListBoxRowNew()
				lbl, _ := gtk.LabelNew(filepath.Base(path))
				row.Add(lbl)
				list.Add(row)
			}
			return nil
		})
		list.ShowAll()
		return false
	})
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
	if err != nil { return nil }
	tag, _ := tab.Lookup(n)
	return tag
}

func renderDiff(p model.ProjectOutput, title string) {
	statsBuf.SetText("")
	statsBuf.Insert(statsBuf.GetEndIter(), fmt.Sprintf("=== %s ===\n\n", strings.ToUpper(title)))
	dmp := diffmatchpatch.New()

	var keys []string
	for k := range p.Files { keys = append(keys, k) }
	sort.Strings(keys)

	renderCount := 0
	const limit = 10

	for _, path := range keys {
		if renderCount >= limit { break }
		newContent := p.Files[path]
		if !utf8.ValidString(newContent) { continue }

		statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("FILE: %s\n", path), getTag("header"))
		old, _ := os.ReadFile(path)
		oldStr := string(old)
		if !utf8.ValidString(oldStr) { oldStr = "[Binary or Invalid UTF-8 Content]" }

		diffs := dmp.DiffMain(oldStr, newContent, false)
		for _, d := range diffs {
			if utf8.ValidString(d.Text) {
				switch d.Type {
				case diffmatchpatch.DiffInsert: statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag("added"))
				case diffmatchpatch.DiffDelete: statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag("deleted"))
				default: statsBuf.Insert(statsBuf.GetEndIter(), d.Text)
				}
			}
		}
		statsBuf.Insert(statsBuf.GetEndIter(), "\n")
		renderCount++
	}

	if len(keys) > limit {
		msg := fmt.Sprintf("\n--- PREVIEW LIMIT REACHED: %d/%d files rendered ---", limit, len(keys))
		statsBuf.Insert(statsBuf.GetEndIter(), msg)
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
			updateStatus("New patch detected")
		}
	}
}

func confirmAction(m string) bool {
	d := gtk.MessageDialogNew(win, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_YES_NO, m)
	r := d.Run()
	d.Destroy()
	return r == gtk.RESPONSE_YES
}
func updateStatus(m string) {
	statusLabel.SetText(fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), m))
}
func mustMarshal(v interface{}) []byte { b, _ := json.MarshalIndent(v, "", "  "); return b }
func newBtn(l string) *gtk.Button      { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	box.PackStart(l, false, false, 0)
}
