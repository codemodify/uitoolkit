package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// TickPlacement says where a slider draws its tick marks.
type TickPlacement uint8

const (
	TicksNone TickPlacement = iota
	TicksBelow
	TicksAbove
	TicksBoth
)

// Slider is a continuous numeric control in [Min, Max].
type Slider struct {
	widget.Base
	Min, Max, Value float32
	OnChange        func(float32)
	// Ticks puts tick marks under, over or on both sides of the track
	// (QSlider's tickPosition, WPF's TickPlacement), every TickInterval
	// (0: a tenth of the range).
	Ticks        TickPlacement
	TickInterval float32
	hovered      bool
	drag         bool
	// grabDX is where on the thumb the pointer took hold of it: pressing
	// the thumb never moves it, only the track does.
	grabDX float32
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

// tickBand is the height of one row of tick marks.
func (s *Slider) tickBand() float32 {
	if s.Ticks == TicksNone {
		return 0
	}
	return style.Dip(s.Look(), 6)
}

// track is the slider's own box: its bounds less the tick rows.
func (s *Slider) track() paintengine2d.Rect {
	b := s.LocalBounds()
	band := s.tickBand()
	if s.Ticks == TicksAbove || s.Ticks == TicksBoth {
		b.Min.Y += band
	}
	if s.Ticks == TicksBelow || s.Ticks == TicksBoth {
		b.Max.Y -= band
	}
	return b
}

func (s *Slider) Measure(c layout.Constraints) paintengine2d.Point {
	h := s.Look().Metrics().SliderH
	switch s.Ticks {
	case TicksBelow, TicksAbove:
		h += s.tickBand()
	case TicksBoth:
		h += 2 * s.tickBand()
	}
	return c.Constrain(paintengine2d.Pt(style.Dip(s.Look(), 160), h))
}

func (s *Slider) Arrange(r paintengine2d.Rect) { s.SetBounds(r) }

// tickXs are the tick positions along the thumb's travel.
func (s *Slider) tickXs(x0, x1 float32) []float32 {
	span := s.Max - s.Min
	step := s.TickInterval
	if step <= 0 {
		step = span / 10
	}
	if span <= 0 || step <= 0 || span/step > 200 {
		return nil
	}
	var xs []float32
	for v := float32(0); v <= span+step*1e-3; v += step {
		xs = append(xs, x0+(x1-x0)*min(v/span, 1))
	}
	return xs
}

func (s *Slider) Paint(ctx *paintengine2d.Context) {
	st := s.State()
	if s.hovered {
		st |= style.StateHovered
	}
	if s.drag {
		st |= style.StatePressed
	}
	lk := s.Look()
	tr := s.track()
	lk.DrawSlider(ctx, tr, st, s.t())
	if s.Ticks == TicksNone {
		return
	}
	x0, x1 := style.SliderTravelOf(lk, tr)
	xs := s.tickXs(x0, x1)
	band := s.tickBand()
	gap := style.Dip(lk, 1)
	b := s.LocalBounds()
	if s.Ticks == TicksAbove || s.Ticks == TicksBoth {
		style.DrawSliderTicksOf(lk, ctx, paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), band-gap), xs, st)
	}
	if s.Ticks == TicksBelow || s.Ticks == TicksBoth {
		style.DrawSliderTicksOf(lk, ctx, paintengine2d.XYWH(b.Min.X, b.Max.Y-band+gap, b.Dx(), band-gap), xs, st)
	}
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
	// On the thumb: take hold of it where it was pressed. On the track:
	// the thumb jumps there (GTK's warping primary button).
	x0, x1 := style.SliderTravelOf(s.Look(), s.track())
	thumb := x0 + (x1-x0)*s.t()
	half := max(s.Look().Metrics().Thumb*0.5, style.Dip(s.Look(), 6))
	s.grabDX = 0
	if d := e.Pos.X - thumb; d >= -half && d <= half {
		s.grabDX = d
		return true
	}
	s.setFromX(e.Pos.X)
	return true
}

func (s *Slider) MouseMove(e widget.MouseEvent) bool {
	if s.drag {
		s.setFromX(e.Pos.X - s.grabDX)
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
	x0, x1 := style.SliderTravelOf(s.Look(), s.track())
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
