package ui

import (
	"encoding/json"
	"fmt"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"goctx/internal/patch"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/sergi/go-diff/diffmatchpatch"
)

var (
	activeContext  model.ProjectOutput
	lastClipboard  string
	statsBuf       *gtk.TextBuffer
	stashPanel     *ActionPanel
	pendingPanel   *ActionPanel
	pendingPatches []model.ProjectOutput
	selectedStash  model.ProjectOutput
	win            *gtk.Window
	statusLabel    *gtk.Label
	btnApplyPatch  *gtk.Button
	btnApplyStash  *gtk.Button
	btnCommit      *gtk.Button
	lastStashCount int
)

func Run() {
	gtk.Init(nil)
	win, _ = gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("GoCtx Manager")
	win.SetDefaultSize(1400, 950)
	win.Connect("destroy", gtk.MainQuit)

	vmain, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	hmain, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	leftBar, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 15)
	leftBar.SetMarginStart(15)
	leftBar.SetMarginEnd(15)
	leftBar.SetMarginTop(15)
	leftBar.SetSizeRequest(320, -1)

	btnBuild := newBtn("CURRENT CONTEXT")
	btnCopy := newBtn("COPY CONTEXT")
	btnApplyPatch = newBtn("APPLY SELECTED PATCH")
	btnApplyStash = newBtn("APPLY SELECTED STASH")
	btnCommit = newBtn("GIT COMMIT")

	// btnCommit.SetSensitive(false)
	btnApplyPatch.SetSensitive(false)
	btnApplyStash.SetSensitive(false)

	leftBar.PackStart(btnBuild, false, false, 0)
	leftBar.PackStart(btnCopy, false, false, 0)
	leftBar.PackStart(btnApplyPatch, false, false, 0)
	leftBar.PackStart(btnApplyStash, false, false, 0)
	leftBar.PackEnd(btnCommit, false, false, 0)

	pendingPanel = NewActionPanel("PENDING PATCHES", 200, clearAllSelections)
	leftBar.PackStart(pendingPanel.Container, false, false, 0)

	stashPanel = NewActionPanel("STASHES", 0, clearAllSelections)
	leftBar.PackStart(stashPanel.Container, true, true, 0)

	rightStack, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 10)
	rightStack.SetMarginStart(20)
	rightStack.SetMarginEnd(20)
	rightStack.SetMarginTop(15)

	label(rightStack, "CONTEXT TOOL GUI (GOCTX)")
	statsScroll, _ := gtk.ScrolledWindowNew(nil, nil)
	statsView, _ := gtk.TextViewNew()
	statsView.SetMonospace(true)
	statsView.SetEditable(false)
	statsView.SetLeftMargin(25)
	statsView.SetTopMargin(25)
	statsBuf, _ = statsView.GetBuffer()
	statsScroll.Add(statsView)
	rightStack.PackStart(statsScroll, true, true, 0)

	setupTags(statsBuf)

	statusPanel, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	statusLabel, _ = gtk.LabelNew("Ready")
	statusPanel.PackStart(statusLabel, false, false, 10)

	hmain.PackStart(leftBar, false, false, 0)
	hmain.PackStart(rightStack, true, true, 0)
	vmain.PackStart(hmain, true, true, 0)
	vmain.PackStart(statusPanel, false, false, 5)

	btnBuild.Connect("clicked", func() {
		go func() {
			out, err := builder.BuildSelectiveContext(".", "Manual Build")
			if err == nil {
				activeContext = out
				glib.IdleAdd(func() {
					renderDiff(activeContext, "Current Workspace State")
					updateStatus(statusLabel, "Context built")
				})
			}
		}()
	})

	btnCopy.Connect("clicked", func() {
		fullPrompt := builder.AI_PROMPT_HEADER + string(mustMarshal(activeContext))
		clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
		clip.SetText(fullPrompt)
		updateStatus(statusLabel, "System Prompt + Context copied")
	})

	pendingPanel.List.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		stashPanel.List.UnselectAll()
		idx := row.GetIndex()
		renderDiff(pendingPatches[idx], "Pending Patch Preview")
		btnApplyPatch.SetSensitive(true)
		btnApplyStash.SetSensitive(false)
	})

	stashPanel.List.Connect("row-selected", func(_ *gtk.ListBox, row *gtk.ListBoxRow) {
		if row == nil {
			return
		}
		pendingPanel.List.UnselectAll()

		lblWidget, _ := row.GetChild()
		lbl, _ := lblWidget.(*gtk.Label)
		fullText, _ := lbl.GetText()

		re := regexp.MustCompile(`stash@\{\d+\}`)
		stashRef := re.FindString(fullText)

		if stashRef != "" {
			showCmd := exec.Command("git", "stash", "show", "-p", stashRef)
			out, _ := showCmd.Output()

			statsBuf.SetText("")
			statsBuf.InsertWithTag(statsBuf.GetEndIter(), "GIT STASH PREVIEW: "+stashRef+"\n\n", getTag("header"))
			statsBuf.Insert(statsBuf.GetEndIter(), string(out))

			btnApplyStash.SetSensitive(true)
			btnApplyPatch.SetSensitive(false)
		}
	})

	btnApplyStash.Connect("clicked", func() {
		row := stashPanel.List.GetSelectedRow()
		if row == nil {
			return
		}

		lblWidget, _ := row.GetChild()
		lbl, _ := lblWidget.(*gtk.Label)
		fullText, _ := lbl.GetText()
		re := regexp.MustCompile(`stash@\{\d+\}`)
		stashRef := re.FindString(fullText)

		if stashRef != "" && confirmAction(win, "Apply "+stashRef+"?") {
			cmd := exec.Command("git", "stash", "apply", stashRef)
			if err := cmd.Run(); err != nil {
				updateStatus(statusLabel, "Conflict or error applying git stash")
			} else {
				updateStatus(statusLabel, "Stash applied successfully")
				refreshStashes(stashPanel.List)
			}
		}
	})

	btnApplyPatch.Connect("clicked", func() {
		// Check for dirty state to warn user about auto-stashing
		stat, _ := exec.Command("git", "status", "--porcelain").Output()
		message := "Apply selected patch?"
		if len(strings.TrimSpace(string(stat))) > 0 {
			message = "Workspace is DIRTY. Current changes will be STASHED before applying. Proceed?"
		}

		if confirmAction(win, message) {
			row := pendingPanel.List.GetSelectedRow()
			if row != nil {
				idx := row.GetIndex()
				err := apply.ApplyPatch(".", pendingPatches[idx])
				if err == nil {
					pendingPatches = append(pendingPatches[:idx], pendingPatches[idx+1:]...)
					pendingPanel.List.Remove(row)

					updateStatus(statusLabel, "Patch applied; previous state stashed")
					clearAllSelections()
					refreshStashes(stashPanel.List)
					btnCommit.SetSensitive(true)
				} else {
					updateStatus(statusLabel, "Error: "+err.Error())
				}
			}
		}
	})

	btnCommit.Connect("clicked", func() {
		if confirmAction(win, "Commit all changes?") {
			exec.Command("git", "add", ".").Run()
			cmd := exec.Command("git", "commit", "-m", "GoCtx: applied surgical patch")
			if err := cmd.Run(); err != nil {
				updateStatus(statusLabel, "Commit failed: "+err.Error())
			} else {
				updateStatus(statusLabel, "Changes committed to git")
				btnCommit.SetSensitive(false)
				btnCommit.SetVisible(false)
			}
		}
	})

	// Background Monitoring Loop
	go func() {
		for {
			time.Sleep(2 * time.Second)
			glib.IdleAdd(func() {
				if btnCommit == nil {
					return
				}

				stat, _ := exec.Command("git", "status", "--porcelain").Output()
				hasChanges := len(strings.TrimSpace(string(stat))) > 0
				btnCommit.SetVisible(hasChanges)
				if !hasChanges {
					btnCommit.SetSensitive(false)
				}

				currentCount := countStashes()
				if currentCount != lastStashCount {
					refreshStashes(stashPanel.List)
					lastStashCount = currentCount
				}

				clip, _ := gtk.ClipboardGet(gdk.SELECTION_CLIPBOARD)
				text, _ := clip.WaitForText()
				if text != "" && text != lastClipboard {
					lastClipboard = text
					processClipboard(text)
				}
			})
		}
	}()

	refreshStashes(stashPanel.List)
	lastStashCount = countStashes()
	win.Add(vmain)
	win.ShowAll()
	gtk.Main()
}

