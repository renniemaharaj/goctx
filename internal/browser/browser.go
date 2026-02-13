package browser

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

var singleton *rod.Browser

func Get() *rod.Browser {
	if singleton == nil {
		l := launcher.New().Headless(false).MustLaunch()
		singleton = rod.New().ControlURL(l).MustConnect()
	}
	return singleton
}