package ui

import (
	"encoding/json"
	"fmt"
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
	win.SetTitle("GO-CTX :: PROJECT STATE MANAGER")
	win.SetDefaultSize(1100, 700)
	win.Connect("destroy", gtk.MainQuit)

	mainBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// SIDEBAR: Stashes and Tree
	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	sidebar.SetSizeRequest(280, -1)
	styleCtx, err := sidebar.GetStyleContext()
	if err == nil {
		styleCtx.AddClass("sidebar")
	}

	stashLabel, _ := gtk.LabelNew("--- STASHES ---")
	stashList, _ := gtk.ListBoxNew()
	refreshStashes(stashList)

	sidebar.PackStart(stashLabel, false, false, 10)
	sidebar.PackStart(stashList, true, true, 0)

	// MAIN CONTENT
	body, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	body.SetMarginStart(15)
	body.SetMarginEnd(15)

	header, _ := gtk.LabelNew("PROJECT STATE CONTROL")
	body.PackStart(header, false, false, 10)

	// Action Bar
	actions, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnGen, _ := gtk.ButtonNewWithLabel("GENERATE PROMPT")
	btnApply, _ := gtk.ButtonNewWithLabel("APPLY FROM CLIPBOARD")
	actions.PackStart(btnGen, true, true, 0)
	actions.PackStart(btnApply, true, true, 0)
	body.PackStart(actions, false, false, 5)

	// Editor Area
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	body.PackStart(sw, true, true, 0)

	// Logic: Generate Prompt
	btnGen.Connect("clicked", func() {
		out, _ := builder.BuildContext(".")
		jsonData, _ := json.MarshalIndent(out, "", "  ")
		prompt := fmt.Sprintf("Instruction: Refer to the following JSON object as the \"project state\". Reconstruct or modify the files as requested and return a modified project state JSON object.\n\nPROJECT STATE:\n%s", string(jsonData))
		buf.SetText(prompt)
	})

	// Logic: Apply from Paste
	// btnApply.Connect("clicked", func() {
	// 	clipboard, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
	// 	clipboard.RequestText(func(_ *gtk.Clipboard, text string) {
	// 		var input model.ProjectOutput
	// 		err := json.Unmarshal([]byte(text), &input)
	// 		if err != nil {
	// 			buf.SetText("ERROR: Invalid Project State JSON in clipboard")
	// 			return
	// 		}
	// 		apply.ApplyPatch(".", input)
	// 		buf.SetText("SUCCESS: Project state applied and stashed.")
	// 		refreshStashes(stashList)
	// 	})
	// })

	mainBox.PackStart(sidebar, false, false, 0)
	mainBox.PackStart(body, true, true, 0)
	win.Add(mainBox)
	win.ShowAll()
	gtk.Main()
}

func refreshStashes(list *gtk.ListBox) {
	list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
	filepath.Walk(".stashes", func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && path != ".stashes" {
			row, _ := gtk.ListBoxRowNew()
			lbl, _ := gtk.LabelNew(filepath.Base(path))
			row.Add(lbl)
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
