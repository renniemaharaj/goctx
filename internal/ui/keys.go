package ui

import (
	"goctx/internal/config"
	"strings"

	"github.com/gotk3/gotk3/gtk"
)

func showKeyManager() {
	dialog, _ := gtk.DialogNew()
	dialog.SetTitle("Gemini API Keys")
	dialog.SetTransientFor(win)
	dialog.SetModal(true)
	dialog.AddButton("Save", gtk.RESPONSE_OK)
	dialog.AddButton("Cancel", gtk.RESPONSE_CANCEL)

	dialog.SetDefaultSize(500, 400)

	content, _ := dialog.GetContentArea()

	vbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	vbox.SetMarginStart(15)
	vbox.SetMarginEnd(15)
	vbox.SetMarginTop(15)
	vbox.SetMarginBottom(15)

	desc, _ := gtk.LabelNew("Enter your Google Gemini API Keys (one per line). Keys are stored locally in goctx.json.")
	desc.SetLineWrap(true)
	desc.SetXAlign(0)
	vbox.PackStart(desc, false, false, 0)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetShadowType(gtk.SHADOW_IN)

	tv, _ := gtk.TextViewNew()
	tv.SetMonospace(true)
	buf, _ := tv.GetBuffer()

	keys, _ := config.LoadKeys(".")
	var sb strings.Builder
	for k, v := range keys {
		sb.WriteString(k + "=" + v + "\n")
	}
	buf.SetText(sb.String())

	sw.Add(tv)
	vbox.PackStart(sw, true, true, 0)

	content.Add(vbox)
	content.ShowAll()

	if dialog.Run() == gtk.RESPONSE_OK {
		start, end := buf.GetBounds()
		text, _ := buf.GetText(start, end, false)

		lines := strings.Split(text, "\n")
		cleaned := make(map[string]string)
		for _, l := range lines {
			if line := strings.TrimSpace(l); line != "" {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					cleaned[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				} else {
					cleaned[strings.TrimSpace(parts[0])] = "gemini-2.0-flash"
				}
			}
		}

		if err := config.SaveKeys(".", cleaned); err != nil {
			updateStatus(statusLabel, "Failed to save keys: "+err.Error())
		} else {
			if aiManager != nil {
				aiManager.SetKeys(cleaned)
			}
			updateStatus(statusLabel, "API Keys updated in keys.json")
		}
	}
	dialog.Destroy()
}
