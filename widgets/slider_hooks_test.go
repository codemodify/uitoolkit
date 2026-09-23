package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/a11y"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func TestSliderPainterTakesOverAndIsToldTheValue(t *testing.T) {
	seen := 0
	var frac float32 = -1
	s := NewSlider(0, 100, 25, nil)
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 200, 28))
	plain := immediatePaint(s, 200, 28)
	s.Painter = func(ctx *paintengine2d.Context, b paintengine2d.Rect, tt float32, _ style.ControlState) bool {
		seen++
		frac = tt
		ctx.DrawRect(b, paintengine2d.Fill(paintengine2d.RGB(0.2, 0.2, 0.2)))
		return true
	}
	art := immediatePaint(s, 200, 28)
	if seen != 1 {
		t.Fatalf("the painter was consulted %d times, want once", seen)
	}
	if frac < 0.24 || frac > 0.26 {
		t.Errorf("the painter was told t=%v for a quarter of the range", frac)
	}
	if sameImage(plain, art) {
		t.Error("the painter drew but the slider still looks like the look's parts")
	}
}

func TestSliderPainterThatRefusesFallsBackToTheParts(t *testing.T) {
	s := NewSlider(0, 100, 25, nil)
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 200, 28))
	plain := immediatePaint(s, 200, 28)
	s.Painter = func(*paintengine2d.Context, paintengine2d.Rect, float32, style.ControlState) bool { return false }
	if !sameImage(plain, immediatePaint(s, 200, 28)) {
		t.Error("a painter that took nothing over changed the slider")
	}
}

func TestPaintedSliderKeepsItsFocusRing(t *testing.T) {
	paint := func(focus bool) *paintengine2d.Image {
		s := NewSlider(0, 100, 25, nil)
		s.SetHost(&host{})
		s.Painter = func(ctx *paintengine2d.Context, b paintengine2d.Rect, _ float32, _ style.ControlState) bool {
			ctx.DrawRect(b, paintengine2d.Fill(paintengine2d.RGB(0.2, 0.2, 0.2)))
			return true
		}
		s.Arrange(paintengine2d.XYWH(0, 0, 200, 28))
		if focus {
			s.RequestFocus()
			s.MarkKeyboardFocus()
		}
		return immediatePaint(s, 200, 28)
	}
	if sameImage(paint(false), paint(true)) {
		t.Error("a focused painted slider looks exactly like a resting one — the ring is gone")
	}
}

// Travel is what makes the pointer and the picture agree about the ends: a
// press at the very edge of the track reaches the extreme, and the middle
// of the travel is the middle of the range.
func TestSliderTravelMovesTheEnds(t *testing.T) {
	s := NewSlider(0, 100, 50, nil)
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 200, 28))
	s.Travel = 40 // a fat painted thumb: the travel is 40 in from each end
	s.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(40, 14)})
	if s.Value != 0 {
		t.Errorf("the start of the travel is %v, want 0", s.Value)
	}
	s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(160, 14)})
	if s.Value != 100 {
		t.Errorf("the end of the travel is %v, want 100", s.Value)
	}
	s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(100, 14)})
	if s.Value < 49 || s.Value > 51 {
		t.Errorf("the middle of the travel is %v", s.Value)
	}
	// Travel is clamped to a quarter of the track: an absurd one still
	// leaves a slider that can be moved.
	s.Travel = 400
	s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(0, 14)})
	if s.Value != 0 {
		t.Errorf("an over-long travel left the start at %v", s.Value)
	}
}

// Up is more: the top of a vertical slider is Max and the bottom is Min.
func TestVerticalSliderRunsUpwards(t *testing.T) {
	s := NewVerticalSlider(0, 100, 50, nil)
	s.SetHost(&host{})
	s.Arrange(paintengine2d.XYWH(0, 0, 28, 200))
	s.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(14, 4)})
	if s.Value < 90 {
		t.Errorf("a press at the top gave %v, want the high end", s.Value)
	}
	s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(14, 196)})
	if s.Value > 10 {
		t.Errorf("a drag to the bottom gave %v, want the low end", s.Value)
	}
	s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(14, 100)})
	if s.Value < 40 || s.Value > 60 {
		t.Errorf("the middle gave %v", s.Value)
	}
	// Up and Right still mean more, as they do lying down.
	s.SetValue(50)
	s.KeyPress(widget.KeyEvent{Key: platform.KeyUp})
	if s.Value <= 50 {
		t.Errorf("Up took a vertical slider down to %v", s.Value)
	}
}

func TestVerticalSliderMeasuresOnEnd(t *testing.T) {
	s := NewVerticalSlider(0, 100, 50, nil)
	s.SetHost(&host{})
	sz := s.Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	if sz.Y <= sz.X {
		t.Errorf("a vertical slider measured %vx%v", sz.X, sz.Y)
	}
	h := NewSlider(0, 100, 50, nil)
	h.SetHost(&host{})
	if hz := h.Measure(layout.Constraints{MaxW: -1, MaxH: -1}); hz.X <= hz.Y {
		t.Errorf("a horizontal slider measured %vx%v", hz.X, hz.Y)
	}
}

// A vertical slider is painted from the same parts, turned: it draws
// something, and it is not the horizontal picture.
func TestVerticalSliderPaintsTurned(t *testing.T) {
	v := NewVerticalSlider(0, 100, 25, nil)
	v.SetHost(&host{})
	v.Arrange(paintengine2d.XYWH(0, 0, 28, 200))
	img := immediatePaint(v, 28, 200)
	top := bandInk(img, 0, 100, 0, 28)
	bottom := bandInk(img, 100, 200, 0, 28)
	if top == 0 && bottom == 0 {
		t.Fatal("a vertical slider painted nothing")
	}
	if top == bottom {
		t.Errorf("a quarter-filled vertical slider is the same above and below: %d / %d", top, bottom)
	}
}

// A screen reader is told which way it runs.
func TestVerticalSliderSaysSoInTheTree(t *testing.T) {
	v := NewVerticalSlider(0, 100, 25, nil)
	v.SetHost(&host{})
	var n a11y.Node
	v.Describe(&n)
	if n.State&a11y.StateVertical == 0 || n.State&a11y.StateHorizontal != 0 {
		t.Errorf("a vertical slider is described %v", n.State)
	}
	h := NewSlider(0, 100, 25, nil)
	h.SetHost(&host{})
	var hn a11y.Node
	h.Describe(&hn)
	if hn.State&a11y.StateHorizontal == 0 {
		t.Error("a horizontal slider stopped saying so")
	}
}
