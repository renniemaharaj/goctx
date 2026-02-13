package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
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
	win.Connect("destroy", gtk.MainQuit)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// --- VS CODE SIDEBAR ---
	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sidebar.SetSizeRequest(260, -1)
	sctx, _ := sidebar.GetStyleContext()
	sctx.AddClass("sidebar")

	label(sidebar, "Explorer: Stashes")
	list, _ := gtk.ListBoxNew()
	refreshStashes(list)
	sidebar.PackStart(list, true, true, 0)

	// --- MAIN EDITOR AREA ---
	editorBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)

	// Top Toolbar
	bar, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	bar.SetMarginTop(10)
	bar.SetMarginBottom(10)
	bar.SetMarginStart(10)

	btnGen, _ := gtk.ButtonNewWithLabel("Build Context")
	btnApply, _ := gtk.ButtonNewWithLabel("Apply Change")
	bar.PackStart(btnGen, false, false, 0)
	bar.PackStart(btnApply, false, false, 0)

	editorBox.PackStart(bar, false, false, 0)

	// Output/Editor View
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetLeftMargin(10)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	editorBox.PackStart(sw, true, true, 0)

	// --- BOTTOM PROMPT INPUT ---
	inputBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)
	inputBox.SetMarginBottom(10)
	inputBox.SetMarginStart(10)
	inputBox.SetMarginEnd(10)
	
	label(inputBox, "AI Instruction Prompt")
	entry, _ := gtk.EntryNew()
	entry.SetPlaceholderText("e.g., Rewrite the UI to use a grid layout...")
	sctxE, _ := entry.GetStyleContext()
	sctxE.AddClass("input-area")
	inputBox.PackStart(entry, false, false, 0)
	
	editorBox.PackEnd(inputBox, false, false, 0)

	// LOGIC
	btnGen.Connect("clicked", func() {
		instructions, _ := entry.GetText()
		out, _ := builder.BuildContext(".")
		jsonData, _ := json.MarshalIndent(out, "", "  ")
		prompt := fmt.Sprintf("Task: %s\n\nRefer to the following JSON as the \"project state\". Respond only with the updated JSON project state.\n\n%s", instructions, string(jsonData))
		buf.SetText(prompt)
	})

	btnApply.Connect("clicked", func() {
		cb, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		text, err := cb.WaitForText()
		if err == nil && text != "" {
			var input model.ProjectOutput
			if json.Unmarshal([]byte(text), &input) == nil {
				apply.ApplyPatch(".", input)
				refreshStashes(list)
				buf.SetText("// State Applied Successfully")
			}
		}
	})

	hbox.PackStart(sidebar, false, false, 0)
	hbox.PackStart(editorBox, true, true, 0)
	win.Add(hbox)
	win.ShowAll()
	gtk.Main()
}

func label(box *gtk.Box, text string) {
	l, _ := gtk.LabelNew(text)
	l.SetXAlign(0)
	l.SetMarginStart(10)
	l.SetMarginTop(5)
	sctx, _ := l.GetStyleContext()
	sctx.AddClass("header")
	box.PackStart(l, false, false, 2)
}

func refreshStashes(list *gtk.ListBox) {
	list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
	filepath.Walk(".stashes", func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && path != ".stashes" && filepath.Dir(path) == ".stashes" {
			row, _ := gtk.ListBoxRowNew()
			lbl, _ := gtk.LabelNew(" 📁 " + filepath.Base(path))
			lbl.SetXAlign(0)
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
