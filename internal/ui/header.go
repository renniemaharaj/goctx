package ui

import "github.com/gotk3/gotk3/gtk"

func headerComponent() *gtk.HeaderBar {
	hb, _ := gtk.HeaderBarNew()
	hb.SetShowCloseButton(true)
	hb.SetTitle("GoCtx Manager")
	hb.SetSubtitle("Stash-Apply-Commit Workflow")

	btnBuild = createToolBtn("document-open-symbolic", "Build current workspace context")
	btnCopy = createToolBtn("edit-copy-symbolic", "Copy AI system prompt + context")
	btnApplyPatch = createToolBtn("document-save-symbolic", "Apply selected pending patch")
	btnApplyCommit = createToolBtn("edit-undo-symbolic", "Restore to this commit state")
	btnCommit = createToolBtn("emblem-ok-symbolic", "Commit all changes")
	btnKeys = createToolBtn("dialog-password-symbolic", "Manage API Keys")
	btnRunBuild = createToolBtn("system-run-symbolic", "Run Build")
	btnRunTest = createToolBtn("media-playback-start-symbolic", "Run Tests")

	btnApplyPatch.SetSensitive(false)
	btnApplyCommit.SetSensitive(false)
	btnCommit.SetSensitive(false)

	hb.PackStart(btnBuild)
	hb.PackStart(btnCopy)
	hb.PackStart(btnApplyCommit)

	hb.PackEnd(btnKeys)
	hb.PackEnd(btnCommit)
	hb.PackEnd(btnRunTest)
	hb.PackEnd(btnRunBuild)
	hb.PackEnd(btnApplyPatch)

	return hb
}
