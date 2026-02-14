package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/glib"
	"github.com/webview/webview_go"
	"os"
	"path/filepath"
	"regexp"
)

var currentPayload string

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Orchestrator")
	win.SetDefaultSize(1200, 800)
	win.Connect("destroy", gtk.MainQuit)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sidebar.SetSizeRequest(250, -1)
	label(sidebar, "STASHES")
	list, _ := gtk.ListBoxNew()

	mainStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	mainStack.SetMarginStart(20)
	mainStack.SetMarginEnd(20)

	label(mainStack, "PROJECT STATISTICS")
	statsView, _ := gtk.TextViewNew()
	statsView.SetEditable(false)
	statsView.SetCanFocus(false)
	statsBuf, _ := statsView.GetBuffer()
	mainStack.PackStart(statsView, true, true, 0)

	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Context")
	btnCopy := newBtn("Copy Context")
	btnChat := newBtn("Chat")
	btnPasteApply := newBtn("Apply Clipboard")
	
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnCopy, true, true, 0)
	btnBox.PackStart(btnChat, true, true, 0)
	btnBox.PackStart(btnPasteApply, true, true, 0)
	mainStack.PackStart(btnBox, false, false, 10)

	list.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil { return }
		lblWidget, _ := row.GetChild()
		lbl, _ := lblWidget.(*gtk.Label)
		txt, _ := lbl.GetText()
		path := filepath.Join(".stashes", txt, "patch.json")
		data, _ := os.ReadFile(path)
		currentPayload = string(data)
		statsBuf.SetText(fmt.Sprintf("Loaded Stash: %s\nSize: %d bytes", txt, len(data)))
	})

	btnBuild.Connect("clicked", func() {
		statsBuf.SetText("Analyzing project...")
		go func() {
			out, _ := builder.BuildSelectiveContext(".", nil)
			js, _ := json.Marshal(out)
			currentPayload = string(js)
			
			stats := fmt.Sprintf("Files Indexed: %d\nEstimated Tokens: %d\nTree Size: %d characters", 
				len(out.Files), out.EstimatedTokens, len(out.ProjectTree))
			
			glib.IdleAdd(func() {
				statsBuf.SetText(stats)
			})
		}()
	})

	btnCopy.Connect("clicked", func() {
		if currentPayload == "" { return }
		clipboard, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clipboard.SetText(currentPayload)
	})

	btnChat.Connect("clicked", func() {
		// Running webview without the defer destroy inside the goroutine to prevent premature exit
		go func() {
			w := webview.New(false)
			w.SetTitle("AI Chat")
			w.SetSize(1200, 900, webview.Hint(0))
			w.Navigate("https://aistudio.google.com")
			w.Run()
		}()
	})

	btnPasteApply.Connect("clicked", func() {
		clipboard, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		text, _ := clipboard.WaitForText()
		re := regexp.MustCompile(`(?s)\{.*\"files\".*\}`)
		match := re.FindString(text)
		if match != "" {
			var patch model.ProjectOutput
			json.Unmarshal([]byte(match), &patch)
			apply.ApplyPatch(".", patch)
			refreshStashes(list)
			statsBuf.SetText("Patch applied successfully")
		}
	})

	hbox.PackStart(sidebar, false, false, 0)
	sidebar.PackStart(list, true, true, 0)
	hbox.PackStart(mainStack, true, true, 0)
	refreshStashes(list)

	win.Add(hbox)
	win.ShowAll()
	gtk.Main()
}

func refreshStashes(list *gtk.ListBox) {
	glib.IdleAdd(func() bool {
		list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
		filepath.Walk(".stashes", func(path string, info os.FileInfo, err error) error {
			if err == nil && info.IsDir() && path != ".stashes" && filepath.Dir(path) == ".stashes" {
				row, _ := gtk.ListBoxRowNew()
				lbl, _ := gtk.LabelNew(filepath.Base(path))
				row.Add(lbl)
				list.Add(row)
			}
			return nil
		})
		list.ShowAll()
		return false
	})
}

func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) { l, _ := gtk.LabelNew(t); l.SetXAlign(0); box.PackStart(l, false, false, 5) }
func applyCSS() {
	provider, _ := gtk.CssProviderNew()
	provider.SetData("label { font-weight: bold; padding: 5px; } textView { font-family: monospace; font-size: 14px; }")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
