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
	"regexp"
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
	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sidebar.SetSizeRequest(250, -1)
	label(sidebar, "STASHES (Click to Load)")
	list, _ := gtk.ListBoxNew()

	mainStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	mainStack.SetMarginStart(15)
	mainStack.SetMarginEnd(15)

	label(mainStack, "PROJECT CONTEXT (Read-Only Optimized)")
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	tv.SetEditable(false)
	tv.SetWrapMode(gtk.WRAP_NONE)
	buf, _ := tv.GetBuffer()
	sw.SetSizeRequest(-1, 300)
	sw.Add(tv)
	mainStack.PackStart(sw, false, false, 0)

	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Context")
	btnCopy := newBtn("📋 Copy")
	btnChat := newBtn("🌐 Chat")
	btnPasteApply := newBtn("📥 Apply Clipboard")
	
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnCopy, true, true, 0)
	btnBox.PackStart(btnChat, true, true, 0)
	btnBox.PackStart(btnPasteApply, true, true, 0)
	mainStack.PackStart(btnBox, false, false, 5)

	// --- Sidebar Logic ---
	list.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil { return }
		lbl, _ := row.GetChild()
		txt, _ := lbl.(*gtk.Label).GetText()
		path := filepath.Join(".stashes", txt, "patch.json")
		data, _ := os.ReadFile(path)
		buf.SetText(string(data))
	})

	// --- Clipboard Logic (Fixed Method) ---
	btnPasteApply.Connect("clicked", func() {
		clipboard, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		text, err := clipboard.WaitForText()
		if err != nil || text == "" {
			buf.SetText("// Error: Clipboard is empty or inaccessible")
			return
		}

		re := regexp.MustCompile(`(?s)\{.*\"files\".*\}`)
		match := re.FindString(text)
		if match == "" { buf.SetText("// Error: No JSON found in clipboard"); return }

		var patch model.ProjectOutput
		if err := json.Unmarshal([]byte(match), &patch); err != nil {
			buf.SetText("// Error: " + err.Error()); return
		}

		apply.ApplyPatch(".", patch)
		refreshStashes(list)
		buf.SetText("// SUCCESS: Patch applied and sidebar updated!")
	})

	btnBuild.Connect("clicked", func() {
		go func() {
			out, _ := builder.BuildSelectiveContext(".", nil)
			js, _ := json.MarshalIndent(out, "", "  ")
			glib.IdleAdd(func() { buf.SetText(string(js)) })
		}()
	})

	btnChat.Connect("clicked", func() {
		go func() {
			w := webview.New(false)
			w.SetTitle("AI Chat")
			w.SetSize(1200, 900, webview.HintNone)
			w.Navigate("https://aistudio.google.com")
			w.Run()
		}()
	})

	refreshStashes(list)
	sidebar.PackStart(list, true, true, 0)
	hbox.PackStart(sidebar, false, false, 0)
	hbox.PackStart(mainStack, true, true, 0)
	win.Add(hbox)
	win.ShowAll()
	gtk.Main()
}

func refreshStashes(list *gtk.ListBox) {
	glib.IdleAdd(func() {
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
	})
}

func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
func label(box *gtk.Box, t string) { l, _ := gtk.LabelNew(t); l.SetXAlign(0); box.PackStart(l, false, false, 5) }
func applyCSS() {
	provider, _ := gtk.CssProviderNew()
	provider.LoadFromPath("assets/style.css")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
