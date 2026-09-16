package dock

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Dragging a panel between the host and a window of its own, with the
// pointer never let go: out of the window it floats, back over the host it
// docks where the indicator says. Both are one gesture — a drag that
// carries a window ([widget.TearOff]) — so a user rearranges an app by
// dragging alone, and the float button is the keyboard's way to the same
// two things rather than the only way.
//
// A panel dragged inside the host is still the host's own drag (drag.go):
// nothing leaves the window while the pointer is in it, so a panel moved
// from one side to the other never flashes a window. The moment the
// pointer leaves, the panel floats into a window that follows it.
//
// Where the desktop cannot carry a window — a Wayland compositor without
// xdg-toplevel-drag-v1 — a panel dragged out floats at the drop instead
// (the same fallback the tab strip takes), and a floating panel's title
// bar goes on moving its window through the desktop's own interactive
// move, with the float button to dock it back.

// PanelMimeType is the private type a panel dragged out of its window is
// offered under. It carries the panel's layout name; the panel itself
// rides along in [widget.DropEvent.Payload], which is the only way it can
// travel, since a panel is a live widget and not a document.
const PanelMimeType = "application/x-uitoolkit-panel"

// tearBand is how far outside the host the pointer goes before a dragged
// panel leaves the window: past the edge by enough that the window's own
// chrome — a status bar under the host, a tool bar above it — is not a
// tear-off by accident.
func tearBand(lk style.LookAndFeel) float32 { return style.Dip(lk, 24) }

// ---- out of the window ---------------------------------------------------

// tornOut takes the panel being dragged out of the window when the
// pointer has left the host far enough, into a window of its own that
// follows the pointer. p is host-local, at in the window's coordinates.
// It reports whether the panel left.
func (h *Host) tornOut(p, at paintengine2d.Point) bool {
	d := h.drag
	if d == nil || !d.active || h.opener == nil {
		return false
	}
	pan := d.panel
	if pan == nil || pan.Floating() || pan.features&FeatureFloatable == 0 {
		return false
	}
	if b := h.LocalBounds(); b.Empty() || b.Inset(-tearBand(h.Look())).Contains(p) {
		return false
	}
	h.drag = nil
	h.hideIndicator()
	// The panel is picked up by the point the user took hold of, not by
	// where the pointer has wandered to since.
	return h.tearOffPanel(pan, d.from, d.start, at)
}

// tearOffPanel floats pan out of stack from and hands its window to a
// drag, so the desktop carries it under the pointer until the user drops
// it: over the host it docks again, anywhere else it stays floating, and
// Escape puts it back where it was.
func (h *Host) tearOffPanel(pan *Panel, from *Stack, grab, at paintengine2d.Point) bool {
	geom, off := h.floatGeometryAt(pan, from, grab, at)
	// Where the desktop carries the window the panel leaves now, so the
	// user drags the panel itself; where it cannot, the panel stays put
	// and the window is made at the drop.
	carried := widget.DragsWindows(h)
	if carried && !h.FloatPanel(pan, geom) {
		return false
	}
	tear := &widget.TearOff{
		Offset: off,
		Open: func() widget.TearOffWindow {
			if !pan.Floating() && !h.FloatPanel(pan, geom) {
				return nil
			}
			return floatTearWindow(pan)
		},
		Done: func(res widget.TearResult, _ widget.TearOffWindow) {
			// Nothing is being dragged any more, so nothing is marked:
			// the desktop can deliver a last motion after the drop, and
			// the indicator would be left standing over the layout it
			// helped make.
			h.hideIndicator()
			switch res {
			case widget.TearCancelled:
				// Nothing happened: the panel goes back where it was,
				// which closes the window it was carried in.
				if carried {
					h.DockPanel(pan)
				}
			case widget.TearKept:
				pan.rememberGeometry()
			}
		},
	}
	if widget.StartTearOff(h, h.panelDrag(pan), tear) {
		return true
	}
	if carried {
		h.DockPanel(pan)
	}
	return false
}

// dragFloatingPanel drags a floating panel's whole window: the desktop
// carries it, its host lights up where it would land, and a drop there
// docks it back. c is the chrome the press landed on — the drag starts in
// the window that holds it, which is the floating one. It reports whether
// the drag started; without a desktop that can carry a window it does
// not, and the title bar falls back to the desktop's interactive move.
func (h *Host) dragFloatingPanel(c widget.Component, pan *Panel, at paintengine2d.Point) bool {
	if c == nil || pan == nil || !pan.Floating() || pan.features&FeatureMovable == 0 {
		return false
	}
	if !widget.DragsWindows(c) {
		return false
	}
	win := floatTearWindow(pan)
	if win == nil {
		return false
	}
	tear := &widget.TearOff{
		// The window is already under the pointer: the offset is where in
		// it the user took hold, so it does not jump.
		Offset: widget.DeviceOrigin(c).Add(at),
		Open:   func() widget.TearOffWindow { return win },
		Done: func(res widget.TearResult, _ widget.TearOffWindow) {
			h.hideIndicator()
			// The window is the panel's own: docking back closes it
			// (Host.Drop), and anything else leaves it standing where the
			// desktop put it — which is worth remembering.
			if res != widget.TearMerged {
				pan.rememberGeometry()
			}
		},
	}
	return widget.StartTearOff(c, h.panelDrag(pan), tear)
}

