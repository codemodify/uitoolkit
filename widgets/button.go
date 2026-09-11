package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Button is a clickable labeled control.
type Button struct {
	widget.Base
	Text    string
	Primary bool
	OnClick func()
	hovered bool
	pressed bool
}

func NewButton(text string, onClick func()) *Button {
	b := &Button{Text: text, OnClick: onClick}
	b.Init(b)
	b.SetWantsFocus(true)
	return b
}

func (b *Button) Measure(c layout.Constraints) paintengine2d.Point {
	lk := b.Look()
	m := lk.Metrics()
	w := lk.Font().Advance(b.Text) + m.Pad*2 + 16
	h := m.ControlH
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (b *Button) Arrange(r paintengine2d.Rect) { b.SetBounds(r) }

func (b *Button) Paint(ctx *paintengine2d.Context) {
	st := b.State()
	if b.hovered {
		st |= style.StateHovered
	}
	if b.pressed {
		st |= style.StatePressed
	}
	if b.Primary {
		st |= style.StatePrimary
	}
	b.Look().DrawButton(ctx, b.LocalBounds(), st, b.Text)
}

func (b *Button) MouseEnter() { b.hovered = true; b.Base.MouseEnter() }
func (b *Button) MouseExit() {
	b.hovered = false
	b.pressed = false
	b.Base.MouseExit()
}

func (b *Button) MousePress(e widget.MouseEvent) bool {
	if !b.Enabled() {
		return false
	}
	b.RequestFocus()
	b.pressed = true
	b.Invalidate()
	return true
}

func (b *Button) MouseRelease(e widget.MouseEvent) bool {
	was := b.pressed
	b.pressed = false
	b.Invalidate()
	if was && b.LocalBounds().Contains(e.Pos) && b.OnClick != nil && b.Enabled() {
		b.OnClick()
	}
	return true
}

func (b *Button) KeyPress(e widget.KeyEvent) bool {
	if e.Key == platform.KeyReturn || e.Key == platform.KeySpace {
		if b.OnClick != nil && b.Enabled() {
			b.OnClick()
		}
		return true
	}
	return false
}
