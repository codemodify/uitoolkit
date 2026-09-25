// Command uitoolkit-settings is the toolkit appearance editor.
//
//	go run ./cmd/uitoolkit-settings
//	go run ./cmd/uitoolkit-settings -headless          # writes settings.png
//	go run ./cmd/uitoolkit-settings -screenshot docs/screenshots
//	go run ./cmd/uitoolkit-settings -page appearance -headless
//
// The theme picker previews embedded starters and exported user packs.
// The icon picker lists classic/sharp plus wide premiere PNG sets
// copied into ~/.config/uitoolkit/icons/<set>/. Apply writes {theme, icons} to
// $XDG_CONFIG_HOME/uitoolkit/look.json and running apps that watch
// the file reload without a restart. Export writes
// themes/<name>/theme.json. Close without Apply discards staged
// changes. See docs/settings.md.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/cmd/uitoolkit-settings/settingsapp"
	"github.com/codemodify/uitoolkit/icons"
	"github.com/codemodify/uitoolkit/platform"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write settings.png")
	shot := flag.String("screenshot", "", "write settings.png into this directory (or file) and exit")
	stage := flag.String("stage", "", "open with this theme staged in the preview (not applied)")
	page := flag.String("page", "", "open on this page: themes (default), appearance, packs, about")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{
		Headless:         *headless || *shot != "",
		DisableLookWatch: true, // stage locally; Apply is the only writer
	})
	// The window icon the desktop shows in its title bar, task bar and switcher.
	a.SetIcon(icons.AppIconRGB("settings", 0x60, 0x70, 0x80)...)
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Settings", Width: 1024, Height: 860, MinWidth: 720, MinHeight: 520,
		Headless: *headless || *shot != "",
	})
	if err != nil {
		log.Fatal(err)
	}
	if *stage != "" || *page != "" {
		win.SetContent(settingsapp.SettingsAppOpen(a, win, *stage, *page))
	} else {
		win.SetContent(settingsapp.SettingsApp(a, win))
	}
	if *shot != "" || *headless {
		out := "settings.png"
		if *shot != "" {
			out = shotPath(*shot)
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				log.Fatal(err)
			}
		}
		if err := win.WritePNG(out); err != nil {
			log.Fatal(err)
		}
		fmt.Println("wrote", out)
		return
	}
	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}

func shotPath(p string) string {
	st, err := os.Stat(p)
	if err == nil && st.IsDir() {
		return filepath.Join(p, "settings.png")
	}
	if err != nil && filepath.Ext(p) == "" {
		return filepath.Join(p, "settings.png")
	}
	return p
}
