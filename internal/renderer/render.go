package renderer

import (
	"regexp"

	"github.com/gotk3/gotk3/gtk"
)

type Renderer struct {
	statsBuf     *gtk.TextBuffer
	isLoading    *bool
	statusLabel  *gtk.Label
	updateStatus func(statusLabel *gtk.Label, m string)
}

func NewRenderer(statsBuf *gtk.TextBuffer, isLoading *bool, statusLabel *gtk.Label,
	updateStatus func(statusLabel *gtk.Label, m string)) *Renderer {
	return &Renderer{
		statsBuf, isLoading, statusLabel, updateStatus,
	}
}

func highlight(buffer *gtk.TextBuffer, pattern string, tag string) {
	re := regexp.MustCompile(pattern)
	text, _ := buffer.GetText(buffer.GetStartIter(), buffer.GetEndIter(), false)
	matches := re.FindAllStringIndex(text, -1)

	for _, m := range matches {
		start := buffer.GetIterAtOffset(m[0])
		end := buffer.GetIterAtOffset(m[1])
		buffer.ApplyTagByName(tag, start, end)
	}
}

func SetupTags(buffer *gtk.TextBuffer) {
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

	tagK, _ := gtk.TextTagNew("keyword")
	tagK.SetProperty("foreground", "#c586c0")
	tab.Add(tagK)

	tagC, _ := gtk.TextTagNew("comment")
	tagC.SetProperty("foreground", "#6a9955")
	tab.Add(tagC)
}

func (r *Renderer) GetTag(n string) *gtk.TextTag {
	tab, err := r.statsBuf.GetTagTable()
	if err != nil {
		return nil
	}
	tag, _ := tab.Lookup(n)
	return tag
}
