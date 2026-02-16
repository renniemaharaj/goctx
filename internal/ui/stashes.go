package ui

import (
	"goctx/internal/stash"

	"github.com/gotk3/gotk3/gtk"
)

func countStashes() int {
	out, err := stash.GetStashes(".")
	if err != nil {
		return 0
	}
	return len(out)
}

func refreshStashes(list *gtk.ListBox) {
	list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })

	lines, err := stash.GetStashes(".")
	if err != nil {
		return
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		row, _ := gtk.ListBoxRowNew()
		lbl, _ := gtk.LabelNew(line)
		lbl.SetXAlign(0)
		row.Add(lbl)
		list.Add(row)
	}
	list.ShowAll()
}
