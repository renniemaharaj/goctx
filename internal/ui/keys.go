package ui

import (
	"goctx/internal/config"
	"strings"

	"github.com/gotk3/gotk3/gtk"
)

func showKeyManager() {
	dialog, _ := gtk.DialogNew()
	dialog.SetTitle("API Key Management")
	dialog.SetTransientFor(win)
	dialog.SetModal(true)
	dialog.AddButton("_Cancel", gtk.RESPONSE_CANCEL)
	dialog.AddButton("_Save Keys", gtk.RESPONSE_OK)

	// Professional sizing: slightly narrower and taller for better list reading
	dialog.SetDefaultSize(450, 480)

	content, _ := dialog.GetContentArea()

	vbox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)
	vbox.SetMarginStart(18)
	vbox.SetMarginEnd(18)
	vbox.SetMarginTop(18)
	vbox.SetMarginBottom(18)

	// Header
	headerLbl, _ := gtk.LabelNew("")
	headerLbl.SetMarkup("<span weight='bold' size='large'>Google Gemini Configuration</span>")
	headerLbl.SetXAlign(0)
	vbox.PackStart(headerLbl, false, false, 0)

	desc, _ := gtk.LabelNew("Format: API_KEY=MODEL_NAME (e.g. AIza...=gemini-2.0-flash). Keys are stored securely in keys.json.")
	desc.SetLineWrap(true)
	desc.SetXAlign(0)
	desc.SetOpacity(0.7)
	vbox.PackStart(desc, false, false, 0)

	sep, _ := gtk.SeparatorNew(gtk.ORIENTATION_HORIZONTAL)
	vbox.PackStart(sep, false, false, 4)

	sw, _ := gtk.ScrolledWindowNew(nil, nil)
	sw.SetShadowType(gtk.SHADOW_IN)
	sw.SetVExpand(true)

	ttv, _ := gtk.TextViewNew()
	ttv.SetMonospace(true)
	ttv.SetLeftMargin(8)
	ttv.SetRightMargin(8)
	ttv.SetTopMargin(8)
	ttv.SetBottomMargin(8)
	buf, _ := ttv.GetBuffer()

	keys, _ := config.LoadKeys(".")
	var sb strings.Builder
	for k, v := range keys {
		sb.WriteString(k + "=" + v + "\n")
	}
	buf.SetText(sb.String())

	sw.Add(ttv)
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
