package ui

import (
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"os/exec"
	"regexp"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var (
	activeContext  model.ProjectOutput
	lastClipboard  string
	statsBuf       *gtk.TextBuffer
	stashPanel     *ActionPanel
	pendingPanel   *ActionPanel
	pendingPatches []model.ProjectOutput
	selectedStash  model.ProjectOutput
	win            *gtk.Window
	statusLabel    *gtk.Label
	btnApplyPatch  *gtk.Button
	btnApplyStash  *gtk.Button
	btnCommit      *gtk.Button
	lastStashCount int
)

func Run() {
	gtk.Init(nil)
	win, _ = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Manager")
	win.SetDefaultSize(1400, 950)
	win.Connect("destroy", gtk.MainQuit)

	vmain, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	hmain, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	leftBar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 15)
	leftBar.SetMarginStart(15)
	leftBar.SetMarginEnd(15)
	leftBar.SetMarginTop(15)
	leftBar.SetSizeRequest(320, -1)

	btnsWrapper, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 15)
	leftBar.PackStart(btnsWrapper, false, false, 0)

	btnBuild := newBtn("")
	btnCopy := newBtn("")
	btnApplyPatch = newBtn("")
	btnApplyStash = newBtn("")
	btnCommit = newBtn("")

	btnBuild.SetTooltipText("Build current workspace context")
	btnCopy.SetTooltipText("Copy AI system prompt + context to clipboard")
	btnApplyPatch.SetTooltipText("Apply selected pending patch")
	btnApplyStash.SetTooltipText("Apply selected git stash")
	btnCommit.SetTooltipText("Commit all changes to git")

	imgBuild, _ := gtk.ImageNewFromIconName("document-open-symbolic", gtk.ICON_SIZE_BUTTON)
	imgCopy, _ := gtk.ImageNewFromIconName("edit-copy-symbolic", gtk.ICON_SIZE_BUTTON)
	imgPatch, _ := gtk.ImageNewFromIconName("document-save-symbolic", gtk.ICON_SIZE_BUTTON)
	imgStash, _ := gtk.ImageNewFromIconName("view-refresh-symbolic", gtk.ICON_SIZE_BUTTON)
	imgCommit, _ := gtk.ImageNewFromIconName("emblem-ok-symbolic", gtk.ICON_SIZE_BUTTON)

	btnBuild.SetImage(imgBuild)
	btnCopy.SetImage(imgCopy)
	btnApplyPatch.SetImage(imgPatch)
	btnApplyStash.SetImage(imgStash)
	btnCommit.SetImage(imgCommit)

	btnBuild.SetAlwaysShowImage(true)
	btnCopy.SetAlwaysShowImage(true)
	btnApplyPatch.SetAlwaysShowImage(true)
	btnApplyStash.SetAlwaysShowImage(true)
	btnCommit.SetAlwaysShowImage(true)

	btnApplyPatch.SetSensitive(false)
	btnApplyStash.SetSensitive(false)

	btnsWrapper.PackStart(btnBuild, false, false, 0)
	btnsWrapper.PackStart(btnCopy, false, false, 0)
	btnsWrapper.PackStart(btnApplyPatch, false, false, 0)
	btnsWrapper.PackStart(btnApplyStash, false, false, 0)
	btnsWrapper.PackEnd(btnCommit, false, false, 0)

	pendingPanel = NewActionPanel("PENDING PATCHES", 200, clearAllSelections)
	leftBar.PackStart(pendingPanel.Container, false, false, 0)

	stashPanel = NewActionPanel("STASHES", 0, clearAllSelections)
	leftBar.PackStart(stashPanel.Container, true, true, 0)

	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	rightStack.SetMarginStart(20)
	rightStack.SetMarginEnd(20)
	rightStack.SetMarginTop(15)

	label(rightStack, "CONTEXT TOOL GUI (GOCTX)")
	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ := gtk.TextViewNew()
	statsView.SetMonospace(true)
	statsView.SetEditable(false)
	statsView.SetLeftMargin(25)
	statsView.SetTopMargin(25)
	statsBuf, _ = statsView.GetBuffer()
	statsScroll.Add(statsView)
	rightStack.PackStart(statsScroll, true, true, 0)

	setupTags(statsBuf)

	statusPanel, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	statusLabel, _ = gtk.LabelNew("Ready")
	statusPanel.PackStart(statusLabel, false, false, 10)

	hmain.PackStart(leftBar, false, false, 0)
	hmain.PackStart(rightStack, true, true, 0)
	vmain.PackStart(hmain, true, true, 0)
	vmain.PackStart(statusPanel, false, false, 5)

	btnBuild.Connect("clicked", func() {
		go func() {
			out, err := builder.BuildSelectiveContext(".", "Manual Build")
			if err == nil {
				activeContext = out
				glib.IdleAdd(func() {
					renderDiff(activeContext, "Current Workspace State")
					updateStatus(statusLabel, "Context built")
				})
			}
		}()
	})

	btnCopy.Connect("clicked", func() {
		fullPrompt := builder.AI_PROMPT_HEADER + string(mustMarshal(activeContext))
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(fullPrompt)
		updateStatus(statusLabel, "System Prompt + Context copied")
	})

	pendingPanel.List.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		stashPanel.List.UnselectAll()
		idx := row.GetIndex()
		renderDiff(pendingPatches[idx], "Pending Patch Preview")
		btnApplyPatch.SetSensitive(true)
		btnApplyStash.SetSensitive(false)
	})

	stashPanel.List.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row != nil {
			pendingPanel.List.UnselectAll()

			lblWidget, _ := row.GetChild()
			lbl, _ := lblWidget.(*gtk.Label)
			fullText, _ := lbl.GetText()

			re := regexp.MustCompile(`stash@\{\d+\}`)
			stashRef := re.FindString(fullText)

			if stashRef != "" {
				showCmd := exec.Command("git", "stash", "show", "-p", stashRef)
				out, _ := showCmd.Output()

				statsBuf.SetText("")
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), "GIT STASH PREVIEW: "+stashRef+"\n\n", getTag("header"))
				statsBuf.Insert(statsBuf.GetEndIter(), string(out))

				btnApplyStash.SetSensitive(true)
				btnApplyPatch.SetSensitive(false)
			}
		}
	})

	btnApplyStash.Connect("clicked", func() {
		row := stashPanel.List.GetSelectedRow()
		if row == nil {
			return
		}

		lblWidget, _ := row.GetChild()
		lbl, _ := lblWidget.(*gtk.Label)
		fullText, _ := lbl.GetText()
		re := regexp.MustCompile(`stash@\{\d+\}`)
		stashRef := re.FindString(fullText)

		if stashRef != "" && confirmAction(win, "Apply "+stashRef+"?") {
			cmd := exec.Command("git", "stash", "apply", stashRef)
			if err := cmd.Run(); err != nil {
				updateStatus(statusLabel, "Conflict or error applying git stash")
			} else {
				updateStatus(statusLabel, "Stash applied successfully")
				refreshStashes(stashPanel.List)
			}
		}
	})

	btnApplyPatch.Connect("clicked", func() {
		// Check for dirty state to warn user about auto-stashing
		stat, _ := exec.Command("git", "status", "--porcelain").Output()
		message := "Apply selected patch?"
		if len(strings.TrimSpace(string(stat))) > 0 {
			message = "Workspace is DIRTY. Current changes will be STASHED before applying. Proceed?"
		}

		if confirmAction(win, message) {
			row := pendingPanel.List.GetSelectedRow()
			if row != nil {
				idx := row.GetIndex()
				err := apply.ApplyPatch(".", pendingPatches[idx])
				if err == nil {
					pendingPatches = append(pendingPatches[:idx], pendingPatches[idx+1:]...)
					pendingPanel.List.Remove(row)

					updateStatus(statusLabel, "Patch applied; previous state stashed")
					clearAllSelections()
					refreshStashes(stashPanel.List)
					btnCommit.SetSensitive(true)
				} else {
					updateStatus(statusLabel, "Error: "+err.Error())
				}
			}
		}
	})

	btnCommit.Connect("clicked", func() {
		if confirmAction(win, "Commit all changes?") {
			exec.Command("git", "add", ".").Run()
			cmd := exec.Command("git", "commit", "-m", "GoCtx: applied surgical patch")
			if err := cmd.Run(); err != nil {
				updateStatus(statusLabel, "Commit failed: "+err.Error())
			} else {
				updateStatus(statusLabel, "Changes committed to git")
				btnCommit.SetSensitive(false)
				btnCommit.SetVisible(false)
			}
		}
	})

	// Background Monitoring Loop
	backgroundMonitoringLoop()
	refreshStashes(stashPanel.List)
	lastStashCount = countStashes()
	win.Add(vmain)
	win.ShowAll()
	gtk.Main()
}
