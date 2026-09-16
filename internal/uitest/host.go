package uitest

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Host is a widget.Host for tests (no display). It records focus, damage,
// layout requests, and the last pointer cursor. It also hosts popups.
type Host struct {
	focus    widget.Component
	look     style.LookAndFeel
	scale    float32
	cursor   platform.Cursor
	layoutN  int
	damageN  int
	lastRect paintengine2d.Rect
	popup    widget.Component
	overlay  widget.Component
	surfW    int
	surfH    int
	// posX, posY are where the desktop put this window and posOK whether
	// it says at all — a Wayland toplevel has no position, so a test can
	// take it away. carries is whether a drag from here can carry a
	// window under the pointer, and tear the last tear-off started.
	posX, posY int
	posOK      bool
	carries    bool
	tear       TearOff
}

// NewHost builds a dark, 1× host unless look/scale are set later. It
// stands in for a desktop that tells a window where it is and can carry
// one under a drag, as X11 does; SetPosition and SetCarriesWindows take
// either away.
func NewHost() *Host {
	return &Host{scale: 1, surfW: 800, surfH: 600, posOK: true, carries: true}
}

// TearOff is a tear-off drag started on this host: what it offers, its
// window half, and the window that was opened for it. A test drives the
// endings from Tear.Done ([widget.TearOff]).
type TearOff struct {
	Drag   *widget.Drag
	Tear   *widget.TearOff
	Window widget.TearOffWindow
	Starts int
	// Refuse makes StartTearOff fail, as a drag already running does.
	Refuse bool
}

// SetCarriesWindows says whether a drag from this host can carry a window
// under the pointer (Wayland's xdg-toplevel-drag-v1, X11 always).
func (h *Host) SetCarriesWindows(v bool) { h.carries = v }

// DragsWindows answers [widget.DragsWindows].
func (h *Host) DragsWindows() bool { return h.carries }

// StartTearOff answers [widget.TearOffHost]: it records the drag and, on
// a host that carries windows, opens the tear-off's window at once —
// which is what app.Window does.
func (h *Host) StartTearOff(d *widget.Drag, t *widget.TearOff) bool {
	if h.tear.Refuse {
		return false
	}
	h.tear.Drag, h.tear.Tear, h.tear.Window = d, t, nil
	h.tear.Starts++
	if h.carries && t != nil && t.Open != nil {
		h.tear.Window = t.Open()
	}
	return true
}

// TearOff is the last tear-off drag started on this host.
func (h *Host) TearOff() *TearOff { return &h.tear }

// SetPosition puts the window at x, y on the test desktop; ok false is a
// window that is not told where it is, as every Wayland toplevel is.
func (h *Host) SetPosition(x, y int, ok bool) { h.posX, h.posY, h.posOK = x, y, ok }

// Position answers where the desktop put this window, for whoever asks
// (app.Window answers the same way).
func (h *Host) Position() (int, int, bool) { return h.posX, h.posY, h.posOK }

func (h *Host) Invalidate(_ widget.Component, r paintengine2d.Rect) {
	h.damageN++
	h.lastRect = r
}
func (h *Host) RequestFocus(c widget.Component) { h.focus = c }
func (h *Host) Focus() widget.Component         { return h.focus }
func (h *Host) Scale() float32 {
	if h.scale <= 0 {
		return 1
	}
	return h.scale
}
func (h *Host) Look() style.LookAndFeel {
	if h.look != nil {
		return h.look
	}
	return style.DarkLook()
}
func (h *Host) RequestLayout() { h.layoutN++ }

func (h *Host) SetLook(l style.LookAndFeel) { h.look = l }
func (h *Host) SetScale(s float32)          { h.scale = s }

func (h *Host) SetCursor(c platform.Cursor) { h.cursor = c }
func (h *Host) Cursor() platform.Cursor     { return h.cursor }

func (h *Host) LayoutCount() int { return h.layoutN }
func (h *Host) DamageCount() int { return h.damageN }

func (h *Host) SetSurfaceSize(w, hh int) { h.surfW, h.surfH = w, hh }
func (h *Host) SurfaceSize() (int, int) {
	if h.surfW < 1 {
		h.surfW = 800
	}
	if h.surfH < 1 {
		h.surfH = 600
	}
	return h.surfW, h.surfH
}

func (h *Host) SetPopup(c widget.Component) {
	if h.popup != nil && h.popup != c {
		if d, ok := h.popup.(widget.Dismisser); ok {
			d.Dismissed()
		}
	}
	h.popup = c
	if c != nil {
		c.SetHost(h)
	}
}
func (h *Host) Popup() widget.Component { return h.popup }
func (h *Host) DismissPopup() {
	if h.popup == nil {
		return
	}
	if d, ok := h.popup.(widget.Dismisser); ok {
		d.Dismissed()
	}
	h.popup = nil
}

func (h *Host) SetOverlay(c widget.Component) {
	if h.overlay == c {
		return
	}
	old := h.overlay
	h.overlay = c
	if c != nil {
		c.SetHost(h)
	}
	if old != nil && old != c {
		if d, ok := old.(widget.Dismisser); ok {
			d.Dismissed()
		}
	}
}

func (h *Host) Overlay() widget.Component { return h.overlay }

var _ widget.Host = (*Host)(nil)
var _ widget.CursorHost = (*Host)(nil)
var _ widget.PopupHost = (*Host)(nil)
var _ widget.OverlayHost = (*Host)(nil)
var _ widget.SurfaceSizer = (*Host)(nil)
