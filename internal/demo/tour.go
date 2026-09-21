package demo

import (
	"strconv"
	"strings"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// The tour: one page for each thing a gallery of controls cannot show.
//
// The gallery answers "which widgets are there". This answers "what can a
// window of this toolkit do" — carry its own tabs in its caption, hand a
// panel to a window of its own and take it back, start a drag that leaves
// the application, draw its own frame or let the desktop draw it, cut a
// hole you can see the desktop through, wear a skin — and the parts that
// exist only while something is running: the accessibility tree, the
// clipboard, the tray, the display's scale.
//
// The navigation is the first of those capabilities rather than a sidebar
// beside it. These pages are tabs in the window's title bar: drag one to
// reorder it, close it, open it again from "+", and pull one clear of the
// strip to get a second tour window running that page. Page one is
// therefore demonstrated by the thing the user is already holding to move
// around, which is the only honest way to show a tab strip that is a
// window's caption rather than a widget under one.

// The pages, in the order the tour opens them.
const (
	pageTabs = iota
	pageDock
	pageDrag
	pageFrames
	pageShapes
	pageSkins
	pageDesktop
	pageAccess
)

// tourPage is one capability: its tab, its heading, the single line that
// says what the page proves, the line in the status bar that says what to
// do, and the build that makes it.
type tourPage struct {
	name  string
	title string
	proof string
	try   string
	build func(*tourState) widget.Component
}

// tourPages is filled in by each page's own file (tour_tabs.go and the
// rest) so that a page's text and its build sit together; only the order
// lives here.
var tourPages = []tourPage{
	pageTabs:    {name: "Tabs"},
	pageDock:    {name: "Docking"},
	pageDrag:    {name: "Drag and drop"},
	pageFrames:  {name: "Frames"},
	pageShapes:  {name: "Shapes"},
	pageSkins:   {name: "Skins"},
	pageDesktop: {name: "Desktop"},
	pageAccess:  {name: "Access"},
}

// allTourPages is every page, in order.
func allTourPages() []int {
	out := make([]int, len(tourPages))
	for i := range tourPages {
		out[i] = i
	}
	return out
}

// tourState is one tour window: the strip that is its title bar and the
// page showing under it. Which pages are open is the strip's own business
// — each tab carries its page in BrowserTab.Data — so a tab torn out,
// dropped back or reordered needs no bookkeeping here to go wrong.
type tourState struct {
	a   *app.Application
	win *app.Window
	// strip is the window's title bar; its tabs are the open pages.
	strip *widgets.BrowserTabs
	// built keeps a page across tab switches, so a dock layout, a scroll
	// position and a list that has been reordered are still there when
	// the user comes back to them.
	built map[int]widget.Component
	// owned is each built page's own state. Pages share a window, so what
	// one of them changes — the frame, the look — another is showing;
	// this is how the tour tells them all to restate themselves, and how
	// a test reaches into a page to drive it.
	owned map[int]any
	// body holds the page showing now; head and proof say which it is.
	body   *widgets.FlexBox
	head   *widgets.Label
	proof  *widgets.Label
	status *widgets.StatusBar

	// look is the appearance the tour has applied, whole.
	//
	// It is kept rather than read back because style.LookAppearance
	// recovers from a look only what the look carries — the pack, the
	// palette, the corners, the icons — and not the preferences that
	// belong to the application rather than to the look: who draws the
	// frame, where the caption buttons go, reduced motion, the desktop's
	// file dialogs. Reading one back and applying it with a new pack in
	// it would quietly reset the other four, so the Skins page would undo
	// whatever the Frames page had just been used for.
	look style.Appearance

	// closers undo what a page left running when the window closes: a
	// window it opened, a tray item it registered, a look watch.
	closers []func()
}

// initAppearance seeds the tour's record: the saved preferences, with the
// look the application actually came up in written over the parts of it
// that a look carries (so `-theme` and UITK_THEME are reflected).
func (t *tourState) initAppearance() {
	saved := style.LoadAppearance().Normalize()
	from := style.LookAppearance(t.a.Look())
	saved.Name, saved.Theme = from.Name, from.Theme
	saved.Corners, saved.Icons, saved.IconSize = from.Corners, from.Icons, from.IconSize
	t.look = saved.Normalize()
}

// apply changes one thing about the appearance and hands the whole of it
// to the application, then lets every built page restate itself: more
// than one of them is showing some part of what just changed.
func (t *tourState) apply(change func(*style.Appearance)) {
	change(&t.look)
	t.look = t.look.Normalize()
	t.a.ApplyAppearance(t.look)
	t.refreshAll()
}

// TourApp is the tour with every page open. It puts the tab strip in the
// window's title bar and returns the window's content.
func TourApp(a *app.Application, win *app.Window) widget.Component {
	return newTour(a, win).content(allTourPages())
}

// TourAppOpen is TourApp with only the named pages open, the first
// selected: a torn-off window runs one page this way, and the screenshot
// and e2e passes open one page at a time.
func TourAppOpen(a *app.Application, win *app.Window, pages ...int) widget.Component {
	if len(pages) == 0 {
		pages = allTourPages()
	}
	return newTour(a, win).content(pages)
}

// TourPageIndex is the page a name opens; -1 for none.
func TourPageIndex(name string) int {
	for i, p := range tourPages {
		if strings.EqualFold(p.name, name) {
			return i
		}
	}
	// The tab titles are what a user reads; these are what a command line
	// is likely to carry.
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "tab", "tabs", "titlebar", "title-bar":
		return pageTabs
	case "dock", "docking", "panels":
		return pageDock
	case "drag", "dnd", "drop", "drag-and-drop":
		return pageDrag
	case "frame", "frames", "decorations", "chrome":
		return pageFrames
	case "shape", "shapes", "transparent", "glass":
		return pageShapes
	case "skin", "skins", "themes", "theme":
		return pageSkins
	case "desktop", "tray", "clipboard":
		return pageDesktop
	case "access", "a11y", "accessibility", "keyboard":
		return pageAccess
	}
	return -1
}

