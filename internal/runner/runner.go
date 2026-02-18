package runner

import (
	"bufio"
	"io"
	"os/exec"
	"runtime"
	"sync"
)

// Run executes a command string and streams output to a callback.
func Run(root, cmdStr string, onLog func(string)) ([]byte, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}
	cmd.Dir = root

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var fullOutput []byte
	var mu sync.Mutex
	wg := sync.WaitGroup{}

	stream := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			fullOutput = append(fullOutput, (line + "\n")...)
			mu.Unlock()
			if onLog != nil {
				onLog(line)
			}
		}
	}

	wg.Add(2)
	go stream(stdout)
	go stream(stderr)

	wg.Wait()
	err := cmd.Wait()
	return fullOutput, err
}
