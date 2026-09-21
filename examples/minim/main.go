// Command minim is the compact player demo: a strip 275 by 116 design
// pixels, with an equaliser and a playlist that snap to its edges and
// travel with it.
//
// It is a **visual demo**. Nothing is decoded and nothing is played: the
// clock counts, the seek bar scrubs the clock, the analyser is drawn from
// the clock, and the equaliser's faders move and filter nothing. There is no
// audio stack in this toolkit and there is not going to be one.
//
//	minim                          # the strip, the equaliser and the playlist
//	minim -theme minim-classic     # in the classic panel skin
//	minim -theme minim-silver      # in the rounded silver one
//	minim -theme breeze-night      # the same app with the skin dropped
//	minim -only strip              # the strip on its own
//	minim -scale 1.75              # at a fractional scale
//	minim -shot out/               # paint one frame of each window, write, exit
//
// The three windows snap flush when one is dragged within a few pixels of
// another's edge, and the stack travels with whichever one is dragged.
// That needs two things of the desktop — being told where a window is, and
// being able to put one somewhere — and only X11 offers both; a Wayland
// toplevel has no position at all. The strip says which of the two it got.
//
// The skin key in the strip, Ctrl+K, or a right-click anywhere on the face
// switches between Minim's three skins live, all three windows at once;
// Ctrl+Shift+K drops the skin and puts it back. The last choice is kept in
// $XDG_CONFIG_HOME/uitoolkit/minim.json and worn again on the next run
// unless -theme says otherwise.
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
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/internal/players/minim"
	"github.com/codemodify/uitoolkit/style"
)

func main() {
	theme := flag.String("theme", "", "the pack to run in: minim, minim-classic, minim-silver, or any other (default: the last one chosen, else minim)")
	only := flag.String("only", "", "strip | strip+eq | strip+list (default: all three)")
	scale := flag.Float64("scale", 0, "display scale (0: the desktop's, or $UITK_SCALE)")
	shot := flag.String("shot", "", "paint one frame of each window into this directory and exit")
	at := flag.Duration("at", 97*time.Second, "where the head is in -shot mode")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("minim: ")

	headless := *shot != ""
	if *theme == "" {
		// The skin the player was last switched to, which is the one piece
		// of state it keeps. A shot is posed, so it is never remembered.
		*theme = minim.Skin
		if r := minim.Remembered(); r != "" && !headless {
			*theme = r
		}
	}
	// The same override UITK_THEME=<pack> gives any app. The look is built
	// when the application is, so it has to be set first.
	os.Setenv(style.ThemeEnv, *theme)
	a := uitoolkit.New(uitoolkit.Options{Headless: headless, Scale: float32(*scale)})

	opts := minim.Options{Headless: headless, Scale: float32(*scale), Remember: !headless}
	switch *only {
	case "strip":
		opts.NoEq, opts.NoList = true, true
	case "strip+eq":
		opts.NoList = true
	case "strip+list":
		opts.NoEq = true
	}
	p, err := minim.New(a, opts)
	if err != nil {
		log.Fatal(err)
	}

	if headless {
		if err := os.MkdirAll(*shot, 0o755); err != nil {
			log.Fatal(err)
		}
		// Posed rather than run: the same position gives the same picture
		// every time, which is what makes one shot comparable with the one
		// before it.
		p.Pose(*at)
		a.PumpOnce()
		for _, w := range []struct {
			name string
			win  *app.Window
		}{{"strip", p.Main}, {"equaliser", p.Eq}, {"playlist", p.List}} {
			if w.win == nil {
				continue
			}
			path := filepath.Join(*shot, "minim-"+w.name+".png")
			if err := w.win.WritePNG(path); err != nil {
				log.Fatal(err)
			}
			fmt.Println("wrote", path)
		}
		return
	}

	p.Start()
	a.Post(func() {
		log.Printf("backend=%s theme=%s scale=%.2f shaped=%v places-windows=%v",
			a.BackendName(), *theme, p.Main.Scale(), p.Main.ShapeActive(), p.Desk.Places())
	})
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
