package ui

import (
	"fmt"
	"goctx/internal/builder"
	"goctx/internal/model"
	"os"
	"sort"
	"unicode/utf8"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	activeContext  model.ProjectOutput
	statsBuf       *gtk.TextBuffer
	win            *gtk.Window
)

func Run() {
	gtk.Init(nil)
	win, _ = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Manager v2.3")
	win.SetDefaultSize(1200, 800)
	win.Connect("destroy", gtk.MainQuit)

	grid, _ := gtk.GridNew()
	grid.SetColumnSpacing(10)
	grid.SetRowSpacing(10)
	grid.SetBorderWidth(10)

	sidebar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	sidebar.SetSizeRequest(250, -1)

	btnBuild := newBtn("BUILD CONTEXT")
	sidebar.PackStart(btnBuild, false, false, 0)
	grid.Attach(sidebar, 0, 0, 1, 1)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetHExpand(true)
	sw.SetVExpand(true)
	sw.SetPolicy(gtk.POLICY_AUTOMATIC, gtk.POLICY_AUTOMATIC)

	statsView, _ := gtk.TextViewNew()
	statsView.SetMonospace(true)
	statsView.SetEditable(false)
	
	statsBuf, _ = statsView.GetBuffer()
	setupTags(statsBuf)
	sw.Add(statsView)
	grid.Attach(sw, 1, 0, 1, 1)

	btnBuild.Connect("clicked", func() {
		go func() {
			out, err := builder.BuildSelectiveContext(".", "Snapshot")
			if err == nil {
				activeContext = out
				glib.IdleAdd(func() {
					renderDiff(activeContext)
				})
			}
		}()
	})

	win.Add(grid)
	win.ShowAll()
	gtk.Main()
}

func setupTags(b *gtk.TextBuffer) {
	tab, _ := b.GetTagTable()
	a, _ := gtk.TextTagNew("added"); a.SetProperty("foreground", "#2ecc71"); tab.Add(a)
	d, _ := gtk.TextTagNew("deleted"); d.SetProperty("foreground", "#e74c3c"); tab.Add(d)
	h, _ := gtk.TextTagNew("header"); h.SetProperty("foreground", "#3498db"); h.SetProperty("weight", 700); tab.Add(h)
	war, _ := gtk.TextTagNew("warn"); war.SetProperty("foreground", "#f39c12"); war.SetProperty("style", gtk.STYLE_ITALIC); tab.Add(war)
}

func renderDiff(p model.ProjectOutput) {
	statsBuf.SetText("")
	dmp := diffmatchpatch.New()
	
	// Get sorted keys for deterministic display
	var keys []string
	for k := range p.Files { keys = append(keys, k) }
	sort.Strings(keys)

	renderCount := 0
	const limit = 10

	for _, path := range keys {
		if renderCount >= limit { break }

		content := p.Files[path]
		if !utf8.ValidString(content) { continue }

		statsBuf.InsertWithTag(statsBuf.GetEndIter(), "\nFILE: "+path+"\n", getTag("header"))
		old, _ := os.ReadFile(path)
		diffs := dmp.DiffMain(string(old), content, false)
		
		for _, d := range diffs {
			if !utf8.ValidString(d.Text) { continue }
			switch d.Type {
			case diffmatchpatch.DiffInsert: statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag("added"))
			case diffmatchpatch.DiffDelete: statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag("deleted"))
			default: statsBuf.Insert(statsBuf.GetEndIter(), d.Text)
			}
		}
		renderCount++
	}

	if len(keys) > limit {
		msg := fmt.Sprintf("\n\n--- TRUNCATED: Displaying %d of %d files. Check ctx.json for full content. ---", limit, len(keys))
		statsBuf.InsertWithTag(statsBuf.GetEndIter(), msg, getTag("warn"))
	}
}

func getTag(n string) *gtk.TextTag { tab, _ := statsBuf.GetTagTable(); t, _ := tab.Lookup(n); return t }
func newBtn(l string) *gtk.Button { b, _ := gtk.ButtonNewWithLabel(l); return b }
