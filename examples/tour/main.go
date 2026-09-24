// Command tour is the demonstration of what uitoolkit does beyond
// ordinary controls.
//
// The gallery (now inside Settings) has the widgets. This has the things
// a page of widgets cannot show: tabs that are the window's caption and
// tear out into windows of their own, panels that dock and float, a drag
// that leaves the application, the toolkit's frame against the desktop's,
// a window with a hole you can click through, a skin, and the parts that
// exist only while something runs — the accessibility tree, the
// clipboard, the tray, the display's scale.
//
//	tour                        # every page, in one window
//	tour -page shapes           # one page (see -page list)
//	tour -page tabs,frames      # a window holding just those
//	tour -theme win95           # any of the packs, skins included
//	tour -scale 1.75            # at a fractional scale
//	tour -shot out/             # one PNG per page, then exit
//	tour -sheet pages.png       # every page on one contact sheet, then exit
//	tour -headless              # paint offscreen, write tour.png, exit
//
// The tour's own navigation is its first page: the tabs along the top are
// the window's caption, and one dragged out of the strip opens a second
// tour window running that page.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/examples/tour/tourapp"
	"github.com/codemodify/uitoolkit/icons"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

func main() {
	page := flag.String("page", "", "the page(s) to open, comma separated (\"list\" names them)")
	theme := flag.String("theme", "", "the pack to run in (a skin, or any of the others)")
	scale := flag.Float64("scale", 0, "display scale (0: the desktop's, or $UITK_SCALE)")
	shot := flag.String("shot", "", "write one PNG per page into this directory and exit")
	sheet := flag.String("sheet", "", "write a contact sheet of the pages to this PNG and exit")
	headless := flag.Bool("headless", false, "paint offscreen and write tour.png")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("tour: ")

	if strings.EqualFold(*page, "list") {
		fmt.Println(strings.Join(tourapp.TourPageNames(), "\n"))
		return
	}
	pages, err := parsePages(*page)
	if err != nil {
		log.Fatal(err)
	}

	if *theme != "" {
		if _, ok := style.LoadTheme(*theme); !ok {
			log.Fatalf("unknown theme %q", *theme)
		}
		// The same override UITK_THEME=<pack> gives any app. The look is
		// built when the application is, so it has to be set first.
		os.Setenv(style.ThemeEnv, *theme)
	}

	offscreen := *headless || *shot != "" || *sheet != ""
	if offscreen && os.Getenv(style.AnimationsEnv) == "" {
		// Stills are the same every run: no fade or busy bar caught
		// halfway through.
		os.Setenv(style.AnimationsEnv, "0")
	}

	a := uitoolkit.New(uitoolkit.Options{Headless: offscreen, Scale: float32(*scale)})
	// The window icon the desktop shows in its title bar, task bar and switcher.
	a.SetIcon(icons.AppIconRGB("home", 0x3f, 0x51, 0xb5)...)

	if *shot != "" || *sheet != "" {
		if *shot != "" {
			if err := writeShots(a, *shot, pages); err != nil {
				log.Fatal(err)
			}
		}
		if *sheet != "" {
			if err := writeSheet(a, *sheet, pages); err != nil {
				log.Fatal(err)
			}
			fmt.Println("wrote", *sheet)
		}
		return
	}

	win, err := tourapp.TourWindow(a, pages...)
	if err != nil {
		log.Fatal(err)
	}
	if offscreen {
		if err := win.WritePNG("tour.png"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote tour.png")
		return
	}
	a.Post(func() {
		log.Printf("backend=%s scale=%.2f theme=%s decorations=%s carries-windows=%v",
			a.BackendName(), win.Scale(), style.LookAppearance(a.Look()).Name,
			win.Decorations(), win.DragsWindows())
	})
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

// parsePages turns "shapes,frames" into page numbers; empty is all of
// them, in order.
func parsePages(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []int
	for _, name := range strings.Split(s, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		i := tourapp.TourPageIndex(name)
		if i < 0 {
			return nil, fmt.Errorf("unknown page %q (one of: %s)", name,
				strings.Join(tourapp.TourPageNames(), ", "))
		}
		out = append(out, i)
	}
	return out, nil
}

// writeShots paints each page on its own, in a window of the size the
// tour opens at, so that every still shows one page whole.
func writeShots(a *app.Application, dir string, pages []int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, p := range shotPages(pages) {
		img, err := shootPage(a, p)
		if err != nil {
			return err
		}
		name := filepath.Join(dir, "tour-"+shotName(tourapp.TourPageNames()[p])+".png")
		if err := img.WritePNGFile(name); err != nil {
			return err
		}
		fmt.Println("wrote", name)
	}
	return nil
}

// shotPages is the pages a still or a sheet shows: the ones asked for, or
// every page in order.
func shotPages(pages []int) []int {
	if len(pages) > 0 {
		return pages
	}
	all := make([]int, len(tourapp.TourPageNames()))
	for i := range all {
		all[i] = i
	}
	return all
}

// Every still is the same page design: shotW x shotH logical pixels, which
// is shotW*scale device pixels across at a display scale — window sizes
// are logical, so a still at 1.75 is the same page drawn finer, not a
// bigger page.
const shotW, shotH = 1180, 820

// shootPage paints page p alone in an offscreen tour window and returns
// the picture.
func shootPage(a *app.Application, p int) (*paintengine2d.Image, error) {
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit tour", Width: shotW, Height: shotH, Headless: true,
	})
	if err != nil {
		return nil, err
	}
	defer win.Close()
	win.SetContent(tourapp.TourAppOpen(a, win, p))
	// Two pumps: the first lays the page out, the second lets anything the
	// page posted for after its first frame land.
	a.PumpOnce()
	a.PumpOnce()
	img := win.Capture()
	if img == nil {
		return nil, fmt.Errorf("page %s: no picture", tourapp.TourPageNames()[p])
	}
	return img, nil
}

func shotName(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), " ", "-")
}
