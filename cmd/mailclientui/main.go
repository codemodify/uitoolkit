// Command mailclientui is the Thunderbird-chrome Mail UI. It connects to
// mailclientd over a Unix socket and never speaks IMAP.
//
//	go run ./cmd/mailclientd          # other terminal (empty until add-account)
//	go run ./cmd/mailclientui
//
//	UITK_MAIL_SOCK=/tmp/mail.sock go run ./cmd/mailclientui
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

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
	sock := flag.String("socket", mail.DefaultSocket(), "mailclientd Unix socket")
	flag.Parse()

	if *shot != "" {
		if err := mail.WriteScreenshots(*shot); err != nil {
			log.Fatal(err)
		}
		return
	}

	cli, err := mail.DialWait(*sock, 3*time.Second)
	if err != nil {
		log.Fatalf("%v\nStart the daemon first: go run ./cmd/mailclientd", err)
	}
	defer cli.Close()

	look := style.PreferredLook()
	if *light {
		look = style.WithTheme(look, style.ThemeLight)
	}
	layout := mail.LayoutVertical
	if *classic {
		layout = mail.LayoutClassic
	}

	a := uitoolkit.New(uitoolkit.Options{Look: look, Headless: *headless, WatchLook: true})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, MinWidth: 860, MinHeight: 560,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetContent(mail.Open(a, win, cli, mail.AppOptions{
		Light: style.LookAppearance(look).Theme == style.ThemeLight, Layout: layout, ShowFilter: true,
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
