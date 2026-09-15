package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Switch is a labeled on/off toggle (pill track).
type Switch struct {
	widget.Base
	Text     string
	On       bool
	OnChange func(bool)
	hovered  bool
	pressed  bool
	fade     stateFade // hover / focus cross-fade (the look's HintHoverFadeMs)
}

// NewSwitch builds a toggle. on is the initial value.
func NewSwitch(text string, on bool, change func(bool)) *Switch {
	s := &Switch{Text: text, On: on, OnChange: change}
	s.Init(s)
	s.SetWantsFocus(true)
	s.SetFocusVisibleOnly(true)
	return s
}

func (s *Switch) SetOn(v bool) {
	if s.On == v {
		return
	}
	s.On = v
	s.Invalidate()
	if s.OnChange != nil {
		s.OnChange(v)
	}
}

func (s *Switch) Measure(c layout.Constraints) paintengine2d.Point {
	lk := s.Look()
	m := lk.Metrics()
	tw := m.SwitchW
	if tw <= 0 {
		tw = 42
	}
	w := tw + style.Dip(lk, 8) + style.ControlFontOf(lk, style.RoleCheck).Advance(s.Text) + style.Dip(lk, 4)
	return c.Constrain(paintengine2d.Pt(w, m.ControlH))
}

func (s *Switch) Arrange(r paintengine2d.Rect) { s.SetBounds(r) }

func (s *Switch) Paint(ctx *paintengine2d.Context) {
	st := s.State()
	if s.hovered {
		st |= style.StateHovered
	}
	if s.On {
		st |= style.StateChecked
	}
	lk, r := s.Look(), s.LocalBounds()
	s.fade.paint(s, ctx, r, st, func(ctx *paintengine2d.Context, st style.ControlState) { lk.DrawSwitch(ctx, r, st, s.On, s.Text) })
}

func (s *Switch) MouseEnter() { s.hovered = true; s.Base.MouseEnter() }
func (s *Switch) MouseExit() {
	s.hovered = false
	s.pressed = false
	s.Base.MouseExit()
}

func (s *Switch) MousePress(widget.MouseEvent) bool {
	if !s.Enabled() {
		return false
	}
	s.MarkPointerFocus()
	s.RequestFocus()
	s.pressed = true
	s.Invalidate()
	return true
}

func (s *Switch) MouseRelease(e widget.MouseEvent) bool {
	was := s.pressed
	s.pressed = false
	s.Invalidate()
	if was && s.Enabled() && s.LocalBounds().Contains(e.Pos) {
		s.SetOn(!s.On)
	}
	return true
}

func (s *Switch) KeyPress(e widget.KeyEvent) bool {
	if !s.Enabled() {
		return false
	}
	s.MarkKeyboardFocus()
	if e.Key == platform.KeySpace || e.Key == platform.KeyReturn {
		s.SetOn(!s.On)
		return true
	}
	return false
}
