// Command shell is an application shell of the shape code editors and web
// dashboards have settled on, built to find out whether this toolkit can
// make one. Two applications asked; this is the answer, in widgets.
//
//	go run ./examples/uitoolkit-sample-shell            # a window
//	go run ./examples/uitoolkit-sample-shell -headless  # shell.png
package main

import (
	"flag"
	"log"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/examples/uitoolkit-sample-shell/shellapp"
	"github.com/codemodify/uitoolkit/platform"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write shell.png")
	browser := flag.Bool("browser", false, "the browser shape: controls, then back/forward, then a wide field")
	mail := flag.Bool("mail", false, "the mail client shape: a pane toggle, named actions, a search field")
	bare := flag.Bool("bare", false, "hide every window control, including the window-menu button")
	out := flag.String("out", "shell.png", "where -headless writes")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "comms-mail", Width: 1100, Height: 700, MinWidth: 640, MinHeight: 420,
		Decorations: platform.DecorationsClient,
	})
	if err != nil {
		log.Fatal(err)
	}
	switch {
	case *browser:
		win.SetContent(shellapp.NewBrowser(win))
	case *mail:
		win.SetContent(shellapp.NewMail(win))
	default:
		win.SetContent(shellapp.New(win))
	}
	if *bare {
		// An application that closes and resizes from its own chrome can
		// take the window's buttons away entirely — the window-menu
		// button with them.
		for _, b := range []platform.CaptionButton{
			platform.CaptionMenu, platform.CaptionMinimize,
			platform.CaptionMaximize, platform.CaptionClose,
			platform.CaptionKeepAbove,
		} {
			win.SetCaptionButtonVisible(b, false)
		}
	}
	if *headless {
		if err := win.WritePNG(*out); err != nil {
			log.Fatal(err)
		}
		return
	}
	a.Run()
}
