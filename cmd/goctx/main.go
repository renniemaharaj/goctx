package main

import (
	"fmt"
	"os"

	"goctx/internal/ui"
)

type Command func()

var commands = map[string]Command{
	"ui": ui.Run,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("expected command")
		return
	}

	cmd, ok := commands[os.Args[1]]
	if !ok {
		fmt.Println("unknown command")
		return
	}

	cmd()
}
