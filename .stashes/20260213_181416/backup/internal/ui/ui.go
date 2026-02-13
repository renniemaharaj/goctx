package ui

import (
	"fmt"
	"os/exec"

	"github.com/gotk3/gotk3/gtk"
)

func Run() {
	gtk.Init(nil)

	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetTitle("goctx GUI")
	win.SetDefaultSize(800, 600)
	win.Connect("destroy", gtk.MainQuit)

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 5)

	btnContext, _ := gtk.ButtonNewWithLabel("Capture Context")
	btnContext.Connect("clicked", func() {
		cmd := exec.Command("goctx")
		out, err := cmd.Output()
		if err == nil {
			fmt.Println(string(out))
		}
	})

	box.Add(btnContext)
	win.Add(box)

	win.ShowAll()
	gtk.Main()
}
