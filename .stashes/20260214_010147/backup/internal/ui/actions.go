package ui

import (
	"encoding/json"
	"strings"
	"time"

	"goctx/internal/model"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
)

func (a *App) startClipboardWatcher() {
	ticker := time.NewTicker(1 * time.Second)
	var last string

	go func() {
		for range ticker.C {
			clip, err := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
			if err != nil { continue }

			text, err := clip.WaitForText()
			if err != nil || text == "" { continue }
			if text == last { continue }
			if !strings.Contains(text, "{\"") { continue }

			var patch model.ProjectOutput
			if err := json.Unmarshal([]byte(text), &patch); err != nil { continue }
			if len(patch.Files) == 0 { continue }

			a.mu.Lock()
			a.pendingPatches = append(a.pendingPatches, patch)
			a.mu.Unlock()

			last = text
		}
	}()
}
