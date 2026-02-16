package ui

import "github.com/gotk3/gotk3/gtk"

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
