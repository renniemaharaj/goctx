package overlay

import (
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gtk"
)

type X11Overlay struct {
	window *gtk.Window
}

func NewX11Overlay() *X11Overlay {
	win, _ := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	win.SetDecorated(false)
	win.SetKeepAbove(true)
	win.SetAppPaintable(true)
	win.SetAcceptFocus(false)
	win.SetSkipTaskbarHint(true)
	win.SetSkipPagerHint(true)

	screen := win.GetScreen()
	visual, _ := screen.GetRGBAVisual()
	if visual != nil {
		win.SetVisual(visual)
	}

	win.Connect("draw", func(w *gtk.Window, cr *cairo.Context) bool {
		cr.SetSourceRGBA(0, 0, 0, 0.5)
		cr.Paint()
		return false
	})

	win.Fullscreen()
	return &X11Overlay{window: win}
}

func (o *X11Overlay) Start() error           { o.window.ShowAll(); return nil }
func (o *X11Overlay) SetOpacity(val float64) { o.window.SetOpacity(val) }
func (o *X11Overlay) Show()                  { o.window.Show() }
func (o *X11Overlay) Hide()                  { o.window.Hide() }
