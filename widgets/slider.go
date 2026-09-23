package widgets

import (
	"math"

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

// SliderPainter paints a slider in place of the look's track, fill and
// thumb and reports whether it did; t is the value as a fraction of the
// range, 0 at Min. It is what lets a skin draw an orange volume bar beside
// a green balance bar, which one set of slider parts cannot say. The focus
// ring is drawn over the art either way.
type SliderPainter func(ctx *paintengine2d.Context, b paintengine2d.Rect, t float32, st style.ControlState) bool

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
	// Vertical stands the slider on end: it takes its width from the
	// slider metric and draws down its height, and up is more. TicksAbove
	// is then the left-hand row and TicksBelow the right-hand one.
	Vertical bool
	// Painter, when set, paints the slider in place of the look's parts.
	Painter SliderPainter
	// Travel is how far in from each end of the track the thumb's centre
	// stops, in design pixels: half the thumb a Painter draws, so the
	// pointer and the picture agree about where the ends are. Zero leaves
	// it to the look (style.SliderTravelOf), which knows its own thumb.
	Travel  float32
	hovered bool
	drag    bool
	// grabD is where along the track the pointer took hold of the thumb:
	// pressing the thumb never moves it, only the track does.
	grabD float32
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

// NewVerticalSlider stands one on end: up is more. It is the same slider
// and the same skin parts — a rack of them is a mixer, a row of horizontal
// ones is a form.
func NewVerticalSlider(min, max, value float32, on func(float32)) *Slider {
	s := NewSlider(min, max, value, on)
	s.Vertical = true
	s.SetPreferred(28, 160)
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

// trackIn is box less the tick rows, which are trimmed off the cross axis.
// box is the slider's own box for a horizontal slider and the turned box
// for a vertical one, where the cross axis is Y either way.
func trackIn(box paintengine2d.Rect, ticks TickPlacement, band float32) paintengine2d.Rect {
	if ticks == TicksAbove || ticks == TicksBoth {
		box.Min.Y += band
	}
	if ticks == TicksBelow || ticks == TicksBoth {
		box.Max.Y -= band
	}
	return box
}

// turned is a box with its width and height swapped, the frame a vertical
// slider is painted and measured in.
func turned(b paintengine2d.Rect) paintengine2d.Rect {
	return paintengine2d.XYWH(b.Min.Y, b.Min.X, b.Dy(), b.Dx())
}

// track is the slider's own box less the tick rows, in its own coordinates.
func (s *Slider) track() paintengine2d.Rect {
	b := s.LocalBounds()
	if s.Vertical {
		return turned(trackIn(turned(b), s.Ticks, s.tickBand()))
	}
	return trackIn(b, s.Ticks, s.tickBand())
}

func (s *Slider) Measure(c layout.Constraints) paintengine2d.Point {
	h := s.Look().Metrics().SliderH
	switch s.Ticks {
	case TicksBelow, TicksAbove:
		h += s.tickBand()
	case TicksBoth:
		h += 2 * s.tickBand()
	}
	long := style.Dip(s.Look(), 160)
	if s.Vertical {
		return c.Constrain(paintengine2d.Pt(h, long))
	}
	return c.Constrain(paintengine2d.Pt(long, h))
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

// PaintState is the state the track and the thumb are painted in.
func (s *Slider) PaintState() style.ControlState {
	st := s.State()
	if s.hovered {
		st |= style.StateHovered
	}
	if s.drag {
		st |= style.StatePressed
	}
	return st
}

// Paint draws the look's slider, or hands the box to a Painter that has art
// of its own. A vertical slider is the same parts inside a turned frame, so
// it is skinned from slider.track, slider.fill and slider.thumb with no art
// of its own — and the turn is what makes up more.
func (s *Slider) Paint(ctx *paintengine2d.Context) {
	lk, b := s.Look(), s.LocalBounds()
	st := s.PaintState()
	if s.Painter != nil && s.Painter(ctx, b, s.t(), st) {
		// A picture of a slider is still a slider, and one the keyboard is
		// on says so: the painter never gets to leave the ring out.
		if st.Focused() {
			lk.DrawFocusRing(ctx, b)
		}
		return
	}
	if !s.Vertical {
		s.paintParts(ctx, lk, st, b)
		return
	}
	ctx.Save()
	// Rotate about the box's centre and draw into the turned frame, as
	// players.Fader does.
	cx, cy := (b.Min.X+b.Max.X)/2, (b.Min.Y+b.Max.Y)/2
	ctx.Translate(cx, cy)
	ctx.Rotate(-90 * math.Pi / 180)
	ctx.Translate(-cy, -cx)
	s.paintParts(ctx, lk, st, turned(b))
	ctx.Restore()
	if st.Focused() {
		lk.DrawFocusRing(ctx, b)
	}
}

// paintParts draws the track and the tick rows inside box, which is the
// slider's own box for a horizontal slider and the turned one for a
// vertical slider.
func (s *Slider) paintParts(ctx *paintengine2d.Context, lk style.LookAndFeel, st style.ControlState, box paintengine2d.Rect) {
	band := s.tickBand()
	tr := trackIn(box, s.Ticks, band)
	lk.DrawSlider(ctx, tr, st, s.t())
	if s.Ticks == TicksNone {
		return
	}
	lo, hi := s.travelIn(tr)
	xs := s.tickXs(lo, hi)
	gap := style.Dip(lk, 1)
	if s.Ticks == TicksAbove || s.Ticks == TicksBoth {
		style.DrawSliderTicksOf(lk, ctx, paintengine2d.XYWH(box.Min.X, box.Min.Y, box.Dx(), band-gap), xs, st)
	}
	if s.Ticks == TicksBelow || s.Ticks == TicksBoth {
		style.DrawSliderTicksOf(lk, ctx, paintengine2d.XYWH(box.Min.X, box.Max.Y-band+gap, box.Dx(), band-gap), xs, st)
	}
}

// travelIn is where the thumb's centre stops at each end of track tr, along
// tr's length. Travel is the Painter's answer — half the thumb its art
// draws — and without one the look's, which knows its own thumb.
func (s *Slider) travelIn(tr paintengine2d.Rect) (lo, hi float32) {
	if s.Travel > 0 {
		pad := min(style.Dip(s.Look(), s.Travel), tr.Dx()/4)
		return tr.Min.X + pad, tr.Max.X - pad
	}
	return style.SliderTravelOf(s.Look(), tr)
}

// travel is where the thumb's centre stops, along the slider's own axis in
// its own coordinates: left to right, or — since up is more — bottom to top.
func (s *Slider) travel() (at0, at1 float32) {
	tr := s.track()
	if !s.Vertical {
		return s.travelIn(tr)
	}
	// In the turned frame the ends run with the axis; turning them back
	// puts t = 0 at the bottom.
	lo, hi := s.travelIn(turned(tr))
	return tr.Min.Y + tr.Max.Y - lo, tr.Min.Y + tr.Max.Y - hi
}

// along is where p falls on the slider's axis.
func (s *Slider) along(p paintengine2d.Point) float32 {
	if s.Vertical {
		return p.Y
	}
	return p.X
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
	at0, at1 := s.travel()
	thumb := at0 + (at1-at0)*s.t()
	half := max(s.Look().Metrics().Thumb*0.5, style.Dip(s.Look(), 6))
	s.grabD = 0
	if d := s.along(e.Pos) - thumb; d >= -half && d <= half {
		s.grabD = d
		return true
	}
	s.setFromPos(s.along(e.Pos))
	return true
}

func (s *Slider) MouseMove(e widget.MouseEvent) bool {
	if s.drag {
		s.setFromPos(s.along(e.Pos) - s.grabD)
		return true
	}
	return false
}

func (s *Slider) MouseRelease(e widget.MouseEvent) bool {
	s.drag = false
	s.Invalidate()
	return true
}

// setFromPos puts the value where the pointer is along the slider's axis.
func (s *Slider) setFromPos(at float32) {
	at0, at1 := s.travel()
	if at1 == at0 {
		return
	}
	t := (at - at0) / (at1 - at0)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	s.SetValue(s.Min + t*(s.Max-s.Min))
}
