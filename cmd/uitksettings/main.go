// Command uitksettings is the toolkit appearance editor.
//
//	go run ./cmd/uitksettings
//	go run ./cmd/uitksettings -headless          # writes settings.png
//	go run ./cmd/uitksettings -screenshot docs/screenshots
//
// Theme, corner policy, and icon set preview in this window. Apply
// writes $XDG_CONFIG_HOME/uitoolkit/look.json and running apps that
// watch the file (Application default when Look is PreferredLook)
// reload without a restart. Close without Apply discards staged
// changes. See docs/settings.md.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/internal/demo"
	"github.com/codemodify/uitoolkit/platform"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write settings.png")
	shot := flag.String("screenshot", "", "write settings.png into this directory (or file) and exit")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{
		Headless:         *headless || *shot != "",
		DisableLookWatch: true, // stage locally; Apply is the only writer
	})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "Settings", Width: 960, Height: 780, MinWidth: 720, MinHeight: 520,
		Headless: *headless || *shot != "",
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetContent(demo.SettingsApp(a, win))
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
