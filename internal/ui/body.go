package ui

import "github.com/gotk3/gotk3/gtk"

func bodyComponent() *gtk.Paned {
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
	statsView.SetWrapMode(gtk.WRAP_WORD_CHAR)
	statsView.SetLeftMargin(15)
	statsView.SetTopMargin(15)
	statsBuf, _ = statsView.GetBuffer()
	statsScroll.Add(statsView)

	hPaned.Pack1(vSidebarOuter, false, false)
	hPaned.Pack2(statsScroll, true, false)

	return hPaned
}
