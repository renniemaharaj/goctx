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
	win.SetDefaultSize(1200, 850)
	win.Connect("destroy", gtk.MainQuit)

	hmain, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// LEFT BAR
	leftBar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	leftBar.SetMarginStart(10)
	leftBar.SetMarginEnd(10)

	btnBuild := newBtn("Build Context")
	btnCopy := newBtn("Copy Context")
	btnPaste := newBtn("Apply Patch")

	leftBar.PackStart(btnBuild, false, false, 5)
	leftBar.PackStart(btnCopy, false, false, 5)
	leftBar.PackStart(btnPaste, false, false, 5)

	label(leftBar, "STASHES")
	swStash, _ := gtk.ScrolledWindowNew(nil, nil)
	stashList, _ := gtk.ListBoxNew()
	swStash.Add(stashList)
	leftBar.PackStart(swStash, true, true, 0)

	// RIGHT CONTENT
	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	rightStack.SetMarginStart(15)
	rightStack.SetMarginEnd(15)

	label(rightStack, "DASHBOARD")
	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ := gtk.TextViewNew()
	statsView.SetMonospace(true)
	statsView.SetEditable(false)
	statsBuf, _ := statsView.GetBuffer()
	statsScroll.Add(statsView)
	rightStack.PackStart(statsScroll, true, true, 0)

	// INPUT MONITOR
	inputBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	urlEntry, _ := gtk.EntryNew()
	urlEntry.SetPlaceholderText("URL to launch or JSON to parse...")
	btnRun := newBtn("Go")
	inputBox.PackStart(urlEntry, true, true, 0)
	inputBox.PackStart(btnRun, false, false, 0)
	rightStack.PackStart(inputBox, false, false, 10)

	// SELECTION LOGIC
	stashList.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		lblWidget, _ := row.GetChild()
		lbl, _ := lblWidget.(*gtk.Label)
		txt, _ := lbl.GetText()

		path := filepath.Join(".stashes", txt, "patch.json")
		data, err := os.ReadFile(path)
		if err == nil {
			currentPayload = string(data)
			var p model.ProjectOutput
			if err := json.Unmarshal(data, &p); err == nil {
				statsBuf.SetText(formatStats(p, "Stash: "+txt))
			}
		}
	})

	// BUILD LOGIC
	btnBuild.Connect("clicked", func() {
		statsBuf.SetText("Scanning project files...")
		go func() {
			out, err := builder.BuildSelectiveContext(".", nil)
			if err == nil {
				activeContext = out
				js, _ := json.Marshal(out)
				currentPayload = string(js)
				glib.IdleAdd(func() {
					statsBuf.SetText(formatStats(activeContext, "Current Workspace"))
				})
			}
		}()
	})

	btnRun.Connect("clicked", func() {
		txt, _ := urlEntry.GetText()
		if strings.HasPrefix(txt, "http") {
			go launchWebview(txt)
		} else {
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

func formatStats(p model.ProjectOutput, source string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("SOURCE: %s\n", source))
	sb.WriteString(fmt.Sprintf("TOKENS: %d\n", p.EstimatedTokens))
	sb.WriteString(fmt.Sprintf("FILES:  %d\n", len(p.Files)))
	sb.WriteString("\n--- DIRECTORY TREE ---\n")
	if p.ProjectTree != "" {
		sb.WriteString(p.ProjectTree)
	} else {
		sb.WriteString("[Tree Data Missing]")
	}

	sb.WriteString("\n\n--- FILE LIST ---\n")
	for f := range p.Files {
		sb.WriteString("  " + f + "\n")
	}
	return sb.String()
}

func monitorInput(input string, buf *gtk.TextBuffer) {
	re := regexp.MustCompile(`(?s)\{.*\"files\".*\}`)
	match := re.FindString(input)
	if match != "" {
		var patch model.ProjectOutput
		if err := json.Unmarshal([]byte(match), &patch); err == nil {
			buf.SetText(formatStats(patch, "Incoming AI Patch"))
		}
	}
}

func launchWebview(url string) {
	glib.IdleAdd(func() {
		w := webview.New(false)
		w.SetTitle("AI Chat")
		w.SetSize(1100, 800, webview.Hint(0))
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
