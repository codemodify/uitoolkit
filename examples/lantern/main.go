// Command lantern is the shaped player demo: a main window cut to its
// skin's outline, a playlist window anchored to its edge, and one control
// that drops the skin and runs the same app in an ordinary theme.
//
// It is a **visual demo**. Nothing is decoded and nothing is played: the
// status bar says so, the About box says so, and every menu item that would
// need a media stack says so when it is picked. There is no audio stack in
// this toolkit and there is not going to be one.
//
//	lantern                       # the skin, with the playlist anchored right
//	lantern -themed               # the same app with the skin already dropped
//	lantern -only main            # the main window on its own
//	lantern -scale 1.75
//	lantern -shot out/            # one frame of each window and each look
//
// Ctrl+K is the switch, and it is the argument: a skin here is a *pack*, so
// taking one off is the same call Settings makes for any theme. The same
// widget tree comes back in another look, with the same tab order and the
// same accessibility tree, and the window is a rectangle again because the
// outline belonged to the skin.
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
	"github.com/codemodify/uitoolkit/internal/players/lantern"
	"github.com/codemodify/uitoolkit/style"
)

func main() {
	theme := flag.String("theme", "", "the pack to start in (default: the skin, or -themed's pack)")
	themed := flag.Bool("themed", false, "start with the skin dropped")
	only := flag.String("only", "", "main (default: the player and its playlist)")
	scale := flag.Float64("scale", 0, "display scale (0: the desktop's, or $UITK_SCALE)")
	shot := flag.String("shot", "", "paint one frame of each window and each look into this directory")
	at := flag.Duration("at", 97*time.Second, "where the head is in -shot mode")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("lantern: ")

	pack := *theme
	if pack == "" {
		pack = lantern.Skin
		if *themed {
			pack = lantern.Themed
		}
	}
	os.Setenv(style.ThemeEnv, pack)

	headless := *shot != ""
	a := uitoolkit.New(uitoolkit.Options{Headless: headless, Scale: float32(*scale)})
	p, err := lantern.New(a, lantern.Options{
		Headless: headless, Scale: float32(*scale),
		NoList: *only == "main", Themed: *themed,
	})
	if err != nil {
		log.Fatal(err)
	}

	if headless {
		if err := os.MkdirAll(*shot, 0o755); err != nil {
			log.Fatal(err)
		}
		for _, look := range []struct {
			name string
			skin bool
		}{{"skinned", true}, {"themed", false}} {
			p.SetSkinned(look.skin)
			p.Pose(*at)
			a.PumpOnce()
			for _, w := range []struct {
				name string
				win  *app.Window
			}{{"main", p.Main}, {"playlist", p.List}} {
				if w.win == nil {
					continue
				}
				path := filepath.Join(*shot, "lantern-"+look.name+"-"+w.name+".png")
				if err := w.win.WritePNG(path); err != nil {
					log.Fatal(err)
				}
				fmt.Println("wrote", path)
			}
		}
		return
	}

	p.Start()
	a.Post(func() {
		log.Printf("backend=%s theme=%s scale=%.2f shaped=%v places-windows=%v",
			a.BackendName(), pack, p.Main.Scale(), p.Main.ShapeActive(), p.Desk.Places())
	})
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
