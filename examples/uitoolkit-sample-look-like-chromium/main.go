// Command look-like-chromium builds a browser's shell the way Chromium
// arranges one, which differs from Firefox's in one deliberate way: the
// tabs are the title bar — there is no separate caption row above them —
// and everything else lives on a single toolbar under it.
//
//	go run -tags theme_engine_web ./examples/uitoolkit-sample-look-like-chromium
//	go run -tags theme_engine_web ./examples/uitoolkit-sample-look-like-chromium -headless
package main

import (
	"flag"
	"log"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// themePack is the look this sample wears. Theme engines are chosen at
// build time (docs/engines.md), so a plain build has only the default
// Chrome's own relationship between the three bands: the tab strip
// is a shade darker than the toolbar, and the selected tab is lighter
// than the strip and meets the toolbar, so the two read as one surface.
// adwaita draws an underline tab instead, which is GNOME's idiom and the
// reason this sample did not read as Chromium however correct its layout.
//
// one and the look falls back to the default — which the toolkit
// says at start-up, and app.Application.Diagnostics collects for a test.
const themePack = "linear"

// chromeTint is the blue Chromium wears on a desktop whose accent is blue
// — #D2E2FC, measured off a real one. A shipping application would read
// the desktop's accent rather than fix it; the sample states it so the
// screenshot is the same on every machine.
var chromeTint = paintengine2d.RGB(0xD2/255.0, 0xE2/255.0, 0xFC/255.0)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write chromium.png")
	flag.Parse()

	// The shape is only half of looking like chromium; the other half is the
	// paint. The sample states the pack it wants and nothing else, so a
	// user's icon set, corner policy and typefaces still come from their
	// own settings. Chrome states the two surfaces Chromium themes.
	a := uitoolkit.New(uitoolkit.Options{
		Headless: *headless,
		// Without this the id would be the binary's name, and a sample
		// built as "chromium" would claim the real chromium's identity on the
		// desktop — its task-bar slot, its icon, its window rules.
		AppID: "uitoolkit-sample-look-like-chromium",
		Theme: uitoolkit.ThemeOverride{Pack: themePack},
		// Chromium paints its tab strip in the desktop's accent and its
		// tool bar a shade lighter, and the engine fills the selected tab
		// with the tool bar's colour — so stating these two is the whole
		// of looking themed. Without them the chrome is the pack's
		// neutral grey: a browser, but not Chromium on this desktop.
		Chrome: map[string]paintengine2d.Color{
			"titleBar": chromeTint,
			"toolBar":  style.Mix(chromeTint, paintengine2d.RGB(1, 1, 1), 0.55),
		},
	})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit-sample-look-like-chromium", Width: 1180, Height: 720,
		MinWidth: 640, MinHeight: 420, Decorations: platform.DecorationsClient,
	})
	if err != nil {
		log.Fatal(err)
	}
	// Chromium's tabs are its title bar on every desktop it runs on,
	// whatever the system theme is: it never grows a second caption above
	// them. That is a shape the application states, not one the look picks.
	win.SetTitleBar(tabStrip())
	win.SetCaptionStyle(uitoolkit.CaptionMerged)
	win.SetBorderless(true)

	bar := toolbar(win)
	doc := page()
	col := widgets.NewColumn(bar, doc).WithGap(0)
	col.AddFlex(doc, 1)
	win.SetContent(col)

	if *headless {
		if err := win.WritePNG("chromium.png"); err != nil {
			log.Fatal(err)
		}
		return
	}
	a.Run()
}

// tabStrip is the whole title bar: tabs and nothing else, so the strip
// runs to the window controls and the selected tab meets the toolbar.
func tabStrip() *widgets.HeaderBar {
	tabs := widgets.NewBrowserTabs(
		"uitoolkit — a UI toolkit in Go",
		"Issues",
		"Pull requests",
	)
	tabs.OnNew = func() { tabs.AddTab(widgets.BrowserTab{Title: "New Tab"}) }
	tabs.OnClose = func(i int) { tabs.RemoveTab(i) }

	head := widgets.NewHeaderBar(nil, tabs, nil)
	head.ShowTitle = false
	return head
}

// toolbar is Chromium's single row under the tabs.
func toolbar(win *app.Window) widget.Component {
	// Chromium's order, left to right: back, forward, reload, then the
	// omnibox, then extensions, profile and the three-dot menu. The two
	// marks that look like toolbar buttons are not — the site-information
	// lock and the bookmark star live *inside* the omnibox, at its two
	// ends, which is what a FieldBox is for.
	address := widgets.NewTextField("github.com/codemodify/uitoolkit", "Search Google or type a URL", nil)
	omnibox := widgets.NewFieldBox(
		[]widget.Component{flat(widgets.NewIconButton(style.IconLock, "View site information", nil))},
		address,
		[]widget.Component{flat(widgets.NewIconButton(style.IconStar, "Bookmark this tab", nil))},
	)

	bar := widgets.NewToolBar(
		widgets.ToolIconBtn(style.IconArrowLeft, "", nil),
		widgets.ToolIconBtn(style.IconArrowRight, "", nil),
		widgets.ToolIconBtn(style.IconSync, "", nil),
		widgets.ToolGrow(omnibox),
		widgets.ToolIconBtn(style.IconArchive, "", nil),
		widgets.ToolIconBtn(style.IconUser, "", nil),
		widgets.ToolWidget(flatMenu(widgets.NewMenuButton(style.IconMore, "Customize and control",
			&widgets.MenuItem{Text: "New tab", Shortcut: "Ctrl+T"},
			&widgets.MenuItem{Text: "New window", Shortcut: "Ctrl+N"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "History"},
			&widgets.MenuItem{Text: "Downloads", Shortcut: "Ctrl+J"},
			&widgets.MenuItem{Text: "Bookmarks"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "Settings"},
			&widgets.MenuItem{Text: "Exit", Shortcut: "Ctrl+Q", OnClick: win.Close}))),
	)
	return bar
}

// flat and flatMenu draw a mark with no frame until the pointer is over
// it. Chromium's lock, star and three-dot menu are all drawn that way —
// a framed button inside the omnibox reads as a control in a control.
func flat(b *widgets.IconButton) *widgets.IconButton {
	b.Flat = true
	return b
}

func flatMenu(b *widgets.MenuButton) *widgets.MenuButton {
	b.Flat = true
	return b
}

func page() widget.Component {
	title := widgets.NewLabel("codemodify / uitoolkit")
	title.Title = true
	body := widgets.NewLabel("A UI toolkit in pure Go. 135 theme packs. Linux, Windows and macOS.")
	body.Wrap = true
	return widgets.NewPad(48, widgets.NewColumn(title, body).WithGap(12))
}
