package ui

import (
	"encoding/json"
	"goctx/internal/builder"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx Hybrid Orchestrator")
	win.SetDefaultSize(1400, 1000)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// --- SIDEBAR ---
	sidebar := createSidebar()
	hbox.PackStart(sidebar, false, false, 0)

	// --- MAIN STACK ---
	mainStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	mainStack.SetMarginStart(15)
	mainStack.SetMarginEnd(15)

	label(mainStack, "PROJECT CONTEXT (Read-Only Optimized)")
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	tv.SetEditable(false)         // No CPU waste on cursor/input tracking
	tv.SetWrapMode(gtk.WRAP_NONE) // Wrap kills performance on large lines
	buf, _ := tv.GetBuffer()
	sw.SetSizeRequest(-1, 200)
	sw.Add(tv)
	mainStack.PackStart(sw, false, false, 0)

	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Context")
	btnCopy := newBtn("📋 Copy")
	btnChat := newBtn("🌐 Open AI Chat")
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnCopy, true, true, 0)
	btnBox.PackStart(btnChat, true, true, 0)
	mainStack.PackStart(btnBox, false, false, 5)

	// Logic
	btnBuild.Connect("clicked", func() {
		buf.SetText("// Indexing files...")
		go func() {
			out, _ := builder.BuildSelectiveContext(".", nil)
			js, _ := json.MarshalIndent(out, "", "  ")
			finalStr := string(js)
			glib.IdleAdd(func() {
				buf.SetText(finalStr)
			})
		}()
	})

	btnChat.Connect("clicked", func() {
		// Launching a separate webview window to keep the main GUI fluid
		go func() {
			w := webview.New(false)
			defer w.Destroy()
			w.SetTitle("AI Studio / Claude / ChatGPT")
			w.SetSize(1200, 900, webview.HintNone)
			w.Navigate("https://aistudio.google.com")
			w.Run()
		}()
	})

	btnCopy.Connect("clicked", func() {
		start, end := buf.GetBounds()
		text, _ := buf.GetText(start, end, false)
		clipboard, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clipboard.SetText(text)
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
func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	box.PackStart(l, false, false, 5)
}
func applyCSS() {
	provider, _ := gtk.CssProviderNew()
	provider.LoadFromPath("assets/style.css")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
