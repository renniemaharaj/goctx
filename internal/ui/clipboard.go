package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/model"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gotk3/gotk3/gtk"
)

func processClipboard(text string) {
	var input model.ProjectOutput

	// 1. Try JSON extraction
	reJSON := regexp.MustCompile(`(?s)\{.*\"files\".*\}`)
	jsonMatch := reJSON.FindString(text)

	if jsonMatch != "" && json.Unmarshal([]byte(jsonMatch), &input) == nil {
		// Successfully parsed JSON
	} else if strings.Contains(text, "<<<<<< SEARCH") && strings.Contains(text, "======") {
		// 2. Fallback: Raw surgical block - try to find a file path header
		path := "unknown_file.go"
		rePath := regexp.MustCompile(`(?i)FILE:\s*([^\s\n]+)`)
		pathMatch := rePath.FindStringSubmatch(text)
		if len(pathMatch) > 1 {
			path = pathMatch[1]
		}

		input = model.ProjectOutput{
			ShortDescription: "Manual: " + filepath.Base(path),
			Files: map[string]string{
				path: text,
			},
		}
	} else {
		return
	}

	// Add to list with Trash Icon
	pendingPatches = append(pendingPatches, input)
	row, _ := gtk.ListBoxRowNew()
	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)

	lbl, _ := gtk.LabelNew(input.ShortDescription)
	if input.ShortDescription == "" {
		lbl.SetText(fmt.Sprintf("Patch %d", len(pendingPatches)))
	}
	lbl.SetXAlign(0)
	hbox.PackStart(lbl, true, true, 5)

	delBtn, _ := gtk.ButtonNewFromIconName("edit-delete-symbolic", gtk.ICON_SIZE_MENU)
	delBtn.SetRelief(gtk.RELIEF_NONE)
	delBtn.Connect("clicked", func() {
		cIdx := row.GetIndex()
		if cIdx >= 0 && cIdx < len(pendingPatches) {
			pendingPatches = append(pendingPatches[:cIdx], pendingPatches[cIdx+1:]...)
			pendingPanel.List.Remove(row)
			resetView()
			updateStatus(statusLabel, "Patch removed")
		}
	})

	hbox.PackEnd(delBtn, false, false, 2)
	row.Add(hbox)

	pendingPanel.List.Add(row)
	pendingPanel.List.ShowAll()
	updateStatus(statusLabel, "New patch detected")
}
