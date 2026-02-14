package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/builder"
	"goctx/internal/model"
	"os"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

var (
	activeContext  model.ProjectOutput
	currentPayload string
	lastClipboard  string
	statsView      *gtk.TextView
	statsBuf       *gtk.TextBuffer
	stashList      *gtk.ListBox
	pendingList    *gtk.ListBox
	pendingPatches []model.ProjectOutput
	win            *gtk.Window
	statusLabel    *gtk.Label
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
	leftBar.SetMarginTop(15)
	leftBar.SetSizeRequest(320, -1)

	btnBuild := newBtn("CURRENT CONTEXT")
	btnCopy := newBtn("COPY CONTEXT")
	btnApplyPatch := newBtn("APPLY SELECTED PATCH")

	leftBar.PackStart(btnBuild, false, false, 0)
	leftBar.PackStart(btnCopy, false, false, 0)
	leftBar.PackStart(btnApplyPatch, false, false, 0)

	label(leftBar, "PENDING PATCHES")
	swPending, _ := gtk.ScrolledWindowNew(nil, nil)
	swPending.SetShadowType(gtk.SHADOW_IN)
	swPending.SetSizeRequest(-1, 300)
	pendingList, _ = gtk.ListBoxNew()
	swPending.Add(pendingList)
	leftBar.PackStart(swPending, false, false, 0)

	label(leftBar, "STASHES")
	swStash, _ := gtk.ScrolledWindowNew(nil, nil)
	stashList, _ = gtk.ListBoxNew()
	swStash.Add(stashList)
	leftBar.PackStart(swStash, true, true, 0)

	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	rightStack.SetMarginStart(20)
	rightStack.SetMarginEnd(20)
	rightStack.SetMarginTop(15)

	label(rightStack, "DASHBOARD / DIFF PREVIEW")
	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ = gtk.TextViewNew()
	statsView.SetMonospace(true)
	statsView.SetEditable(false)
	statsBuf, _ = statsView.GetBuffer()
	setupTags(statsBuf)
	statsScroll.Add(statsView)
	rightStack.PackStart(statsScroll, true, true, 0)

	statusPanel, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	statusLabel, _ = gtk.LabelNew("Ready")
	statusPanel.PackStart(statusLabel, false, false, 5)

	hmain.PackStart(leftBar, false, false, 0)
	hmain.PackStart(rightStack, true, true, 0)
	vmain.PackStart(hmain, true, true, 0)
	vmain.PackStart(statusPanel, false, false, 5)

	btnBuild.Connect("clicked", func() {
		go func() {
			out, _ := builder.BuildSelectiveContext(".", nil)
			activeContext = out
			currentPayload = string(mustMarshal(out))
			glib.IdleAdd(func() { renderText(formatStats(activeContext, "Current Workspace State")) })
		}()
	})

	pendingList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		idx := row.GetIndex()
		if idx < len(pendingPatches) {
			renderDiff(pendingPatches[idx])
			btnApplyPatch.SetSensitive(true)
		}
	})

	win.Add(vmain)
	win.ShowAll()
	gtk.Main()
}

func setupTags(buf *gtk.TextBuffer) {
	buf.CreateTag("add", map[string]interface{}{"foreground": "#2ea44f"})
	buf.CreateTag("rem", map[string]interface{}{"foreground": "#cf222e"})
	buf.CreateTag("hdr", map[string]interface{}{"foreground": "#0969da", "weight": 700})
}

func renderText(text string) {
	statsBuf.SetText(text)
}

func renderDiff(p model.ProjectOutput) {
	statsBuf.SetText("")
	// iter := statsBuf.GetStartIter()

	insertWithTag := func(t string, tagName string) {
		if tagName != "" {
			statsBuf.InsertWithTagByName(statsBuf.GetEndIter(), t, tagName)
		} else {
			statsBuf.Insert(statsBuf.GetEndIter(), t)
		}
	}

	insertWithTag("=== DIFF PREVIEW ===\n\n", "hdr")

	for path, newContent := range p.Files {
		oldContent, _ := os.ReadFile(path)
		edits := myers.ComputeEdits(span.URIFromPath(path), string(oldContent), newContent)
		diff := fmt.Sprint(gotextdiff.ToUnified(path, path, string(oldContent), edits))

		lines := strings.Split(diff, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				insertWithTag(line+"\n", "add")
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				insertWithTag(line+"\n", "rem")
			} else if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
				insertWithTag(line+"\n", "hdr")
			} else {
				insertWithTag(line+"\n", "")
			}
		}
	}
}

func mustMarshal(v interface{}) []byte { b, _ := json.Marshal(v); return b }
func newBtn(l string) *gtk.Button      { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	box.PackStart(l, false, false, 0)
}
func formatStats(p model.ProjectOutput, title string) string { return "Tree rendering..." }
