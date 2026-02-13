package ui

import (
	"encoding/json"
	"goctx/internal/apply"
	"goctx/internal/browser"
	"goctx/internal/builder"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx AI Orchestrator")
	win.SetDefaultSize(1200, 850)
	win.Connect("destroy", gtk.MainQuit)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

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

	btnAI.Connect("clicked", func() {
		start, end := pBuf.GetBounds()
		instruction, _ := pBuf.GetText(start, end, false)
		currCtx, _ := builder.BuildContext(".")
		ctxJS, _ := json.Marshal(currCtx)

		fullPrompt := "TASK: " + instruction + "\n\nPROJECT_STATE: " + string(ctxJS)
		buf.SetText("// Launching browser... waiting for AI response...")

		go func() {
			updatedState, err := browser.ProcessWithAI(fullPrompt)
			// Use glib.IdleAdd to update UI from goroutine
			glib.IdleAdd(func() {
				if err != nil {
					buf.SetText("// ERROR: " + err.Error())
					return
				}
				apply.ApplyPatch(".", updatedState)
				buf.SetText("// SUCCESS: Project updated via AI.")
			})
		}()
	})

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
	box.PackStart(l, false, false, 5)
}

func applyCSS() {
	provider, _ := gtk.CssProviderNew()
	provider.LoadFromPath("assets/style.css")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
