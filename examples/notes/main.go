// Command notes is a small real desktop app built on uitoolkit.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/internal/demo"
	"github.com/codemodify/uitoolkit/platform"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write notes.png")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{Look: uitoolkit.PreferredLook(), Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Notes", Width: 860, Height: 540, MinWidth: 520, MinHeight: 360,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetContent(demo.NotesApp(win))
	if *headless {
		if err := win.WritePNG("notes.png"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote notes.png")
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