// TourPageNames are the page names a command line takes, in order.
func TourPageNames() []string {
	out := make([]string, len(tourPages))
	for i, p := range tourPages {
		out[i] = p.name
	}
	return out
}

func newTour(a *app.Application, win *app.Window) *tourState {
	return &tourState{a: a, win: win, built: map[int]widget.Component{}, owned: map[int]any{}}
}

// tourRefresher is a page that can restate what it shows. Everything on
// these pages is read off the running window, so a change one page makes
// is a change another is already displaying.
type tourRefresher interface{ refresh() }

// own records a page's state as it is built.
func (t *tourState) own(page int, state any) { t.owned[page] = state }

// page is the state of a page that has been built, nil for one that has
// not been opened yet.
func (t *tourState) page(p int) any { return t.owned[p] }

// refreshAll tells every built page to restate itself.
func (t *tourState) refreshAll() {
	for _, pg := range t.owned {
		if r, ok := pg.(tourRefresher); ok {
			r.refresh()
		}
	}
}

// content builds the window: the strip goes in the title bar, the page
// under it, and the whole is wrapped in the root that gives the strip its
// browser shortcuts (Ctrl+T, Ctrl+W, Ctrl+Tab) wherever the focus is.
func (t *tourState) content(open []int) widget.Component {
	if len(open) == 0 {
		open = allTourPages()
	}
	t.initAppearance()
	t.strip = widgets.NewBrowserTabs()
	t.strip.SetAccessibleName("Tour pages")
	t.wireStrip()
	for _, p := range open {
		if p >= 0 && p < len(tourPages) {
			t.strip.AddTab(t.tabFor(p))
		}
	}
	t.syncCloseButtons()
	t.win.SetTitleBar(t.strip)
	t.installCloseHook()

	first := t.pageAt(0)
	t.head = widgets.NewTitle(tourPages[first].title)
	t.proof = widgets.NewLabel(tourPages[first].proof)
	t.proof.Wrap = true
	t.proof.MinLines = 2
	t.body = widgets.NewColumn()
	t.status = widgets.NewStatusBar("", "")

	header := widgets.NewColumn(t.head, t.proof).WithGap(2).WithPadding(12, 9, 12, 8)
	col := widgets.NewColumn(header, widgets.NewSeparator(), t.body, t.status).WithGap(0)
	col.AddFlex(t.body, 1)
	t.strip.Select(0)
	t.show(0)
	return newTourRoot(t, col)
}

