package browser

import (
	"encoding/json"
	"errors"
	"goctx/internal/model"
	"regexp"
	"time"
)

func ProcessWithAI(prompt string) (model.ProjectOutput, error) {
	b := Get()
	page := b.MustPage("https://aistudio.google.com/app/prompts/new") // Direct to AI Studio
	
	// Note: Selectors here depend on the specific UI state
	// This assumes a standard content-editable or textarea
	textarea := page.MustElement("textarea, [contenteditable="true"]").MustWaitVisible()
	textarea.MustInput(prompt)
	
	// Simulate hitting CMD+Enter or clicking send
	page.Keyboard.MustType("\n") 
	
	// Wait for the AI to finish generating
	time.Sleep(5 * time.Second)
	
	content := page.MustElement("body").MustText()
	return ExtractProjectState(content)
}

func ExtractProjectState(input string) (model.ProjectOutput, error) {
	re := regexp.MustCompile(`(?s)\{.*\"project_tree\".*\"files\".*\}`) 
	match := re.FindString(input)
	if match == "" {
		return model.ProjectOutput{}, errors.New("AI response did not contain a valid JSON project state")
	}

	var output model.ProjectOutput
	err := json.Unmarshal([]byte(match), &output)
	return output, err
}