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
	pendingPatch    model.ProjectOutput
)

func Run() {
	gtk.Init(nil)

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Management Console")
	win.SetDefaultSize(1200, 800)
	win.Connect("destroy", gtk.MainQuit)

	hmain, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// --- LEFT BAR ---
	leftBar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 15)
	leftBar.SetMarginStart(15)
	leftBar.SetMarginEnd(15)
	leftBar.SetMarginTop(15)
	leftBar.SetSizeRequest(280, -1)

	btnBuild := newBtn("BUILD CONTEXT")
	btnCopy := newBtn("COPY CONTEXT")
	btnApply := newBtn("APPLY PENDING PATCH")
	btnApply.SetSensitive(false)

	leftBar.PackStart(btnBuild, false, false, 0)
	leftBar.PackStart(btnCopy, false, false, 0)
	leftBar.PackStart(btnApply, false, false, 0)

	label(leftBar, "SESSION HISTORY")
	swStash, _ := gtk.ScrolledWindowNew(nil, nil)
	stashList, _ = gtk.ListBoxNew()
	swStash.Add(stashList)
	leftBar.PackStart(swStash, true, true, 0)

	statusLabel, _ := gtk.LabelNew("CLIPBOARD: Watching...")
	leftBar.PackEnd(statusLabel, false, false, 5)

	// --- RIGHT CONTENT ---
	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	rightStack.SetMarginStart(20)
	rightStack.SetMarginEnd(20)
	rightStack.SetMarginTop(15)

	label(rightStack, "PROJECT DASHBOARD")
	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ := gtk.TextViewNew()
	statsView.SetMonospace(true)
	statsView.SetEditable(false)
	statsBuf, _ = statsView.GetBuffer()
	statsScroll.Add(statsView)
	rightStack.PackStart(statsScroll, true, true, 0)

	// --- LOGIC: BUILD ---
	btnBuild.Connect("clicked", func() {
		go func() {
			out, err := builder.BuildSelectiveContext(".", nil)
			if err == nil {
				activeContext = out
				js, _ := json.Marshal(out)
				currentPayload = string(js)
				glib.IdleAdd(func() {
					statsBuf.SetText(formatStats(activeContext, "Active Workspace"))
				})
			}
		}()
	})

	// --- LOGIC: COPY ---
	btnCopy.Connect("clicked", func() {
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(currentPayload)
		lastClipboard = currentPayload // Avoid re-parsing our own copy
		sta, _ := statsBuf.GetText(statsBuf.GetStartIter(), statsBuf.GetEndIter(), true)
		glib.IdleAdd(func() { statsBuf.SetText("CONTEXT COPIED\n" + sta) })
	})

	// --- LOGIC: APPLY ---
	btnApply.Connect("clicked", func() {
		apply.ApplyPatch(".", pendingPatch)
		btnApply.SetSensitive(false)
		refreshStashes(stashList)
		statsBuf.SetText("PATCH APPLIED SUCCESSFULLY")
	})

	// --- CLIPBOARD MONITOR (Polling) ---
	go func() {
		for {
			time.Sleep(1 * time.Second)
			glib.IdleAdd(func() {
				clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
				text, err := clip.WaitForText()
				if err == nil && text != "" && text != lastClipboard {
					lastClipboard = text
					monitorClipboard(text, btnApply)
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

func monitorClipboard(text string, applyBtn *gtk.Button) {
	re := regexp.MustCompile(`(?s)\{.*\"files\".*\}`)
	match := re.FindString(text)
	if match != "" {
		var patch model.ProjectOutput
		if err := json.Unmarshal([]byte(match), &patch); err == nil {
			pendingPatch = patch
			statsBuf.SetText(formatStats(patch, "Clipboard Patch Detected"))
			applyBtn.SetSensitive(true)
		}
	}
}

func formatStats(p model.ProjectOutput, title string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== %s ===\n", strings.ToUpper(title)))
	sb.WriteString(fmt.Sprintf("TOKENS:  %d\n", p.EstimatedTokens))
	sb.WriteString(fmt.Sprintf("FILES:   %d\n\n", len(p.Files)))
	sb.WriteString("DIRECTORY TREE:\n")
	if p.ProjectTree != "" { sb.WriteString(p.ProjectTree) } else { sb.WriteString("(No tree data available)") }
	sb.WriteString("\n\nPROPOSED CHANGES:\n")
	for f := range p.Files { sb.WriteString("  [MOD] " + f + "\n") }
	return sb.String()
}

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
		list.ShowAll()
		return false
	})
}

func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) { l, _ := gtk.LabelNew(t); l.SetXAlign(0); box.PackStart(l, false, false, 0) }
