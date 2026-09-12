package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Slider is a continuous numeric control in [Min, Max].
type Slider struct {
	widget.Base
	Min, Max, Value float32
	OnChange        func(float32)
	hovered, drag   bool
}

func NewSlider(min, max, value float32, on func(float32)) *Slider {
	if max <= min {
		max = min + 1
	}
	s := &Slider{Min: min, Max: max, Value: value, OnChange: on}
	s.Init(s)
	s.SetWantsFocus(true)
	s.SetFocusVisibleOnly(true)
	s.SetPreferred(160, 28)
	return s
}

func (s *Slider) SetValue(v float32) {
	if v < s.Min {
		v = s.Min
	}
	if v > s.Max {
		v = s.Max
	}
	if s.Value == v {
		return
	}
	s.Value = v
	s.Invalidate()
	if s.OnChange != nil {
		s.OnChange(v)
	}
}

func (s *Slider) t() float32 {
	if s.Max <= s.Min {
		return 0
	}
	return (s.Value - s.Min) / (s.Max - s.Min)
}

func (s *Slider) Measure(c layout.Constraints) paintengine2d.Point {
	h := s.Look().Metrics().SliderH
	return c.Constrain(paintengine2d.Pt(160, h))
}

func (s *Slider) Arrange(r paintengine2d.Rect) { s.SetBounds(r) }

func (s *Slider) Paint(ctx *paintengine2d.Context) {
	st := s.State()
	if s.hovered {
		st |= style.StateHovered
	}
	if s.drag {
		st |= style.StatePressed
	}
	s.Look().DrawSlider(ctx, s.LocalBounds(), st, s.t())
}

func (s *Slider) MouseEnter() { s.hovered = true; s.Base.MouseEnter() }
func (s *Slider) MouseExit() {
	s.hovered = false
	if !s.drag {
		s.Base.MouseExit()
	}
}

func (s *Slider) KeyPress(e widget.KeyEvent) bool {
	if !s.Enabled() {
		return false
	}
	s.MarkKeyboardFocus()
	span := s.Max - s.Min
	step := span / 20
	if e.Mods.Shift() {
		step = span / 80
	}
	if step == 0 {
		step = 1
	}
	switch e.Key {
	case platform.KeyLeft, platform.KeyDown:
		s.SetValue(s.Value - step)
		return true
	case platform.KeyRight, platform.KeyUp:
		s.SetValue(s.Value + step)
		return true
	case platform.KeyPageDown:
		s.SetValue(s.Value - span*0.1)
		return true
	case platform.KeyPageUp:
		s.SetValue(s.Value + span*0.1)
		return true
	case platform.KeyHome:
		s.SetValue(s.Min)
		return true
	case platform.KeyEnd:
		s.SetValue(s.Max)
		return true
	}
	return false
}

func (s *Slider) MousePress(e widget.MouseEvent) bool {
	if !s.Enabled() {
		return false
	}
	s.MarkPointerFocus()
	s.RequestFocus()
	s.drag = true
	s.setFromX(e.Pos.X)
	return true
}

func (s *Slider) MouseMove(e widget.MouseEvent) bool {
	if s.drag {
		s.setFromX(e.Pos.X)
		return true
	}
	return false
}

func (s *Slider) MouseRelease(e widget.MouseEvent) bool {
	s.drag = false
	s.Invalidate()
	return true
}

func (s *Slider) setFromX(x float32) {
	m := s.Look().Metrics()
	x0 := m.Thumb * 0.5
	x1 := s.LocalBounds().Dx() - m.Thumb*0.5
	if x1 <= x0 {
		return
	}
	t := (x - x0) / (x1 - x0)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	s.SetValue(s.Min + t*(s.Max-s.Min))
}
