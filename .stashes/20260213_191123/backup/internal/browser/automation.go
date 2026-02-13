package browser

import (
	"encoding/json"
	"errors"
	"goctx/internal/model"
	"regexp"
	"time"
	"github.com/go-rod/rod/lib/input"
)

func ProcessWithAI(prompt string) (model.ProjectOutput, error) {
	b := Get()
	page := b.MustPage("https://aistudio.google.com/app/prompts/new") 
	textarea := page.MustElement("textarea, [contenteditable=u0027trueu0027]").MustWaitVisible()
	textarea.MustInput(prompt)
	page.Keyboard.MustType(input.Enter)
	time.Sleep(10 * time.Second)
	content := page.MustElement("body").MustText()
	return ExtractProjectState(content)
}

func ExtractProjectState(input string) (model.ProjectOutput, error) {
	re := regexp.MustCompile(`(?s)\{.*\"files\".*\}`) 
	match := re.FindString(input)
	if match == "" { return model.ProjectOutput{}, errors.New("no JSON found") }
	var out model.ProjectOutput
	err := json.Unmarshal([]byte(match), &out)
	return out, err
}