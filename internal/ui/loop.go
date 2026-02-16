package ui

import (
	"os/exec"
	"strings"
	"time"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func backgroundMonitoringLoop() {
	// Background Monitoring Loop
	go func() {
		for {
			time.Sleep(2 * time.Second)
			glib.IdleAdd(func() {
				if btnCommit == nil {
					return
				}

				stat, _ := exec.Command("git", "status", "--porcelain").Output()
				hasChanges := len(strings.TrimSpace(string(stat))) > 0
				btnCommit.SetVisible(hasChanges)
				if !hasChanges {
					btnCommit.SetSensitive(false)
				}

				currentCount := countCommits()
				if currentCount != lastHistoryCount {
					refreshHistory(historyPanel.List)
					lastHistoryCount = currentCount
				}

				clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
				text, _ := clip.WaitForText()
				if text != "" && text != lastClipboard {
					lastClipboard = text
					processClipboard(text)
				}
			})
		}
	}()
}
