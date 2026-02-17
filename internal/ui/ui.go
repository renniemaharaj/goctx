package ui

import (
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var (
	activeContext      model.ProjectOutput
	lastClipboard      string
	lastAppliedDesc    string
	statsBuf           *gtk.TextBuffer
	statsView          *gtk.TextView
	treeStore          *gtk.TreeStore
	currentEditingPath string
	pathMu             sync.RWMutex
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
	tokenScale         *gtk.Scale
	smartCheck         *gtk.CheckButton
	header             *gtk.HeaderBar
)

func Run() {
	gtk.Init(nil)
	win, _ = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetDefaultSize(1400, 950)
	win.Connect("destroy", gtk.MainQuit)

	// --- HeaderBar (Toolbar with Close Button) ---
	header, _ = gtk.HeaderBarNew()
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
	btnKeys := createToolBtn("preferences-desktop-remote-symbolic", "Manage Gemini API Keys")

	btnApplyPatch.SetSensitive(false)
	btnApplyCommit.SetSensitive(false)
	btnCommit.SetSensitive(false)

	header.PackStart(btnBuild)
	header.PackStart(btnCopy)
	header.PackStart(btnApplyPatch)
	header.PackStart(btnApplyCommit)
	header.PackEnd(btnCommit)
	header.PackEnd(btnKeys)

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

	// Context Tree & Controls
	contextTreeBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)
	label(contextTreeBox, "CONTEXT SELECTION")

	// Token Budget Control
	boxBudget, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	boxBudget.SetMarginStart(10)
	boxBudget.SetMarginEnd(10)
	lblBudget, _ := gtk.LabelNew("Token Budget")
	lblBudget.SetXAlign(0)
	tokenScale, _ = gtk.ScaleNewWithRange(gtk.ORIENTATION_HORIZONTAL, 1000, 128000, 1000)
	tokenScale.SetValue(32000) // Default 32k
	tokenScale.SetDrawValue(true)
	boxBudget.PackStart(lblBudget, false, false, 0)
	boxBudget.PackStart(tokenScale, false, false, 0)
	contextTreeBox.PackStart(boxBudget, false, false, 5)

	// Smart Context Checkbox
	smartCheck, _ = gtk.CheckButtonNewWithLabel("Smart Context (LSP Aware)")
	smartCheck.SetTooltipText("Uses local Go tools to find related symbol definitions automatically")
	smartCheck.SetMarginStart(10)
	contextTreeBox.PackStart(smartCheck, false, false, 5)

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

	// Live Ignore Auto-save with Debounce and Path-Locking
	statsBuf.Connect("changed", func() {
		if isLoading || !statsView.GetEditable() {
			return
		}

		pathMu.RLock()
		frozenPath := currentEditingPath
		pathMu.RUnlock()

		if frozenPath == "" || !strings.HasSuffix(frozenPath, ".ctxignore") {
			return
		}

		if debounceID != 0 {
			glib.SourceRemove(debounceID)
		}

		debounceID = glib.TimeoutAdd(500, func() bool {
			pathMu.RLock()
			activePath := currentEditingPath
			pathMu.RUnlock()

			// Ensure we are still looking at the same file that triggered the save
			if activePath != frozenPath {
				return false
			}

			text, _ := statsBuf.GetText(statsBuf.GetStartIter(), statsBuf.GetEndIter(), false)
			err := os.WriteFile(activePath, []byte(text), 0644)
			if err != nil {
				updateStatus(statusLabel, "Error saving: "+err.Error())
			}

			isRefreshing = true
			refreshTreeData(treeStore)
			SelectPath(mainTreeView, treeStore, activePath)
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

	// Overlay for Floating Chat
	overlay, _ := gtk.OverlayNew()
	vmain, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	vmain.PackStart(hPaned, true, true, 0)
	vmain.PackStart(statusPanel, false, false, 0)

	overlay.Add(vmain)
	setupChatInterface(overlay)
	win.Add(overlay)

	// --- Logic ---
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
					renderDiff(activeContext, "Current Workspace State")
					updateStatus(statusLabel, "Context built successfully")
				})
			}
		}()
	})

	btnKeys.Connect("clicked", func() {
		showKeyManager()
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

		pathMu.Lock()
		currentEditingPath = ""
		pathMu.Unlock()
		statsView.SetEditable(false)

		idx := row.GetIndex()
		renderDiff(pendingPatches[idx], "Pending Patch Preview")
		btnApplyPatch.SetSensitive(true)
		btnApplyCommit.SetSensitive(false)
	})

	historyPanel.List.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row != nil {
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

				isLoading = true
				statsBuf.SetText("")
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), "COMMIT PREVIEW: "+hash+"\n\n", getTag("header"))
				statsBuf.Insert(statsBuf.GetEndIter(), string(out))
				isLoading = false

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
		row := pendingPanel.List.GetSelectedRow()
		if row == nil {
			return
		}

		idx := row.GetIndex()
		patchToApply := pendingPatches[idx]

		// Check if workspace is dirty
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
			err := apply.ApplyPatch(".", patchToApply)

			appliedFunc := func() {
				pendingPatches = append(pendingPatches[:idx], pendingPatches[idx+1:]...)
				pendingPanel.List.Remove(row)
				updateStatus(statusLabel, "Patch applied and verified")
				clearAllSelections()
				refreshHistory(historyPanel.List)
				lastAppliedDesc = patchToApply.ShortDescription
			}

			if err == nil {
				appliedFunc()
			} else if strings.Contains(err.Error(), "PATCH_ERROR") {
				updateStatus(statusLabel, "Patch failed to apply")
				RenderError(err)
			} else {
				RenderError(err)
				confirmMsg := "Verification failed (Build/Test). Changes were stashed. Pop stash to keep them anyway?"
				if confirmAction(win, confirmMsg) {
					exec.Command("git", "stash", "pop").Run()
					appliedFunc()
					updateStatus(statusLabel, "Patch integrated (verification ignored)")
				} else {
					updateStatus(statusLabel, "Verification failed (changes stashed)")
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

			pathMu.Lock()
			currentEditingPath = pathStr
			pathMu.Unlock()

			isLoading = true
			if strings.HasSuffix(pathStr, ".ctxignore") {
				statsView.SetEditable(true)
			} else {
				statsView.SetEditable(false)
			}

			RenderFile(pathStr)
			isLoading = false
		}
	})

	btnCommit.Connect("clicked", func() {
		defaultMsg := lastAppliedDesc
		if defaultMsg == "" {
			defaultMsg = "GoCtx: Manual Commit"
		}

		msg, ok := askForString(win, "Commit Message", defaultMsg)
		if !ok || strings.TrimSpace(msg) == "" {
			return
		}

		exec.Command("git", "add", ".").Run()
		if err := exec.Command("git", "commit", "-m", msg).Run(); err != nil {
			updateStatus(statusLabel, "Failed: "+err.Error())
		} else {
			updateStatus(statusLabel, "Committed")
			refreshHistory(historyPanel.List)
			lastAppliedDesc = ""
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
	dialog.FormatSecondaryText("%s", truncate(msg, 100))
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