// tourRoot is the tour's content root. It hands the strip the browser
// shortcuts — Ctrl+Tab, Ctrl+W, Ctrl+T — which no focused widget takes
// and which the strip itself rarely has the focus to see; and it is how
// anything holding the window finds the tour inside it.
type tourRoot struct {
	widget.Base
	tour *tourState
}

func newTourRoot(t *tourState, child widget.Component) *tourRoot {
	r := &tourRoot{tour: t}
	r.Init(r)
	r.Add(child)
	return r
}

func (r *tourRoot) Measure(c layout.Constraints) paintengine2d.Point {
	return r.Children()[0].Measure(c)
}

func (r *tourRoot) Arrange(b paintengine2d.Rect) {
	r.SetBounds(b)
	r.Children()[0].Arrange(paintengine2d.XYWH(0, 0, b.Dx(), b.Dy()))
}

func (r *tourRoot) KeyPress(e widget.KeyEvent) bool {
	return r.tour != nil && r.tour.strip != nil && r.tour.strip.Shortcut(e)
}

// tourIn is the tour running in win, nil for a window that is not one.
func tourIn(win *app.Window) *tourState {
	if win == nil {
		return nil
	}
	if r, ok := win.Content().(*tourRoot); ok {
		return r.tour
	}
	return nil
}

func (t *tourState) tabFor(p int) widgets.BrowserTab {
	return widgets.BrowserTab{Title: tourPages[p].name, Tip: tourPages[p].title, Data: p}
}

// pageAt is the page tab i holds (the first page for a tab that has none,
// which cannot happen but must not panic if it does).
func (t *tourState) pageAt(i int) int {
	if p, ok := t.strip.Tab(i).Data.(int); ok && p >= 0 && p < len(tourPages) {
		return p
	}
	return pageTabs
}

func (t *tourState) isOpen(p int) bool { return t.tabOf(p) >= 0 }

func (t *tourState) tabOf(p int) int {
	for i := 0; i < t.strip.Len(); i++ {
		if q, ok := t.strip.Tab(i).Data.(int); ok && q == p {
			return i
		}
	}
	return -1
}

