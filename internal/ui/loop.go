package ui

import (
	"fmt"
	"goctx/internal/config"
	"goctx/internal/git"
	"goctx/internal/renderer"
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
		// Use a lock-based guard to prevent overlapping refreshes if Git is slow
		var running bool
		for {
			time.Sleep(5 * time.Second)
			if running {
				continue
			}
			running = true
			glib.IdleAdd(func() {
				refreshGitState()
				running = false
			})
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

func runVerification(mode string, verbose bool, r *renderer.Renderer) {
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
			isLoadingState = true
			statsBuf.SetText("")
			statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("=== MANUAL %s STARTING ===\n", strings.ToUpper(mode)), r.GetTag("header"))
			updateStatus(statusLabel, "Running "+mode+"...")
		})
	}

	// Use internal runner to execute and stream logs if verbose
	out, err := runner.Run(".", cmd, func(line string) {
		if verbose {
			glib.IdleAdd(func() {
				statsBuf.Insert(statsBuf.GetEndIter(), line+"\n")
				// Auto-scroll to keep logs visible without shifting X-axis
				mark := statsBuf.CreateMark("bottom", statsBuf.GetEndIter(), false)
				statsView.ScrollToMark(mark, 0.0, false, 0.0, 1.0)
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
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("\nFAILED: %v\n", err), r.GetTag("deleted"))
				updateStatus(statusLabel, mode+" failed")
			}
		} else {
			ctx.AddClass("btn-success")
			btn.SetTooltipText(fmt.Sprintf("Last %s passed", mode))
			if verbose {
				successMsg := fmt.Sprintf("\nSUCCESS: %s completed successfully.\n", strings.ToUpper(mode))
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), successMsg, r.GetTag("added"))
				updateStatus(statusLabel, mode+" passed")
			}
		}

		if verbose {
			isLoadingState = false
		} else if err != nil && !isLoadingState {
			// If background check failed and the user is not currently looking at something else
			updateStatus(statusLabel, fmt.Sprintf("Background %s failed", mode))
			r.RenderError(fmt.Errorf("%s output:\n%s", mode, string(out)))
		}
	})
}
