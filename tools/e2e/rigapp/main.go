// Command rigapp is the nested-KWin rig's fixture for the two title-bar
// behaviours that need a real compositor to see: rolling the window up to
// its title bar, and keeping it above the others. It is a harness: one window
// with the toolkit's own frame (UITK_DECORATIONS=client, or look.json's
// "decorations": "toolkit"), a loud body so a roll-up is obvious, and flags
// for the title and the size.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	title := flag.String("title", "Roll Up", "window title")
	w := flag.Int("w", 520, "width")
	h := flag.Int("h", 320, "height")
	body := flag.String("body", "CONTENT", "body text")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{})
	win, err := a.NewWindow(platform.WindowOptions{Title: *title, Width: *w, Height: *h})
	if err != nil {
		log.Fatal(err)
	}
	win.SetContent(widgets.NewColumn(
		widgets.NewLabel(*body),
		widgets.NewButton("Shade", func() { win.ToggleShade() }),
		widgets.NewButton("Keep above", func() { win.ToggleKeepAbove() }),
	))
	// Report what this backend can honestly do, once the window is up.
	go func() {
		time.Sleep(1500 * time.Millisecond)
		a.Post(func() {
			fmt.Fprintf(os.Stderr, "RIGAPP %q decorations=%v shaded=%v canShade=%v canKeepAbove=%v keepAbove=%v\n",
				*title, win.Decorations(), win.Shaded(), win.CanShade(), win.CanKeepAbove(), win.KeepAbove())
		})
	}()
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