// wireStrip is the whole of page one: what the strip does when a tab is
// chosen, closed, asked for, dragged along the strip, pulled out of the
// window, or dropped in from another one.
func (t *tourState) wireStrip() {
	s := t.strip
	s.OnSelect = func(i int) { t.show(i) }

	// Closing: a tour window keeps at least one page, so its last tab has
	// no close button and Ctrl+W on it does nothing.
	s.OnClose = func(i int) {
		if s.Len() < 2 {
			t.note("A window keeps its last page.")
			return
		}
		name := tourPages[t.pageAt(i)].name
		s.RemoveTab(i)
		t.syncCloseButtons()
		t.note("Closed " + name + " — reopen it from +.")
	}

	// "+" offers the pages this window does not hold.
	s.OnNew = func() { t.openMenu() }

	s.OnReorder = func(from, to int) {
		t.note("Moved " + tourPages[t.pageAt(to)].name + " to position " + strconv.Itoa(to+1) + ".")
	}

	// Out of the strip and into a window of its own: the strip asks for
	// the window, the tour opens a second tour running that one page. The
	// strip takes the tab out and puts it back itself, so there is nothing
	// to undo here if the drag is cancelled.
	s.OnTearOff = func(_ int, tab widgets.BrowserTab) widget.TearOffWindow {
		p, ok := tab.Data.(int)
		if !ok {
			return nil
		}
		win, err := TourWindow(t.a, p)
		if err != nil {
			t.note("Could not open a window: " + err.Error())
			return nil
		}
		t.syncCloseButtons()
		return win
	}

	// And back in: a tab dropped on this strip joins it at the caret.
	s.OnMergeTab = func(at int, tab widgets.BrowserTab, _ widget.DropEvent) bool {
		p, ok := tab.Data.(int)
		if !ok {
			// From another tour process the private type carries only the
			// title, which is enough to name the page again.
			if p = TourPageIndex(tab.Title); p < 0 {
				return false
			}
		}
		if t.isOpen(p) {
			t.note(tourPages[p].name + " is already open in this window.")
			return false
		}
		if at < 0 || at > s.Len() {
			at = s.Len()
		}
		s.InsertTab(at, t.tabFor(p))
		t.syncCloseButtons()
		s.Select(at)
		t.note("Took " + tourPages[p].name + " from another window.")
		return true
	}
	s.DropActions = platform.DragMove

	// Besides the tab itself a dragged tab carries a line of text, so
	// dropping one in an editor or a terminal says what it was.
	s.OnTabDrag = func(_ int, tab widgets.BrowserTab) *widget.Drag {
		p, ok := tab.Data.(int)
		if !ok {
			return nil
		}
		return widget.DragText("uitoolkit tour — " + tourPages[p].title)
	}

	s.OnContextMenu = func(i int, at paintengine2d.Point) bool {
		if i < 0 {
			return false // the caption space keeps the desktop's window menu
		}
		items := []*widgets.MenuItem{
			widgets.Item("Move to a new window", func() { t.tearOff(i) }),
			widgets.Sep(),
			widgets.ItemAccel("Close", "Ctrl+W", func() { s.CloseTab(i) }),
			widgets.ItemAccel("New page…", "Ctrl+T", func() { t.openMenu() }),
		}
		widgets.ShowContextMenu(s, at, items...)
		return true
	}
}

// syncCloseButtons hides the close button of a strip's only tab: a tour
// window always holds a page.
func (t *tourState) syncCloseButtons() {
	only := t.strip.Len() < 2
	for i := 0; i < t.strip.Len(); i++ {
		tab := t.strip.Tab(i)
		if tab.NoClose != only {
			tab.NoClose = only
			t.strip.SetTab(i, tab)
		}
	}
}

// tearOff moves tab i into a window of its own with no drag at all, which
// is what the tab's menu item does and what a keyboard-only user gets. The
// drag does the same thing through OnTearOff.
func (t *tourState) tearOff(i int) {
	if i < 0 || i >= t.strip.Len() {
		return
	}
	if t.strip.Len() < 2 {
		t.note("This window holds one page; there is nothing to move out of it.")
		return
	}
	p := t.pageAt(i)
	win, err := TourWindow(t.a, p)
	if err != nil {
		t.note("Could not open a window: " + err.Error())
		return
	}
	t.strip.RemoveTab(i)
	t.syncCloseButtons()
	win.Show()
	t.note("Moved " + tourPages[p].name + " into a window of its own.")
}

// openMenu offers the pages this window does not hold.
func (t *tourState) openMenu() {
	var items []*widgets.MenuItem
	for p := range tourPages {
		if t.isOpen(p) {
			continue
		}
		page := p
		items = append(items, widgets.Item(tourPages[p].name, func() { t.openPage(page) }))
	}
	if len(items) == 0 {
		t.note("Every page is already open in this window.")
		return
	}
	at := widget.DeviceOrigin(t.strip)
	at.Y += t.strip.Bounds().Dy()
	widgets.ShowContextMenu(t.strip, at, items...)
}

