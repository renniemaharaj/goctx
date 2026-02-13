package browser

import (
	"goctx/internal/model"
	"time"

	"github.com/go-rod/rod/lib/input"
)

func ProcessWithAI(prompt string) (model.ProjectOutput, error) {
	b := Get()
	page := b.MustPage("https://aistudio.google.com/app/prompts/new")

	textarea := page.MustElement("textarea, [contenteditable=u0027trueu0027]").MustWaitVisible()
	textarea.MustInput(prompt)

	// Use the proper input.Key constant
	page.Keyboard.MustPress(input.Enter)

	time.Sleep(10 * time.Second)

	content := page.MustElement("body").MustText()
	return ExtractProjectState(content)
}
