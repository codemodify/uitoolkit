// Command files is a Files / Projects dogfood app built on uitoolkit.
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
	headless := flag.Bool("headless", false, "paint offscreen and write files.png")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{Look: uitoolkit.DarkLook(), Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Files", Width: 1040, Height: 680, MinWidth: 720, MinHeight: 480,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetContent(demo.FilesApp(win))
	if *headless {
		if err := win.WritePNG("files.png"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote files.png")
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
