package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func isRepo(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".git"))
	return err == nil
}

// IsDirty checks if the workspace has uncommitted changes
func IsDirty(root string) bool {
	if !isRepo(root) {
		return false
	}
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	out, _ := cmd.Output()
	return len(strings.TrimSpace(string(out))) > 0
}

// GetStatusFiles returns paths of modified/added files in the workspace
func GetStatusFiles(root string) ([]string, error) {
	if !isRepo(root) {
		return nil, nil
	}
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if len(line) > 3 {
			// Porcelain format: XY path
			files = append(files, strings.TrimSpace(line[3:]))
		}
	}
	return files, nil
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
