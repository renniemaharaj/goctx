package ui

import (
	"encoding/json"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/browser"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/glib"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx AI Orchestrator")
	win.SetDefaultSize(1300, 900)
	win.Connect("destroy", gtk.MainQuit)

	body, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	body.SetMarginStart(20)
	body.SetMarginEnd(20)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	body.PackStart(sw, true, true, 0)

	pView, _ := gtk.TextViewNew()
	pView.SetSizeRequest(-1, 150)
	pBuf, _ := pView.GetBuffer()
	body.PackStart(pView, false, false, 0)

	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Context")
	btnApply := newBtn("🚀 APPLY AI PATCH")
	btnClear := newBtn("🧹 Clear Console")
	
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnApply, true, true, 0)
	btnBox.PackStart(btnClear, false, false, 0)
	body.PackStart(btnBox, false, false, 10)

	// Logic
	btnClear.Connect("clicked", func() { buf.SetText("") })

	btnBuild.Connect("clicked", func() {
		buf.SetText("// Building...")
		go func() {
			out, _ := builder.BuildSelectiveContext(".", nil)
			js, _ := json.MarshalIndent(out, "", "  ")
			glib.IdleAdd(func() { buf.SetText(string(js)) })
		}()
	})

	btnApply.Connect("clicked", func() {
		start, end := pBuf.GetBounds()
		instr, _ := pBuf.GetText(start, end, false)
		go func() {
			glib.IdleAdd(func() { buf.SetText("[PIPELINE] Running...") })
			currCtx, _ := builder.BuildSelectiveContext(".", nil)
			ctxJS, _ := json.Marshal(currCtx)
			updated, err := browser.ProcessWithAI("TASK: " + instr + "\n\nSTATE: " + string(ctxJS))
			glib.IdleAdd(func() {
				if err != nil { buf.SetText("// Error: " + err.Error()); return }
				apply.ApplyPatch(".", updated)
				buf.SetText("// SUCCESS")
			})
		}()
	})

	win.Add(body)
	win.ShowAll()
	gtk.Main()
}

func newBtn(l string) *gtk.Button {
	b, _ := gtk.ButtonNewWithLabel(l)
	return b
}

func applyCSS() {
	provider, _ := gtk.CssProviderNew()
	provider.LoadFromPath("assets/style.css")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
