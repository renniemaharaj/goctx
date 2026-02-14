package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/builder"
	"goctx/internal/model"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	webview "github.com/webview/webview_go"
)

var activeContext model.ProjectOutput
var currentPayload string

func Run() {
	gtk.Init(nil)

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Manager")
	win.SetDefaultSize(1400, 900)
	win.Connect("destroy", gtk.MainQuit)

	hmain, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// LEFT: VERTICAL BUTTON STACK
	leftBar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	leftBar.SetMarginStart(10)
	leftBar.SetMarginEnd(10)
	leftBar.SetMarginTop(10)

	btnBuild := newBtn("Build Context")
	btnCopy := newBtn("Copy Context")
	btnPaste := newBtn("Apply Patch")

	leftBar.PackStart(btnBuild, false, false, 0)
	leftBar.PackStart(btnCopy, false, false, 0)
	leftBar.PackStart(btnPaste, false, false, 0)

	label(leftBar, "STASHES")
	stashList, _ := gtk.ListBoxNew()
	leftBar.PackStart(stashList, true, true, 0)

	// RIGHT: SPLIT VIEW (STATS TOP / WEB BOTTOM)
	rightStack, _ := gtk.PanedNew(gtk.ORIENTATION_VERTICAL)

	// Stats Area
	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ := gtk.TextViewNew()
	statsView.SetEditable(false)
	statsBuf, _ := statsView.GetBuffer()
	statsScroll.Add(statsView)
	rightStack.Pack1(statsScroll, true, false)

	// Webview Logic Panel
	webContainer, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	urlEntry, _ := gtk.EntryNew()
	urlEntry.SetPlaceholderText("Paste AI URL or JSON content here...")
	webContainer.PackStart(urlEntry, false, false, 5)

	// Placeholder for where the Webview lives (it is a native overlay)
	webArea, _ := gtk.DrawingAreaNew()
	webArea.SetSizeRequest(-1, 400)
	webContainer.PackStart(webArea, true, true, 0)

	rightStack.Pack2(webContainer, true, true)

	// LOGIC: BUILD
	btnBuild.Connect("clicked", func() {
		statsBuf.SetText("Building...")
		go func() {
			out, err := builder.BuildSelectiveContext(".", nil)
			if err == nil {
				activeContext = out
				js, _ := json.Marshal(out)
				currentPayload = string(js)
				glib.IdleAdd(func() {
					statsBuf.SetText(formatStats(activeContext))
				})
			}
		}()
	})

	// LOGIC: COPY
	btnCopy.Connect("clicked", func() {
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(currentPayload)
	})

	// LOGIC: WEBVIEW INTEGRATION
	urlEntry.Connect("activate", func() {
		txt, _ := urlEntry.GetText()
		if strings.HasPrefix(txt, "http") {
			go launchWebview(txt)
		} else {
			// If it is not a URL, treat it as a JSON input monitor
			monitorInput(txt, statsBuf)
		}
	})

	hmain.PackStart(leftBar, false, false, 0)
	hmain.PackStart(rightStack, true, true, 0)

	refreshStashes(stashList)
	win.Add(hmain)
	win.ShowAll()
	gtk.Main()
}

func formatStats(p model.ProjectOutput) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("PROJ: %d FILES\n", len(p.Files)))
	sb.WriteString(fmt.Sprintf("TOKENS: %d\n", p.EstimatedTokens))
	sb.WriteString("\nDIRECTORY STRUCTURE:\n")
	sb.WriteString(p.ProjectTree)
	return sb.String()
}

func monitorInput(input string, buf *gtk.TextBuffer) {
	re := regexp.MustCompile(`(?s)\{.*\"files\".*\}`)
	match := re.FindString(input)
	if match != "" {
		var patch model.ProjectOutput
		json.Unmarshal([]byte(match), &patch)
		list := "FILES TO CHANGE:\n"
		for f := range patch.Files {
			list += "- " + f + "\n"
		}
		buf.SetText(list + "\nClick Apply Patch to finalize.")
	}
}

func launchWebview(url string) {
	glib.IdleAdd(func() {
		w := webview.New(false)
		w.SetTitle("AI Interface")
		w.SetSize(1000, 800, webview.Hint(0))
		w.Navigate(url)
		w.Run()
	})
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
func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	box.PackStart(l, false, false, 5)
}
