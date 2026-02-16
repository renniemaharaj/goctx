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
	clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
	clip.Connect("owner-change", func() {
		text, _ := clip.WaitForText()
		if text != "" && text != lastClipboard {
			lastClipboard = text
			processClipboard(text)
		}
	})

	go func() {
		for {
			time.Sleep(5 * time.Second)
			glib.IdleAdd(func() {
				refreshGitState()
			})
		}
	}()
}

func refreshGitState() {
	if btnCommit == nil { return }
	stat, _ := exec.Command("git", "status", "--porcelain").Output()
	hasChanges := len(strings.TrimSpace(string(stat))) > 0
	btnCommit.SetSensitive(hasChanges)

	currentCount := countCommits()
	if currentCount != lastHistoryCount {
		refreshHistory(historyPanel.List)
		lastHistoryCount = currentCount
	}
}
