// Command marquee is the big player demo: a cabinet with a display, a
// queue and a curved transport shelf, which folds down into a small shaped
// window and back.
//
// It is a **visual demo**. Nothing is decoded and nothing is played: the
// analyser is drawn from the transport's own clock and the shelf's controls
// move that clock and nothing else. There is no audio stack in this toolkit
// and there is not going to be one.
//
//	marquee                       # the cabinet
//	marquee -compact              # folded down, in its own stadium outline
//	marquee -theme breeze-night   # the same app with the skin dropped
//	marquee -scale 1.75
//	marquee -shot out/            # one frame of each mode, written, exit
//
// The fold is the thing to look at. Ctrl+M, or the button at the right-hand
// end of the shelf, changes the window's size, its layout *and* its
// silhouette at once: the cabinet wears the skin's outline, which the app
// never mentions, and the folded player wears one of its own.
//
// Escape quits.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/icons"
	"github.com/codemodify/uitoolkit/internal/players/marquee"
	"github.com/codemodify/uitoolkit/style"
)

func main() {
	theme := flag.String("theme", marquee.Skin, "the pack to run in (a skin, or any of the others)")
	compact := flag.Bool("compact", false, "open folded down")
	scale := flag.Float64("scale", 0, "display scale (0: the desktop's, or $UITK_SCALE)")
	shot := flag.String("shot", "", "paint one frame of each mode into this directory and exit")
	at := flag.Duration("at", 97*time.Second, "where the head is in -shot mode")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("marquee: ")

	if *theme != "" {
		os.Setenv(style.ThemeEnv, *theme)
	}
	headless := *shot != ""
	a := uitoolkit.New(uitoolkit.Options{Headless: headless, Scale: float32(*scale)})
	// The window icon the desktop shows in its title bar, task bar and switcher.
	a.SetIcon(icons.AppIconRGB("star", 0x7b, 0x4f, 0xc9)...)

	p, err := marquee.New(a, marquee.Options{
		Headless: headless, Scale: float32(*scale), Compact: *compact,
	})
	if err != nil {
		log.Fatal(err)
	}

	if headless {
		if err := os.MkdirAll(*shot, 0o755); err != nil {
			log.Fatal(err)
		}
		for _, mode := range []struct {
			name string
			fold bool
		}{{"full", false}, {"compact", true}} {
			p.SetCompact(mode.fold)
			p.Pose(*at)
			a.PumpOnce()
			path := filepath.Join(*shot, "marquee-"+mode.name+".png")
			if err := p.Window.WritePNG(path); err != nil {
				log.Fatal(err)
			}
			fmt.Println("wrote", path)
		}
		return
	}

	p.Start()
	a.Post(func() {
		log.Printf("backend=%s theme=%s scale=%.2f compact=%v shaped=%v",
			a.BackendName(), *theme, p.Window.Scale(), p.Compact(), p.Window.ShapeActive())
	})
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
