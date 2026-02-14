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
	"strings"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const AI_PROMPT_WRAPPER = `You are an AI developer assistant. Below is the current state of a project.\nWhen providing changes, respond ONLY with a JSON object in the following format:\n{\n  "short_description": "brief summary of changes",\n  "files": {\n    "path/to/file.go": "full content of the file to be overwritten..."\n  }\n}\nStrictly follow this structure so the local tool can apply the patch.\n\nPROJECT DESCRIPTION: %s\n\n--- PROJECT DATA ---\n` 

var (
	activeContext  model.ProjectOutput
	currentPayload string
	lastClipboard  string
	statsBuf       *gtk.TextBuffer
	stashList      *gtk.ListBox
	pendingList    *gtk.ListBox
	pendingPatches []model.ProjectOutput
	selectedStash  model.ProjectOutput
	win            *gtk.Window
	statusLabel    *gtk.Label
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
	leftBar.SetMarginStart(15); leftBar.SetMarginEnd(15); leftBar.SetMarginTop(15)
	leftBar.SetSizeRequest(320, -1)

	btnBuild := newBtn("CURRENT CONTEXT")
	btnCopy := newBtn("COPY CONTEXT")
	btnApplyPatch := newBtn("APPLY SELECTED PATCH")
	btnApplyStash := newBtn("APPLY SELECTED STASH")
	btnApplyPatch.SetSensitive(false)
	btnApplyStash.SetSensitive(false)

	leftBar.PackStart(btnBuild, false, false, 0)
	leftBar.PackStart(btnCopy, false, false, 0)
	leftBar.PackStart(btnApplyPatch, false, false, 0)
	leftBar.PackStart(btnApplyStash, false, false, 0)

	label(leftBar, "PENDING PATCHES")
	swPending, _ := gtk.ScrolledWindowNew(nil, nil)
	swPending.SetSizeRequest(-1, 200)
	pendingList, _ = gtk.ListBoxNew()
	swPending.Add(pendingList)
	leftBar.PackStart(swPending, false, false, 0)

	label(leftBar, "STASHES")
	swStash, _ := gtk.ScrolledWindowNew(nil, nil)
	stashList, _ = gtk.ListBoxNew()
	swStash.Add(stashList)
	leftBar.PackStart(swStash, true, true, 0)

	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	rightStack.SetMarginStart(20); rightStack.SetMarginEnd(20); rightStack.SetMarginTop(15)

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
			out, err := builder.BuildSelectiveContext(".", nil)
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
		desc := "Core project context for review or modification."
		fullPrompt := fmt.Sprintf(AI_PROMPT_WRAPPER, desc) + string(mustMarshal(activeContext))
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(fullPrompt)
		updateStatus("Instructional Prompt + JSON copied")
	})

	stashList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil { return }
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
		if selectedStash.Files != nil {
			apply.ApplyPatch(".", selectedStash)
			updateStatus("Stash applied successfully")
			refreshStashes(stashList)
		}
	})

	btnApplyPatch.Connect("clicked", func() {
		row := pendingList.GetSelectedRow()
		if row != nil {
			apply.ApplyPatch(".", pendingPatches[row.GetIndex()])
			updateStatus("Patch applied successfully")
			refreshStashes(stashList)
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

func refreshStashes(list *gtk.ListBox) {
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

func setupTags(buffer *gtk.TextBuffer) {
	tab, _ := buffer.GetTagTable()
	tagA, _ := gtk.TextTagNew("added"); tagA.SetProperty("background", "#1e3a1e"); tagA.SetProperty("foreground", "#afffbc"); tab.Add(tagA)
	tagD, _ := gtk.TextTagNew("deleted"); tagD.SetProperty("background", "#4b1818"); tagD.SetProperty("foreground", "#ffa1a1"); tab.Add(tagD)
	tagH, _ := gtk.TextTagNew("header"); tagH.SetProperty("weight", 700); tagH.SetProperty("foreground", "#569cd6"); tab.Add(tagH)
}

func renderDiff(p model.ProjectOutput, title string) {
	statsBuf.SetText("")
	statsBuf.Insert(statsBuf.GetEndIter(), fmt.Sprintf("=== %s ===\n\n", strings.ToUpper(title)))
	dmp := diffmatchpatch.New()
	for path, newContent := range p.Files {
		statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("FILE: %s\n", path), getTag("header"))
		old, _ := os.ReadFile(path)
		diffs := dmp.DiffMain(string(old), newContent, false)
		for _, d := range diffs {
			switch d.Type {
			case diffmatchpatch.DiffInsert: statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag("added"))
			case diffmatchpatch.DiffDelete: statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag("deleted"))
			default: statsBuf.Insert(statsBuf.GetEndIter(), d.Text)
			}
		}
		statsBuf.Insert(statsBuf.GetEndIter(), "\n")
	}
}

func getTag(n string) *gtk.TextTag { tab, _ := statsBuf.GetTagTable(); t, _ := tab.Lookup(n); return t }
func updateStatus(m string) { statusLabel.SetText(fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), m)) }
func mustMarshal(v interface{}) []byte { b, _ := json.MarshalIndent(v, "", "  "); return b }
func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) { l, _ := gtk.LabelNew(t); l.SetXAlign(0); box.PackStart(l, false, false, 0) }
