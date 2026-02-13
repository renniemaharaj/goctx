package ui

import (
	"encoding/json"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/browser"
	"goctx/internal/model"
	"github.com/gotk3/gotk3/gtk"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx AI Orchestrator")
	win.SetDefaultSize(1200, 850)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// SIDEBAR
	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sidebar.SetSizeRequest(280, -1)
	sidebar.GetStyleContext().AddClass("sidebar")
	label(sidebar, "HISTORY / STASHES")
	stashList, _ := gtk.ListBoxNew()
	refreshStashes(stashList)
	sidebar.PackStart(stashList, true, true, 0)

	// MAIN CONTENT
	body, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	body.SetMarginStart(20)
	body.SetMarginEnd(20)

	// Editor View (JSON Output)
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	body.PackStart(sw, true, true, 0)

	// LARGE PROMPT AREA
	label(body, "AI INSTRUCTIONS")
	promptView, _ := gtk.TextViewNew()
	promptView.SetWrapMode(gtk.WRAP_WORD)
	promptView.SetSizeRequest(-1, 120)
	promptView.GetStyleContext().AddClass("input-area")
	pBuf, _ := promptView.GetBuffer()
	body.PackStart(promptView, false, false, 0)

	// BOTTOM ACTION ROW
	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Current Context")
	btnAI := newBtn("🚀 SYNCHRONIZE WITH AI")
	btnAI.GetStyleContext().AddClass("suggested-action")
	
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnAI, true, true, 0)
	body.PackStart(btnBox, false, false, 20)

	// LOGIC
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
	sctx, _ := l.GetStyleContext()
	sctx.AddClass("header")
	box.PackStart(l, false, false, 5)
}
