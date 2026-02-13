package ui

import (
	"encoding/json"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/browser"
	"goctx/internal/model"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/glib"
	"strings"
)

func Run() {
	gtk.Init(nil)
	applyCSS()

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("● GoCtx Patch Mode")
	win.SetDefaultSize(1200, 850)

	body, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	body.SetMarginStart(20)
	body.SetMarginEnd(20)

	// View of what is being sent
	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	buf, _ := tv.GetBuffer()
	sw.Add(tv)
	body.PackStart(sw, true, true, 0)

	label(body, "AI INSTRUCTIONS (Target specific files with @filename)")
	promptView, _ := gtk.TextViewNew()
	pBuf, _ := promptView.GetBuffer()
	body.PackStart(promptView, false, false, 0)

	btnAI := newBtn("🚀 APPLY SMART PATCH")
	body.PackStart(btnAI, false, false, 20)

	btnAI.Connect("clicked", func() {
		start, end := pBuf.GetBounds()
		instruction, _ := pBuf.GetText(start, end, false)
		
		// Extract file mentions from prompt (e.g., "change @ui.go")
		targets := []string{}
		words := strings.Split(instruction, " ")
		for _, w := range words {
			if strings.HasPrefix(w, "@") {
				targets = append(targets, strings.TrimPrefix(w, "@"))
			}
		}

		// Build a selective context to save tokens/bandwidth
		currCtx, _ := builder.BuildSelectiveContext(".", targets)
		ctxJS, _ := json.Marshal(currCtx)
		
		fullPrompt := "TASK: " + instruction + "\n\nONLY return a JSON patch for these files:\n" + string(ctxJS)
		buf.SetText("// Sending partial state for: " + strings.Join(targets, ", "))
		
		go func() {
			updatedState, err := browser.ProcessWithAI(fullPrompt)
			glib.IdleAdd(func() {
				if err != nil {
					buf.SetText("// ERROR: " + err.Error())
					return
				}
				// apply.ApplyPatch handles partial file maps naturally
				apply.ApplyPatch(".", updatedState)
				buf.SetText("// PATCH APPLIED SUCCESS")
			})
		}()
	})

	win.Add(body)
	win.ShowAll()
	gtk.Main()
}
