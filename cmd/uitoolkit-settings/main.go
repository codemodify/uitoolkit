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
// a small live application window in the staged pack, with the settings
// in a folding row over it and the three config paths under it.
//
// That row is four check boxes — Animations, OS colors, OS open/save
// dialogs, OS window borders — and then the four choosers that say what
// a pack is drawn with and what draws it: Corners, Icons, Size, Paint.
// It folds onto two lines at the default window size, the options on the
// first and the choosers on the second, and onto three at the 720x520
// minimum. Paint is the renderer (look.json "renderer", UITK_PAINT's
// three values) and is the one setting here that a window already open
// cannot take; the chooser says so. The icon chooser lists
// classic/sharp plus wide premiere PNG sets copied into
// ~/.config/uitoolkit/icons/<set>/.
//
// Where the caption buttons of a frame the toolkit draws go is not a box
// of its own: unticking OS window borders puts them where the theme says, which
// is the only opinion there is once the toolkit is drawing the frame.
//
// The window in the pane is a sample, with one exception: its File menu's
// Open… and Save… open the file dialog OS open/save dialogs is asking
// for — the desktop's or the toolkit's — so the option can be looked at
// instead of read. They open nothing: a chosen path is said on the
// sample's status bar and dropped. -plain-preview takes even that away,
// for pictures.
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
	plain := flag.Bool("plain-preview", false, "make the previewed window make-believe all through: its File menu opens no dialog (for screenshots)")
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
