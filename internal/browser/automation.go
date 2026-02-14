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
	page.MustWaitIdle()

	editor := page.MustElement(`div[contenteditable="true"], .prompt-input, textarea`).MustWaitVisible()
	editor.MustClick()

	// Use KeyActions to simulate Ctrl+A accurately without character literal issues
	page.KeyActions().Press(input.ControlLeft, input.KeyA).Do()
	page.Keyboard.Press(input.Backspace)

	editor.MustInput(prompt)
	time.Sleep(500 * time.Millisecond)
	page.Keyboard.Press(input.Enter)

	time.Sleep(18 * time.Second)
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