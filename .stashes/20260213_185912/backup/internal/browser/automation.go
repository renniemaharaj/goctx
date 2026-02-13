package browser

import (
	"encoding/json"
	"errors"
	"goctx/internal/model"
	"regexp"

	"github.com/go-rod/rod"
)

type AIWorker struct {
	Browser *rod.Browser
}

func NewAIWorker() *AIWorker {
	return &AIWorker{Browser: rod.New().MustConnect()}
}

func (w *AIWorker) FetchAIUpdate(prompt string) (model.ProjectOutput, error) {
	// page := w.Browser.MustPage("https://google.com") // Placeholder for specific AI URL

	// Logic to inject prompt into textarea and wait for response
	// This is a template for the scraping logic:

	// 1. Find textarea: page.MustElement("textarea").MustInput(prompt).MustType(input.Enter)
	// 2. Wait for code blocks: page.MustWaitIdle()

	// For now, we simulate the scraping result of the page content:
	pageContent := "{...}" // page.MustElement("body").MustText()

	return ExtractProjectState(pageContent)
}

func ExtractProjectState(input string) (model.ProjectOutput, error) {
	re := regexp.MustCompile(`(?s)\{.*\"project_tree\".*\"files\".*\}`)
	match := re.FindString(input)
	if match == "" {
		return model.ProjectOutput{}, errors.New("no valid project state JSON found in response")
	}

	var output model.ProjectOutput
	err := json.Unmarshal([]byte(match), &output)
	return output, err
}
