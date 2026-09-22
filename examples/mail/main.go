// Command mail is a convenience launcher: in-process mailclientd
// (MemoryStore) plus mailclientui on a temp Unix socket. Prefer the split
// binaries for a real two-process run:
//
//	go run ./cmd/mailclientd
//	go run ./cmd/mailclientui
//
//	UITK_SCENE=auto go run ./examples/mail
//	go run ./examples/mail -headless
//	go run ./examples/mail -screenshot docs/screenshots
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/icons"
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

	sock, stop, err := mail.StartDemo(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer stop()
	cli, err := mail.DialWait(sock, 2*time.Second)
	if err != nil {
		log.Fatal(err)
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
	// The window icon the desktop shows in its title bar, task bar and switcher.
	a.SetIcon(icons.AppIconRGB("mail", 0x2f, 0x6f, 0xd0)...)
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Mail", Width: 1280, Height: 800, MinWidth: 860, MinHeight: 560,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetContent(mail.Open(a, win, cli, mail.AppOptions{
		Light: style.LookAppearance(look).Theme == style.ThemeLight, Layout: layout,
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
