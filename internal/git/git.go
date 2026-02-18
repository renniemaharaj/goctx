package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// IsDirty checks if the workspace has uncommitted changes
func IsDirty(root string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	out, _ := cmd.Output()
	return len(strings.TrimSpace(string(out))) > 0
}

// StashPush saves changes to a stash
func StashPush(root, message string) error {
	cmd := exec.Command("git", "stash", "push", "-m", message)
	cmd.Dir = root
	return cmd.Run()
}

// StashPop restores the top stash entry
func StashPop(root string) error {
	cmd := exec.Command("git", "stash", "pop")
	cmd.Dir = root
	return cmd.Run()
}

// GetLog returns a list of recent commit hashes and messages
func GetLog(root string, limit int) ([]string, error) {
	cmd := exec.Command("git", "log", "--oneline", "-n", fmt.Sprintf("%d", limit))
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
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

// Show returns the diff/content for a specific hash
func Show(root, hash string) (string, error) {
	cmd := exec.Command("git", "show", "--color=never", hash)
	cmd.Dir = root
	out, err := cmd.Output()
	return string(out), err
}

// Checkout restores files from a specific commit
func Checkout(root, hash string) error {
	cmd := exec.Command("git", "checkout", hash, "--", ".")
	cmd.Dir = root
	return cmd.Run()
}

// AddAll stages all changes
func AddAll(root string) error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = root
	return cmd.Run()
}

// Commit creates a new commit
func Commit(root, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = root
	return cmd.Run()
}
