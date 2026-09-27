package tourapp

import (
	"strconv"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// One window, three packs. The Skins page swaps the *application's*
// theme, which every window follows; this page is the other half of the
// answer — a pane inside a window, and a pane inside that pane, each in
// a pack of its own while everything around them keeps the window's.
//
// What makes it a cascade rather than two unrelated looks is what the
// panes do *not* say. Neither states corners, an icon set or a typeface,
// so all three are still whatever the desktop's look.json chose, in both
// panes; and neither can state a display scale, because a pane derives
// its look from the one above it, which already carries the window's.
// Pick "square corners" from the cascade row and watch it reach the
// window and both panes at once, through three different packs.
//
// The combo box and the menu button inside the inner pane are the part
// a naive cascade gets wrong. A menu is not in the window's widget tree
// at all — on Wayland it is a surface of its own, placed by the
// compositor — so nothing can reach it by walking up from the widget
// that opened it. It comes up in the pane's pack anyway.

func init() {
	p := &tourPages[pageCascade]
	p.title = "A theme per pane"
	p.proof = "Three packs in one window: the window's, a pane's, and a pane inside that pane's — " +
		"each one inheriting everything it does not state, the way CSS cascades."
	p.try = "Give the panes different packs, then open the combo box or the menu inside the inner one."
	p.build = buildCascadePage
}

// cascadePicks are the packs the two pickers offer: loud, old, new, and
// one whose chrome is pictures — enough that a wrong level is obvious at
// a glance rather than a shade of grey away.
var cascadePicks = []string{"win95", "system7", "luna", "aqua", "metal-ocean", "breeze6", "tahoe", "deck"}

type cascadePage struct {
	t *tourState
	// outer is the pane the window's tree hands a pack to, inner the
	// pane inside it that hands its own subtree another.
	outer, inner *widgets.Panel
	// plain is a pane beside outer that states nothing: the control for
	// the demonstration, and the proof that a level reaches its subtree
	// and no further.
	plain     *widgets.Panel
	outerPack int
	innerPack int
	// outerPick / innerPick are the two choosers, kept so that retheme
	// can restate them: a test (and the screenshot pass) sets the packs
	// directly, and a chooser showing something else would make the
	// picture a lie.
	// corners is the one part the panes deliberately do not state, so
	// that changing it in the *window* is seen to reach both of them.
	outerPick, innerPick *widgets.ComboBox
	corners              style.CornerStyle
	facts                *widgets.TextArea
}

func buildCascadePage(t *tourState) widget.Component {
	p := &cascadePage{t: t, outerPack: 4, innerPack: 0, corners: style.CornersTheme}
	t.own(pageCascade, p)

	p.outerPick = widgets.NewComboBox(cascadePackLabels(), p.outerPack, func(i int) {
		p.outerPack = i
		p.retheme()
	})
	p.outerPick.SetAccessibleName("The pane's pack")
	p.innerPick = widgets.NewComboBox(cascadePackLabels(), p.innerPack, func(i int) {
		p.innerPack = i
		p.retheme()
	})
	p.innerPick.SetAccessibleName("The inner pane's pack")

	corners := widgets.NewSegmented([]string{"The pack's own", "Round", "Square"}, 0, func(i int) {
		p.corners = []style.CornerStyle{style.CornersTheme, style.CornersRound, style.CornersSquare}[i]
		// The corner policy is a preference of the *application's*, not
		// of a pack: it is handed to the whole appearance, and both
		// panes inherit it through packs that never mention it.
		p.t.apply(func(ap *style.Appearance) { ap.Corners = p.corners })
		p.refresh()
	})
	corners.SetAccessibleName("Corners, for the whole window")

	p.inner = widgets.NewPanel("Inner pane", p.innerBody())
	p.inner.SetAccessibleName("The inner pane")
	p.outer = widgets.NewPanel("Pane", p.outerBody())
	p.outer.SetAccessibleName("The themed pane")
	p.plain = widgets.NewPanel("A pane that states nothing",
		widgets.NewRow(widgets.NewButton("Untouched", nil), widgets.NewCheckbox("Still the window's", true, nil)).WithGap(8),
	)
	p.plain.SetAccessibleName("The untouched pane")

	panes := widgets.NewColumn(p.outer, p.plain).WithGap(10)
	panes.AddFlex(p.outer, 1)

	stage := widgets.NewColumn(
		tourNote("This page is the window's own pack — whatever the Skins page or look.json last chose."),
		tourRow("Pane:", p.outerPick),
		tourRow("Pane inside it:", p.innerPick),
		tourRow("Corners:", corners),
		panes,
	).WithGap(8)
	stage.AddFlex(panes, 1)

	panel, facts := tourReadout("What each level says, and what it inherits")
	p.facts = facts

	p.retheme()
	return tourStage(stage, panel)
}

func cascadePackLabels() []string {
	out := make([]string, len(cascadePicks))
	for i, name := range cascadePicks {
		out[i] = name
		if pack, ok := style.LoadTheme(name); ok {
			out[i] = pack.Display()
		}
	}
	return out
}

// outerBody is what the themed pane holds: enough chrome that a pack is
// unmistakable, and the inner pane under it.
func (p *cascadePage) outerBody() widget.Component {
	ok := widgets.NewButton("Default", nil)
	ok.Primary = true
	field := widgets.NewTextField("A field in this pane", "", nil)
	field.SetAccessibleName("A field in the themed pane")
	bar := widgets.NewProgressBar(0.45)
	bar.SetAccessibleName("Progress in the themed pane")
	body := widgets.NewColumn(
		widgets.NewRow(ok, widgets.NewButton("Cancel", nil),
			widgets.NewCheckbox("Checked", true, nil)).WithGap(8),
		field,
		bar,
		p.inner,
	).WithGap(8)
	// The inner pane takes the room the outer one has left, so the two
	// levels are drawn at a size worth looking at rather than a band of
	// chrome over an empty pane.
	body.AddFlex(p.inner, 1)
	return body
}

// innerBody holds the two controls that open something of their own: a
// combo box's list and a menu, both of them surfaces outside the
// window's tree, both of them in this pane's pack.
func (p *cascadePage) innerBody() widget.Component {
	list := widgets.NewComboBox([]string{"This list opens", "in the inner pane's pack", "not the window's"}, 0, nil)
	list.SetAccessibleName("A combo box in the inner pane")
	menu := widgets.NewButton("Menu…", nil)
	menu.SetAccessibleName("A menu from the inner pane")
	menu.OnClick = func() {
		b := widget.DeviceBounds(menu)
		widgets.ShowContextMenu(menu, paintengine2d.Pt(b.Min.X, b.Max.Y),
			widgets.Item("In the inner pane's pack", nil),
			widgets.Item("Even on its own surface", nil),
			widgets.Sep(),
			widgets.Submenu("And so is a submenu",
				widgets.Item("All the way down", nil)),
		)
	}
	slider := widgets.NewSlider(0, 100, 70, nil)
	slider.SetAccessibleName("A slider in the inner pane")
	return widgets.NewColumn(
		widgets.NewRow(menu, widgets.NewButton("Nearest wins", nil)).WithGap(8),
		list,
		slider,
	).WithGap(8)
}

// retheme states the two levels. This is the whole of the API this page
// demonstrates: one call per level, naming only the pack.
func (p *cascadePage) retheme() {
	outer, inner := cascadePicks[p.outerPack], cascadePicks[p.innerPack]
	widget.SetTheme(p.outer, style.ThemeOverride{Pack: outer})
	widget.SetTheme(p.inner, style.ThemeOverride{Pack: inner})
	if p.outerPick != nil {
		p.outerPick.Selected = p.outerPack
		p.innerPick.Selected = p.innerPack
	}
	p.outer.Title = "A pane in " + packLabel(outer)
	p.inner.Title = "A pane inside it, in " + packLabel(inner)
	p.refresh()
}

func packLabel(name string) string {
	if pack, ok := style.LoadTheme(name); ok {
		return pack.Display()
	}
	return name
}

// refresh restates the readout (tourRefresher): another page changing
// the application appearance changes what the two panes inherit.
func (p *cascadePage) refresh() {
	if p.facts == nil {
		return
	}
	win := style.LookAppearance(p.t.win.Look())
	out := style.LookAppearance(p.outer.Look())
	in := style.LookAppearance(p.inner.Look())
	scale := func(lk style.LookAndFeel) string {
		if c, ok := lk.(*style.Classic); ok {
			return strconv.FormatFloat(float64(c.Scale()), 'f', -1, 32) + "x"
		}
		return "?"
	}
	p.facts.SetText(tourFacts(
		[2]string{"the window", win.Name},
		[2]string{"  the pane", out.Name},
		[2]string{"    inside", in.Name},
		[2]string{"", ""},
		[2]string{"stated", "the pack, and nothing else"},
		[2]string{"inherited", "corners " + string(win.Corners) + ", icons " + string(win.Icons) +
			", size " + string(win.IconSize)},
		[2]string{"  the pane", "corners " + string(out.Corners) + ", icons " + string(out.Icons) +
			", size " + string(out.IconSize)},
		[2]string{"    inside", "corners " + string(in.Corners) + ", icons " + string(in.Icons) +
			", size " + string(in.IconSize)},
		[2]string{"", ""},
		[2]string{"scale", "the display's, and no level of the cascade can state one"},
		[2]string{"the window", scale(p.t.win.Look())},
		[2]string{"  the pane", scale(p.outer.Look())},
		[2]string{"    inside", scale(p.inner.Look())},
		[2]string{"", ""},
		[2]string{"the frame", "the window's look, always: the desktop and the compositor " +
			"need one answer for a window, and a pane is not a window"},
		[2]string{"popups", "a menu, a list and a tip take the look of the widget that " +
			"opened them, wherever the window system puts the surface"},
	))
}
