package ui

import (
	"encoding/json"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/browser"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/glib"
	"os"
	"path/filepath"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx AI Orchestrator")
	win.SetDefaultSize(1300, 900)
	win.Connect("destroy", gtk.MainQuit)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sidebar.SetSizeRequest(300, -1)
	sctx, err := sidebar.GetStyleContext()
	if err == nil { sctx.AddClass("sidebar") }

	label(sidebar, "EXPLORER: STASHES")
	stashList, _ := gtk.ListBoxNew()
	refreshStashes(stashList)
	sidebar.PackStart(stashList, true, true, 0)

	body, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	body.SetMarginStart(20)
	body.SetMarginEnd(20)

	label(body, "CONTEXT EDITOR (Optimized Rendering)")
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	tv.SetEditable(false) // Disable editing large blobs to save CPU
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	body.PackStart(sw, true, true, 0)

	label(body, "INSTRUCTIONS")
	pView, _ := gtk.TextViewNew()
	pView.SetSizeRequest(-1, 100)
	pBuf, _ := pView.GetBuffer()
	body.PackStart(pView, false, false, 0)

	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Context")
	btnApply := newBtn("🚀 APPLY")
	btnClear := newBtn("🧹 Clear")
	
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnApply, true, true, 0)
	btnBox.PackStart(btnClear, false, false, 0)
	body.PackStart(btnBox, false, false, 10)

	btnClear.Connect("clicked", func() { buf.SetText("") })

	btnBuild.Connect("clicked", func() {
		buf.SetText("// SYSTEM: Building... UI will stay responsive.")
		go func() {
			out, _ := builder.BuildSelectiveContext(".", nil)
			js, _ := json.MarshalIndent(out, "", "  ")
			finalStr := string(js)

			glib.IdleAdd(func() {
				// Performance Trick: Clear buffer before inserting massive text
				buf.SetText("")
				// Insert text
				buf.SetText(finalStr)
				println("Render Complete")
			})
		}()
	})

	btnApply.Connect("clicked", func() {
		start, end := pBuf.GetBounds()
		instr, _ := pBuf.GetText(start, end, false)
		go func() {
			glib.IdleAdd(func() { buf.SetText("[PIPELINE] Building...") })
			currCtx, _ := builder.BuildSelectiveContext(".", nil)
			ctxJS, _ := json.Marshal(currCtx)
			
			glib.IdleAdd(func() { buf.SetText("[PIPELINE] Browser interacting...") })
			updated, err := browser.ProcessWithAI("TASK: " + instr + "\n\nSTATE: " + string(ctxJS))
			
			glib.IdleAdd(func() {
				if err != nil { buf.SetText("// Error: " + err.Error()); return }
				apply.ApplyPatch(".", updated)
				refreshStashes(stashList)
				buf.SetText("// SUCCESS: Patch applied.")
			})
		}()
	})

	hbox.PackStart(sidebar, false, false, 0)
	hbox.PackStart(body, true, true, 0)
	win.Add(hbox)
	win.ShowAll()
	gtk.Main()
}

func refreshStashes(list *gtk.ListBox) {
	list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
	filepath.Walk(".stashes", func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && path != ".stashes" && filepath.Dir(path) == ".stashes" {
			row, _ := gtk.ListBoxRowNew()
			lbl, _ := gtk.LabelNew(filepath.Base(path))
			lbl.SetXAlign(0)
			row.Add(lbl)
			list.Add(row)
		}
		return nil
	})
	list.ShowAll()
}

func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) { l, _ := gtk.LabelNew(t); l.SetXAlign(0); box.PackStart(l, false, false, 5) }
func applyCSS() {
	provider, _ := gtk.CssProviderNew()
	provider.LoadFromPath("assets/style.css")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