// panelDrag is what a panel dragged out of its window offers: the dock's
// private type, with the panel itself riding along in the process — a
// panel is a live widget, so nothing else could carry it — and a move,
// since a panel is in one place or the other.
func (h *Host) panelDrag(pan *Panel) *widget.Drag {
	d := &widget.Drag{
		Types:     []string{PanelMimeType},
		Payload:   pan,
		Actions:   platform.DragMove,
		Preferred: platform.DragMove,
		Data: func(m string) ([]byte, bool) {
			if m != PanelMimeType {
				return nil, false
			}
			return []byte(pan.Name()), true
		},
	}
	// The host's own look and scale: a panel on its way out of the tree
	// may have no host of its own to ask.
	scale := float32(1)
	if host := h.Host(); host != nil && host.Scale() > 0 {
		scale = host.Scale()
	}
	d.Image, d.Hotspot = widget.DragLabel(h.Look(), pan.Title(), scale)
	return d
}

// floatTearWindow is a floating panel's window as a drag can carry it.
func floatTearWindow(pan *Panel) widget.TearOffWindow {
	if pan == nil || pan.win == nil {
		return nil
	}
	win := pan.win.TearOffWindow()
	if win == nil {
		return nil
	}
	return win
}

// floatGeometryAt is where a panel torn out of the host floats: the size
// it would float at anyway, put so that the point the user took hold of
// stays under the pointer. grab is the press, in the host's coordinates,
// and at the pointer now, in the window's; the offset that comes back is
// where in the floating window the pointer will sit.
//
// The position is a request. X11 honours it, and a Wayland compositor is
// not even asked — it places the window itself and then carries it under
// the pointer, which is the whole point of the drag protocol.
func (h *Host) floatGeometryAt(pan *Panel, from *Stack, grab, at paintengine2d.Point) (paintengine2d.Rect, paintengine2d.Point) {
	geom := pan.geom
	if geom.Empty() {
		geom = h.firstFloatGeometry(pan)
	}
	// Where in the panel's own box the pointer took hold, kept inside the
	// window it is about to be in.
	off := paintengine2d.Pt(geom.Dx()*0.5, style.Dip(h.Look(), 12))
	if from != nil {
		if b := h.rectOf(from); !b.Empty() {
			off = paintengine2d.Pt(
				min(max(grab.X-b.Min.X, 0), max(geom.Dx()-1, 0)),
				min(max(grab.Y-b.Min.Y, 0), max(geom.Dy()-1, 0)))
		}
	}
	x, y, ok := hostWindowPosition(h)
	if !ok {
		// Nobody is telling us where this window is, so nothing sensible
		// can be asked for: the desktop places it.
		return paintengine2d.XYWH(0, 0, geom.Dx(), geom.Dy()), off
	}
	return paintengine2d.XYWH(float32(x)+at.X-off.X, float32(y)+at.Y-off.Y, geom.Dx(), geom.Dy()), off
}

// hostWindowPosition is where the desktop put the host's window, where
// the backend is told at all (X11 is, Wayland is not).
func hostWindowPosition(h *Host) (int, int, bool) {
	w, ok := h.Host().(interface{ Position() (int, int, bool) })
	if !ok {
		return 0, 0, false
	}
	return w.Position()
}

// rememberGeometry keeps where a floating panel's window is now, so a
// saved layout brings it back there. It is best effort: a Wayland client
// is not told where its windows are, and the size comes back either way.
func (p *Panel) rememberGeometry() {
	if p == nil || p.win == nil {
		return
	}
	if g := p.win.Geometry(); !g.Empty() {
		p.geom = g
	}
}

// ---- back into the host --------------------------------------------------

// DropTypes implements [widget.DropTarget]: the host takes panels of its
// own, dragged out of the windows they float in.
func (h *Host) DropTypes() []string { return []string{PanelMimeType} }

// Drop docks a panel dragged back over the host, where the indicator said
// it would land. A drop that lands nowhere in particular is refused, and
// the panel stays in the window it is in.
func (h *Host) Drop(e widget.DropEvent) bool {
	h.hideIndicator()
	pan := panelFromDrop(e)
	if pan == nil || pan.owner != h {
		// A panel of another host's is another host's business: its tree
		// holds it, and this one would leave a hole behind.
		return false
	}
	t := h.targetAt(e.Pos)
	if t.kind == dropNone || t.kind == dropFloat {
		return false
	}
	h.applyDrop(pan, t)
	return true
}

// DragOver shows where the panel under the pointer would land, with the
// same indicator a drag inside the host draws.
func (h *Host) DragOver(pos paintengine2d.Point) {
	h.showIndicator(h.targetAt(pos))
}

// DragLeave takes the indicator away as the drag goes.
func (h *Host) DragLeave() { h.hideIndicator() }

// DropActionFor implements [widget.DropActions]: docking a panel moves
// it, and a panel copied into two windows would be two of the same live
// widget.
func (h *Host) DropActionFor(platform.DragAction) platform.DragAction { return platform.DragMove }

// panelFromDrop is the panel a drop carries: one of this host's own, from
// a drag inside this application. Nothing else can be a panel — the type
// carries a name, and the widget it names belongs to a process.
func panelFromDrop(e widget.DropEvent) *Panel {
	pan, _ := e.Payload.(*Panel)
	return pan
}
