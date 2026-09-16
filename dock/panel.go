package dock

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// Panel is one dockable panel (Qt's QDockWidget, a VS Code view): the
// content, the title it is called by, and the name a saved layout finds it
// under. The chrome — the title bar with its drag handle, collapse, float
// and close buttons, and the tab when panels share a box — belongs to the
// [Stack] the panel sits in, so a stack of three panels has one title bar
// rather than three.
//
// A panel's layout name is its component name ([widget.Base.SetName], as
// Qt saves docks by objectName); it must be stable across runs.
type Panel struct {
	widget.Base
	title   string
	content widget.Component
	min     paintengine2d.Point

	closed    bool
	collapsed bool
	features  Features

	// homeSide and homeStack are where the panel last was docked, so a
	// floating or closed panel comes back where it left.
	homeSide  Side
	homeStack *Stack

	// win is the window the panel floats in, nil while it is docked.
	win FloatWindow
	// owner is the host the panel belongs to while it floats, since the
	// component tree no longer reaches it then.
	owner *Host
	// geom is where the panel floats, kept across dock and undock so a
	// panel floated twice comes back the same size. Empty until it has
	// floated once.
	geom paintengine2d.Rect

	// OnClose runs as the panel is about to close; returning false keeps
	// it open (a panel with unsaved work).
	OnClose func() bool
	// OnShown runs when the panel is closed or shown again.
	OnShown func(shown bool)
	// OnFloat runs when the panel floats or docks back.
	OnFloat func(floating bool)
}

// NewPanel is a panel called title showing content. name is what a saved
// layout calls it and must be stable across runs; it is what [Host.Panel]
// looks up.
func NewPanel(name, title string, content widget.Component) *Panel {
	p := &Panel{title: title, content: content, features: DefaultFeatures}
	p.Init(p)
	p.SetName(name)
	if content != nil {
		p.Add(content)
	}
	return p
}

// Features is what the panel's title bar offers.
func (p *Panel) Features() Features { return p.features }

// SetFeatures says what the panel's title bar offers: a panel without
// FeatureClosable has no close button and ignores Ctrl+W, one without
// FeatureMovable cannot be dragged elsewhere (Qt's setFeatures).
func (p *Panel) SetFeatures(f Features) {
	if p.features == f {
		return
	}
	p.features = f
	if st := p.Stack(); st != nil {
		st.head.Invalidate()
	}
}

// Title is what the title bar and the tab show.
func (p *Panel) Title() string { return p.title }

// SetTitle renames the panel.
func (p *Panel) SetTitle(t string) {
	if p.title == t {
		return
	}
	p.title = t
	if st := p.Stack(); st != nil {
		st.Invalidate()
	}
	if p.win != nil {
		p.win.SetTitle(t)
	}
}

// Content is what the panel shows, or nil.
func (p *Panel) Content() widget.Component { return p.content }

// SetContent replaces the panel's content.
func (p *Panel) SetContent(c widget.Component) {
	if p.content == c {
		return
	}
	if p.content != nil {
		p.Remove(p.content)
	}
	p.content = c
	if c != nil {
		p.Add(c)
	}
	p.RequestLayout()
	p.Invalidate()
}

// SetMinSize is the smallest content box the panel will be squeezed into.
// Splits honour it: a sash stops rather than shrink a panel past it.
func (p *Panel) SetMinSize(w, h float32) {
	p.min = paintengine2d.Pt(w, h)
	p.RequestLayout()
}

// MinContent is the panel's own minimum, never under the floor every panel
// gets so no neighbour can squeeze it away entirely.
func (p *Panel) MinContent() paintengine2d.Point {
	return maxPt(p.min, minPanelSize(p.Look()))
}

// Closed reports whether the panel is hidden. A closed panel keeps its
// place in the layout, so showing it again puts it back where it was.
func (p *Panel) Closed() bool { return p.closed }

// Close hides the panel, asking OnClose first. It reports whether the
// panel closed.
func (p *Panel) Close() bool {
	if p.closed {
		return true
	}
	if p.OnClose != nil && !p.OnClose() {
		return false
	}
	p.setClosed(true)
	return true
}

