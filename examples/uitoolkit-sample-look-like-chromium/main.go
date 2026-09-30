// Command look-like-chromium builds a browser's shell the way Chromium
// arranges one, which differs from Firefox's in one deliberate way: the
// tabs are the title bar — there is no separate caption row above them —
// and everything else lives on a single toolbar under it.
//
//	go run ./examples/uitoolkit-sample-look-like-chromium
//	go run ./examples/uitoolkit-sample-look-like-chromium -headless
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
	headless := flag.Bool("headless", false, "paint offscreen and write chromium.png")
	flag.Parse()

	a := uitoolkit.New(uitoolkit.Options{Headless: *headless})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit", Width: 1180, Height: 720,
		MinWidth: 640, MinHeight: 420, Decorations: platform.DecorationsClient,
	})
	if err != nil {
		log.Fatal(err)
	}
	win.SetTitleBar(tabStrip())
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
	address := widgets.NewTextField("github.com/codemodify/uitoolkit", "Search Google or type a URL", nil)

	row := widgets.NewRow(
		widgets.NewIconButton(style.IconArrowLeft, "Click to go back", nil),
		widgets.NewIconButton(style.IconArrowRight, "Click to go forward", nil),
		widgets.NewIconButton(style.IconRedo, "Reload this page", nil),
		widgets.NewIconButton(style.IconLock, "View site information", nil),
		address,
		widgets.NewIconButton(style.IconStar, "Bookmark this tab", nil),
		widgets.NewIconButton(style.IconArchive, "Extensions", nil),
		widgets.NewIconButton(style.IconUser, "Profile", nil),
		widgets.NewMenuButton(style.IconMore, "Customize and control",
			&widgets.MenuItem{Text: "New tab", Shortcut: "Ctrl+T"},
			&widgets.MenuItem{Text: "New window", Shortcut: "Ctrl+N"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "History"},
			&widgets.MenuItem{Text: "Downloads", Shortcut: "Ctrl+J"},
			&widgets.MenuItem{Text: "Bookmarks"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "Settings"},
			&widgets.MenuItem{Text: "Exit", Shortcut: "Ctrl+Q", OnClick: win.Close}),
	).WithGap(4).WithPad(6)
	row.AddFlex(address, 1)
	return row
}

func page() widget.Component {
	title := widgets.NewLabel("codemodify / uitoolkit")
	title.Title = true
	body := widgets.NewLabel("A UI toolkit in pure Go. 135 theme packs. Linux, Windows and macOS.")
	body.Wrap = true
	return widgets.NewPad(48, widgets.NewColumn(title, body).WithGap(12))
}
