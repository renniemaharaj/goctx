package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/model"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func processClipboard(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	var outputs []model.ProjectOutput

	// Try single object
	var single model.ProjectOutput
	if err := json.Unmarshal([]byte(text), &single); err == nil && len(single.Files) > 0 {
		outputs = append(outputs, single)
	} else {
		// Try array of objects
		var multiple []model.ProjectOutput
		if err := json.Unmarshal([]byte(text), &multiple); err == nil {
			for _, p := range multiple {
				if len(p.Files) > 0 {
					outputs = append(outputs, p)
				}
			}
		}
	}

	if len(outputs) > 0 {
		glib.IdleAdd(func() {
			dispatchPatches(outputs)
		})
	}
}

// dispatchPatches sends valid ProjectOutput objects to the UI
func dispatchPatches(outputs []model.ProjectOutput) {
	if len(outputs) == 0 {
		return
	}

	glib.IdleAdd(func() {
		for _, p := range outputs {
			addPatchToSidebar(p)
		}
		updateStatus(statusLabel, fmt.Sprintf("Detected %d new patches", len(outputs)))
	})
}

// Helper to encapsulate the UI row creation logic
func addPatchToSidebar(input model.ProjectOutput) {
	pendingPatches = append(pendingPatches, input)

	row, _ := gtk.ListBoxRowNew()
	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)

	title := input.ShortDescription

	// Fallback to InstructionHeader if ShortDescription is empty
	if title == "" && input.InstructionHeader != "" {
		for _, line := range strings.Split(input.InstructionHeader, "\n") {
			cleanLine := strings.TrimSpace(line)
			if cleanLine != "" {
				title = cleanLine
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				break
			}
		}
	}

	if title == "" {
		title = fmt.Sprintf("Patch %d", len(pendingPatches))
	}

	lbl, _ := gtk.LabelNew(title)
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
		}
	})

	hbox.PackEnd(delBtn, false, false, 2)
	row.Add(hbox)
	pendingPanel.List.Add(row)
	pendingPanel.List.ShowAll()
}