// Show puts a closed panel back where it was. A floating panel's window
// comes back with it.
func (p *Panel) Show() {
	if !p.closed {
		return
	}
	p.setClosed(false)
}

// setClosed hides or shows the panel and tells everything that cares.
func (p *Panel) setClosed(v bool) {
	if p.closed == v {
		return
	}
	p.closed = v
	st := p.Stack()
	if st != nil {
		st.panelClosedChanged(p)
	}
	if p.win != nil {
		if v {
			p.win.Hide()
		} else {
			p.win.Show()
		}
	}
	if h := p.DockHost(); h != nil {
		h.relayout()
	}
	if p.OnShown != nil {
		p.OnShown(!v)
	}
}

// Collapsed reports whether only the panel's title bar shows.
func (p *Panel) Collapsed() bool { return p.collapsed }

// SetCollapsed hides or shows the panel's content, leaving its title bar
// (the twisty of a VS Code view, a collapsed Qt Creator pane).
func (p *Panel) SetCollapsed(v bool) {
	if p.collapsed == v {
		return
	}
	p.collapsed = v
	if st := p.Stack(); st != nil {
		st.Invalidate()
	}
	if h := p.DockHost(); h != nil {
		h.relayout()
	}
}

// ToggleCollapsed collapses an expanded panel and expands a collapsed one.
func (p *Panel) ToggleCollapsed() { p.SetCollapsed(!p.collapsed) }

// Floating reports whether the panel is in a window of its own.
func (p *Panel) Floating() bool { return p.win != nil }

// FloatGeometry is where the panel floats: the geometry its window has now,
// else the one it will open at. An empty rect means it has never floated.
func (p *Panel) FloatGeometry() paintengine2d.Rect {
	if p.win != nil {
		if g := p.win.Geometry(); !g.Empty() {
			return g
		}
	}
	return p.geom
}

// SetFloatGeometry is where the panel floats next.
func (p *Panel) SetFloatGeometry(g paintengine2d.Rect) { p.geom = g }

// Stack is the stack the panel is docked in, or nil while it floats.
func (p *Panel) Stack() *Stack {
	st, _ := p.Parent().(*Stack)
	return st
}

// DockHost is the host the panel belongs to, or nil. A floating panel
// still answers with the host that floated it.
func (p *Panel) DockHost() *Host {
	if h := hostOf(p); h != nil {
		return h
	}
	return p.owner
}

// Float moves the panel into a window of its own. It reports whether it
// did: a host with no [WindowOpener] cannot float anything.
func (p *Panel) Float() bool {
	h := p.DockHost()
	if h == nil {
		return false
	}
	return h.FloatPanel(p, p.FloatGeometry())
}

// Dock brings a floating panel back to where it last was docked.
func (p *Panel) Dock() bool {
	h := p.DockHost()
	if h == nil {
		return false
	}
	return h.DockPanel(p)
}

func (p *Panel) Measure(c layout.Constraints) paintengine2d.Point {
	if p.content == nil {
		return c.Constrain(p.MinContent())
	}
	return c.Constrain(maxPt(p.content.Measure(c), p.MinContent()))
}

func (p *Panel) Arrange(r paintengine2d.Rect) {
	p.SetBounds(r)
	if p.content != nil {
		p.content.Arrange(p.LocalBounds())
	}
}

// Paint fills the panel with the look's surface, so content that does not
// paint its own background sits on the pack's face rather than on whatever
// happened to be behind it.
func (p *Panel) Paint(ctx *paintengine2d.Context) {
	b := p.LocalBounds()
	if b.Empty() {
		return
	}
	ctx.DrawRect(b, paintengine2d.Fill(p.Look().Palette().Surface))
}

// Describe implements widget.Accessible: a panel is a named group, as
// ATK's panel role and Qt's QDockWidget are.
func (p *Panel) Describe(n *a11y.Node) {
	n.Role = a11y.RoleGroup
	if n.Name == "" {
		n.Name = widget.PlainText(p.title)
	}
	n.State |= a11y.StateExpandable
	if !p.collapsed {
		n.State |= a11y.StateExpanded
	}
}
