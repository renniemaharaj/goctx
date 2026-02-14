package browser

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

var singleton *rod.Browser

func Get() *rod.Browser {
	if singleton == nil {
		// Use UserMode to bypass "Untrusted Browser" alerts
		l := launcher.NewUserMode().Leakless(true).MustLaunch()
		singleton = rod.New().ControlURL(l).MustConnect()
	}
	return singleton
}