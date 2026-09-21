package demo

import (
	"strconv"
	"strings"

	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// Page six: a look is a value, and swapping it under a running window is
// a field assignment and a relayout, not a rebuild. Settings already has
// the browser for all of them; what this page is for is the swap itself
// — the same widget tree, the same tab order, the same accessibility
// tree, in a pack from 1991 and then in one made of photographs.
//
// A skin is a pack like any other, which is the point worth making: it
// is listed, loaded, previewed and applied through exactly the same
// calls, and the only thing that marks it out is that its shapes are
// pictures rather than code — and that some of them cut the window.

func init() {
	p := &tourPages[pageSkins]
	p.title = "Themes and skins, live"
	p.proof = "Every pack the toolkit has, swapped under this running window without rebuilding it — " +
		"including a skin, whose chrome is pictures and whose silhouette becomes the window's."
	p.try = "Pick a pack on the left; \"Use this look\" gives it to the whole app."
	p.build = buildSkinsPage
}

// tourExtremes are the packs the tour is checked against, and the ones
// worth trying first: the two that predate anti-aliasing, the two that
// made gradients normal, the two that are mostly blur, and a skin.
var tourExtremes = []string{"win95", "system7", "luna", "aqua", "bigsur", "tahoe", "sourcegit", "deck"}

type skinsPage struct {
	t *tourState
	// rows is what the list shows now, after the filter and the query.
	rows  []style.ThemePack
	list  *widgets.ListView
	scope *widgets.ThemeScope
	box   *widgets.Panel
	facts *widgets.TextArea
	query string
	// only narrows the list: 0 every pack, 1 skins, 2 the extremes.
	only int
	// started is the look the tour was in, so it can be put back.
	started style.Appearance
}

func buildSkinsPage(t *tourState) widget.Component {
	p := &skinsPage{t: t, started: t.look}
	t.own(pageSkins, p)

	search := widgets.NewTextField("", "Search packs", func(s string) {
		p.query = s
		p.reload()
	})
	search.SetAccessibleName("Search packs")

	filter := widgets.NewSegmented([]string{"All", "Skins", "Extremes"}, 0, func(i int) {
		p.only = i
		p.reload()
	})
	filter.SetAccessibleName("Which packs")

	p.list = widgets.NewListView(0, func(i int) string {
		if i < 0 || i >= len(p.rows) {
			return ""
		}
		return skinRowText(p.rows[i])
	}, func(i int) { p.choose(i) })
	p.list.SetAccessibleName("Packs")
	p.list.RowHeight = 30

	use := widgets.NewButton("Use this look", func() {
		pack, ok := p.selected()
		if !ok {
			return
		}
		// Only the pack changes: the tour hands the application the whole
		// of its appearance, so the frame, the caption buttons and the
		// motion preference the other pages set stay as they were.
		p.t.apply(func(ap *style.Appearance) {
			ap.Name = pack.Name
			ap.Theme = pack.Palette
			ap.FollowDesktop = false
		})
		p.note("Every window of the tour is in " + pack.Display() + " now — nothing was rebuilt.")
	})
	use.Primary = true
	back := widgets.NewButton("Back to where we started", func() {
		p.t.apply(func(ap *style.Appearance) { *ap = p.started })
		p.note("Back in " + p.started.Name + ".")
	})

	p.box = widgets.NewPanel("Preview", widgets.NewLabel(""))
	p.scope = nil

	left := widgets.NewColumn(search, filter, p.list).WithGap(8)
	left.AddFlex(p.list, 1)

	about := widgets.NewPanel("What a swap costs",
		tourNote("Picking a pack here previews it in a ThemeScope — one subtree running a look of its "+
			"own while the rest of the window keeps the window's, which is how a preview can be honest "+
			"without a second process."),
		tourNote("\"Use this look\" hands the look to the application, which re-applies each window's "+
			"own display scale to it and relayouts. The widget tree is untouched: the same components, "+
			"the same tab order, the same accessibility tree, different metrics and different paint."),
		tourNote("A skin is a pack whose parts are pictures — nine-patch sheets and sprites — with a "+
			"base pack behind it for anything the pictures do not cover. Some skins also state a "+
			"silhouette, and then the window takes it: pick \"deck\" and this window stops being a "+
			"rectangle."),
	)
	about.Content().Spec.Gap = 7

	right := widgets.NewColumn(p.box, widgets.NewRow(use, back).WithGap(6), about).WithGap(10)
	right.AddFlex(p.box, 1)

	split := widgets.NewSplitter(true, left, right)
	split.Ratio = 0.34

	panel, facts := tourReadout("The pack, and what it does to a window")
	p.facts = facts

	p.reload()
	t.onClose(func() { p.t.apply(func(ap *style.Appearance) { *ap = p.started }) })
	return tourStage(split, panel)
}

// reload refills the list for the filter and the query, keeping the
// selection on the pack it was on where that pack is still shown.
func (p *skinsPage) reload() {
	was := ""
	if pack, ok := p.selected(); ok {
		was = pack.Name
	}
	p.rows = p.rows[:0]
	for _, pack := range style.ListThemes() {
		if !p.shows(pack) {
			continue
		}
		p.rows = append(p.rows, pack)
	}
	p.list.Count = len(p.rows)
	p.list.Selected = 0
	for i, pack := range p.rows {
		if pack.Name == was {
			p.list.Selected = i
			break
		}
	}
	p.list.Invalidate()
	p.list.RequestLayout()
	p.choose(p.list.Selected)
}

func (p *skinsPage) shows(pack style.ThemePack) bool {
	switch p.only {
	case 1:
		if !style.IsSkin(pack.Name) {
			return false
		}
	case 2:
		found := false
		for _, n := range tourExtremes {
			if n == pack.Name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	q := strings.ToLower(strings.TrimSpace(p.query))
	if q == "" {
		return true
	}
	hay := strings.ToLower(strings.Join([]string{
		pack.Name, pack.Label, pack.Era, pack.Lineage, pack.Summary, strconv.Itoa(pack.Year),
	}, " "))
	return strings.Contains(hay, q)
}

func (p *skinsPage) selected() (style.ThemePack, bool) {
	i := p.list.Selected
	if i < 0 || i >= len(p.rows) {
		return style.ThemePack{}, false
	}
	return p.rows[i], true
}

// choose previews row i without applying it.
func (p *skinsPage) choose(i int) {
	p.list.Selected = i
	pack, ok := p.selected()
	if !ok {
		p.box.Title = "Preview"
		p.refresh()
		return
	}
	look := pack.Look()
	if p.scope == nil {
		p.scope = widgets.NewThemeScope(look, skinPreviewTree())
		p.box.Content().ClearChildren()
		p.box.Content().AddFlex(p.scope, 1)
	} else {
		p.scope.SetTheme(look)
	}
	p.box.Title = "Preview — " + pack.Display()
	p.box.RequestLayout()
	p.refresh()
}

// skinPreviewTree is the handful of controls the preview shows: enough
// that a pack's bevel, its hot track and its chrome are all visible, and
// few enough to read at a glance.
func skinPreviewTree() widget.Component {
	ok := widgets.NewButton("Default", nil)
	ok.Primary = true
	row := widgets.NewRow(ok, widgets.NewButton("Cancel", nil)).WithGap(8)
	list := widgets.NewListView(3, func(i int) string {
		return []string{"Inbox", "Drafts", "Sent"}[i]
	}, nil)
	list.Selected = 0
	list.RowHeight = 24
	list.SetAccessibleName("Preview list")
	field := widgets.NewTextField("A field", "", nil)
	field.SetAccessibleName("Preview field")
	slider := widgets.NewSlider(0, 100, 60, nil)
	slider.SetAccessibleName("Preview slider")
	bar := widgets.NewProgressBar(0.45)
	bar.SetAccessibleName("Preview progress")
	col := widgets.NewColumn(
		widgets.NewTitleBar("A window in this pack", "chrome, controls and a list"),
		widgets.NewPad(8, widgets.NewColumn(
			row,
			widgets.NewRow(widgets.NewCheckbox("Checked", true, nil), widgets.NewSwitch("On", true, nil)).WithGap(10),
			field,
			slider,
			bar,
			list,
		).WithGap(8)),
	).WithGap(0)
	return col
}

func (p *skinsPage) note(s string) {
	p.t.note(s)
	// A pack can cut the window's silhouette, which the Shapes page
	// reports, and change its frame, which the Tabs page does.
	p.t.refreshAll()
}

func (p *skinsPage) refresh() {
	if p.facts == nil {
		return
	}
	all, skins := 0, 0
	for _, pack := range style.ListThemes() {
		all++
		if style.IsSkin(pack.Name) {
			skins++
		}
	}
	now := p.t.look
	pack, ok := p.selected()
	if !ok {
		p.facts.SetText(tourFacts(
			[2]string{"packs", strconv.Itoa(all) + " (" + strconv.Itoa(skins) + " of them skins)"},
			[2]string{"showing", "nothing matches that search"},
			[2]string{"in use", now.Name},
		))
		return
	}
	look := pack.Look()
	p.facts.SetText(tourFacts(
		[2]string{"highlighted", pack.Name},
		[2]string{"label", pack.Display()},
		[2]string{"year", strconv.Itoa(pack.Year)},
		[2]string{"era", pack.Era},
		[2]string{"lineage", pack.Lineage},
		[2]string{"family", string(pack.Palette)},
		[2]string{"a skin", yesNo(style.IsSkin(pack.Name))},
		[2]string{"cuts windows", yesNo(style.WindowShaped(look))},
		[2]string{"shapes controls", yesNo(style.ControlShapes(look))},
		[2]string{"wants glass", yesNo(style.WantsGlass(look))},
		[2]string{"", ""},
		[2]string{"in use now", now.Name},
		[2]string{"this window", windowShapeState(p)},
		[2]string{"", ""},
		[2]string{"showing", strconv.Itoa(len(p.rows)) + " of " + strconv.Itoa(all) + " packs"},
		[2]string{"skins", strconv.Itoa(skins)},
		[2]string{"", ""},
		[2]string{"about it", pack.Summary},
	))
}

func windowShapeState(p *skinsPage) string {
	if p.t.win.ShapeActive() {
		return "cut to the look's own outline"
	}
	if style.WindowShaped(p.t.win.Look()) {
		return "the look cuts windows, but this one is maximized or tiled"
	}
	return "a rectangle"
}

func skinRowText(pack style.ThemePack) string {
	mark := ""
	if style.IsSkin(pack.Name) {
		mark = "  · skin"
	}
	return pack.Display() + "   " + strconv.Itoa(pack.Year) + mark
}