func countStashes() int {
	out, err := exec.Command("git", "stash", "list").Output()
	if err != nil {
		return 0
	}
	return strings.Count(string(out), "\n")
}

func clearAllSelections() {
	pendingPanel.List.UnselectAll()
	stashPanel.List.UnselectAll()
	resetView()
}

func resetView() {
	btnApplyPatch.SetSensitive(false)
	btnApplyStash.SetSensitive(false)
	statsBuf.SetText("")
	updateStatus(statusLabel, "Selection cleared")
}

func refreshStashes(list *gtk.ListBox) {
	list.GetChildren().Foreach(func(item interface{}) { list.Remove(item.(gtk.IWidget)) })

	out, err := exec.Command("git", "stash", "list").Output()
	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		row, _ := gtk.ListBoxRowNew()
		lbl, _ := gtk.LabelNew(line)
		lbl.SetXAlign(0)
		row.Add(lbl)
		list.Add(row)
	}
	list.ShowAll()
}

func setupTags(buffer *gtk.TextBuffer) {
	tab, _ := buffer.GetTagTable()
	tagA, _ := gtk.TextTagNew("added")
	tagA.SetProperty("background", "#1e3a1e")
	tagA.SetProperty("foreground", "#afffbc")
	tab.Add(tagA)
	tagD, _ := gtk.TextTagNew("deleted")
	tagD.SetProperty("background", "#4b1818")
	tagD.SetProperty("foreground", "#ffa1a1")
	tab.Add(tagD)
	tagH, _ := gtk.TextTagNew("header")
	tagH.SetProperty("weight", 700)
	tagH.SetProperty("foreground", "#569cd6")
	tab.Add(tagH)
}

