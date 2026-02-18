package ui

import (
	"os"
	"strings"

	"github.com/gotk3/gotk3/glib"
)

func setupDebounceAutoSave() {
	statsBuf.Connect("changed", func() {
		if isLoadingState || !statsView.GetEditable() {
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
