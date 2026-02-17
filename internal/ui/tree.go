package ui

import (
	"goctx/internal/builder"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func setupContextTree() (*gtk.TreeView, *gtk.TreeStore) {
	store, _ := gtk.TreeStoreNew(glib.TYPE_BOOLEAN, glib.TYPE_STRING, glib.TYPE_STRING)
	tree, _ := gtk.TreeViewNewWithModel(store)

	renderer, _ := gtk.CellRendererToggleNew()
	renderer.Connect("toggled", func(r *gtk.CellRendererToggle, pathStr string) {
		path, _ := gtk.TreePathNewFromString(pathStr)
		iter, _ := store.GetIter(path)
		val, _ := store.GetValue(iter, 0)
		boolVal, _ := val.GoValue()
		store.SetValue(iter, 0, !boolVal.(bool))
	})

	colToggle, _ := gtk.TreeViewColumnNew()
	colToggle.PackStart(renderer, false)
	colToggle.AddAttribute(renderer, "active", 0)
	tree.AppendColumn(colToggle)

	textRenderer, _ := gtk.CellRendererTextNew()
	colPath, _ := gtk.TreeViewColumnNew()
	colPath.SetTitle("Path")
	colPath.PackStart(textRenderer, true)
	colPath.AddAttribute(textRenderer, "text", 1)
	tree.AppendColumn(colPath)

	refreshTreeData(store)
	return tree, store
}

// SelectPath attempts to find and select a specific file path in the tree
func SelectPath(tree *gtk.TreeView, store *gtk.TreeStore, target string) {
	store.ForEach(func(model *gtk.TreeModel, path *gtk.TreePath, iter *gtk.TreeIter) bool {
		val, _ := model.GetValue(iter, 2)
		s, _ := val.GoValue()
		if s.(string) == target {
			sel, _ := tree.GetSelection()
			sel.SelectIter(iter)
			tree.ScrollToCell(path, nil, false, 0, 0)
			return true
		}
		return false
	})
}

func refreshTreeData(store *gtk.TreeStore) {
	store.Clear()
	files, _ := builder.GetFileList(".")
	for _, f := range files {
		iter := store.Append(nil)
		store.SetValue(iter, 0, true)
		store.SetValue(iter, 1, f)
		store.SetValue(iter, 2, f)
	}
}

func getCheckedFiles(store *gtk.TreeStore) []string {
	checked := []string{}
	store.ForEach(func(model *gtk.TreeModel, path *gtk.TreePath, iter *gtk.TreeIter) bool {
		val, _ := model.GetValue(iter, 0)
		active, _ := val.GoValue()
		if active.(bool) {
			pathVal, _ := model.GetValue(iter, 2)
			fileStr, _ := pathVal.GoValue()
			checked = append(checked, fileStr.(string))
		}
		return false
	})
	return checked
}