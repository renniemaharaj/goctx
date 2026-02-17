package ui

import (
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"os"
	"os/exec"
	"strings"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var (
	activeContext      model.ProjectOutput
	lastClipboard      string
	statsBuf           *gtk.TextBuffer
	statsView          *gtk.TextView
	treeStore          *gtk.TreeStore
	currentEditingPath string
	historyPanel       *ActionPanel
	pendingPanel       *ActionPanel
	pendingPatches     []model.ProjectOutput
	win                *gtk.Window
	statusLabel        *gtk.Label
	btnApplyPatch      *gtk.Button
	btnApplyCommit     *gtk.Button
	btnCommit          *gtk.Button
	lastHistoryCount   int
	isLoading          bool
	isRefreshing       bool
	debounceID         glib.SourceHandle
	mainTreeView       *gtk.TreeView
)

func Run() {
	gtk.Init(nil)
	win, _ = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetDefaultSize(1400, 950)
	win.Connect("destroy", gtk.MainQuit)

	// --- HeaderBar (Toolbar with Close Button) ---
	header, _ := gtk.HeaderBarNew()
	header.SetShowCloseButton(true)
	header.SetTitle("GoCtx Manager")
	header.SetSubtitle("Stash-Apply-Commit Workflow")
	win.SetTitlebar(header)

	// Toolbar Buttons
	btnBuild := createToolBtn("document-open-symbolic", "Build current workspace context")
	btnCopy := createToolBtn("edit-copy-symbolic", "Copy AI system prompt + context to clipboard")
	btnApplyPatch = createToolBtn("document-save-symbolic", "Apply selected pending patch")
	btnApplyCommit = createToolBtn("edit-undo-symbolic", "Restore workspace to this commit's state")
	btnCommit = createToolBtn("emblem-ok-symbolic", "Commit all changes to git")

	btnApplyPatch.SetSensitive(false)
	btnApplyCommit.SetSensitive(false)
	btnCommit.SetSensitive(false)

	header.PackStart(btnBuild)
	header.PackStart(btnCopy)
	header.PackStart(btnApplyPatch)
	header.PackStart(btnApplyCommit)
	header.PackEnd(btnCommit)

	// --- Layout: Resizable Panes ---
	// Root Paned: [ Sidebar (Left) | Diff View (Right) ]
	hPaned, _ := gtk.PanedNew(gtk.ORIENTATION_HORIZONTAL)
	hPaned.SetPosition(350)

	// Nested Resizable Sidebar: [ Pending | [ History | Explorer ] ]
	pendingPanel = NewActionPanel("PENDING PATCHES", clearAllSelections)
	historyPanel = NewActionPanel("COMMIT HISTORY", clearAllSelections)

	vSidebarOuter, _ := gtk.PanedNew(gtk.ORIENTATION_VERTICAL)
	vSidebarInner, _ := gtk.PanedNew(gtk.ORIENTATION_VERTICAL)

	vSidebarOuter.Pack1(pendingPanel.Container, true, false)
	vSidebarOuter.Pack2(vSidebarInner, true, false)

	vSidebarInner.Pack1(historyPanel.Container, true, false)

	// Context Tree
	contextTreeBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)
	label(contextTreeBox, "CONTEXT SELECTION")
	mainTreeView, treeStore = setupContextTree()
	treeScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	treeScroll.Add(mainTreeView)
	contextTreeBox.PackStart(treeScroll, true, true, 0)

	vSidebarInner.Pack2(contextTreeBox, true, false)

	vSidebarOuter.SetPosition(250)
	vSidebarInner.SetPosition(250)

	// Content Area
	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ = gtk.TextViewNew()
	statsView.SetMonospace(true)
	statsView.SetEditable(false)
	statsView.SetLeftMargin(15)
	statsView.SetTopMargin(15)
	statsBuf, _ = statsView.GetBuffer()
	setupTags(statsBuf)

	// Live Ignore Auto-save with Debounce and Selection Persistence
	statsBuf.Connect("changed", func() {
		if isLoading || currentEditingPath == "" || !strings.HasSuffix(currentEditingPath, ".ctxignore") {
			return
		}

		if debounceID != 0 {
			glib.SourceRemove(debounceID)
		}

		debounceID = glib.TimeoutAdd(500, func() bool {
			text, _ := statsBuf.GetText(statsBuf.GetStartIter(), statsBuf.GetEndIter(), false)
			_ = os.WriteFile(currentEditingPath, []byte(text), 0644)

			isRefreshing = true
			refreshTreeData(treeStore)
			SelectPath(mainTreeView, treeStore, currentEditingPath)
			isRefreshing = false

			debounceID = 0
			return false
		})
	})

	statsScroll.Add(statsView)
	rightStack.PackStart(statsScroll, true, true, 0)

	hPaned.Pack1(vSidebarOuter, false, false)
	hPaned.Pack2(rightStack, true, false)

	// Status Bar
	statusPanel, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	statusLabel, _ = gtk.LabelNew("Ready")
	statusLabel.SetMarginStart(10)
	statusLabel.SetMarginBottom(5)
	statusPanel.PackStart(statusLabel, false, false, 0)

	vmain, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	vmain.PackStart(hPaned, true, true, 0)
	vmain.PackStart(statusPanel, false, false, 0)
	win.Add(vmain)

	// --- Logic ---
	btnBuild.Connect("clicked", func() {
		go func() {
			selected := getCheckedFiles(treeStore)
			out, err := builder.BuildSelectiveContext(".", "Manual Build", selected)
			if err == nil {
				activeContext = out
				glib.IdleAdd(func() {
					renderDiff(activeContext, "Current Workspace State")
					updateStatus(statusLabel, "Context built (filtered)")
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
		historyPanel.List.UnselectAll()
		idx := row.GetIndex()
		renderDiff(pendingPatches[idx], "Pending Patch Preview")
		btnApplyPatch.SetSensitive(true)
		btnApplyCommit.SetSensitive(false)
	})

	historyPanel.List.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row != nil {
			pendingPanel.List.UnselectAll()
			lblWidget, _ := row.GetChild()
			lbl, _ := lblWidget.(*gtk.Label)
			fullText, _ := lbl.GetText()
			parts := strings.Fields(fullText)
			if len(parts) > 0 {
				hash := parts[0]
				showCmd := exec.Command("git", "show", "--color=never", hash)
				out, _ := showCmd.Output()
				statsBuf.SetText("")
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), "COMMIT PREVIEW: "+hash+"\n\n", getTag("header"))
				statsBuf.Insert(statsBuf.GetEndIter(), string(out))
				btnApplyCommit.SetSensitive(true)
				btnApplyPatch.SetSensitive(false)
			}
		}
	})

	btnApplyCommit.Connect("clicked", func() {
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
	})

	btnApplyPatch.Connect("clicked", func() {
		stat, _ := exec.Command("git", "status", "--porcelain").Output()
		msg := "Apply selected patch?"
		if len(strings.TrimSpace(string(stat))) > 0 {
			msg = "Workspace is DIRTY. Stashing current changes first. Proceed?"
		}
		if confirmAction(win, msg) {
			row := pendingPanel.List.GetSelectedRow()
			if row != nil {
				idx := row.GetIndex()
				patchToApply := pendingPatches[idx]
				err := apply.ApplyPatch(".", patchToApply)

				// helper to clean up the UI after a successful (or forced) apply
				appliedFunc := func() {
					pendingPatches = append(pendingPatches[:idx], pendingPatches[idx+1:]...)
					pendingPanel.List.Remove(row)
					updateStatus(statusLabel, "Patch applied and verified")
					clearAllSelections()
					refreshHistory(historyPanel.List)
				}

				if err == nil {
					appliedFunc()
				} else if strings.Contains(err.Error(), "PATCH_ERROR") {
					// Hard failure: Hunk mismatch or FS error (no stash created by apply.go)
					updateStatus(statusLabel, "Patch failed to apply")
					RenderError(err)
				} else {
					// Verification failed (Build/Test). ApplyPatch stashed the failing changes.
					RenderError(err)
					confirmMsg := "Verification failed (Build/Test). Changes were stashed. Pop stash to keep them anyway?"
					if confirmAction(win, confirmMsg) {
						// Restore the stashed changes that caused the build failure
						exec.Command("git", "stash", "pop").Run()
						appliedFunc()
						updateStatus(statusLabel, "Patch integrated (verification ignored)")
					} else {
						updateStatus(statusLabel, "Verification failed (changes stashed)")
					}
				}
			}
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

			isLoading = true
			if strings.HasSuffix(pathStr, ".ctxignore") {
				currentEditingPath = pathStr
				statsView.SetEditable(true)
			} else {
				currentEditingPath = ""
				statsView.SetEditable(false)
			}

			RenderFile(pathStr)
			isLoading = false
		}
	})

	btnCommit.Connect("clicked", func() {
		if confirmAction(win, "Commit changes?") {
			exec.Command("git", "add", ".").Run()
			if err := exec.Command("git", "commit", "-m", "GoCtx: patch").Run(); err != nil {
				updateStatus(statusLabel, "Failed: "+err.Error())
			} else {
				updateStatus(statusLabel, "Committed")
				refreshHistory(historyPanel.List)
			}
		}
	})

	backgroundMonitoringLoop()
	refreshHistory(historyPanel.List)
	lastHistoryCount = countCommits()
	win.ShowAll()
	gtk.Main()
}

func showDetailedError(title, msg string) {
	dialog := gtk.MessageDialogNew(win, gtk.DIALOG_MODAL, gtk.MESSAGE_ERROR, gtk.BUTTONS_OK, "%s", title)
	dialog.FormatSecondaryText("%s", msg)
	dialog.Run()
	dialog.Destroy()
}

func createToolBtn(iconName, tooltip string) *gtk.Button {
	btn, _ := gtk.ButtonNew()
	img, _ := gtk.ImageNewFromIconName(iconName, gtk.ICON_SIZE_BUTTON)
	btn.SetImage(img)
	btn.SetAlwaysShowImage(true)
	btn.SetTooltipText(tooltip)
	return btn
}
