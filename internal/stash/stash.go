package stash

import (
	"goctx/internal/git"
)

// Push creates a stash with a message
func Push(root, message string) error {
	return git.StashPush(root, message)
}

// GetCommits returns the recent git commit history
func GetCommits(root string) ([]string, error) {
	return git.GetLog(root, 30)
}
