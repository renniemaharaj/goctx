package ui

import (
	"encoding/json"
	"goctx/internal/builder"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/webkit2"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx Hybrid Orchestrator")
	win.SetDefaultSize(1400, 1000)
	win.Connect("destroy", gtk.MainQuit)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// --- SIDEBAR ---
	sidebar := createSidebar()
	hbox.PackStart(sidebar, false, false, 0)

	// --- MAIN STACK ---
	mainStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	mainStack.SetMarginStart(15)
	mainStack.SetMarginEnd(15)

	// 1. Context Editor (READ ONLY)
	label(mainStack, "PROJECT CONTEXT (Read-Only Console)")
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	tv.SetEditable(false) // CRITICAL: Fixes sluggishness
	tv.SetCursorVisible(false)
	buf, _ := tv.GetBuffer()
	sw.SetSizeRequest(-1, 250)
	sw.Add(tv)
	mainStack.PackStart(sw, false, false, 0)

	// 2. Control Buttons
	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Context")
	btnCopy := newBtn("📋 Copy to Clipboard")
	btnClear := newBtn("🧹 Clear")
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnCopy, true, true, 0)
	btnBox.PackStart(btnClear, false, false, 0)
	mainStack.PackStart(btnBox, false, false, 5)

	// 3. Web View (AI Chat Environment)
	label(mainStack, "AI CHAT ENVIRONMENT")
	wv := webkit2.NewWebView()
	wv.LoadURI("https://aistudio.google.com")
	wvScrolled, _ := gtk.ScrolledWindowNew(nil, nil)
	wvScrolled.Add(wv)
	mainStack.PackStart(wvScrolled, true, true, 0)

	// --- LOGIC ---
	btnBuild.Connect("clicked", func() {
		buf.SetText("// Building...")
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

	btnClear.Connect("clicked", func() { buf.SetText("") })

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
	// Logic for stashes removed for brevity but follows same pattern
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
