package ui

import (
	"context"
	"goctx/internal/builder"
	"goctx/internal/config"
	"goctx/internal/google"
	"regexp"
	"strings"
	"time"

	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

var aiManager *google.Manager

func setupChatInterface(overlay *gtk.Overlay) {
	keys, _ := config.LoadKeys(".")
	aiManager = google.NewManager(keys)

	// Floating Container
	chatBox, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	chatBox.SetHAlign(gtk.ALIGN_CENTER)
	chatBox.SetVAlign(gtk.ALIGN_END)
	chatBox.SetMarginBottom(40)
	chatBox.SetSizeRequest(700, -1)

	// Styling
	chatBox.SetHExpand(false)
	chatBox.SetVExpand(false)

	frame, _ := gtk.FrameNew("")
	innerBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	innerBox.SetMarginStart(10)
	innerBox.SetMarginEnd(10)
	innerBox.SetMarginTop(10)
	innerBox.SetMarginBottom(10)

	chatEntry, _ := gtk.EntryNew()
	chatEntry.SetPlaceholderText("Ask AI for a patch (Context is auto-included)...")
	chatEntry.SetHExpand(true)

	spinner, _ := gtk.SpinnerNew()

	chatEntry.Connect("activate", func() {
		prompt, _ := chatEntry.GetText()
		if prompt == "" || isLoading {
			return
		}

		chatEntry.SetSensitive(false)
		spinner.Start()
		updateStatus(statusLabel, "AI is generating patch...")

		go func() {
			sysPrompt := builder.AI_PROMPT_HEADER + "\nCURRENT CONTEXT:\n" + string(mustMarshal(activeContext))

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			resp, err := aiManager.Generate(ctx, sysPrompt, prompt)

			glib.IdleAdd(func() {
				chatEntry.SetSensitive(true)
				chatEntry.SetText("")
				spinner.Stop()

				if err != nil {
					updateStatus(statusLabel, "AI Error: "+err.Error())
					showDetailedError("Generation Failed", err.Error())
					return
				}

				cleaned := stripMarkdown(resp)
				processClipboard(cleaned)
				updateStatus(statusLabel, "AI Patch received and parsed")
			})
		}()
	})

	innerBox.PackStart(chatEntry, true, true, 0)
	innerBox.PackStart(spinner, false, false, 5)
	frame.Add(innerBox)
	chatBox.Add(frame)

	btnToggleChat := createToolBtn("chat-message-new-symbolic", "Toggle AI Prompt")
	btnToggleChat.Connect("clicked", func() {
		if chatBox.IsVisible() {
			chatBox.Hide()
		} else {
			chatBox.ShowAll()
			chatEntry.GrabFocus()
		}
	})

	overlay.AddOverlay(chatBox)
	chatBox.Hide()
	header.PackEnd(btnToggleChat)
}

func stripMarkdown(input string) string {
	// Remove opening ```json, ```text, ```go etc
	reStart := regexp.MustCompile("(?m)^```[a-zA-Z]*\\s*")
	// Remove closing ```
	reEnd := regexp.MustCompile("(?m)^```\\s*$")

	res := reStart.ReplaceAllString(input, "")
	res = reEnd.ReplaceAllString(res, "")
	return strings.TrimSpace(res)
}
