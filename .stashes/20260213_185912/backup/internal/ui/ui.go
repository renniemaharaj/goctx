package ui

import (
	"encoding/json"
	"goctx/internal/builder"
	"os"
	"path/filepath"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx Editor")
	win.SetDefaultSize(1200, 800)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sidebar.SetSizeRequest(300, -1)
	sctx, _ := sidebar.GetStyleContext()
	sctx.AddClass("sidebar")

	label(sidebar, "EXPLORER: STASHES")
	stashList, _ := gtk.ListBoxNew()
	refreshStashes(stashList)
	sidebar.PackStart(stashList, true, true, 0)

	editorBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	bar, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	bar.SetMarginTop(10)
	bar.SetMarginBottom(10)
	bar.SetMarginStart(10)

	btnGen := newBtn("Build Context")
	btnAI := newBtn("Ask AI (Rod)")
	btnApply := newBtn("Apply Manual")

	bar.PackStart(btnGen, false, false, 0)
	bar.PackStart(btnAI, false, false, 0)
	bar.PackStart(btnApply, false, false, 0)

	editorBox.PackStart(bar, false, false, 0)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetLeftMargin(10)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	editorBox.PackStart(sw, true, true, 0)

	promptEntry, _ := gtk.EntryNew()
	promptEntry.SetPlaceholderText("Describe changes...")
	editorBox.PackEnd(promptEntry, false, false, 10)

	// ACTIONS
	btnGen.Connect("clicked", func() {
		ctx, _ := builder.BuildContext(".")
		js, _ := json.MarshalIndent(ctx, "", "  ")
		buf.SetText(string(js))
	})

	hbox.PackStart(sidebar, false, false, 0)
	hbox.PackStart(editorBox, true, true, 0)
	win.Add(hbox)
	win.ShowAll()
	gtk.Main()
}

func refreshStashes(list *gtk.ListBox) {
	list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
	filepath.Walk(".stashes", func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && path != ".stashes" && filepath.Dir(path) == ".stashes" {
			row, _ := gtk.ListBoxRowNew()
			hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
			lbl, _ := gtk.LabelNew(filepath.Base(path))
			btn := newBtn("Apply")
			btn.SetSizeRequest(60, 20)
			hbox.PackStart(lbl, true, true, 5)
			hbox.PackEnd(btn, false, false, 5)
			row.Add(hbox)
			list.Add(row)
		}
		return nil
	})
	list.ShowAll()
}

func label(box *gtk.Box, text string) {
	l, _ := gtk.LabelNew(text)
	l.SetXAlign(0)
	l.SetMarginStart(10)
	sctx, _ := l.GetStyleContext()
	sctx.AddClass("header")
	box.PackStart(l, false, false, 5)
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
