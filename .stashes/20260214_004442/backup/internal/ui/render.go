package ui

import "github.com/gotk3/gotk3/gtk"

func (a *App) setupDiffPane(textView *gtk.TextView) {
	textView.SetEditable(false)
	textView.SetCursorVisible(false)

	textView.Connect("button-press-event", func() {
		a.mu.Lock()
		defer a.mu.Unlock()

		a.selectedIndex = -1
	})
}
