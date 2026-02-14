package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var (
	activeContext  model.ProjectOutput
	currentPayload string
	statsBuf       *gtk.TextBuffer
	stashList      *gtk.ListBox
	pendingPatch   model.ProjectOutput
)

func Run() {
	gtk.Init(nil)

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Management Console")
	win.SetDefaultSize(1200, 800)
	win.Connect("destroy", gtk.MainQuit)

	hmain, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

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

	apiStatus, _ := gtk.LabelNew("API: Listening on :8080")
	leftBar.PackEnd(apiStatus, false, false, 5)

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

	btnBuild.Connect("clicked", func() {
		statsBuf.SetText("Scanning project files...")
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

	btnCopy.Connect("clicked", func() {
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(currentPayload)
		sta, _ := statsBuf.GetText(statsBuf.GetStartIter(), statsBuf.GetEndIter(), true)
		glib.IdleAdd(func() { statsBuf.SetText("Context copied to clipboard.\n" + sta) })
	})

	btnApply.Connect("clicked", func() {
		apply.ApplyPatch(".", pendingPatch)
		btnApply.SetSensitive(false)
		refreshStashes(stashList)
		statsBuf.SetText("PATCH APPLIED SUCCESSFULLY")
	})

	stashList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		lblWidget, _ := row.GetChild()
		lbl, _ := lblWidget.(*gtk.Label)
		txt, _ := lbl.GetText()
		data, _ := os.ReadFile(filepath.Join(".stashes", txt, "patch.json"))
		var p model.ProjectOutput
		json.Unmarshal(data, &p)
		statsBuf.SetText(formatStats(p, "Stash: "+txt))
	})

	go startAPIServer(btnApply)

	hmain.PackStart(leftBar, false, false, 0)
	hmain.PackStart(rightStack, true, true, 0)
	refreshStashes(stashList)
	win.Add(hmain)
	win.ShowAll()
	gtk.Main()
}

func startAPIServer(applyBtn *gtk.Button) {
	http.HandleFunc("/patch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}
		var patch model.ProjectOutput
		if err := json.NewDecoder(r.Body).Decode(&patch); err == nil {
			pendingPatch = patch
			glib.IdleAdd(func() {
				statsBuf.SetText(formatStats(patch, "INCOMING PATCH DETECTED"))
				applyBtn.SetSensitive(true)
			})
		}
	})
	http.ListenAndServe(":8080", nil)
}

func formatStats(p model.ProjectOutput, title string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== %s ===\n", strings.ToUpper(title)))
	sb.WriteString(fmt.Sprintf("TOKENS:  %d\n", p.EstimatedTokens))
	sb.WriteString(fmt.Sprintf("FILES:   %d\n\n", len(p.Files)))
	sb.WriteString("DIRECTORY TREE:\n")
	if p.ProjectTree != "" {
		sb.WriteString(p.ProjectTree)
	} else {
		sb.WriteString("(No tree data available)")
	}
	sb.WriteString("\n\nPROPOSED CHANGES:\n")
	for f := range p.Files {
		sb.WriteString("  [MOD] " + f + "\n")
	}
	return sb.String()
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

func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	box.PackStart(l, false, false, 0)
}
