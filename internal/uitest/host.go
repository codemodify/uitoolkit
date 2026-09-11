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
	surfW    int
	surfH    int
}

// NewHost builds a dark, 1× host unless look/scale are set later.
func NewHost() *Host { return &Host{scale: 1, surfW: 800, surfH: 600} }

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

var _ widget.Host = (*Host)(nil)
var _ widget.CursorHost = (*Host)(nil)
var _ widget.PopupHost = (*Host)(nil)
var _ widget.SurfaceSizer = (*Host)(nil)
