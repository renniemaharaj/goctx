package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/glib"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	activeContext   model.ProjectOutput
	currentPayload  string
	lastClipboard   string
	statsBuf        *gtk.TextBuffer
	stashList       *gtk.ListBox
	pendingList     *gtk.ListBox
	pendingPatches  []model.ProjectOutput
)

func Run() {
	gtk.Init(nil)

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Management Console")
	win.SetDefaultSize(1300, 850)
	win.Connect("destroy", gtk.MainQuit)

	hmain, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// --- LEFT BAR ---
	leftBar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 15)
	leftBar.SetMarginStart(15); leftBar.SetMarginEnd(15); leftBar.SetMarginTop(15)
	leftBar.SetSizeRequest(300, -1)

	btnBuild := newBtn("CURRENT CONTEXT")
	btnCopy  := newBtn("COPY CONTEXT")
	btnApply := newBtn("APPLY SELECTED PATCH")
	btnApply.SetSensitive(false)

	leftBar.PackStart(btnBuild, false, false, 0)
	leftBar.PackStart(btnCopy, false, false, 0)
	leftBar.PackStart(btnApply, false, false, 0)

	label(leftBar, "PENDING PATCHES")
	swPending, _ := gtk.ScrolledWindowNew(nil, nil)
	pendingList, _ = gtk.ListBoxNew()
	swPending.Add(pendingList)
	leftBar.PackStart(swPending, true, true, 0)

	label(leftBar, "STASHES")
	swStash, _ := gtk.ScrolledWindowNew(nil, nil)
	stashList, _ = gtk.ListBoxNew()
	swStash.Add(stashList)
	leftBar.PackStart(swStash, true, true, 0)

	// --- RIGHT CONTENT ---
	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	rightStack.SetMarginStart(20); rightStack.SetMarginEnd(20); rightStack.SetMarginTop(15)

	label(rightStack, "DASHBOARD")
	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ := gtk.TextViewNew()
	statsView.SetMonospace(true); statsView.SetEditable(false)
	statsBuf, _ = statsView.GetBuffer()
	statsScroll.Add(statsView)
	rightStack.PackStart(statsScroll, true, true, 0)

	// --- DESELECT LOGIC (Click Away) ---
	win.Connect("button-press-event", func(w *gtk.Window, event *gdk.Event) {
		// If we click anywhere that isnt the listbox, clear selections
		pendingList.UnselectAll()
		stashList.UnselectAll()
		btnApply.SetSensitive(false)
	})

	btnBuild.Connect("clicked", func() {
		go func() {
			out, err := builder.BuildSelectiveContext(".", nil)
			if err == nil {
				activeContext = out; currentPayload = string(mustMarshal(out))
				glib.IdleAdd(func() { statsBuf.SetText(formatStats(activeContext, "Active Workspace")) })
			}
		}()
	})

	btnCopy.Connect("clicked", func() {
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(currentPayload); lastClipboard = currentPayload
	})

	pendingList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil { return }
		stashList.UnselectAll() // Ensure mutually exclusive selection
		idx := row.GetIndex()
		if idx < len(pendingPatches) {
			statsBuf.SetText(formatStats(pendingPatches[idx], "Pending Patch"))
			btnApply.SetSensitive(true)
		}
	})

	stashList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil { return }
		pendingList.UnselectAll()
		lblWidget, _ := row.GetChild()
		lbl, _ := lblWidget.(*gtk.Label)
		txt, _ := lbl.GetText()
		
		data, err := os.ReadFile(filepath.Join(".stashes", txt, "patch.json"))
		if err == nil {
			var p model.ProjectOutput
			if err := json.Unmarshal(data, &p); err == nil {
				statsBuf.SetText(formatStats(p, "Stash: "+txt))
				btnApply.SetSensitive(false) // Stashes are historical, not pending
			}
		}
	})

	btnApply.Connect("clicked", func() {
		row := pendingList.GetSelectedRow()
		if row == nil { return }
		patch := pendingPatches[row.GetIndex()]
		apply.ApplyPatch(".", patch)
		btnApply.SetSensitive(false)
		refreshStashes(stashList)
		statsBuf.SetText("ACTION COMPLETE: Files updated/removed.")
	})

	go func() {
		for {
			time.Sleep(1 * time.Second)
			glib.IdleAdd(func() {
				clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
				text, err := clip.WaitForText()
				if err == nil && text != "" && text != lastClipboard {
					lastClipboard = text
					processClipboard(text)
				}
			})
		}
	}()

	hmain.PackStart(leftBar, false, false, 0)
	hmain.PackStart(rightStack, true, true, 0)
	refreshStashes(stashList)
	win.Add(hmain)
	win.ShowAll()
	gtk.Main()
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

func formatStats(p model.ProjectOutput, title string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== %s ===\n", strings.ToUpper(title)))
	sb.WriteString(fmt.Sprintf("TOKENS:  %d\nFILES:   %d\n\n", p.EstimatedTokens, len(p.Files)))
	sb.WriteString("DIRECTORY TREE:\n")
	if p.ProjectTree != "" { sb.WriteString(p.ProjectTree) } else { sb.WriteString("(No tree data available)") }
	sb.WriteString("\n\nFILES IN SCOPE:\n")
	for f, content := range p.Files {
		if content == "" {
			sb.WriteString("  [DELETE] " + f + "\n")
		} else {
			sb.WriteString("  " + f + "\n")
		}
	}
	return sb.String()
}

func mustMarshal(v interface{}) []byte { b, _ := json.Marshal(v); return b }
func refreshStashes(list *gtk.ListBox) {
	glib.IdleAdd(func() bool {
		list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
		os.MkdirAll(".stashes", 0755)
		filepath.Walk(".stashes", func(path string, info os.FileInfo, err error) error {
			if err == nil && info.IsDir() && path != ".stashes" && filepath.Dir(path) == ".stashes" {
				row, _ := gtk.ListBoxRowNew(); lbl, _ := gtk.LabelNew(filepath.Base(path)); row.Add(lbl); list.Add(row)
			}
			return nil
		})
		list.ShowAll(); return false
	})
}
func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) { l, _ := gtk.LabelNew(t); l.SetXAlign(0); box.PackStart(l, false, false, 0) }
