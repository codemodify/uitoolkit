// Command uitoolkit-settings is the toolkit appearance editor.
//
//	go run ./cmd/uitoolkit-settings
//	go run ./cmd/uitoolkit-settings -headless          # writes settings.png
//	go run ./cmd/uitoolkit-settings -screenshot docs/screenshots
//	go run ./cmd/uitoolkit-settings -page appearance -headless
//	go run ./cmd/uitoolkit-settings -plain-preview -screenshot out.png
//	go run ./cmd/uitoolkit-settings -version
//
// Settings is one page: the theme browser down a column on the left,
// and on the right, never scrolling away, the thing it is browsing for —
// a small live application window in the staged pack, with the five
// on/off options in a row over it (Animations, OS open/save dialogs, OS
// borders, Theme buttons, OS colors) and the three config paths under it.
//
// The preview sets three of the things it shows: the icon set, the size
// its glyphs are drawn at and its corner style are combo boxes on a bar
// of their own over the sample's menu bar, each behind the word that
// says what it sets — Icons, Size, Corners. Everything else in that
// window is a sample. The icon chooser lists classic/sharp plus wide
// premiere PNG sets copied into ~/.config/uitoolkit/icons/<set>/.
// -plain-preview leaves that bar off, for pictures of the sample alone;
// nothing can be chosen while it is set.
// The column is four bare controls — search, decade filter, the list of
// packs, Export — with no heading over them and no group box around
// them: the window's title bar says which application this is, and the
// only list on the page does not need a legend to say it lists themes.
// -version is where the version went when that heading's version label
// went with it; it is the only place the command states it.
//
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
	page := flag.String("page", "", "the section of the page you came for: theme (default), behaviour, preview, files — there is one page and nothing on it is under a fold, so the flag names what is already on screen and changes nothing; the old page names themes/appearance/packs/about/desktop still resolve rather than fail")
	plain := flag.Bool("plain-preview", false, "draw the preview as the sample application alone, without the bar of live settings (for screenshots: nothing can be chosen)")
	showVersion := flag.Bool("version", false, "print the toolkit version and exit")
	flag.Parse()

	// The version used to be a label under a "Settings" heading at the
	// top of the column. The heading is gone — a window says what it is
	// in its title bar — and the version came off the page with it. This
	// is where it is now, which is where a command's version belongs.
	if *showVersion {
		fmt.Println("uitoolkit-settings " + uitoolkit.Version)
		return
	}

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
