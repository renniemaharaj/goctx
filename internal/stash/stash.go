package stash

import (
	"fmt"
	"os/exec"
	"strings"
)

// Push creates a stash with a message
func Push(root, message string) error {
	cmd := exec.Command("git", "stash", "push", "-m", message)
	cmd.Dir = root
	return cmd.Run()
}

// GetCommits returns the recent git commit history
func GetCommits(root string) ([]string, error) {
	cmd := exec.Command("git", "log", "--oneline", "-n", "30")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list commits: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []string
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}
