package ui

import (
	"encoding/json"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/browser"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/glib"
	"strings"
)

func Run() {
	gtk.Init(nil)
	applyCSS()
	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx Selective Patch")
	win.SetDefaultSize(1200, 800)
	win.Connect("destroy", gtk.MainQuit)

	body, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	body.PackStart(sw, true, true, 0)

	label(body, "INSTRUCTIONS (Use @filename to target files)")
	promptView, _ := gtk.TextViewNew()
	pBuf, _ := promptView.GetBuffer()
	body.PackStart(promptView, false, false, 0)

	btnAI := newBtn("🚀 APPLY SMART PATCH")
	body.PackStart(btnAI, false, false, 20)

	btnAI.Connect("clicked", func() {
		start, end := pBuf.GetBounds()
		instr, _ := pBuf.GetText(start, end, false)
		targets := []string{}
		for _, w := range strings.Split(instr, " ") {
			if strings.HasPrefix(w, "@") { targets = append(targets, strings.TrimPrefix(w, "@")) }
		}

		currCtx, _ := builder.BuildSelectiveContext(".", targets)
		ctxJS, _ := json.Marshal(currCtx)
		go func() {
			updated, err := browser.ProcessWithAI("TASK: " + instr + "\n\nSTATE: " + string(ctxJS))
			glib.IdleAdd(func() {
				if err != nil { buf.SetText("// Error: " + err.Error()); return }
				apply.ApplyPatch(".", updated)
				buf.SetText("// Patch applied successfully")
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

func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	l.SetMarginStart(10)
	box.PackStart(l, false, false, 5)
}

func applyCSS() {
	provider, _ := gtk.CssProviderNew()
	provider.LoadFromPath("assets/style.css")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
