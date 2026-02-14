package stash

import (
	"fmt"
	"os/exec"
	"goctx/internal/model"
)

// CreateStash uses git stash push to create a native git backup
func CreateStash(root string, patch model.ProjectOutput) (string, error) {
	msg := fmt.Sprintf("GoCtx: %s", patch.ShortDescription)
	cmd := exec.Command("git", "stash", "push", "-m", msg, "--include-untracked")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git stash failed: %w", err)
	}
	return "git", nil
}

// MarkApplied is a no-op for git-native system as git handles the state
func MarkApplied(root, id string) error {
	return nil
}

// DeleteStash drops a specific git stash by its ref (e.g., stash@{0})
func DeleteStash(root, id string) error {
	cmd := exec.Command("git", "stash", "drop", id)
	cmd.Dir = root
	return cmd.Run()
}
