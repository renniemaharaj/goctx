package ui

import (
	"goctx/internal/model"
	"goctx/internal/renderer"
	"os"
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
	btnRunBuild        *gtk.Button
	btnRunTest         *gtk.Button
	btnBuild           *gtk.Button
	btnCopy            *gtk.Button
	btnKeys            *gtk.Button
	lastHistoryCount   int
	isLoading          bool
	isRefreshing       bool
	debounceID         glib.SourceHandle
	mainTreeView       *gtk.TreeView
	tokenScale         *gtk.Scale
	smartCheck         *gtk.CheckButton
	header             *gtk.HeaderBar
)

func setupCSS() {
	css, _ := gtk.CssProviderNew()
	css.LoadFromData(`
		.btn-success { background-image: none; background-color: #28a745; color: white; text-shadow: none; }
		.btn-failure { background-image: none; background-color: #dc3545; color: white; text-shadow: none; }
		.btn-neutral { background-image: none; }
	`)
	screen, _ := gdk.ScreenGetDefault()
	gtk.AddProviderForScreen(screen, css, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
}

func Run() {
	gtk.Init(nil)
	setupCSS()

	win, _ = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetDefaultSize(1400, 950)
	win.Connect("destroy", gtk.MainQuit)

	header = createHeaderBar()
	win.SetTitlebar(header)

	hPaned := createMainLayout()

	statusPanel, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	statusLabel, _ = gtk.LabelNew("Ready")
	statusLabel.SetMarginStart(10)
	statusLabel.SetMarginBottom(5)
	statusPanel.PackStart(statusLabel, false, false, 0)

	vmain, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	vmain.PackStart(hPaned, true, true, 0)
	vmain.PackStart(statusPanel, false, false, 0)

	overlay, _ := gtk.OverlayNew()
	overlay.Add(vmain)
	setupChatInterface(overlay)

	win.Add(overlay)

	// Rendering logic init
	renderStruct := renderer.NewRenderer(statsBuf, isLoading, statusLabel, updateStatus)
	renderer.SetupTags(statsBuf)

	bindEvents(renderStruct)
	setupDebounceAutoSave()

	backgroundMonitoringLoop()
	refreshHistory(historyPanel.List)
	lastHistoryCount = countCommits()

	win.ShowAll()
	gtk.Main()
}

func createHeaderBar() *gtk.HeaderBar {
	hb, _ := gtk.HeaderBarNew()
	hb.SetShowCloseButton(true)
	hb.SetTitle("GoCtx Manager")
	hb.SetSubtitle("Stash-Apply-Commit Workflow")

	btnBuild = createToolBtn("document-open-symbolic", "Build current workspace context")
	btnCopy = createToolBtn("edit-copy-symbolic", "Copy AI system prompt + context")
	btnApplyPatch = createToolBtn("document-save-symbolic", "Apply selected pending patch")
	btnApplyCommit = createToolBtn("edit-undo-symbolic", "Restore to this commit state")
	btnCommit = createToolBtn("emblem-ok-symbolic", "Commit all changes")
	btnKeys = createToolBtn("dialog-password-symbolic", "Manage API Keys")
	btnRunBuild = createToolBtn("system-run-symbolic", "Run Build")
	btnRunTest = createToolBtn("media-playback-start-symbolic", "Run Tests")

	btnApplyPatch.SetSensitive(false)
	btnApplyCommit.SetSensitive(false)
	btnCommit.SetSensitive(false)

	hb.PackStart(btnBuild)
	hb.PackStart(btnCopy)
	hb.PackStart(btnApplyCommit)

	hb.PackEnd(btnKeys)
	hb.PackEnd(btnCommit)
	hb.PackEnd(btnRunTest)
	hb.PackEnd(btnRunBuild)
	hb.PackEnd(btnApplyPatch)

	return hb
}

func createMainLayout() *gtk.Paned {
	hPaned, _ := gtk.PanedNew(gtk.ORIENTATION_HORIZONTAL)
	hPaned.SetPosition(350)

	pendingPanel = NewActionPanel("PENDING PATCHES", clearAllSelections)
	historyPanel = NewActionPanel("COMMIT HISTORY", clearAllSelections)

	vSidebarOuter, _ := gtk.PanedNew(gtk.ORIENTATION_VERTICAL)
	vSidebarInner, _ := gtk.PanedNew(gtk.ORIENTATION_VERTICAL)

	vSidebarOuter.Pack1(pendingPanel.Container, true, false)
	vSidebarOuter.Pack2(vSidebarInner, true, false)
	vSidebarInner.Pack1(historyPanel.Container, true, false)

	contextTreeBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)
	label(contextTreeBox, "CONTEXT SELECTION")

	boxBudget, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	boxBudget.SetMarginStart(10)
	boxBudget.SetMarginEnd(10)
	lblBudget, _ := gtk.LabelNew("Token Budget")
	lblBudget.SetXAlign(0)
	tokenScale, _ = gtk.ScaleNewWithRange(gtk.ORIENTATION_HORIZONTAL, 1000, 128000, 1000)
	tokenScale.SetValue(32000)
	tokenScale.SetDrawValue(true)
	boxBudget.PackStart(lblBudget, false, false, 0)
	boxBudget.PackStart(tokenScale, false, false, 0)
	contextTreeBox.PackStart(boxBudget, false, false, 5)

	smartCheck, _ = gtk.CheckButtonNewWithLabel("Smart Context (LSP Aware)")
	smartCheck.SetMarginStart(10)
	contextTreeBox.PackStart(smartCheck, false, false, 5)

	mainTreeView, treeStore = setupContextTree()
	treeScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	treeScroll.Add(mainTreeView)
	contextTreeBox.PackStart(treeScroll, true, true, 0)

	vSidebarInner.Pack2(contextTreeBox, true, false)
	vSidebarOuter.SetPosition(250)
	vSidebarInner.SetPosition(250)

	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ = gtk.TextViewNew()
	statsView.SetMonospace(true)
	statsView.SetEditable(false)
	statsView.SetLeftMargin(15)
	statsView.SetTopMargin(15)
	statsBuf, _ = statsView.GetBuffer()
	statsScroll.Add(statsView)

	hPaned.Pack1(vSidebarOuter, false, false)
	hPaned.Pack2(statsScroll, true, false)

	return hPaned
}

func setupDebounceAutoSave() {
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

			if activePath != frozenPath {
				return false
			}

			text, _ := statsBuf.GetText(statsBuf.GetStartIter(), statsBuf.GetEndIter(), false)
			_ = os.WriteFile(activePath, []byte(text), 0644)

			isRefreshing = true
			refreshTreeData(treeStore)
			SelectPath(mainTreeView, treeStore, activePath)
			isRefreshing = false

			debounceID = 0
			return false
		})
	})
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
