package tourapp

import (
	"os"
	"strconv"
	"time"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Page seven: the four things an application asks the desktop for that
// have nothing to do with drawing — the clipboard, an icon in the tray,
// a notification, and a file dialog that is the desktop's rather than
// ours. Each of them can be absent, and a toolkit that pretends
// otherwise ships apps with dead buttons, so every one of them is
// reported here as available or not before it is offered.

func init() {
	p := &tourPages[pageDesktop]
	p.title = "The desktop around the window"
	p.proof = "The clipboard both ways, an icon in the system tray with a menu, a notification, " +
		"and the desktop's own file dialog beside the toolkit's — each reported present or absent."
	p.try = "Copy the text and paste it back, put an icon in the tray, or open either file dialog."
	p.build = buildDesktopPage
}

type desktopPage struct {
	t     *tourState
	field *widgets.TextField
	seen  *widgets.Label
	// Two items, because the interesting thing about the tray is who
	// draws the menu: tray is HostMenu (the desktop draws a dbusmenu),
	// trayOurs is ToolkitMenu (we draw a PopupMenu in this theme). Both
	// can be up at once, which is the only way to compare them.
	tray     platform.StatusItem
	trayOurs platform.StatusItem
	trayB    *widgets.Button
	trayOurB *widgets.Button
	notes    *widgets.Button
	// notifier sends the page's notifications (made on first use), and
	// sent counts them so each replaces the last.
	notifier *platform.Notifier
	sent     int
	// opened is how the last link went.
	opened string
	facts  *widgets.TextArea
	// picked is the last path a file dialog came back with.
	picked string
	// clip is what the app last read out of the clipboard.
	clip string
}

func buildDesktopPage(t *tourState) widget.Component {
	p := &desktopPage{t: t}
	t.own(pageDesktop, p)

	// ---- the clipboard ---------------------------------------------------

	p.field = widgets.NewTextField("Cut me, copy me, paste me somewhere else.", "", nil)
	p.field.SetAccessibleName("Clipboard text")
	p.seen = tourNote("")

	copyB := widgets.NewButton("Copy to the clipboard", func() {
		platform.ClipboardSet(p.field.Text)
		p.note("Put on the desktop's CLIPBOARD — paste it into any other application.")
	})
	copyB.Primary = true
	pasteB := widgets.NewButton("Read it back", func() {
		p.clip = platform.ClipboardGet()
		if p.clip == "" {
			p.note("The clipboard is empty, or holds something that is not text.")
			return
		}
		p.field.SetText(p.clip)
		p.note("Read " + plural(len(p.clip), "byte") + " back off the clipboard.")
	})

	clip := widgets.NewPanel("The clipboard",
		p.field,
		widgets.NewRow(copyB, pasteB).WithGap(6),
		tourNote("Ctrl+C, Ctrl+X and Ctrl+V in any field here go to the desktop's CLIPBOARD, and a "+
			"middle click pastes the PRIMARY selection — the X11 convention that Wayland kept as "+
			"primary-selection-v1. The readout shows both, as they are right now."),
		p.seen,
	)
	clip.Content().Spec.Gap = 8

	// ---- the tray ---------------------------------------------------------

	p.trayB = widgets.NewButton("Put an icon in the tray", func() { p.toggleTray(platform.HostMenu) })
	p.trayOurB = widgets.NewButton("…and one we draw ourselves", func() { p.toggleTray(platform.ToolkitMenu) })
	p.notes = widgets.NewButton("Send a notification", func() { p.notify() })
	tray := widgets.NewPanel("The system tray and notifications",
		widgets.NewRow(p.trayB, p.trayOurB, p.notes).WithGap(6),
		tourNote("Two icons, one difference: the first exports a dbusmenu and the desktop draws the "+
			"menu; the second says Menu=/NO_DBUSMENU, so the desktop asks us and we draw it in this "+
			"theme. Right-click each. The tree is the same, submenu and all."),
		tourNote("The notification needs no tray icon: it goes to the notification portal, or "+
			"straight to the desktop's notification server outside a sandbox, with two buttons; "+
			"clicking it raises this window and says which was clicked."),
		tourNote("The icon is a StatusNotifierItem on Linux, which is what Plasma, Waybar and the "+
			"GNOME extensions read; the menu under it is a real dbusmenu the desktop draws itself, so "+
			"it looks like every other tray menu rather than like us."),
		tourNote("With no tray host running the toolkit hands back a stub whose methods quietly "+
			"succeed, so an app is never broken by a desktop that has no tray — the readout says which "+
			"you got."),
	)
	tray.Content().Spec.Gap = 8

	// ---- file dialogs ------------------------------------------------------

	ours := widgets.NewButton("The toolkit's dialog", func() { p.pick(false) })
	theirs := widgets.NewButton("The desktop's dialog", func() { p.pick(true) })
	dialogs := widgets.NewPanel("File dialogs",
		widgets.NewRow(ours, theirs).WithGap(6),
		tourNote("The toolkit's dialog is an overlay in this window, themed with everything else, and "+
			"it works anywhere — including on a machine with no portal at all. The desktop's is "+
			"xdg-desktop-portal's FileChooser: KDE's dialog under Plasma, GNOME's under GNOME, with "+
			"their bookmarks and their recent files, which is what a sandboxed app must use."),
		tourNote("Settings has a switch that makes every dialog in every uitoolkit app the desktop's; "+
			"UITK_NATIVE_DIALOGS=1 does the same for one run. Without a portal the toolkit's shows "+
			"instead of nothing."),
	)
	dialogs.Content().Spec.Gap = 8

	// ---- links ---------------------------------------------------------------

	web := widgets.NewLinkButton("The project's page", "https://github.com/codemodify/uitoolkit")
	home := widgets.NewLinkButton("Your home folder", homeOrRoot())
	for _, l := range []*widgets.LinkButton{web, home} {
		l := l
		l.OnError = func(err error) { p.opened = err.Error(); p.note("Could not open " + l.URI + ": " + err.Error()) }
	}
	web.OnOpen = func(uri string) { p.open(web, uri) }
	home.OnOpen = func(uri string) { p.open(home, uri) }
	links := widgets.NewPanel("Links",
		widgets.NewRow(web, home).WithGap(12),
		tourNote("A link opens in the desktop's application for it through the OpenURI portal — the "+
			"browser, the file manager — as this window's request, so an \"Open with…\" the desktop "+
			"asks is this window's child. With no portal, xdg-open opens it."),
	)
	links.Content().Spec.Gap = 8

	panel, facts := tourReadout("What this desktop offers")
	p.facts = facts

	stage := widgets.NewColumn(clip, tray, links, dialogs, widgets.NewSpacer()).WithGap(10)
	stage.AddFlex(widgets.NewSpacer(), 1)

	p.refresh()
	t.onClose(p.dropTray)
	return tourStage(tourScroll("Desktop page", stage), panel)
}

func (p *desktopPage) toggleTray(chrome platform.StatusMenuChrome) {
	ours := chrome == platform.ToolkitMenu
	if p.itemFor(chrome) != nil {
		p.dropTrayItem(chrome)
		p.note("Tray icon removed.")
		return
	}
	id, title, tip := "uitoolkit-tour", "uitoolkit tour", "The tour — the desktop draws this menu"
	icon := "applications-graphics"
	if ours {
		id, title = "uitoolkit-tour-ours", "uitoolkit tour (our chrome)"
		tip, icon = "The tour — we draw this menu, in the tour's theme", "applications-development"
	}
	item, err := p.t.a.NewStatusItem(platform.StatusItemOptions{
		ID:         id,
		Title:      title,
		Tooltip:    tip,
		Icon:       platform.StatusIcon{Name: icon},
		MenuChrome: chrome,
		Menu:       app.StatusMenuFromItems(p.trayMenu()),
		OnClick: func() {
			p.t.win.Show()
			p.t.win.Raise()
			p.note("The tray icon was clicked.")
		},
	})
	if err != nil {
		p.note("The tray refused: " + err.Error())
		return
	}
	if ours {
		p.trayOurs = item
	} else {
		p.tray = item
	}
	drawn := "the desktop"
	if ours {
		drawn = "us"
	}
	p.note("Registered " + title + " through " + item.Backend() + " — live host: " +
		yesNo(item.Alive()) + ", menu drawn by " + drawn + ".")
	p.refresh()
}

// trayMenu is the same tree for both icons, so the only difference a
// person sees when they right-click is who drew it. It is nested two deep
// on purpose: one level proves less, and a child of a child is the depth
// that used to make the layout signature panic.
func (p *desktopPage) trayMenu() []*widgets.MenuItem {
	return []*widgets.MenuItem{
		widgets.Item("Raise the tour", func() {
			p.t.win.Show()
			p.t.win.Raise()
		}),
		widgets.Submenu("Open a page",
			widgets.Item("Shapes", func() { p.goToPage(pageShapes) }),
			widgets.Item("Skins", func() { p.goToPage(pageSkins) }),
			widgets.Item("Docking", func() { p.goToPage(pageDock) }),
			widgets.Sep(),
			widgets.Submenu("Further in",
				widgets.Item("Access", func() { p.goToPage(pageAccess) }),
				widgets.Item("Drag and drop", func() { p.goToPage(pageDrag) }),
			),
		),
		widgets.Sep(),
		widgets.Item("Take both icons away", func() {
			p.dropTray()
			p.note("Both tray icons removed, from a tray menu.")
		}),
	}
}

// goToPage opens a tour page from the tray menu and brings the window
// forward with it: a menu item that quietly changed something behind a
// window nobody can see has not done what it said.
func (p *desktopPage) goToPage(page int) {
	p.t.openPage(page)
	p.t.win.Show()
	p.t.win.Raise()
	p.note("Opened the " + tourPages[page].name + " page from the tray menu.")
}

func (p *desktopPage) itemFor(chrome platform.StatusMenuChrome) platform.StatusItem {
	if chrome == platform.ToolkitMenu {
		return p.trayOurs
	}
	return p.tray
}

func (p *desktopPage) notify() {
	if p.notifier == nil {
		p.notifier = p.t.a.NewNotifier(platform.NotifierOptions{AppName: "uitoolkit tour"})
	}
	p.sent++
	n := platform.DesktopNotification{
		ID:       "tour",
		Title:    "uitoolkit tour",
		Body:     "Notification " + strconv.Itoa(p.sent) + ", sent at " + time.Now().Format("15:04:05") + ".",
		IconName: "dialog-information",
		Actions: []platform.NotificationAction{
			{ID: "raise", Label: "Raise the tour"},
			{ID: "thanks", Label: "Thanks"},
		},
		OnActivate: func(action string) {
			p.t.win.Show()
			p.t.win.Raise()
			if action == "" {
				action = "the notification itself"
			}
			p.note("Clicked: " + action + ".")
		},
	}
	notifier := p.notifier
	go func() {
		// The desktop is asked off the UI goroutine; a notification
		// service D-Bus has to start first may take a moment.
		_, err := notifier.Send(n)
		p.t.a.Post(func() {
			if err != nil {
				p.note("Nothing to show it: " + err.Error())
				return
			}
			p.note("Sent through " + notifier.Backend() + ".")
		})
	}()
}

// open opens a link as its window asks, and says how it went.
func (p *desktopPage) open(l *widgets.LinkButton, uri string) {
	p.t.win.OpenURI(uri, func(err error) {
		if err != nil {
			p.opened = err.Error()
			if l.OnError != nil {
				l.OnError(err)
			}
			return
		}
		p.opened = uri
		p.note("Asked the desktop to open " + uri + ".")
	})
}

// dropTray takes both icons away. It is also the window's close handler:
// an icon left behind by an application that has quit is a bug people
// remember.
func (p *desktopPage) dropTray() {
	p.dropTrayItem(platform.HostMenu)
	p.dropTrayItem(platform.ToolkitMenu)
}

func (p *desktopPage) dropTrayItem(chrome platform.StatusMenuChrome) {
	item := p.itemFor(chrome)
	if item == nil {
		return
	}
	item.Close()
	if chrome == platform.ToolkitMenu {
		p.trayOurs = nil
	} else {
		p.tray = nil
	}
	p.refresh()
}

// pick opens a file dialog, the desktop's or the toolkit's. Both report
// through the same callbacks, which is the whole point of the flag.
func (p *desktopPage) pick(native bool) {
	which := "the toolkit's"
	if native {
		which = "the desktop's"
	}
	widgets.ShowFileDialog(p.t.win.Content(), widgets.FileDialogOptions{
		Title:  "Open a file — " + which + " dialog",
		Mode:   widgets.FileOpen,
		Path:   homeOrRoot(),
		Native: native,
		OnPick: func(path string) {
			p.picked = path
			p.note("Picked " + path + " (" + which + " dialog).")
		},
		OnCancel: func() { p.note("Cancelled the " + which + " dialog.") },
	})
	p.refresh()
}

func homeOrRoot() string {
	if h, err := os.UserHomeDir(); err == nil && h != "" {
		return h
	}
	return "/"
}

func (p *desktopPage) note(s string) {
	p.t.note(s)
	p.refresh()
}

func (p *desktopPage) refresh() {
	if p.trayB != nil {
		if p.tray != nil {
			p.trayB.Text = "Take the desktop's icon away"
		} else {
			p.trayB.Text = "Put an icon in the tray"
		}
		p.trayB.Invalidate()
	}
	if p.trayOurB != nil {
		if p.trayOurs != nil {
			p.trayOurB.Text = "Take our icon away"
		} else {
			p.trayOurB.Text = "…and one we draw ourselves"
		}
		p.trayOurB.Invalidate()
	}
	if p.seen != nil {
		p.seen.SetText("CLIPBOARD now: " + oneLine(platform.ClipboardGet()) +
			"\nPRIMARY now: " + oneLine(platform.ClipboardPrimaryGet()))
	}
	if p.facts == nil {
		return
	}
	trayState := "not registered"
	if p.tray != nil {
		trayState = p.tray.Backend() + ", live host: " + yesNo(p.tray.Alive())
	}
	oursState := "not registered"
	if p.trayOurs != nil {
		oursState = p.trayOurs.Backend() + ", we draw the menu"
	}
	picked := p.picked
	if picked == "" {
		picked = "nothing picked yet"
	}
	p.facts.SetText(tourFacts(
		[2]string{"backend", p.t.a.BackendName()},
		[2]string{"", ""},
		[2]string{"clipboard", oneLine(platform.ClipboardGet())},
		[2]string{"primary", oneLine(platform.ClipboardPrimaryGet())},
		[2]string{"", ""},
		[2]string{"tray host", yesNo(platform.StatusItemAvailable())},
		[2]string{"our own icon", oursState},
		[2]string{"tray item", trayState},
		[2]string{"menu chrome", trayMenuChrome()},
		[2]string{"notifications", notifierState(p.notifier)},
		[2]string{"last link", orNone(p.opened)},
		[2]string{"", ""},
		[2]string{"portal dialog", yesNo(platform.FileChooserAvailable())},
		[2]string{"preferred", nativeDialogName()},
		[2]string{"last pick", picked},
	))
}

// trayMenuChrome is the chrome the tour's tray item really gets, not the
// one it asked for: on Wayland without zwlr_layer_shell_v1 a ToolkitMenu
// is demoted to HostMenu, because its window could not be put where the
// tray clicked.
func trayMenuChrome() string {
	chrome, why := app.StatusMenuChromeFor(platform.HostMenu)
	name := "the desktop's (dbusmenu)"
	if chrome == platform.ToolkitMenu {
		name = "the toolkit's (PopupMenu)"
	}
	return name + " — " + why
}

func nativeDialogName() string {
	if style.NativeDialogs() {
		return "the desktop's (look.json nativeDialogs)"
	}
	return "the toolkit's"
}

// oneLine is a clipboard's content as a readout can show it: one line,
// cut, with its length so that an empty string and a missing selection
// are told apart.
func oneLine(s string) string {
	if s == "" {
		return "(empty)"
	}
	one := ""
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			one += " "
			continue
		}
		one += string(r)
	}
	if len([]rune(one)) > 44 {
		one = string([]rune(one)[:44]) + "…"
	}
	return strconv.Quote(one) + " (" + plural(len(s), "byte") + ")"
}

func notifierState(n *platform.Notifier) string {
	if n == nil || n.Backend() == "" {
		return "none sent yet"
	}
	return "through " + n.Backend()
}

func orNone(s string) string {
	if s == "" {
		return "none opened yet"
	}
	return s
}
