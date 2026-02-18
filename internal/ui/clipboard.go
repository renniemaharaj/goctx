package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/model"
	"goctx/internal/patch"
	"os/exec"
	"strings"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func processClipboard(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
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

	if len(outputs) == 0 {
		// Fallback: Try Native Dialect (file: SEARCH/REPLACE blocks)
		if native, ok := patch.ParseNative(text); ok {
			outputs = append(outputs, native)
		}
	}

	if len(outputs) > 0 {
		glib.IdleAdd(func() {
			dispatchPatches(outputs)
		})
	}
	return len(outputs)
}

// dispatchPatches sends valid ProjectOutput objects to the UI
func dispatchPatches(outputs []model.ProjectOutput) {
	if len(outputs) == 0 {
		return
	}

	glib.IdleAdd(func() {
		for _, p := range outputs {
			addPatchToSidebar(p)

			// Extract meaningful metadata for the notification
			title := "Patch Ingested"
			desc := p.ShortDescription
			if desc == "" {
				desc = "AI-generated update"
			}

			fileCount := len(p.Files)
			var body string
			if fileCount == 1 {
				// If it's a single file, identify it in the notification
				var targetFile string
				for path := range p.Files {
					targetFile = path
					break
				}
				body = fmt.Sprintf("%s\nTarget: %s", desc, targetFile)
			} else {
				body = fmt.Sprintf("%s\nModified %d files", desc, fileCount)
			}

			sendNotification(title, body)
		}
		updateStatus(statusLabel, fmt.Sprintf("Detected %d new patches", len(outputs)))
	})
}

func sendNotification(title, msg string) {
	// Use notify-send (common on Linux/GTK environments)
	_ = exec.Command("notify-send", "-a", "GoCtx", "-i", "emblem-symbolic", title, msg).Run()
}

// Helper to encapsulate the UI row creation logic
func addPatchToSidebar(input model.ProjectOutput) {
	pendingPatches = append(pendingPatches, input)

	row, _ := gtk.ListBoxRowNew()
	hbox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)

	title := input.ShortDescription

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

	// Add tooltip showing number of files in the patch
	tooltipText := fmt.Sprintf("Contains %d file(s)", len(input.Files))
	row.SetTooltipText(tooltipText)

	pendingPanel.List.ShowAll()
}
