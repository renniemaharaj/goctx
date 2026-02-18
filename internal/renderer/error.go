package renderer

// RenderError displays application or verification failures in the main panel
func (r *Renderer) RenderError(err error) {
	r.statsBuf.SetText("")
	r.statsBuf.InsertWithTag(r.statsBuf.GetEndIter(), "=== APPLICATION / VERIFICATION FAILURE ===\n\n", r.GetTag("deleted"))

	msg := err.Error()
	// If the error contains build/test output with newlines, it will be preserved here
	r.statsBuf.Insert(r.statsBuf.GetEndIter(), msg+"\n")

	// Apply syntax highlighting to the error output to help identify issues
	highlight(r.statsBuf, `(?i)error:.*`, "deleted")
	highlight(r.statsBuf, `(?i)failed:.*`, "deleted")
	highlight(r.statsBuf, `line \d+`, "header")
	highlight(r.statsBuf, `\./.*\.go:\d+:\d+`, "header")

	r.updateStatus(r.statusLabel, "Error details rendered to panel")
}
