package ui

import (
	"fmt"
	"time"

	"github.com/gotk3/gotk3/gtk"
)

// ActionPanel represents a standardized list container with a label and scroll area
type ActionPanel struct {
	Container *gtk.Box
	List      *gtk.ListBox
}

// NewActionPanel creates a labeled, scrollable list box used for Stashes and Patches
func NewActionPanel(title string, height int, onEmptyClick func()) *ActionPanel {
	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)

	lbl, _ := gtk.LabelNew(title)
	lbl.SetXAlign(0)
	box.PackStart(lbl, false, false, 0)

	list, _ := gtk.ListBoxNew()
	eb, _ := gtk.EventBoxNew()
	sw, _ := gtk.ScrolledWindowNew(nil, nil)

	if height > 0 {
		sw.SetSizeRequest(-1, height)
	}

	sw.Add(list)
	eb.Add(sw)

	if onEmptyClick != nil {
		eb.Connect("button-press-event", onEmptyClick)
	}

	box.PackStart(eb, true, true, 0)

	return &ActionPanel{
		Container: box,
		List:      list,
	}
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
