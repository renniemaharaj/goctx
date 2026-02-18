package ui

import (
	"fmt"
	"goctx/internal/config"
	"goctx/internal/git"
	"goctx/internal/runner"
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

	// Periodic Workspace Status Refresh (Every 5s)
	go func() {
		for {
			time.Sleep(5 * time.Second)
			glib.IdleAdd(func() {
				refreshGitState()
			})
		}
	}()

	// Periodic Build/Test Verification Loop (Every 30s)
	go func() {
		for {
			runVerification("build", false)
			runVerification("test", false)
			time.Sleep(30 * time.Second)
		}
	}()
}

func refreshGitState() {
	if btnCommit == nil {
		return
	}
	hasChanges := git.IsDirty(".")
	btnCommit.SetSensitive(hasChanges)

	currentCount := countCommits()
	if currentCount != lastHistoryCount {
		refreshHistory(historyPanel.List)
		lastHistoryCount = currentCount
	}
}

func runVerification(mode string, verbose bool) {
	cfg, _ := config.Load(".")
	var cmd string
	var btn *gtk.Button

	if mode == "build" {
		cmd = cfg.Scripts.Build
		btn = btnRunBuild
	} else {
		cmd = cfg.Scripts.Test
		btn = btnRunTest
	}

	if cmd == "" || btn == nil {
		return
	}

	if verbose {
		glib.IdleAdd(func() {
			isLoading = true
			statsBuf.SetText("")
			statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("=== MANUAL %s STARTING ===\n", strings.ToUpper(mode)), getTag("header"))
			updateStatus(statusLabel, "Running "+mode+"...")
		})
	}

	// Use internal runner to execute and stream logs if verbose
	out, err := runner.Run(".", cmd, func(line string) {
		if verbose {
			glib.IdleAdd(func() {
				statsBuf.Insert(statsBuf.GetEndIter(), line+"\n")
				// Auto-scroll to keep logs visible
				mark := statsBuf.CreateMark("bottom", statsBuf.GetEndIter(), false)
				statsView.ScrollToMark(mark, 0.0, true, 0.0, 1.0)
			})
		}
	})

	glib.IdleAdd(func() {
		ctx, _ := btn.GetStyleContext()
		ctx.RemoveClass("btn-success")
		ctx.RemoveClass("btn-failure")

		if err != nil {
			ctx.AddClass("btn-failure")
			btn.SetTooltipText(fmt.Sprintf("Last %s failed: %v", mode, err))
			if verbose {
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("\nFAILED: %v\n", err), getTag("deleted"))
				updateStatus(statusLabel, mode+" failed")
			}
		} else {
			ctx.AddClass("btn-success")
			btn.SetTooltipText(fmt.Sprintf("Last %s passed", mode))
			if verbose {
				successMsg := fmt.Sprintf("\nSUCCESS: %s completed successfully.\n", strings.ToUpper(mode))
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), successMsg, getTag("added"))
				updateStatus(statusLabel, mode+" passed")
			}
		}

		if verbose {
			isLoading = false
		} else if err != nil && !isLoading {
			// If background check failed and the user is not currently looking at something else
			updateStatus(statusLabel, fmt.Sprintf("Background %s failed", mode))
			RenderError(fmt.Errorf("%s output:\n%s", mode, string(out)))
		}
	})
}
