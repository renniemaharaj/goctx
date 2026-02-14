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
	"strings"
)

var currentPayload string

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Management Console")
	win.SetDefaultSize(1400, 900)
	win.Connect("destroy", gtk.MainQuit)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// SIDEBAR
	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sidebar.SetSizeRequest(260, -1)
	label(sidebar, "STASH HISTORY")
	list, _ := gtk.ListBoxNew()

	// MAIN CONTENT
	mainStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	mainStack.SetMarginStart(20)
	mainStack.SetMarginEnd(20)
	mainStack.SetMarginTop(10)

	// Stats Panel (Upper Half)
	label(mainStack, "PROJECT OVERVIEW")
	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsScroll.SetSizeRequest(-1, 300)
	statsView, _ := gtk.TextViewNew()
	statsView.SetEditable(false)
	statsView.SetCanFocus(false)
	statsBuf, _ := statsView.GetBuffer()
	statsScroll.Add(statsView)
	mainStack.PackStart(statsScroll, false, false, 0)

	// Navigator Panel (Lower Half)
	label(mainStack, "WEB NAVIGATOR")
	navBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	urlEntry, _ := gtk.EntryNew()
	urlEntry.SetText("https://aistudio.google.com")
	btnGo := newBtn("Launch Chat")
	navBox.PackStart(urlEntry, true, true, 0)
	navBox.PackStart(btnGo, false, false, 0)
	mainStack.PackStart(navBox, false, false, 0)

	// Action Buttons
	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Context")
	btnCopy := newBtn("Copy Context")
	btnPasteApply := newBtn("Apply Patch")
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnCopy, true, true, 0)
	btnBox.PackStart(btnPasteApply, true, true, 0)
	mainStack.PackStart(btnBox, false, false, 10)

	// LOGIC
	list.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil { return }
		lblWidget, _ := row.GetChild()
		lbl, _ := lblWidget.(*gtk.Label)
		txt, _ := lbl.GetText()
		path := filepath.Join(".stashes", txt, "patch.json")
		data, _ := os.ReadFile(path)
		currentPayload = string(data)
		
		var p model.ProjectOutput
		json.Unmarshal(data, &p)
		statsBuf.SetText(formatStats(p, "Stash: "+txt))
	})

	btnBuild.Connect("clicked", func() {
		statsBuf.SetText("Scanning filesystem and calculating tokens...")
		go func() {
			out, _ := builder.BuildSelectiveContext(".", nil)
			js, _ := json.Marshal(out)
			currentPayload = string(js)
			
			glib.IdleAdd(func() {
				statsBuf.SetText(formatStats(out, "Active Session"))
			})
		}()
	})

	btnCopy.Connect("clicked", func() {
		if currentPayload == "" { return }
		clipboard, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clipboard.SetText(currentPayload)
	})

	btnGo.Connect("clicked", func() {
		target, _ := urlEntry.GetText()
		go func() {
			glib.IdleAdd(func() {
				w := webview.New(false)
				w.SetTitle("GoCtx AI Environment")
				w.SetSize(1200, 900, webview.Hint(0))
				w.Navigate(target)
				w.Run()
			})
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
			statsBuf.SetText("Update: Patch applied successfully to " + fmt.Sprint(len(patch.Files)) + " files.")
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

func formatStats(p model.ProjectOutput, title string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("--- %s ---\n", title))
	sb.WriteString(fmt.Sprintf("Files Indexed:    %d\n", len(p.Files)))
	sb.WriteString(fmt.Sprintf("Estimated Tokens: %d\n", p.EstimatedTokens))
	sb.WriteString("\n--- Project Tree ---\n")
	sb.WriteString(p.ProjectTree)
	return sb.String()
}

func refreshStashes(list *gtk.ListBox) {
	glib.IdleAdd(func() bool {
		list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
		os.MkdirAll(".stashes", 0755)
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
	provider.LoadFromData("label { font-weight: bold; color: #555; margin-bottom: 2px; } textView { font-family: monospace; font-size: 13px; padding: 15px; background: #fdfdfd; }")
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, provider, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}