// openPage adds a page at the end of the strip and shows it.
func (t *tourState) openPage(p int) {
	if p < 0 || p >= len(tourPages) || t.isOpen(p) {
		return
	}
	i := t.strip.AddTab(t.tabFor(p))
	t.syncCloseButtons()
	t.strip.Select(i)
}

// show puts tab i's page in the body, building it the first time.
func (t *tourState) show(i int) {
	if t.body == nil || i < 0 || i >= t.strip.Len() {
		return
	}
	p := t.pageAt(i)
	page := t.built[p]
	if page == nil {
		page = tourPages[p].build(t)
		t.built[p] = page
	}
	t.body.ClearChildren()
	t.body.AddFlex(page, 1)
	t.head.SetText(tourPages[p].title)
	t.proof.SetText(tourPages[p].proof)
	t.status.Set(0, tourPages[p].try)
	t.status.Set(1, "")
	t.win.RequestLayout()
}

// current is the page showing now, -1 for none.
func (t *tourState) current() int {
	i := t.strip.Selected()
	if i < 0 || i >= t.strip.Len() {
		return -1
	}
	return t.pageAt(i)
}

// note says what just happened, in the right-hand half of the status bar.
// It is how every page reports the thing the user cannot see: the action a
// drop performed, the side a panel docked to, which frame is in effect.
func (t *tourState) note(s string) {
	if t.status != nil {
		t.status.Set(1, s)
	}
}

// onClose registers something to undo when the tour window closes: a
// window a page opened, a tray item it registered.
func (t *tourState) onClose(fn func()) {
	if fn != nil {
		t.closers = append(t.closers, fn)
	}
}

// installCloseHook puts the tour's own close handler back. A page that
// calls something which takes the hook for itself — app.DockHost does,
// to close a host's floating windows — calls this afterwards, and
// registers what it displaced with onClose.
func (t *tourState) installCloseHook() {
	t.win.SetOnCloseRequest(func() bool {
		t.closeAll()
		return true
	})
}

func (t *tourState) closeAll() {
	for i := len(t.closers) - 1; i >= 0; i-- {
		t.closers[i]()
	}
	t.closers = nil
}

// TourWindow opens a tour window of its own holding the given pages: what
// a tab torn out of the strip lands in, and what `-page` opens.
func TourWindow(a *app.Application, pages ...int) (*app.Window, error) {
	title := "uitoolkit tour"
	if len(pages) == 1 && pages[0] >= 0 && pages[0] < len(tourPages) {
		title = tourPages[pages[0]].title + " — tour"
	}
	win, err := a.NewWindow(platform.WindowOptions{
		Title: title, Width: 1060, Height: 740, MinWidth: 620, MinHeight: 430, Resizable: true,
	})
	if err != nil {
		return nil, err
	}
	win.SetContent(TourAppOpen(a, win, pages...))
	return win, nil
}

// ---- the shape of a page -------------------------------------------------------

// tourStage is the shape every page takes: the thing to try on the left,
// and on the right what the toolkit says about it — the facts a
// screenshot cannot carry, in the mono face so that they line up.
func tourStage(stage, readout widget.Component) widget.Component {
	sp := widgets.NewSplitter(true, widgets.NewPad(10, stage), widgets.NewPad(10, readout))
	sp.Ratio = 0.6
	return sp
}

// tourReadout is the right-hand panel of a page: a heading and a
// read-only mono view the page keeps up to date.
func tourReadout(title string) (*widgets.Panel, *widgets.TextArea) {
	view := widgets.NewMonoTextView("", "")
	view.SetAccessibleName(title)
	p := widgets.NewPanel(title, view)
	p.Content().AddFlex(view, 1)
	return p, view
}

