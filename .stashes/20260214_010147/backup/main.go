package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	// Simple toggle logic for demo: in a real app, track current in meta.json
	fmt.Printf("Stash found: %s. Use GUI to select specific version or implement index tracking.\n", list[len(list)-1])
}

func tidyStashes() {
	os.RemoveAll(".stashes")
	os.Mkdir(".stashes", 0755)
	fmt.Println("Stashes tidied.")
}

func runBuild() {
	output, _ := builder.BuildContext(".")
	json.NewEncoder(os.Stdout).Encode(output)
}

func runApply() {
	var input model.ProjectOutput
	json.NewDecoder(os.Stdin).Decode(&input)
	apply.ApplyPatch(".", input)
	fmt.Println("Done.")
}
