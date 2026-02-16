package runner

import (
	"os/exec"
	"runtime"
)

// Run executes a command string in the given root directory using the system shell.
// It returns the combined output (stdout+stderr) and any error encountered.
func Run(root, cmdStr string) ([]byte, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}
	cmd.Dir = root
	return cmd.CombinedOutput()
}