func getTag(n string) *gtk.TextTag {
	tab, err := statsBuf.GetTagTable()
	if err != nil {
		return nil
	}
	tag, _ := tab.Lookup(n)
	return tag
}

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

func renderDiff(p model.ProjectOutput, title string) {
	statsBuf.SetText("")
	statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("=== %s ===\n\n", strings.ToUpper(title)), getTag("header"))

	if p.ProjectTree != "" {
		statsBuf.Insert(statsBuf.GetEndIter(), "PROJECT STRUCTURE:\n")
		statsBuf.Insert(statsBuf.GetEndIter(), p.ProjectTree+"\n")
		statsBuf.Insert(statsBuf.GetEndIter(), "---\n\n")
	}

	dmp := diffmatchpatch.New()
	var keys []string
	for k := range p.Files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, path := range keys {
		newContent := p.Files[path]
		if !utf8.ValidString(newContent) {
			continue
		}

		statsBuf.InsertWithTag(statsBuf.GetEndIter(), fmt.Sprintf("FILE: %s\n", path), getTag("header"))

		old, _ := os.ReadFile(path)
		oldStr := string(old)

		hunks := patch.ParseHunks(newContent)
		if len(hunks) > 0 {
			for _, h := range hunks {
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), "--- SURGICAL MODIFICATION ---\n", getTag("header"))
				statsBuf.Insert(statsBuf.GetEndIter(), "[EXISTING CODE]:\n")
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), h.Search+"\n", getTag("deleted"))
				statsBuf.Insert(statsBuf.GetEndIter(), "\n[REPLACEMENT]:\n")
				statsBuf.InsertWithTag(statsBuf.GetEndIter(), h.Replace+"\n", getTag("added"))

				_, ok := patch.ApplyHunk(oldStr, h)
				if !ok {
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), "\nERROR: Match not found!\n", getTag("header"))
				} else {
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), "\nREADY: Hunk match validated.\n", getTag("added"))
				}
				statsBuf.Insert(statsBuf.GetEndIter(), "\n---\n\n")
			}
		} else {
			diffs := dmp.DiffMain(oldStr, newContent, false)
			for _, d := range diffs {
				tag := ""
				switch d.Type {
				case diffmatchpatch.DiffInsert:
					tag = "added"
				case diffmatchpatch.DiffDelete:
					tag = "deleted"
				}
				if tag != "" {
					statsBuf.InsertWithTag(statsBuf.GetEndIter(), d.Text, getTag(tag))
				} else {
					statsBuf.Insert(statsBuf.GetEndIter(), d.Text)
				}
			}
			statsBuf.Insert(statsBuf.GetEndIter(), "\n\n")
		}
	}
}

func mustMarshal(v interface{}) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return b
}
