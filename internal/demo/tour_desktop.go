package demo

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
	tray  platform.StatusItem
	trayB *widgets.Button
	notes *widgets.Button
	facts *widgets.TextArea
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

	p.trayB = widgets.NewButton("Put an icon in the tray", func() { p.toggleTray() })
	p.notes = widgets.NewButton("Send a notification", func() {
		if p.tray == nil {
			p.note("Put the icon in the tray first — the notification goes through it.")
			return
		}
		err := p.tray.Notify(platform.Notification{
			Title: "uitoolkit tour",
			Body:  "This came from the tray item the page registered, at " + time.Now().Format("15:04:05") + ".",
		})
		if err != nil {
			p.note("The notification was refused: " + err.Error())
			return
		}
		p.note("Sent — the desktop shows it if it has a notification daemon.")
	})
	tray := widgets.NewPanel("The system tray",
		widgets.NewRow(p.trayB, p.notes).WithGap(6),
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

	panel, facts := tourReadout("What this desktop offers")
	p.facts = facts

	stage := widgets.NewColumn(clip, tray, dialogs, widgets.NewSpacer()).WithGap(10)
	stage.AddFlex(widgets.NewSpacer(), 1)

	p.refresh()
	t.onClose(p.dropTray)
	return tourStage(tourScroll("Desktop page", stage), panel)
}

func (p *desktopPage) toggleTray() {
	if p.tray != nil {
		p.dropTray()
		p.note("Tray icon removed.")
		return
	}
	item, err := p.t.a.NewStatusItem(platform.StatusItemOptions{
		ID:      "uitoolkit-tour",
		Title:   "uitoolkit tour",
		Tooltip: "The tour — click to raise its window",
		Icon:    platform.StatusIcon{Name: "applications-graphics"},
		Menu: app.StatusMenuFromItems([]*widgets.MenuItem{
			widgets.Item("Raise the tour", func() {
				p.t.win.Show()
				p.t.win.Raise()
			}),
			widgets.Sep(),
			widgets.Item("Take the icon away", func() {
				p.dropTray()
				p.note("Tray icon removed from its own menu.")
			}),
		}),
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
	p.tray = item
	p.note("Registered through " + item.Backend() + " — live host: " + yesNo(item.Alive()) + ".")
}

func (p *desktopPage) dropTray() {
	if p.tray == nil {
		return
	}
	p.tray.Close()
	p.tray = nil
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
			p.trayB.Text = "Take the icon away"
		} else {
			p.trayB.Text = "Put an icon in the tray"
		}
		p.trayB.Invalidate()
	}
	if p.notes != nil {
		p.notes.SetEnabled(p.tray != nil)
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
		[2]string{"tray item", trayState},
		[2]string{"host draws menus", yesNo(platform.HostMenuNative())},
		[2]string{"", ""},
		[2]string{"portal dialog", yesNo(platform.FileChooserAvailable())},
		[2]string{"preferred", nativeDialogName()},
		[2]string{"last pick", picked},
	))
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
