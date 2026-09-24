// Command inspector is a small preferences app built on uitoolkit.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/examples/inspector/inspectorapp"
	"github.com/codemodify/uitoolkit/icons"
	"github.com/codemodify/uitoolkit/platform"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write inspector.png")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	// The window icon the desktop shows in its title bar, task bar and switcher.
	a.SetIcon(icons.AppIconRGB("search", 0x2a, 0x9d, 0x8f)...)
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Inspector", Width: 860, Height: 580, MinWidth: 560, MinHeight: 400,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetContent(inspectorapp.InspectorApp(win))
	if *headless {
		if err := win.WritePNG("inspector.png"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote inspector.png")
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
