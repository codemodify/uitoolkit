// Command demo runs the widget gallery headless with the AT-SPI2 bridge on,
// for the accessibility smoke test (tools/a11y/smoke.sh). It never opens a
// window.
package main

import (
	"os"
	"time"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/internal/demo"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

func main() {
	os.Setenv(app.A11yEnv, "1")
	a := app.New(app.Options{Look: style.DarkLook(), Headless: true, Scale: 1, DisableLookWatch: true})
	w, err := a.NewWindow(platform.WindowOptions{Title: "Gallery", Width: 1280, Height: 860, Headless: true})
	if err != nil {
		panic(err)
	}
	w.SetContent(demo.Gallery(a, w, false))
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		a.PumpOnce()
		time.Sleep(10 * time.Millisecond)
	}
}
