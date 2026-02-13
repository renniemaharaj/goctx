package ui

import (
	"encoding/json"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/browser"
	"goctx/internal/model"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"os"
	"path/filepath"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx AI Orchestrator")
	win.SetDefaultSize(1200, 850)
	win.Connect("destroy", gtk.MainQuit)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// SIDEBAR
	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sidebar.SetSizeRequest(280, -1)
	sctx, _ := sidebar.GetStyleContext()
	sctx.AddClass("sidebar")
	label(sidebar, "HISTORY / STASHES")
	stashList, _ := gtk.ListBoxNew()
	refreshStashes(stashList)
	sidebar.PackStart(stashList, true, true, 0)

	// MAIN CONTENT
	body, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	body.SetMarginStart(20)
	body.SetMarginEnd(20)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	body.PackStart(sw, true, true, 0)

	label(body, "AI INSTRUCTIONS")
	promptView, _ := gtk.TextViewNew()
	promptView.SetWrapMode(gtk.WRAP_WORD)
	promptView.SetSizeRequest(-1, 150)
	pBuf, _ := promptView.GetBuffer()
	body.PackStart(promptView, false, false, 0)

	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Context")
	btnAI := newBtn("🚀 SYNCHRONIZE WITH AI")
	
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnAI, true, true, 0)
	body.PackStart(btnBox, false, false, 20)

	btnBuild.Connect("clicked", func() {
		out, _ := builder.BuildContext(".")
		jsonData, _ := json.MarshalIndent(out, "", "  ")
		buf.SetText(string(jsonData))
	})

	btnAI.Connect("clicked", func() {
		start, end := pBuf.GetBounds()
		instruction, _ := pBuf.GetText(start, end, false)
		currCtx, _ := builder.BuildContext(".")
		ctxJS, _ := json.Marshal(currCtx)
		
		fullPrompt := "TASK: " + instruction + "\n\nPROJECT_STATE: " + string(ctxJS)
		buf.SetText("// Launching browser... waiting for AI response...")
		
		go func() {
			updatedState, err := browser.ProcessWithAI(fullPrompt)
			gtk.FunctionsInMain(func() {
				if err != nil {
					buf.SetText("// ERROR: " + err.Error())
					return
				}
				apply.ApplyPatch(".", updatedState)
				refreshStashes(stashList)
				buf.SetText("// SUCCESS: Project updated via AI.")
			})
		}()
	})

	hbox.PackStart(sidebar, false, false, 0)
	hbox.PackStart(body, true, true, 0)
	win.Add(hbox)
	win.ShowAll()
	gtk.Main()
}

func newBtn(l string) *gtk.Button {
	btn, _ := gtk.ButtonNewWithLabel(l)
	return btn
}

func label(box *gtk.Box, text string) {
	l, _ := gtk.LabelNew(text)
	l.SetXAlign(0)
	l.SetMarginStart(10)
	box.PackStart(l, false, false, 5)
}

func refreshStashes(list *gtk.ListBox) {
	list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
	filepath.Walk(".stashes", func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && path != ".stashes" && filepath.Dir(path) == ".stashes" {
			row, _ := gtk.ListBoxRowNew()
			hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
			lbl, _ := gtk.LabelNew(filepath.Base(path))
			btn := newBtn("Apply")
			hbox.PackStart(lbl, true, true, 5)
			hbox.PackEnd(btn, false, false, 5)
			row.Add(hbox)
			list.Add(row)
		}
		return nil
	})
	list.ShowAll()
}

func applyCSS() {
	provider, _ := gtk.CssProviderNew()
	provider.LoadFromPath("assets/style.css")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
