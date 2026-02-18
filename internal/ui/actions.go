package ui

import (
	"fmt"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/git"
	"goctx/internal/renderer"
	"os/exec"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

// bindEvents connects signals to logic. We pass in a generic interface for the renderer
func bindEvents(r *renderer.Renderer) {
	btnBuild.Connect("clicked", func() {
		limit := int(tokenScale.GetValue())
		smart := smartCheck.GetActive()
		go func() {
			selected := getCheckedFiles(treeStore)
			out, err := builder.BuildSelectiveContext(".", "Manual Build", selected, limit, smart)
			if err == nil {
				activeContext = out
				glib.IdleAdd(func() {
					pathMu.Lock()
					currentEditingPath = ""
					pathMu.Unlock()
					statsView.SetEditable(false)
					r.RenderSummary(activeContext)
					updateStatus(statusLabel, "Context built successfully")
				})
			}
		}()
	})

	btnKeys.Connect("clicked", func() {
		showKeyManager()
	})

	btnRunBuild.Connect("clicked", func() {
		go runVerification("build", true, r)
	})

	btnRunTest.Connect("clicked", func() {
		go runVerification("test", true, r)
	})

	btnCopy.Connect("clicked", func() {
		fullPrompt := builder.AI_PROMPT_HEADER + string(mustMarshal(activeContext))
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(fullPrompt)
		updateStatus(statusLabel, "System Prompt + Context copied")
	})

	btnCommit.Connect("clicked", handleCommitAction)
	btnApplyPatch.Connect("clicked", func() { handleApplyPatchAction(r) })
	btnApplyCommit.Connect("clicked", handleRestoreCommitAction)

	// List selections
	pendingPanel.List.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		historyPanel.List.UnselectAll()
		pathMu.Lock()
		currentEditingPath = ""
		pathMu.Unlock()
		statsView.SetEditable(false)
		idx := row.GetIndex()
		r.RenderDiff(pendingPatches[idx], "Pending Patch Preview")
		btnApplyPatch.SetSensitive(true)
		btnApplyCommit.SetSensitive(false)
	})

	historyPanel.List.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		pendingPanel.List.UnselectAll()
		pathMu.Lock()
		currentEditingPath = ""
		pathMu.Unlock()
		statsView.SetEditable(false)

		lblWidget, _ := row.GetChild()
		lbl, _ := lblWidget.(*gtk.Label)
		fullText, _ := lbl.GetText()
		parts := strings.Fields(fullText)
		if len(parts) > 0 {
			hash := parts[0]
			showCmd := exec.Command("git", "show", "--color=never", hash)
			out, _ := showCmd.Output()
			isLoadingState = true
			statsBuf.SetText("")
			statsBuf.InsertWithTag(statsBuf.GetEndIter(), "COMMIT PREVIEW: "+hash+"\n\n", r.GetTag("header"))
			statsBuf.Insert(statsBuf.GetEndIter(), string(out))
			isLoadingState = false
			btnApplyCommit.SetSensitive(true)
			btnApplyPatch.SetSensitive(false)
		}
	})

	mainTreeView.Connect("cursor-changed", func() {
		if isRefreshing {
			return
		}
		selection, _ := mainTreeView.GetSelection()
		_, iter, ok := selection.GetSelected()
		if ok {
			pendingPanel.List.UnselectAll()
			historyPanel.List.UnselectAll()
			pathVal, _ := treeStore.GetValue(iter, 2)
			pathRaw, _ := pathVal.GoValue()
			pathStr := pathRaw.(string)
			pathMu.Lock()
			currentEditingPath = pathStr
			pathMu.Unlock()
			isLoadingState = true
			statsView.SetEditable(strings.HasSuffix(pathStr, ".ctxignore"))
			r.RenderFile(pathStr)
			isLoadingState = false
		}
	})
}

