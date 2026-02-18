package ui

import (
	"goctx/internal/model"
	"goctx/internal/renderer"
	"sync"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var (
	pathMu sync.RWMutex

	isLoadingState     bool
	lastHistoryCount   int
	activeContext      model.ProjectOutput
	lastClipboard      string
	lastAppliedDesc    string
	currentEditingPath string
	pendingPatches     []model.ProjectOutput
	isRefreshing       bool
	debounceID         glib.SourceHandle
	mainTreeView       *gtk.TreeView
	tokenScale         *gtk.Scale
	smartCheck         *gtk.CheckButton
	header             *gtk.HeaderBar
	mainRenderer       *renderer.Renderer

	win          *gtk.Window
	historyPanel *ActionPanel
	pendingPanel *ActionPanel
	statusLabel  *gtk.Label
	statsBuf     *gtk.TextBuffer
	statsView    *gtk.TextView
	treeStore    *gtk.TreeStore

	btnApplyPatch  *gtk.Button
	btnApplyCommit *gtk.Button
	btnCommit      *gtk.Button
	btnRunBuild    *gtk.Button
	btnRunTest     *gtk.Button
	btnBuild       *gtk.Button
	btnCopy        *gtk.Button
	btnKeys        *gtk.Button
)
