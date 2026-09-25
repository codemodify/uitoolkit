package tourapp

import (
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/showcase"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The controls themselves.
//
// These used to be a separate program, examples/gallery, which put the
// whole showcase package in one window. It answered "which widgets are
// there" and the tour answered "what can a window of this toolkit do",
// and there is no reason for a person who wants to see what the toolkit
// looks like to have to know which of the two to run. So the showcase
// came in here, and it is the first thing the tour shows.
//
// It comes in as three pages rather than one. The gallery was already
// more than a screenful — a scrolling column of small controls beside a
// tab view of eight big ones — and pouring all of that into one tour
// page would have been the one page you cannot take in, next to seven
// that each make a single point. The division is by what the widgets
// are for:
//
//   - Controls: the things you set. Menu bar, title bar, tool bar,
//     buttons, dialogs, every field, and the form that lines them up.
//   - Views: the things that hold rows. Scroll, list and cards, tree,
//     table — what a file manager or a mail client is made of.
//   - Documents: the three composites that are nearly applications in
//     themselves. Rich text with its format bar, windows inside a
//     window, a wizard that validates page by page.
//
// Each page builds the showcase for itself with [showcase.Build] and
// lays out the parts it wants; the parts it does not want are simply not
// put on screen. That is one build per page rather than one shared
// between them, because a widget belongs to one parent and the three
// pages are three trees.

func init() {
	c := &tourPages[pageControls]
	c.title = "Every control the toolkit has"
	c.proof = "The whole widget set, drawn by whichever pack is in force: the menu bar and tool bar, " +
		"every button and field, and the dialogs they open. It is the showcase package — the same " +
		"widgets Settings puts under its theme preview."
	c.try = "Press anything; the status bar under the stage says what it did."
	c.build = func(t *tourState) widget.Component { return buildGalleryPage(t, pageControls, layControls) }

	v := &tourPages[pageViews]
	v.title = "Lists, trees and tables"
	v.proof = "The views that hold rows, each with rows in it: a scrolling column, a list beside the " +
		"card list a mail client uses, a tree, and a table whose headers sort it."
	v.try = "Sort a column, right-click a row, and scroll with the wheel."
	v.build = func(t *tourState) widget.Component { return buildGalleryPage(t, pageViews, layViews) }

	d := &tourPages[pageDocs]
	d.title = "Documents, windows in a window, a wizard"
	d.proof = "The three composites big enough to be applications on their own: a rich-text editor " +
		"with its format bar, an MDI area you can cascade and tile, and a wizard that validates a " +
		"page before it lets you leave it."
	d.try = "Type in the editor, press Tile, and walk the wizard to Finish."
	d.build = func(t *tourState) widget.Component { return buildGalleryPage(t, pageDocs, layDocs) }
}

// galleryPage is one of the three showcase pages. It holds the build in
// a box of its own so that a change of palette — made here, on the Skins
// page or by Settings while the tour runs — can replace it: the Dark
// switch and the View menu's radio state which side the showcase was
// built for, and there is no way to restate that but to build it again.
type galleryPage struct {
	t    *tourState
	page int
	// lay turns one build of the showcase into this page's stage.
	lay func(*galleryPage, showcase.Parts) widget.Component
	// body holds the stage; rebuild swaps what is in it.
	body  *widgets.FlexBox
	facts *widgets.TextArea
	// light is the palette the stage in the body was built for.
	light bool
	// kinds is the inventory of the stage now in the body.
	kinds string
	count int
	types int
}

func buildGalleryPage(t *tourState, page int, lay func(*galleryPage, showcase.Parts) widget.Component) widget.Component {
	p := &galleryPage{t: t, page: page, lay: lay}
	t.own(page, p)
	p.body = widgets.NewColumn()
	panel, facts := tourReadout("What is on this page")
	p.facts = facts
	p.rebuild()
	return tourStage(p.body, panel)
}

// host is what the showcase asks of the application around it. The tour
// owns the look, so SwitchTheme goes through the tour's own appearance
// rather than re-theming this window behind the other pages' backs.
func (p *galleryPage) host() showcase.Host {
	t := p.t
	return showcase.Host{
		Light: p.light,
		Root:  func() widget.Component { return t.win.Content() },
		NewWindow: func() error {
			w2, err := t.a.NewWindow(platform.WindowOptions{
				Title: "Second window", Width: 420, Height: 280,
			})
			if err != nil {
				return err
			}
			w2.SetContent(widgets.NewPanel("Dialog window",
				widgets.NewLabel("This is a second OS window, opened by the showcase."),
				widgets.NewButton("Close", func() { w2.Close() }),
			))
			// It is the tour's window that opened it, so the tour closes it.
			t.onClose(w2.Close)
			t.note("Opened a second window.")
			return nil
		},
		SwitchTheme: func(light bool) {
			// One appearance, the application's: the Skins page's packs and
			// this switch are the same setting seen from two pages.
			t.apply(func(ap *style.Appearance) {
				ap.Theme = style.ThemeDark
				if light {
					ap.Theme = style.ThemeLight
				}
				ap.FollowDesktop = false
			})
			t.note("Every window of the tour is in the " + string(t.a.Appearance().Theme) + " palette now.")
		},
		Quit: t.a.Quit,
	}
}

// lightNow is the palette the application is in.
func (p *galleryPage) lightNow() bool {
	return style.LookAppearance(p.t.a.Look()).Theme == style.ThemeLight
}

// rebuild builds the showcase again and puts it in the body.
func (p *galleryPage) rebuild() {
	p.light = p.lightNow()
	stage := p.lay(p, showcase.Build(p.host()))
	p.body.ClearChildren()
	p.body.AddFlex(stage, 1)
	p.types, p.count, p.kinds = galleryInventory(stage)
	p.restate()
	p.t.win.RequestLayout()
}

// recount reads the inventory off whatever is in the body now, which a
// tab view changes under it, and says so.
func (p *galleryPage) recount() {
	if p.body == nil || len(p.body.Children()) == 0 {
		return
	}
	p.types, p.count, p.kinds = galleryInventory(p.body.Children()[0])
	p.restate()
}

// refresh is what the tour calls when the appearance changed anywhere.
// Only a change of palette needs the page built again; a change of pack
// is metrics and paint, which the widgets already showing take
// themselves.
func (p *galleryPage) refresh() {
	if p.light != p.lightNow() {
		p.rebuild()
		return
	}
	p.restate()
}

// restate writes the readout: how many widgets of how many kinds are on
// this page, and which. The pictures show what the pack does to them;
// this is the part a picture cannot carry — that these really are the
// toolkit's own widgets, and which ones.
func (p *galleryPage) restate() {
	if p.facts == nil {
		return
	}
	ap := style.LookAppearance(p.t.a.Look())
	pack := ap.Name
	if t, ok := style.LoadTheme(ap.Name); ok {
		pack = t.Display()
	}
	p.facts.SetText(tourFacts(
		[2]string{"page", tourPages[p.page].name},
		[2]string{"widget kinds", strconv.Itoa(p.types)},
		[2]string{"widgets", strconv.Itoa(p.count)},
		[2]string{"", ""},
		[2]string{"pack", pack},
		[2]string{"palette", string(ap.Theme)},
		[2]string{"built for", paletteWord(p.light)},
		[2]string{"scale", strconv.FormatFloat(float64(p.t.win.Scale()), 'f', -1, 32) + "×"},
		[2]string{"animations", yesNo(!ap.ReduceMotion)},
		[2]string{"", ""},
	) + p.kinds)
}

func paletteWord(light bool) string {
	if light {
		return "light"
	}
	return "dark"
}

// ---- the three layouts ---------------------------------------------------------

// layControls is the things you set: the window furniture across the top,
// then the buttons, the fields and the form in a column that scrolls.
func layControls(p *galleryPage, s showcase.Parts) widget.Component {
	col := widgets.NewColumn(s.Chrome, s.Tools, widgets.NewWrap(s.Buttons, s.Fields)).WithGap(10)
	if form, ok := s.Named("Form"); ok {
		col.Add(form.Content)
	}
	col.Add(tourNote("Every one of these is a widget of the toolkit, drawn by the pack in force and " +
		"by nothing else: there is no native control under any of them. The menu bar above is a real " +
		"one — its menus drop, its accelerators work — and the bar along the bottom is where they all " +
		"report what they just did."))
	scroll := tourScroll("Controls", widgets.NewPad(2, col))
	body := widgets.NewColumn(s.Menu, scroll, s.Status).WithGap(0)
	body.AddFlex(scroll, 1)
	return body
}

// layViews is the collections, tabbed: they want height rather than a
// column each, and one of them at a time is how an application shows
// them anyway.
func layViews(p *galleryPage, s showcase.Parts) widget.Component {
	// The list first, not the scroll pane: the page opens on the tab
	// worth opening on, and forty labels in a scroller is what the
	// ScrollView tab is for rather than a first impression.
	return galleryTabs(p, s, s.Some("List", "Tree", "Table", "Scroll"))
}

// layDocs is the three composites, tabbed for the same reason: each of
// them wants the whole page.
func layDocs(p *galleryPage, s showcase.Parts) widget.Component {
	return galleryTabs(p, s, s.Some("Rich text", "MDI", "Wizard"))
}

// galleryTabs is a tab view of these views over the showcase's own
// status bar, which is where they say what was selected or sorted.
func galleryTabs(p *galleryPage, s showcase.Parts, views []showcase.View) widget.Component {
	pages := make([]widgets.Tab, len(views))
	for i, v := range views {
		pages[i] = widgets.Tab{Title: v.Title, Content: v.Content}
	}
	tabs := widgets.NewTabView(pages...)
	tabs.SetAccessibleName(tourPages[p.page].name)
	tabs.Bar().SetAccessibleName(tourPages[p.page].name)
	tabs.OnChange = func(i int) {
		if i >= 0 && i < len(views) {
			s.Status.Set(0, "Tab: "+views[i].Title)
		}
		// A tab view holds only the page showing, so the inventory is of
		// that page: it is counted again whenever the tab changes.
		p.recount()
	}
	col := widgets.NewColumn(widgets.NewPad(2, tabs), s.Status).WithGap(0)
	col.AddFlex(col.Children()[0], 1)
	return col
}

// ---- the inventory -------------------------------------------------------------

// galleryInventory counts the widgets on a page: how many kinds, how
// many in all, and the roll of them, folded to the readout's width.
//
// Only the toolkit's own widgets/ types are counted. The layout boxes,
// the pads and the tour's own wrappers are scaffolding a reader of the
// samples would write themselves, and counting them would flatter the
// number the page is there to state.
func galleryInventory(root widget.Component) (kinds, total int, list string) {
	seen := map[string]int{}
	widget.Walk(root, func(c widget.Component) {
		if n := widgetKind(c); n != "" {
			seen[n]++
			total++
		}
	})
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)
	items := make([]string, len(names))
	for i, n := range names {
		items[i] = n + "×" + strconv.Itoa(seen[n])
	}
	// The items are single words, so the readout's folder breaks the roll
	// between them and never inside one.
	return len(names), total, foldValue(strings.Join(items, " "), tourFactCols, "")
}

// widgetKind is c's type name where c is one of the toolkit's widgets,
// and empty for anything else.
func widgetKind(c widget.Component) string {
	t := reflect.TypeOf(c)
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.PkgPath() != widgetsPkg {
		return ""
	}
	return t.Name()
}

// widgetsPkg is the import path of the toolkit's widget package, read off
// a widget rather than written down, so a move of the package cannot
// leave the inventory silently empty.
var widgetsPkg = reflect.TypeOf(widgets.Label{}).PkgPath()
