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

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const AI_PROMPT_WRAPPER = `SYSTEM: You are a GoCtx AI agent...`

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

	vmain, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	hmain, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	leftBar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 15)
	leftBar.SetMarginStart(15)
	leftBar.SetMarginEnd(15)
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
	swP, _ := gtk.ScrolledWindowNew(nil, nil)
	swP.SetSizeRequest(-1, 200)
	swP.Add(pendingList)
	leftBar.PackStart(swP, false, false, 0)

	label(leftBar, "STASHES")
	stashList, _ = gtk.ListBoxNew()
	swS, _ := gtk.ScrolledWindowNew(nil, nil)
	swS.Add(stashList)
	leftBar.PackStart(swS, true, true, 0)

	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
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
		go func() {
			out, err := builder.BuildSelectiveContext(".", "Manual Build")
			glib.IdleAdd(func() {
				if err == nil {
					activeContext = out
					renderDiff(out, "Current Context")
				}
			})
		}()
	})

	pendingList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		idx := row.GetIndex()
		renderDiff(pendingPatches[idx], "Pending Patch")
		btnApplyPatch.SetSensitive(true)
		btnApplyStash.SetSensitive(false)
	})

	stashList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		widj, _ := row.GetChild()
		lbl, _ := widj.(*gtk.Label)
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
			refreshStashes(stashList)
		}
	})

	btnApplyPatch.Connect("clicked", func() {
		row := pendingList.GetSelectedRow()
		if row != nil && confirmAction("Apply selected patch?") {
			apply.ApplyPatch(".", pendingPatches[row.GetIndex()])
			refreshStashes(stashList)
		}
	})

	go func() {
		for {
			time.Sleep(1 * time.Second)
			glib.IdleAdd(func() {
				clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
				txt, _ := clip.WaitForText()
				if txt != "" && txt != lastClipboard {
					lastClipboard = txt
					processClipboard(txt)
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
	statsBuf.InsertWithTag(statsBuf.GetEndIter(), "=== "+strings.ToUpper(title)+" ===\n\n", getTag("header"))
	if p.ProjectTree != "" {
		statsBuf.Insert(statsBuf.GetEndIter(), p.ProjectTree+"\n---\n\n")
	}

	keys := make([]string, 0, len(p.Files))
	for k := range p.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	dmp := diffmatchpatch.New()
	for _, path := range keys {
		statsBuf.InsertWithTag(statsBuf.GetEndIter(), "FILE: "+path+"\n", getTag("header"))
		newContent := p.Files[path]
		old, _ := os.ReadFile(path)
		diffs := dmp.DiffMain(string(old), newContent, false)
		if len(diffs) == 1 && diffs[0].Type == diffmatchpatch.DiffEqual {
			statsBuf.Insert(statsBuf.GetEndIter(), newContent+"\n\n")
		} else {
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

func refreshStashes(list *gtk.ListBox) {
	list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
	os.MkdirAll(".stashes", 0755)
	items, _ := os.ReadDir(".stashes")
	for _, item := range items {
		if item.IsDir() {
			row, _ := gtk.ListBoxRowNew()
			lbl, _ := gtk.LabelNew(item.Name())
			row.Add(lbl)
			list.Add(row)
		}
	}
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
func confirmAction(m string) bool {
	d := gtk.MessageDialogNew(win, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_YES_NO, m)
	r := d.Run()
	d.Destroy()
	return r == gtk.RESPONSE_YES
}
func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	box.PackStart(l, false, false, 0)
}
