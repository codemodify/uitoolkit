// Command look-like-firefox builds a browser's shell the way Firefox
// arranges one: the tab strip in the title bar, and the navigation
// controls on a second row under it.
//
// The tab strip is a BrowserTabs, which reports FillsCaption — so in a
// header bar it takes the caption's whole height and the selected tab
// meets the row below it, instead of being centred in the band like an
// ordinary item.
//
//	go run -tags theme_engine_web ./examples/uitoolkit-sample-look-like-firefox
//	go run -tags theme_engine_web ./examples/uitoolkit-sample-look-like-firefox -headless
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
// the same three-band relationship Firefox uses: a tab strip a shade
// darker than the toolbar, with the selected tab lighter than the strip
// and meeting the toolbar below it.
//
// one and the look falls back to the default — which the toolkit
// says at start-up, and app.Application.Diagnostics collects for a test.
const themePack = "linear"

// firefoxBlue is the tab-strip colour measured off a real Firefox on a
// blue desktop. A shipping application would read the desktop's own
// title-bar colour rather than fix it.
var firefoxBlue = paintengine2d.RGB(0x60/255.0, 0x94/255.0, 0xCF/255.0)

func main() {
	headless := flag.Bool("headless", false, "paint offscreen and write firefox.png")
	flag.Parse()

	// The shape is only half of looking like firefox; the other half is the
	// paint. The sample states the pack it wants and nothing else, so a
	// user's icon set, corner policy and typefaces still come from their
	// own settings — GNOME's flat chrome, which is what Firefox wears on this desktop.
	a := uitoolkit.New(uitoolkit.Options{
		Headless: *headless,
		// Without this the id would be the binary's name, and a sample
		// built as "firefox" would claim the real firefox's identity on the
		// desktop — its task-bar slot, its icon, its window rules.
		AppID: "uitoolkit-sample-look-like-firefox",
		Theme: uitoolkit.ThemeOverride{Pack: themePack},
		// Firefox takes the desktop's title-bar colour for its tab strip
		// and leaves the tool bar a light grey. The colour is stated as
		// measured — a tint carries its own ink, so the labels on a strip
		// this dark come out light without the application saying so.
		Chrome: map[string]paintengine2d.Color{
			"titleBar": firefoxBlue,
			// White bar, grey pill — not the other way round. Firefox's
			// navigation bar is white and the address bar inside it is a
			// light grey capsule; measured off a real one, the bar is
			// #FFFFFF and the capsule #F2F2F2.
			"toolBar": paintengine2d.RGB(1, 1, 1),
			"field":   style.Hex("#F2F2F2"),
			// Stated, not left to be derived. The toolkit would keep the
			// palette's dark ink here, and be right to: dark on this blue
			// reads at about 5:1. But Firefox takes the *desktop's*
			// title-bar foreground, which on this one is white, and that
			// is a fact about the desktop rather than about contrast. An
			// application that knows says so; one that does not gets an
			// ink that reads.
			"titleBarText": paintengine2d.RGB(1, 1, 1),
		},
	})
	win, err := a.NewWindow(platform.WindowOptions{
		Title: "uitoolkit-sample-look-like-firefox", Width: 1180, Height: 720,
		MinWidth: 640, MinHeight: 420, Decorations: platform.DecorationsClient,
	})
	if err != nil {
		log.Fatal(err)
	}
	// Firefox puts its tabs in the title bar under every desktop theme,
	// so the strip states it rather than inheriting a classic look's
	// separate caption row.
	win.SetTitleBar(tabStrip(win))
	// Firefox's tab strip is taller than the look's caption: 78px against
	// a real one. A design length again, so 99 draws 78. The look still
	// decides everything *in* the strip; this is only how much room the
	// application's own bar is given.
	win.SetCaptionHeight(99)
	win.SetCaptionStyle(uitoolkit.CaptionMerged)
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
	// The shield and the star are *inside* the address bar, at its two
	// ends, the way Firefox draws them — not buttons beside it. A
	// FieldBox is one well around all three, and it takes the focus ring
	// from the field, so the bar reads as one control.
	address := widgets.NewTextField("https://github.com/codemodify/uitoolkit", "Search or enter address", nil)
	omnibox := widgets.NewFieldBox(
		[]widget.Component{flat(widgets.NewIconButton(style.IconLock, "Site information", nil))},
		address,
		[]widget.Component{flat(widgets.NewIconButton(style.IconStar, "Bookmark this page", nil))},
	)
	// Firefox's address capsule is taller than a form's field: measured
	// against a real one it stands 56px in a 70px bar, where the look's
	// own field height leaves it about half that. Both this and the
	// caption height below are 1x *design* lengths, which the look scales
	// by its own density — 74 here draws 56.
	omnibox.MinHeight = 74

	// A ToolBar, not a Row on the window's background: a look paints its
	// tool bar as its own surface, and the engine fills the *selected tab*
	// with that same colour so the two meet and read as one — which is the
	// thing that makes a browser look like a browser. On a plain Row there
	// is no tool bar surface for the tab to merge into.
	bar := widgets.NewToolBar(
		widgets.ToolIconBtn(style.IconArrowLeft, "", nil),
		widgets.ToolIconBtn(style.IconArrowRight, "", nil),
		widgets.ToolIconBtn(style.IconSync, "", nil),
		widgets.ToolGrow(omnibox),
		widgets.ToolIconBtn(style.IconArchive, "", nil),
		widgets.ToolIconBtn(style.IconUser, "", nil),
		widgets.ToolWidget(flatMenu(widgets.NewMenuButton(style.IconMenu, "Open application menu",
			&widgets.MenuItem{Text: "New Tab", Shortcut: "Ctrl+T"},
			&widgets.MenuItem{Text: "New Window", Shortcut: "Ctrl+N"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "Bookmarks"},
			&widgets.MenuItem{Text: "History"},
			&widgets.MenuItem{Separator: true},
			&widgets.MenuItem{Text: "Settings"},
			&widgets.MenuItem{Text: "Quit", Shortcut: "Ctrl+Q", OnClick: win.Close}))),
	)
	return bar
}

// flat and flatMenu draw a mark with no frame until the pointer is over
// it, which is how a browser draws every mark on its chrome.
func flat(b *widgets.IconButton) *widgets.IconButton {
	b.Flat = true
	return b
}

func flatMenu(b *widgets.MenuButton) *widgets.MenuButton {
	b.Flat = true
	return b
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
