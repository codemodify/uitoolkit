// Command gallery shows every control the toolkit has, in one window.
//
//	go run ./examples/gallery
//	go run ./examples/gallery -headless          # writes gallery.png
//	go run ./examples/gallery -tab 3             # open on the Table tab
//
// The controls themselves are the showcase package, because more than one
// application wants them: this window, and the theme preview in Settings.
// What is here is what any application is: a window, an icon, the content,
// and a run loop.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/icons"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/showcase"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write gallery.png")
	tab := flag.Int("tab", 0, "the tab to open on (0 Scroll, 1 List, 2 Tree, 3 Table, 4 Form)")
	flag.Parse()
	if *headless && os.Getenv(widgets.AnimationsEnv) == "" {
		// A still is the same every run: no fade or pulse caught mid-way.
		os.Setenv(widgets.AnimationsEnv, "0")
	}

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	// The window icon the desktop shows in its title bar, task bar and switcher.
	a.SetIcon(icons.AppIconRGB("layout", 0x1f, 0x8a, 0xc0)...)
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit gallery", Width: 1000, Height: 760, MinWidth: 720, MinHeight: 480,
	})
	if err != nil {
		log.Fatal(err)
	}
	// showcase.App wires the showcase's Window, Dark and Quit controls to
	// this application; showcase.Window takes a host of your own instead.
	win.SetContent(showcase.App(a, win, a.Look().Name() == "light"))
	if *tab > 0 {
		selectTab(win, *tab)
	}
	if *headless {
		if err := win.WritePNG("gallery.png"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote gallery.png")
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

// selectTab opens the showcase on one of its tabbed views: walk the
// window's widget tree for the tab view and tell it which page to show.
func selectTab(w *app.Window, i int) {
	widget.Walk(w.Content(), func(c widget.Component) {
		if tv, ok := c.(*widgets.TabView); ok {
			tv.Select(i)
		}
	})
}
