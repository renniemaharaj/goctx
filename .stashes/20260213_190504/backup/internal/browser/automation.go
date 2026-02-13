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
	// Use a reliable prompt URL
	page := b.MustPage("https://aistudio.google.com/app/prompts/new") 
	
	// Fixed syntax: Added missing comma in selector list
	textarea := page.MustElement("textarea, [contenteditable=u0027trueu0027]").MustWaitVisible()
	textarea.MustInput(prompt)
	
	page.Keyboard.MustType("\n") 
	
	// Give the AI time to generate the massive JSON
	time.Sleep(8 * time.Second)
	
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