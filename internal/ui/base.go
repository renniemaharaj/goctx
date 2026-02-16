package ui

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gotk3/gotk3/gtk"
)

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
	statsBuf.SetText("")
	updateStatus(statusLabel, "Selection cleared")
}

func confirmAction(parent *gtk.Window, m string) bool {
	d := gtk.MessageDialogNew(parent, gtk.DIALOG_MODAL, gtk.MESSAGE_QUESTION, gtk.BUTTONS_YES_NO, m)
	r := d.Run()
	d.Destroy()
	return r == gtk.RESPONSE_YES
}

func updateStatus(lbl *gtk.Label, m string) {
	lbl.SetText(fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), m))
}

func newBtn(l string) *gtk.Button {
	b, _ := gtk.ButtonNewWithLabel(l)
	return b
}

func label(box *gtk.Box, t string) {
	l, _ := gtk.LabelNew(t)
	l.SetXAlign(0)
	box.PackStart(l, false, false, 0)
}
