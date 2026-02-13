package overlay

type Overlay interface {
	Start() error
	SetOpacity(float64)
	Show()
	Hide()
}
