package main

import (
	"encoding/json"
	"fmt"
	"os"

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
		runGUI()
	default:
		fmt.Println("Usage:")
		fmt.Println("  goctx           Build context to stdout")
		fmt.Println("  goctx apply     Apply JSON patch from stdin")
		fmt.Println("  goctx gui       Launch GUI")
	}
}

func runBuild() {
	output, err := builder.BuildContext(".")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(output)
}

func runApply() {
	var input model.ProjectOutput
	err := json.NewDecoder(os.Stdin).Decode(&input)
	if err != nil {
		fmt.Println("Invalid JSON input")
		return
	}

	err = apply.ApplyPatch(".", input)
	if err != nil {
		fmt.Println("Apply failed:", err)
		return
	}

	fmt.Println("Patch applied successfully.")
}

func runGUI() {
	ui.Run()
}
