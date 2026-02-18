package ui

import (
	"goctx/internal/renderer"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
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

	header = headerComponent()
	win.SetTitlebar(header)

	hPaned := bodyComponent()

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
	chatComponent(overlay)

	win.Add(overlay)

	// Rendering logic init
	mainRenderer = renderer.NewRenderer(statsBuf, &isLoadingState, statusLabel, updateStatus)
	renderer.SetupTags(statsBuf)

	bindEvents(mainRenderer)
	setupDebounceAutoSave()

	backgroundMonitoringLoop()
	refreshHistory(historyPanel.List)
	lastHistoryCount = countCommits()

	win.ShowAll()

	glib.IdleAdd(func() {
		mainRenderer.RenderMarkdown("# Welcome to GoCtx\nSelect files from the tree to build context, or chat with the AI to generate patches.")
	})

	gtk.Main()
}
