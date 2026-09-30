// Command look-like-firefox builds a browser's shell the way Firefox
// arranges one: the tab strip in the title bar, and the navigation
// controls on a second row under it.
//
// The tab strip is a BrowserTabs, which reports FillsCaption — so in a
// header bar it takes the caption's whole height and the selected tab
// meets the row below it, instead of being centred in the band like an
// ordinary item.
//
//	go run ./examples/uitoolkit-sample-look-like-firefox
//	go run ./examples/uitoolkit-sample-look-like-firefox -headless
package main

import (
	"flag"
	"log"

	"github.com/codemodify/uitoolkit"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write firefox.png")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit — Browser", Width: 1180, Height: 720,
		MinWidth: 640, MinHeight: 420, Decorations: platform.DecorationsClient,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetTitleBar(tabStrip(win))
	win.SetBorderless(true)
	doc := page()
	col := widgets.NewColumn(navigationBar(win), doc).WithGap(0)
	col.AddFlex(doc, 1)
	win.SetContent(col)

	if *headless {
		if err := win.WritePNG("firefox.png"); err != nil {
			log.Fatal(err)
		}
		return
	}
	a.Run()
}

// tabStrip is the title bar: the tabs themselves, filling the caption,
// with the window's controls on whichever side this era puts them.
func tabStrip(win *app.Window) *widgets.HeaderBar {
	tabs := widgets.NewBrowserTabs(
		"uitoolkit — a UI toolkit in Go",
		"Release notes",
		"docs/widgets.md",
	)
	tabs.OnNew = func() { tabs.AddTab(widgets.BrowserTab{Title: "New Tab"}) }
	tabs.OnClose = func(i int) { tabs.RemoveTab(i) }

	head := widgets.NewHeaderBar(
		[]widget.Component{widgets.NewIconButton(style.IconColumns, "Sidebars", nil)},
		tabs,
		[]widget.Component{widgets.NewIconButton(style.IconArrowDown, "List all tabs", nil)})
	head.ShowTitle = false
	return head
}

// navigationBar is Firefox's second row: back, forward, reload, the
// address field taking the rest, then the extras.
func navigationBar(win *app.Window) widget.Component {
	address := widgets.NewTextField("https://github.com/codemodify/uitoolkit", "Search or enter address", nil)

	row := widgets.NewRow(
		widgets.NewIconButton(style.IconArrowLeft, "Back", nil),
		widgets.NewIconButton(style.IconArrowRight, "Forward", nil),
		widgets.NewIconButton(style.IconRedo, "Reload", nil),
		widgets.NewIconButton(style.IconLock, "Site information", nil),
		address,
		widgets.NewIconButton(style.IconStar, "Bookmark", nil),
		widgets.NewIconButton(style.IconArchive, "Extensions", nil),
		widgets.NewIconButton(style.IconUser, "Account", nil),
		widgets.NewMenuButton(style.IconMenu, "Open application menu",
			&widgets.MenuItem{Text: "New Tab", Shortcut: "Ctrl+T"},
			&widgets.MenuItem{Text: "New Window", Shortcut: "Ctrl+N"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "Bookmarks"},
			&widgets.MenuItem{Text: "History"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "Settings"},
			&widgets.MenuItem{Text: "Quit", Shortcut: "Ctrl+Q", OnClick: win.Close}),
	).WithGap(4).WithPad(6)
	row.AddFlex(address, 1)
	return row
}

// page stands in for the rendered document.
func page() widget.Component {
	title := widgets.NewLabel("uitoolkit")
	title.Title = true
	title.Align = style.AlignCenter
	body := widgets.NewLabel("A UI toolkit in pure Go: 135 theme packs, three platforms, no C.")
	body.Align = style.AlignCenter
	body.Wrap = true
	return widgets.NewPad(64, widgets.NewColumn(title, body).WithGap(12))
}
