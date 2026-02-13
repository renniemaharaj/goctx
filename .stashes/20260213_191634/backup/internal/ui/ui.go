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
	win.SetTitle("● GoCtx Patch Engine")
	win.SetDefaultSize(1200, 800)
	win.Connect("destroy", gtk.MainQuit)

	vbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	vbox.PackStart(sw, true, true, 0)

	label(vbox, "PROMPT (@filename to target)")
	pView, _ := gtk.TextViewNew()
	pView.SetWrapMode(gtk.WRAP_WORD)
	pBuf, _ := pView.GetBuffer()
	vbox.PackStart(pView, false, false, 5)

	btnAI := newBtn("🚀 PATCH VIA AI")
	vbox.PackStart(btnAI, false, false, 10)

	btnAI.Connect("clicked", func() {
		start, end := pBuf.GetBounds()
		instr, _ := pBuf.GetText(start, end, false)
		targets := []string{}
		for _, w := range strings.Split(instr, " ") {
			if strings.HasPrefix(w, "@") { targets = append(targets, strings.TrimPrefix(w, "@")) }
		}

		// Ensure package reference is solid
		currCtx, _ := builder.BuildSelectiveContext(".", targets)
		ctxJS, _ := json.Marshal(currCtx)

		go func() {
			updated, err := browser.ProcessWithAI("TASK: " + instr + "\n\nJSON: " + string(ctxJS))
			glib.IdleAdd(func() {
				if err != nil {
					buf.SetText("// AI Error: " + err.Error())
					return
				}
				apply.ApplyPatch(".", updated)
				buf.SetText("// Successfully applied patch to: " + strings.Join(targets, ", "))
			})
		}()
	})

	win.Add(vbox)
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
	box.PackStart(l, false, false, 2)
}

func applyCSS() {
	provider, _ := gtk.CssProviderNew()
	provider.LoadFromPath("assets/style.css")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
