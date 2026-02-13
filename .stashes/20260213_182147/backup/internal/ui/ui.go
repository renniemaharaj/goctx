package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/builder"
	"github.com/gotk3/gotk3/gtk"
)

func Run() {
	gtk.Init(nil)
	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Pro")
	win.SetDefaultSize(1000, 600)
	win.Connect("destroy", gtk.MainQuit)

	paned, _ := gtk.PanedNew(gtk.ORIENTATION_HORIZONTAL)
	
	// Sidebar
	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	sidebar.SetSizeRequest(250, -1)
	label, _ := gtk.LabelNew("PROJ_CTX")
	btnRefresh, _ := gtk.ButtonNewWithLabel("Regenerate JSON")
	sidebar.PackStart(label, false, false, 10)
	sidebar.PackStart(btnRefresh, false, false, 5)

	// Editor View
	vbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	vbox.PackStart(sw, true, true, 0)

	paned.Pack1(sidebar, false, false)
	paned.Pack2(vbox, true, false)

	btnRefresh.Connect("clicked", func() {
		out, err := builder.BuildContext(".")
		if err != nil {
			buf.SetText(fmt.Sprintf("Error: %v", err))
			return
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		buf.SetText(string(data))
	})

	win.Add(paned)
	win.ShowAll()
	gtk.Main()
}