// afterLaidOut wraps a child and runs fn once, on the pump after the
// child is first given a box. A page that reads something measured off
// the window it is in — the accessibility tree, whose nodes have no
// bounds at all until layout has run — needs that moment, and there is
// no event for it.
type afterLaidOut struct {
	widget.Base
	child widget.Component
	post  func(func())
	fn    func()
	done  bool
}

func newAfterLaidOut(child widget.Component, post func(func()), fn func()) *afterLaidOut {
	a := &afterLaidOut{child: child, post: post, fn: fn}
	a.Init(a)
	a.Add(child)
	return a
}

func (a *afterLaidOut) Measure(c layout.Constraints) paintengine2d.Point {
	return a.child.Measure(c)
}

func (a *afterLaidOut) Arrange(b paintengine2d.Rect) {
	a.SetBounds(b)
	a.child.Arrange(paintengine2d.XYWH(0, 0, b.Dx(), b.Dy()))
	if !a.done && b.Dx() > 0 && b.Dy() > 0 {
		a.done = true
		// Not here: the tree is being arranged, and the callback reads it
		// and asks for another layout.
		a.post(a.fn)
	}
}

// tourScroll is a page's scrollable body. A scroll pane takes the
// keyboard focus so that the arrow keys can scroll it, which means a
// screen reader stops on it — so it has to have something to say.
func tourScroll(name string, child widget.Component) *widgets.ScrollView {
	s := widgets.NewScrollView(child)
	s.SetAccessibleName(name)
	return s
}

// tourNote is the small print under a control: why the thing above it is
// worth trying, or what the toolkit could not do here.
func tourNote(text string) *widgets.Label {
	l := widgets.NewLabel(text)
	l.Wrap = true
	return l
}

// tourRow is a labelled row of controls, so that every page lines its
// choices up the same way and the label names the control for a screen
// reader.
func tourRow(label string, children ...widget.Component) *widgets.FlexBox {
	name := widgets.NewLabel(label)
	if len(children) > 0 {
		name.For(children[0])
	}
	row := widgets.NewRow(append([]widget.Component{name}, children...)...).WithGap(8)
	return row.WithAlign(layout.AlignCenter)
}

// tourFactCols is how wide a readout is, in characters of the mono face.
// The readout panel is a fixed share of the page and the mono face is a
// fixed width, so this is a constant rather than a measurement: a value
// wider than this is folded under its own column instead of wrapping
// back to the margin, where it would read as another fact.
const tourFactCols = 58

// tourFacts renders a readout: one "name  value" line per fact, the names
// padded so that the values line up in the mono face. A pair with neither
// is a blank line, which is how a readout gets its paragraphs.
func tourFacts(pairs ...[2]string) string {
	w := 0
	for _, p := range pairs {
		if len(p[0]) > w {
			w = len(p[0])
		}
	}
	gap := strings.Repeat(" ", w+2)
	var b strings.Builder
	for _, p := range pairs {
		if p[0] == "" && p[1] == "" {
			b.WriteByte('\n')
			continue
		}
		b.WriteString(p[0])
		b.WriteString(strings.Repeat(" ", w-len(p[0])+2))
		b.WriteString(foldValue(p[1], tourFactCols-w-2, gap))
		b.WriteByte('\n')
	}
	return b.String()
}

// foldValue breaks a value over as many lines as it needs, each after the
// first indented to the value's own column. Lines the value already has
// are kept.
func foldValue(v string, width int, indent string) string {
	if width < 12 {
		width = 12
	}
	var out []string
	for _, para := range strings.Split(v, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := ""
		for _, word := range words {
			switch {
			case line == "":
				line = word
			case len(line)+1+len(word) <= width:
				line += " " + word
			default:
				out = append(out, line)
				line = word
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"+indent)
}

// yesNo is how a readout says a boolean, so that "no" is as loud as "yes"
// — half the facts on these pages are a capability the desktop withheld.
func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
