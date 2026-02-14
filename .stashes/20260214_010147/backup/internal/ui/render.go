package ui

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

func (a *App) setupDiffPane(textView *gtk.TextView) {
	textView.SetEditable(false)
	textView.SetCursorVisible(false)

	textView.Connect("button-press-event", func(tv *gtk.TextView, ev *gdk.Event) bool {
		a.mu.Lock()
		a.selectedIndex = -1
		a.mu.Unlock()

		buffer, _ := tv.GetBuffer()
		buffer.SelectRange(buffer.GetStartIter(), buffer.GetStartIter())

		return false
	})
}
