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
	leftBar.SetMarginStart(15)
	leftBar.SetMarginEnd(15)
	leftBar.SetMarginTop(15)
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
	swPending.SetShadowType(gtk.SHADOW_IN)
	swPending.SetSizeRequest(-1, 200)
	pendingList, _ = gtk.ListBoxNew()
	swPending.Add(pendingList)
	leftBar.PackStart(swPending, false, false, 0)

	label(leftBar, "STASHES")
	swStash, _ := gtk.ScrolledWindowNew(nil, nil)
	swStash.SetShadowType(gtk.SHADOW_IN)
	stashList, _ = gtk.ListBoxNew()
	swStash.Add(stashList)
	leftBar.PackStart(swStash, true, true, 0)

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
	statusPanel.SetMarginStart(10)
	statusPanel.SetMarginEnd(10)
	statusLabel, _ = gtk.LabelNew("Ready")
	statusPanel.PackStart(statusLabel, false, false, 5)

	hmain.PackStart(leftBar, false, false, 0)
	hmain.PackStart(rightStack, true, true, 0)
	vmain.PackStart(hmain, true, true, 0)
	vmain.PackStart(statusPanel, false, false, 5)

	btnBuild.Connect("clicked", func() {
		go func() {
			out, err := builder.BuildSelectiveContext(".", nil)
			if err == nil {
				activeContext = out
				currentPayload = string(mustMarshal(out))
				glib.IdleAdd(func() {
					renderDiff(activeContext, "Current Workspace State")
					updateStatus("Context built")
				})
			}
		}()
	})

	btnCopy.Connect("clicked", func() {
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(strings.Trim(currentPayload, "\x00"))
		updateStatus("JSON copied to clipboard")
	})

	pendingList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		idx := row.GetIndex()
		if idx < len(pendingPatches) {
			renderDiff(pendingPatches[idx], "Pending Patch Preview")
			btnApplyPatch.SetSensitive(true)
			btnApplyStash.SetSensitive(false)
		}
	})

	stashList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
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

	btnApplyPatch.Connect("clicked", func() {
		if confirmAction("Apply selected patch?") {
			row := pendingList.GetSelectedRow()
			apply.ApplyPatch(".", pendingPatches[row.GetIndex()])
			refreshStashes(stashList)
			updateStatus("Patch applied")
		}
	})

	go func() {
		for {
			time.Sleep(1 * time.Second)
			glib.IdleAdd(func() {
				clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
				text, err := clip.WaitForText()
				if err == nil && text != "" && text != lastClipboard {
					text = strings.ReplaceAll(text, "\x00", "")
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

func renderDiff(p model.ProjectOutput, title string) {
	statsBuf.SetText("")
	iter := statsBuf.GetStartIter()
	statsBuf.Insert(iter, fmt.Sprintf("=== %s ===\nFILES: %d\n\n", strings.ToUpper(title), len(p.Files)))

	dmp := diffmatchpatch.New()
	for path, newContent := range p.Files {
		statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("\nFILE: %s\n", path), getTag("header"))
		statsBuf.Insert(statsBuf.GetEndIter(), strings.Repeat("━", len(path)+6)+"\n")
		old, _ := os.ReadFile(path)
		diffs := dmp.DiffMain(string(old), newContent, false)
		diffs = dmp.DiffCleanupSemantic(diffs)
		for _, d := range diffs {
			end := statsBuf.GetEndIter()
			switch d.Type {
			case diffmatchpatch.DiffInsert:
				statsBuf.InsertWithTag(end, d.Text, getTag("added"))
			case diffmatchpatch.DiffDelete:
				statsBuf.InsertWithTag(end, d.Text, getTag("deleted"))
			default:
				statsBuf.Insert(end, d.Text)
			}
		}
	}
}

func getTag(n string) *gtk.TextTag { tab, _ := statsBuf.GetTagTable(); t, _ := tab.Lookup(n); return t }
func confirmAction(m string) bool {
	d := gtk.MessageDialogNew(win, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_YES_NO, m)
	r := d.Run()
	d.Destroy()
	return r == gtk.RESPONSE_YES
}
func updateStatus(m string) {
	statusLabel.SetText(fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), m))
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

func mustMarshal(v interface{}) []byte { b, _ := json.Marshal(v); return b }
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
func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	box.PackStart(l, false, false, 0)
}
