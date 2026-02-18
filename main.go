package main

import (
	"encoding/json"
	"fmt"
	"goctx/internal/apply"
	"goctx/internal/builder"
	"goctx/internal/model"
	"goctx/internal/ui"
	"io"
	"os"
	"regexp"
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
	default:
		fmt.Println("Commands: apply, gui")
	}
}

func runBuild() {
	// CLI default: 128k context, no smart mode unless specified
	output, _ := builder.BuildSelectiveContext(".", "Manual Build", nil, 128000, false)
	json.NewEncoder(os.Stdout).Encode(output)
}

func runApply() {
	data, _ := io.ReadAll(os.Stdin)
	text := string(data)

	// Strip markdown backticks if present
	re := regexp.MustCompile(`(?s)\{.*\"files\".*\}`)
	match := re.FindString(text)
	if match == "" {
		match = text
	}

	var input model.ProjectOutput
	if err := json.Unmarshal([]byte(match), &input); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// New: Progress tracking for CLI
	err := apply.ApplyPatch(".", input, func(phase, desc, logLine string) {
		if phase != "" {
			fmt.Printf("\n[%s] %s\n", phase, desc)
		}
		if logLine != "" {
			fmt.Printf("  %s\n", logLine)
		}
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nPatch applied successfully.")
}