func handleCommitAction() {
	defaultMsg := lastAppliedDesc
	if defaultMsg == "" {
		defaultMsg = "Commit Msg"
	}
	msg, ok := askForString(win, "Commit Message", defaultMsg)
	if !ok || strings.TrimSpace(msg) == "" {
		return
	}
	git.AddAll(".")
	if err := git.Commit(".", msg); err != nil {
		updateStatus(statusLabel, "Failed: "+err.Error())
	} else {
		updateStatus(statusLabel, "Committed")
		refreshHistory(historyPanel.List)
		lastAppliedDesc = ""
	}
}

func handleRestoreCommitAction() {
	row := historyPanel.List.GetSelectedRow()
	if row == nil {
		return
	}
	lblWidget, _ := row.GetChild()
	lbl, _ := lblWidget.(*gtk.Label)
	fullText, _ := lbl.GetText()
	parts := strings.Fields(fullText)
	if len(parts) > 0 {
		hash := parts[0]
		if confirmAction(win, "Restoring "+hash+" will overwrite current changes. Proceed?") {
			cmd := exec.Command("git", "checkout", hash, "--", ".")
			if err := cmd.Run(); err != nil {
				updateStatus(statusLabel, "Error: "+err.Error())
			} else {
				updateStatus(statusLabel, "Restored "+hash)
				refreshHistory(historyPanel.List)
			}
		}
	}
}

func handleApplyPatchAction(r interface {
	RenderGitStatus(root string)
	RenderError(err error)
	GetTag(n string) *gtk.TextTag
}) {
	row := pendingPanel.List.GetSelectedRow()
	if row == nil {
		return
	}
	idx := row.GetIndex()
	patchToApply := pendingPatches[idx]
	stat, _ := exec.Command("git", "status", "--porcelain").Output()
	isDirty := len(strings.TrimSpace(string(stat))) > 0
	shouldProceed := false
	if isDirty {
		choice := askStashOrApply(win)
		if choice == 1 {
			exec.Command("git", "stash", "push", "-m", "GoCtx: Pre-patch stash").Run()
			shouldProceed = true
		} else if choice == 0 {
			shouldProceed = true
		}
	} else {
		shouldProceed = confirmAction(win, "Apply selected patch?")
	}
	if shouldProceed {
		statsBuf.SetText("")
		isLoadingState = true
		header.SetSubtitle("Applying Patch...")
		go func() {
			err := apply.ApplyPatch(".", patchToApply, func(phase, desc, logLine string) {
				glib.IdleAdd(func() {
					if phase != "" {
						updateStatus(statusLabel, fmt.Sprintf("Phase: %s", phase))
						header.SetSubtitle(fmt.Sprintf("%s: %s", phase, desc))
						statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("\n--- %s ---\n", phase), r.GetTag("header"))
					}
					if logLine != "" {
						statsBuf.Insert(statsBuf.GetEndIter(), logLine+"\n")
						mark := statsBuf.CreateMark("bottom", statsBuf.GetEndIter(), false)
						statsView.ScrollToMark(mark, 0.0, false, 0.0, 1.0)
					}
				})
			})
			glib.IdleAdd(func() {
				isLoadingState = false
				header.SetSubtitle("Stash-Apply-Commit Workflow")
				if err == nil {
					pendingPatches = append(pendingPatches[:idx], pendingPatches[idx+1:]...)
					pendingPanel.List.Remove(row)
					updateStatus(statusLabel, "Patch applied and verified")
					clearAllSelections()
					refreshHistory(historyPanel.List)
					lastAppliedDesc = patchToApply.ShortDescription
					r.RenderGitStatus(".")
				} else {
					r.RenderError(err)
					if !strings.Contains(err.Error(), "PATCH_ERROR") {
						if confirmAction(win, "Verification failed. Pop stash to keep changes?") {
							exec.Command("git", "stash", "pop").Run()
							pendingPatches = append(pendingPatches[:idx], pendingPatches[idx+1:]...)
							pendingPanel.List.Remove(row)
							clearAllSelections()
							refreshHistory(historyPanel.List)
							r.RenderGitStatus(".")
						}
					}
				}
			})
		}()
	}
}
