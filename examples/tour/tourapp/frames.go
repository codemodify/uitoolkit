package tourapp

import (
	"strconv"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Page four: who draws the frame. This is the one page whose subject is
// the window it is in, so every control here changes this window and the
// page simply reports what happened — including the things the desktop
// refused, which is most of what is interesting about window management.
//
// Watch the tab strip while switching: under the toolkit's frame it is
// the caption, with the caption buttons beside the tabs; under the
// desktop's it drops to being the window's first row. Chromium makes the
// same move for the same reason.

func init() {
	p := &tourPages[pageFrames]
	p.title = "Whose frame is it"
	p.proof = "The same window under the desktop's frame and under the toolkit's own, with the caption " +
		"buttons in the desktop's order or the theme's — and what a maximized or tiled window gives up."
	p.try = "Switch the frame and watch where the tab strip goes."
	p.build = buildFramesPage
}

// decorPrefs are the frame choices, in the order the page lists them.
var decorPrefs = []style.DecorationsPref{style.DecorationsAuto, style.DecorationsSystem, style.DecorationsToolkit}

type framesPage struct {
	t     *tourState
	facts *widgets.TextArea
	decor *widgets.RadioGroup
	caps  *widgets.RadioGroup
	// stop ends the poll that keeps the readout honest while the desktop
	// maximizes, tiles and activates the window behind our back.
	stop func()
}

func buildFramesPage(t *tourState) widget.Component {
	p := &framesPage{t: t}
	t.own(pageFrames, p)

	// Who frames it. The preference goes through the appearance, which is
	// what Settings writes and what every open window re-reads: there is
	// no per-window override on purpose, because a desktop with a mix of
	// framed and unframed windows of one app looks broken.
	decorNames := []string{"Auto — let the policy decide", "The desktop's frame", "The toolkit's frame"}
	p.decor = widgets.NewRadioGroup(decorNames, indexOfDecorPref(decorPrefs, t.a.Decorations()), func(i int) {
		t.apply(func(ap *style.Appearance) { ap.Decorations = decorPrefs[i] })
		p.note("Asked for " + decorNames[i] + " — the desktop has the last word.")
	})
	p.decor.SetAccessibleName("Who draws the frame")

	capNames := []string{"The desktop's layout", "The theme's own layout"}
	capPrefs := []style.CaptionButtonsPref{style.CaptionButtonsDesktop, style.CaptionButtonsTheme}
	start := 0
	if t.a.CaptionButtons() == style.CaptionButtonsTheme {
		start = 1
	}
	p.caps = widgets.NewRadioGroup(capNames, start, func(i int) {
		// The caption-button preference is the application's, like the
		// frame: through the appearance, so every tour window hears it and a
		// later change of pack does not undo it.
		t.apply(func(ap *style.Appearance) { ap.CaptionButtons = capPrefs[i] })
		p.note("Caption buttons: " + capNames[i] + ".")
	})
	p.caps.SetAccessibleName("Where the caption buttons go")

	who := widgets.NewPanel("Who draws it",
		p.decor,
		tourNote("Under the toolkit's frame the tab strip above is the caption and the caption buttons "+
			"sit in it. Under the desktop's, the same strip is the window's first row and the desktop's "+
			"title bar is above it. Nothing is rebuilt either way — the header bar is told which it is."),
		widgets.NewSeparator(),
		p.caps,
		tourNote("KDE and GNOME each state an order for the caption buttons, and every era theme here "+
			"has one of its own — Windows 95 puts them at the right in its own shapes, System 7 puts a "+
			"close box at the left. The desktop's order is the default because that is what the rest of "+
			"the user's windows do."),
	)
	who.Content().Spec.Gap = 8

	maxb := widgets.NewButton("Maximize / restore", func() {
		t.win.ToggleMaximize()
		p.note("Asked the desktop to toggle maximize.")
	})
	maxb.Primary = true
	minb := widgets.NewButton("Minimize", func() {
		t.win.Minimize()
		p.note("Minimized — the tray or the task bar brings it back.")
	})
	full := widgets.NewButton("Full screen", func() {
		t.win.SetFullscreen(!t.win.WindowState().Fullscreen)
		p.note("Toggled full screen: a full-screen window has no frame at all.")
	})
	menu := widgets.NewButton("Window menu", func() {
		b := t.win.WindowRect()
		t.win.ShowWindowMenu(paintengine2d.Pt(b.Min.X+40, b.Min.Y+40))
		p.note("Asked for the window menu — the desktop's where it has one, else the toolkit's.")
	})
	move := widgets.NewButton("Move the window", func() {
		if !t.win.StartMove() {
			p.note("This desktop will not move the window on request.")
			return
		}
		p.note("The window is following the pointer — let go to drop it.")
	})

	actions := widgets.NewPanel("What the desktop will do for it",
		widgets.NewRow(maxb, minb, full).WithGap(6),
		widgets.NewRow(menu, move).WithGap(6),
		tourNote("A maximized window gives up its margin, its rounded corners and its shadow, and its "+
			"caption buttons run to the screen edge so that throwing the pointer into the corner hits "+
			"close. A tiled window gives up the borders on its tiled edges only. Both are the desktop's "+
			"decision; the toolkit is told and re-lays the frame."),
		tourNote("Tile it with the desktop's own shortcut — Meta and an arrow key on KDE — and watch "+
			"the tiled edges appear in the readout."),
	)
	actions.Content().Spec.Gap = 8

	panel, facts := tourReadout("What the desktop says")
	p.facts = facts

	stage := widgets.NewColumn(who, actions, widgets.NewSpacer()).WithGap(10)
	stage.AddFlex(widgets.NewSpacer(), 1)

	p.refresh()
	p.poll()
	t.onClose(func() {
		if p.stop != nil {
			p.stop()
		}
	})
	return tourStage(tourScroll("Frames page", stage), panel)
}

// poll re-reads the window's state a few times a second. Maximizing,
// tiling and activating happen outside the application, and there is no
// hook for them to arrive on; a page whose whole subject is that state
// has to ask.
func (p *framesPage) poll() {
	p.stop = p.t.win.AfterFunc(400*time.Millisecond, func() {
		if p.t.win.Closed() {
			return
		}
		p.refresh()
		p.poll()
	})
}

func (p *framesPage) note(s string) {
	p.t.note(s)
	// Which frame is in effect is on the Tabs page too — it is what
	// decides whether the strip above is caption or a row.
	p.t.refreshAll()
}

func (p *framesPage) refresh() {
	if p.facts == nil {
		return
	}
	t := p.t
	// The choices show what the application is set to, which another tour
	// window may have changed.
	selectQuietly(p.decor, indexOfDecorPref(decorPrefs, t.a.Decorations()))
	capsNow := 0
	if t.a.CaptionButtons() == style.CaptionButtonsTheme {
		capsNow = 1
	}
	selectQuietly(p.caps, capsNow)
	st := t.win.WindowState()
	caps := t.win.FrameCaps()
	prefs := t.a.TitleBarPrefs()
	w, h := t.win.Size()
	sw, sh := t.win.SurfaceSize()

	known := "the desktop has not said yet (everything counts as allowed)"
	if caps&platform.CapKnown != 0 {
		known = "maximize " + yesNo(caps.Can(platform.CapMaximize)) +
			" · minimize " + yesNo(caps.Can(platform.CapMinimize)) +
			" · full screen " + yesNo(caps.Can(platform.CapFullscreen)) +
			" · window menu " + yesNo(caps.Can(platform.CapWindowMenu))
	}
	p.facts.SetText(tourFacts(
		[2]string{"backend", t.a.BackendName()},
		[2]string{"asked for", decorPrefName(t.a.Decorations())},
		[2]string{"in effect", t.win.Decorations().String()},
		[2]string{"tab strip", captionOrRow(t)},
		[2]string{"", ""},
		[2]string{"maximized", yesNo(st.Maximized)},
		[2]string{"full screen", yesNo(st.Fullscreen)},
		[2]string{"tiled edges", st.Tiled.String()},
		[2]string{"constrained", st.Constrained.String()},
		[2]string{"active", yesNo(st.Activated)},
		[2]string{"resizing", yesNo(st.Resizing)},
		[2]string{"opaque screen", yesNo(st.Solid)},
		[2]string{"", ""},
		[2]string{"desktop does", known},
		[2]string{"", ""},
		[2]string{"buttons from", captionSourceName(t)},
		[2]string{"button layout", prefs.Layout.String()},
		[2]string{"conventions", prefs.Source},
		[2]string{"double-click", prefs.DoubleClick.String()},
		[2]string{"middle-click", prefs.MiddleClick.String()},
		[2]string{"right-click", prefs.RightClick.String()},
		[2]string{"drag starts", strconv.FormatFloat(float64(prefs.DragThreshold), 'f', 0, 32) + " px"},
		[2]string{"", ""},
		[2]string{"window", strconv.Itoa(w) + " x " + strconv.Itoa(h) + " px"},
		[2]string{"surface", strconv.Itoa(sw) + " x " + strconv.Itoa(sh) + " px"},
		[2]string{"scale", strconv.FormatFloat(float64(t.win.Scale()), 'f', 2, 32)},
	))
}

func indexOfDecorPref(all []style.DecorationsPref, want style.DecorationsPref) int {
	for i, p := range all {
		if p == want {
			return i
		}
	}
	return 0
}

func decorPrefName(p style.DecorationsPref) string {
	switch p {
	case style.DecorationsSystem:
		return "the desktop's frame"
	case style.DecorationsToolkit:
		return "the toolkit's frame"
	}
	return "auto (the policy decides)"
}

func captionSourceName(t *tourState) string {
	if t.a.CaptionButtons() == style.CaptionButtonsTheme {
		return "the theme"
	}
	return "the desktop"
}
