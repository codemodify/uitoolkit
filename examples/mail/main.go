// Command mail is a Thunderbird-chrome desktop email client on uitoolkit.
//
//	UITK_SCENE=auto go run ./examples/mail
//	go run ./examples/mail -headless          # writes mail.png
//	go run ./examples/mail -screenshot docs/screenshots
//
// Backend is an in-memory Store (see internal/mail). No IMAP/SMTP in v1.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/internal/mail"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write mail.png")
	shot := flag.String("screenshot", "", "write mail-*.png into this directory and exit")
	light := flag.Bool("light", false, "start with LightLook")
	classic := flag.Bool("classic", false, "classic layout (preview below the thread list)")
	flag.Parse()

	if *shot != "" {
		if err := mail.WriteScreenshots(*shot); err != nil {
			log.Fatal(err)
		}
		return
	}

	look := style.DarkLook()
	if *light {
		look = style.LightLook()
	}
	layout := mail.LayoutVertical
	if *classic {
		layout = mail.LayoutClassic
	}

	a := uitoolkit.New(uitoolkit.Options{Look: look, Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, MinWidth: 860, MinHeight: 560,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetContent(mail.Open(a, win, mail.NewDemoStore(), mail.AppOptions{
		Light: *light, Layout: layout, ShowFilter: true,
	}))
	if *headless {
		if err := win.WritePNG("mail.png"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote mail.png")
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
