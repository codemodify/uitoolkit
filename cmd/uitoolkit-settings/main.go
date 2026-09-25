// Command uitoolkit-settings is the toolkit appearance editor.
//
//	go run ./cmd/uitoolkit-settings
//	go run ./cmd/uitoolkit-settings -headless          # writes settings.png
//	go run ./cmd/uitoolkit-settings -screenshot docs/screenshots
//	go run ./cmd/uitoolkit-settings -page appearance -headless
//	go run ./cmd/uitoolkit-settings -plain-preview -screenshot out.png
//
// Settings is one page: the theme browser and the behaviour switches
// down a column on the left, and on the right, never scrolling away, the
// thing those choices are about — a small live application window in the
// staged pack, with the switch that says where its colours come from
// over it and the three config paths under it.
//
// The preview sets three of the things it shows: the icon set, the size
// its glyphs are drawn at and its corner style are combo boxes on a bar
// of their own over the sample's menu bar, each behind the word that
// says what it sets — Icons, Size, Corners. Everything else in that
// window is a sample. The icon chooser lists classic/sharp plus wide
// premiere PNG sets copied into ~/.config/uitoolkit/icons/<set>/.
// -plain-preview leaves that bar off, for pictures of the sample alone;
// nothing can be chosen while it is set.
// Apply writes {theme, icons} to
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
	page := flag.String("page", "", "open the page at this section: theme (default), behaviour, desktop, preview, files — the old page names themes/appearance/packs/about still resolve")
	plain := flag.Bool("plain-preview", false, "draw the preview as the sample application alone, without the bar of live settings (for screenshots: nothing can be chosen)")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{
		Headless:         *headless || *shot != "",
		DisableLookWatch: true, // stage locally; Apply is the only writer
	})
	// The window icon the desktop shows in its title bar, task bar and switcher.
	a.SetIcon(icons.AppIconRGB("settings", 0x60, 0x70, 0x80)...)
	win, err := a.NewWindow(platform.WindowOptions{
		Title: windowTitle("Settings"), Width: 1024, Height: 860, MinWidth: 720, MinHeight: 520,
		Headless: *headless || *shot != "",
	})
	if err != nil {
		log.Fatal(err)
	}
	if *stage != "" || *page != "" || *plain {
		win.SetContent(settingsapp.SettingsAppWith(a, win, settingsapp.SettingsOptions{
			Theme: *stage, Page: *page, PlainPreview: *plain,
		}))
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

// windowTitle is what a window of this app is called on the desktop: its
// own name for the window under the toolkit's prefix, so that a desktop
// with several uitoolkit windows open says which toolkit they belong to.
// Settings has one window and never retitles it, but the samples all
// name their windows through a helper like this one and there is no
// reason for the editor of the look to be the odd one out.
func windowTitle(name string) string { return "uitoolkit - " + name }

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
