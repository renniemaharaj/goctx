package ui

import (
	"sync"

	"goctx/internal/model"
	"github.com/gotk3/gotk3/gtk"
)

type App struct {
	win *gtk.Window

	activeContext model.ProjectOutput
	pendingPatches []model.ProjectOutput

	selectedIndex int
	mu sync.Mutex
}

func NewApp() *App {
	return &App{
		selectedIndex: -1,
	}
}
