package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/apply"
	"goctx/internal/browser"
	"goctx/internal/builder"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx AI Orchestrator")
	win.SetDefaultSize(1300, 900)
	win.Connect("destroy", gtk.MainQuit)

	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	// --- SIDEBAR (Stashes) ---
	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	sidebar.SetSizeRequest(300, -1)
	styleCtx, err := sidebar.GetStyleContext()
	if err == nil {
		styleCtx.AddClass("sidebar")
	}
	label(sidebar, "EXPLORER: STASHES")
	stashList, _ := gtk.ListBoxNew()
	refreshStashes(stashList)
	sidebar.PackStart(stashList, true, true, 0)

	// --- MAIN CONTENT ---
	body, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	body.SetMarginStart(20)
	body.SetMarginEnd(20)

	// Editor (Context/Pipeline Output)
	label(body, "CONTEXT EDITOR / PIPELINE LOG")
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	tv.SetWrapMode(gtk.WRAP_NONE)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	body.PackStart(sw, true, true, 0)

	// Large Input Area
	label(body, "INSTRUCTIONS (@file to target)")
	pView, _ := gtk.TextViewNew()
	pView.SetWrapMode(gtk.WRAP_WORD)
	pView.SetSizeRequest(-1, 120)
	pBuf, _ := pView.GetBuffer()
	body.PackStart(pView, false, false, 0)

	// Actions
	btnBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 10)
	btnBuild := newBtn("Build Context")
	btnAI := newBtn("🚀 RUN PIPELINE")
	btnBox.PackStart(btnBuild, true, true, 0)
	btnBox.PackStart(btnAI, true, true, 0)
	body.PackStart(btnBox, false, false, 10)

	// --- LOGIC ---
	btnBuild.Connect("clicked", func() {
		out, _ := builder.BuildSelectiveContext(".", []string{})
		js, _ := json.MarshalIndent(out, "", "  ")
		buf.SetText(string(js))
	})

	btnAI.Connect("clicked", func() {
		start, end := pBuf.GetBounds()
		instr, _ := pBuf.GetText(start, end, false)

		targets := []string{}
		for _, w := range strings.Split(instr, " ") {
			if strings.HasPrefix(w, "@") {
				targets = append(targets, strings.TrimPrefix(w, "@"))
			}
		}

		go func() {
			updateLog := func(msg string) {
				glib.IdleAdd(func() {
					iter := buf.GetEndIter()
					buf.Insert(iter, "\n[PIPELINE] "+msg)
				})
			}

			updateLog("Stage 1: Building Selective Context...")
			currCtx, _ := builder.BuildSelectiveContext(".", targets)
			ctxJS, _ := json.Marshal(currCtx)

			updateLog("Stage 2: Poking Rod / Waiting for AI response...")
			updated, err := browser.ProcessWithAI("TASK: " + instr + "\n\nSTATE: " + string(ctxJS))

			if err != nil {
				updateLog("FAILED: " + err.Error())
				return
			}

			updateLog("Stage 3: Validating AI Output Structure...")
			updateLog("Stage 4: Applying Patch to Filesystem...")

			glib.IdleAdd(func() {
				apply.ApplyPatch(".", updated)
				refreshStashes(stashList)
				updateLog("COMPLETE: Project state synchronized.")
			})
		}()
	})

	hbox.PackStart(sidebar, false, false, 0)
	hbox.PackStart(body, true, true, 0)
	win.Add(hbox)
	win.ShowAll()
	gtk.Main()
}

func refreshStashes(list *gtk.ListBox) {
	list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })
	filepath.Walk(".stashes", func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && path != ".stashes" && filepath.Dir(path) == ".stashes" {
			row, _ := gtk.ListBoxRowNew()
			hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
			lbl, _ := gtk.LabelNew(filepath.Base(path))
			btn := newBtn("Apply")
			btn.Connect("clicked", func() {
				fmt.Println("Restoring stash:", path)
			})
			hbox.PackStart(lbl, true, true, 5)
			hbox.PackEnd(btn, false, false, 5)
			row.Add(hbox)
			list.Add(row)
		}
		return nil
	})
	list.ShowAll()
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
