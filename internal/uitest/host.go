package uitest

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Host is a widget.Host for tests (no display). It records focus, damage,
// layout requests, and the last pointer cursor.
type Host struct {
	focus    widget.Component
	look     style.LookAndFeel
	scale    float32
	cursor   platform.Cursor
	layoutN  int
	damageN  int
	lastRect paintengine2d.Rect
}

// NewHost builds a dark, 1× host unless look/scale are set later.
func NewHost() *Host { return &Host{scale: 1} }

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

var _ widget.Host = (*Host)(nil)
var _ widget.CursorHost = (*Host)(nil)
