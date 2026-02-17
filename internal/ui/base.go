package ui

import (
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/gotk3/gotk3/gtk"
)

func truncate(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}

	runes := []rune(s)
	return string(runes[:limit]) + "..."
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}

func clearAllSelections() {
	pendingPanel.List.UnselectAll()
	historyPanel.List.UnselectAll()
	resetView()
}

func resetView() {
	btnApplyPatch.SetSensitive(false)
	btnApplyCommit.SetSensitive(false)

	pathMu.Lock()
	currentEditingPath = ""
	pathMu.Unlock()

	isLoading = true
	statsBuf.SetText("")
	statsView.SetEditable(false)
	isLoading = false

	updateStatus(statusLabel, "Selection cleared")
}

func confirmAction(parent *gtk.Window, m string) bool {
	d := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_YES_NO, "%s", m)
	r := d.Run()
	d.Destroy()
	return r == gtk.RESPONSE_YES
}

func updateStatus(lbl *gtk.Label, m string) {
	lbl.SetText(fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), truncate(m, 50)))
}

func newBtn(l string) *gtk.Button {
	b, _ := gtk.ButtonNewWithLabel(l)
	return b
}

func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	l.SetMarginStart(10)
	l.SetMarginTop(8)
	l.SetMarginBottom(2)
	box.PackStart(l, false, false, 0)
}

// askStashOrApply returns: 1 for Stash & Apply, 0 for Apply Directly, -1 for Cancel
func askForString(parent *gtk.Window, title, defaultText string) (string, bool) {
	d := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_OTHER, gtk.BUTTONS_OK_CANCEL, "%s", title)

	content, _ := d.GetContentArea()
	entry, _ := gtk.EntryNew()
	entry.SetText(defaultText)
	entry.SetMarginStart(10)
	entry.SetMarginEnd(10)
	entry.Connect("activate", func() { d.Response(gtk.RESPONSE_OK) })

	content.Add(entry)
	content.ShowAll()

	resp := d.Run()
	text, _ := entry.GetText()
	d.Destroy()

	return text, resp == gtk.RESPONSE_OK
}

// askStashOrApply returns: 1 for Stash & Apply, 0 for Apply Directly, -1 for Cancel
func askStashOrApply(parent *gtk.Window) int {
	d := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_NONE, "Workspace is DIRTY. How would you like to proceed?")

	// Add the custom choices as buttons
	d.AddButton("Stash & Apply", gtk.RESPONSE_YES)
	d.AddButton("Apply Directly", gtk.RESPONSE_NO)
	d.AddButton("Cancel", gtk.RESPONSE_CANCEL)

	r := d.Run()
	d.Destroy()

	switch r {
	case gtk.RESPONSE_YES:
		return 1
	case gtk.RESPONSE_NO:
		return 0
	default:
		return -1
	}
}
