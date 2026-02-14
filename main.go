package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"goctx/internal/ui"
)

func main() {
	if len(os.Args) == 1 {
		runBuild()
		return
	}

	switch os.Args[1] {
	case "apply":
		runApply()
	case "gui":
		ui.Run()
	case "back", "forward":
		navigateStash(os.Args[1])
	case "tidy":
		tidyStashes()
	default:
		fmt.Println("Commands: apply, gui, back, forward, tidy")
	}
}

func navigateStash(dir string) {
	stashes, _ := os.ReadDir(".stashes")
	var list []string
	for _, s := range stashes {
		if s.IsDir() { list = append(list, s.Name()) }
	}
	sort.Strings(list)
	if len(list) == 0 {
		fmt.Println("No stashes found."); return
	}

	// Current active is handled by internal/stash refs logic
	fmt.Printf("Navigation [%s] triggered. Implementation: use GUI to select version or 'back' to revert previous stash.\n", dir)
}

func tidyStashes() {
	os.RemoveAll(".stashes")
	os.Mkdir(".stashes", 0755)
	fmt.Println("Stashes tidied.")
}

func runBuild() {
	output, _ := builder.BuildSelectiveContext(".", "Manual Build")
	json.NewEncoder(os.Stdout).Encode(output)
}

func runApply() {
	data, _ := io.ReadAll(os.Stdin)
	text := string(data)
	
	// Strip markdown backticks if present
	re := regexp.MustCompile(`(?s)\{.*\"files\".*\}`)
	match := re.FindString(text)
	if match == "" { match = text }

	var input model.ProjectOutput
	if err := json.Unmarshal([]byte(match), &input); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}
	
	apply.ApplyPatch(".", input)
	fmt.Println("Patch applied successfully.")
}
