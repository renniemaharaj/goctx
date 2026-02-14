package ui

import (
	"encoding/json"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/glib"
	"github.com/webview/webview_go"
	"os"
	"path/filepath"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx Hybrid Orchestrator")
	win.SetDefaultSize(1400, 1000)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	sidebar := createSidebar()
	hbox.PackStart(sidebar, false, false, 0)

	mainStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	mainStack.SetMarginStart(15)
	mainStack.SetMarginEnd(15)

	label(mainStack, "PROJECT CONTEXT (Read-Only)")
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	tv.SetEditable(false)
	tv.SetWrapMode(gtk.WRAP_NONE)
	buf, _ := tv.GetBuffer()
	sw.SetSizeRequest(-1, 200)
	sw.Add(tv)
	mainStack.PackStart(sw, false, false, 0)

	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Context")
	btnCopy := newBtn("📋 Copy Context")
	btnChat := newBtn("🌐 Open AI Chat")
	btnPasteApply := newBtn("📥 Apply from Clipboard")
	
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnCopy, true, true, 0)
	btnBox.PackStart(btnChat, true, true, 0)
	btnBox.PackStart(btnPasteApply, true, true, 0)
	mainStack.PackStart(btnBox, false, false, 5)

	// --- Logic ---
	btnBuild.Connect("clicked", func() {
		buf.SetText("// Indexing...")
		go func() {
			out, _ := builder.BuildSelectiveContext(".", nil)
			js, _ := json.MarshalIndent(out, "", "  ")
			glib.IdleAdd(func() { buf.SetText(string(js)) })
		}()
	})

	btnCopy.Connect("clicked", func() {
		start, end := buf.GetBounds()
		text, _ := buf.GetText(start, end, false)
		clipboard, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clipboard.SetText(text)
	})

	btnChat.Connect("clicked", func() {
		go func() {
			w := webview.New(false)
			defer w.Destroy()
			w.SetTitle("AI Chat")
			w.SetSize(1200, 900, webview.HintNone)
			w.Navigate("https://aistudio.google.com")
			w.Run()
		}()
	})

	btnPasteApply.Connect("clicked", func() {
		clipboard, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clipboard.RequestText(func(_ *gtk.Clipboard, text string) {
			var patch model.ProjectOutput
			err := json.Unmarshal([]byte(text), &patch)
			if err != nil {
				buf.SetText("// Error: Clipboard content is not valid JSON patch\n" + err.Error())
				return
			}
			apply.ApplyPatch(".", patch)
			buf.SetText("// SUCCESS: Patch applied from clipboard!")
		})
	})

	hbox.PackStart(mainStack, true, true, 0)
	win.Add(hbox)
	win.ShowAll()
	gtk.Main()
}

func createSidebar() *gtk.Box {
	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sidebar.SetSizeRequest(250, -1)
	label(sidebar, "STASHES")
	list, _ := gtk.ListBoxNew()
	sidebar.PackStart(list, true, true, 0)
	return sidebar
}

func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) { l, _ := gtk.LabelNew(t); l.SetXAlign(0); box.PackStart(l, false, false, 5) }
func applyCSS() {
	provider, _ := gtk.CssProviderNew()
	provider.LoadFromPath("assets/style.css")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
