package dock

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/widget"
)

// FloatWindow is the window one floating panel lives in — a real toplevel
// the desktop moves, resizes and stacks. app.DockWindows wraps app.Window
// into one; a test supplies its own.
type FloatWindow interface {
	// SetTitle names the window.
	SetTitle(title string)
	// SetContent puts the panel's chrome and content in the window.
	SetContent(c widget.Component)
	// Geometry is where the window is and how big, in screen pixels. The
	// origin is best effort: a Wayland client is not told where its
	// windows are, so it is the position the window was asked for.
	Geometry() paintengine2d.Rect
	// SetOnCloseRequest is called when the desktop asks the window to
	// close; returning false keeps the window open.
	SetOnCloseRequest(fn func() bool)
	Show()
	Hide()
	Raise()
	Close()
}

// WindowOpener opens the windows floating panels live in. An app gives the
// host one with [Host.SetWindowOpener]; without it panels cannot float,
// and the float button does not appear.
//
// Tearing a panel off through the compositor — dragging it straight out of
// the window with xdg-toplevel-drag, so the pointer never lets go — would
// be another implementation of this interface plus a drag hand-off; see
// docs/decorations.md. Nothing in the dock package assumes the window
// appears only on a button press.
type WindowOpener interface {
	// OpenFloat opens a window for a panel. geom is where to put it; an
	// empty rect asks for the desktop's own choice.
	OpenFloat(title string, geom paintengine2d.Rect) (FloatWindow, error)
}

// defaultFloatSize is how big a panel's window is when it has never
// floated and the app did not say.
var defaultFloatSize = paintengine2d.Pt(320, 400)

// FloatPanel takes p out of the tree and into a window of its own. geom is
// where to put the window; an empty rect uses the panel's remembered
// geometry, else a default size. It reports whether the panel floated.
func (h *Host) FloatPanel(p *Panel, geom paintengine2d.Rect) bool {
	if p == nil || h.opener == nil || p.features&FeatureFloatable == 0 {
		return false
	}
	if p.Floating() {
		p.win.Raise()
		return true
	}
	h.adopt(p)
	if geom.Empty() {
		geom = p.geom
	}
	if geom.Empty() {
		b := h.rectOf(p)
		if b.Empty() {
			b = paintengine2d.XYWH(0, 0, defaultFloatSize.X, defaultFloatSize.Y)
		}
		geom = paintengine2d.XYWH(0, 0, b.Dx(), b.Dy())
	}
	h.detach(p)
	// The panel keeps the same chrome it had docked — a stack of one —
	// so the title bar's buttons, its keyboard handling and its
	// accessibility are the same code in both places.
	st := NewStack(p)
	win, err := h.opener.OpenFloat(p.Title(), geom)
	if err != nil || win == nil {
		// The window would not open: put the panel back where it was.
		st.removePanel(p)
		h.redock(p)
		h.after()
		return false
	}
	p.win = win
	p.geom = geom
	win.SetOnCloseRequest(func() bool {
		if p.OnClose != nil && !p.OnClose() {
			return false
		}
		p.setClosed(true)
		return false // the window hides with the panel rather than dying
	})
	win.SetContent(st)
	if !p.closed {
		win.Show()
	}
	if p.OnFloat != nil {
		p.OnFloat(true)
	}
	h.after()
	return true
}

// DockPanel brings a floating panel back into the tree, where it last was
// docked. It reports whether it docked.
func (h *Host) DockPanel(p *Panel) bool {
	if p == nil || !p.Floating() {
		return false
	}
	h.dockBack(p, func() { h.redock(p) })
	return true
}

// dockBack closes a floating panel's window, keeping its geometry, and
// then runs place to put the panel back in the tree.
func (h *Host) dockBack(p *Panel, place func()) {
	win := p.win
	if win != nil {
		if g := win.Geometry(); !g.Empty() {
			p.geom = g
		}
		if st := p.Stack(); st != nil {
			st.removePanel(p)
		}
		win.SetContent(nil)
		win.Close()
	}
	p.win = nil
	place()
	if p.OnFloat != nil {
		p.OnFloat(false)
	}
	h.after()
}

// redock puts p back where it last was docked: into the stack it came from
// while that is still in the tree, otherwise a new stack on the side it
// was on.
func (h *Host) redock(p *Panel) {
	if st := p.homeStack; st != nil && h.holds(st) && len(st.panels) > 0 {
		st.addPanel(p, len(st.panels))
		st.SelectPanel(p)
		st.sync()
		return
	}
	area := h.areas[sideIndex(p.homeSide)]
	area.insert(len(area.kids), NewStack(p), 1)
}

// holds reports whether n is still somewhere in the host's tree.
func (h *Host) holds(n Node) bool {
	parent, _ := h.parentOf(n)
	return parent != nil
}

// FloatingPanels are the panels in windows of their own.
func (h *Host) FloatingPanels() []*Panel {
	var out []*Panel
	for _, p := range h.panels {
		if p.Floating() {
			out = append(out, p)
		}
	}
	return out
}

// CloseFloating closes every floating panel's window, docking the panels
// back. An app calls it as it shuts down so no window outlives the main
// one.
func (h *Host) CloseFloating() {
	for _, p := range h.FloatingPanels() {
		h.DockPanel(p)
	}
}
