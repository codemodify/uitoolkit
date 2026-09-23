package widgets

import (
	"fmt"
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

// ---- the label, the reading, the step and the page ------------------------------

func keyed(s *Slider, k platform.Key, mods platform.Modifiers) {
	s.KeyPress(widget.KeyEvent{Key: k, Mods: mods})
}

// An arrow key moves by the app's Step and a page key by its Page, and a
// slider with neither moves the twentieth and the tenth it always did.
func TestSliderStepAndPageAreTheApps(t *testing.T) {
	s := NewSlider(-12, 12, 0, nil)
	s.SetHost(&host{})
	s.Step, s.Page = 1.5, 6
	keyed(s, platform.KeyRight, 0)
	if s.Value != 1.5 {
		t.Errorf("an arrow key moved to %v, want one step of 1.5", s.Value)
	}
	keyed(s, platform.KeyPageUp, 0)
	if s.Value != 7.5 {
		t.Errorf("a page key moved to %v, want a page of 6 on top of the step", s.Value)
	}
	// Shift is still the finer move: a quarter of a step.
	s.Value = 0
	keyed(s, platform.KeyRight, platform.ModShift)
	if s.Value != 0.375 {
		t.Errorf("Shift and an arrow moved to %v, want a quarter step", s.Value)
	}
	d := NewSlider(0, 100, 50, nil)
	d.SetHost(&host{})
	keyed(d, platform.KeyRight, 0)
	if d.Value != 55 {
		t.Errorf("a slider with no Step moved to %v, want a twentieth of the range", d.Value)
	}
	keyed(d, platform.KeyPageDown, 0)
	if d.Value != 45 {
		t.Errorf("a slider with no Page moved to %v, want a tenth of the range", d.Value)
	}
}

// Up is more on a slider standing on end, so Home is the top. The
// horizontal convention turned on its side would put Home at the bottom and
// read backwards to anyone using it.
func TestVerticalSliderHomeIsTheTop(t *testing.T) {
	v := NewVerticalSlider(0, 100, 50, nil)
	v.SetHost(&host{})
	keyed(v, platform.KeyHome, 0)
	if v.Value != 100 {
		t.Errorf("Home on a vertical slider went to %v, want the top", v.Value)
	}
	keyed(v, platform.KeyEnd, 0)
	if v.Value != 0 {
		t.Errorf("End on a vertical slider went to %v, want the bottom", v.Value)
	}
	h := NewSlider(0, 100, 50, nil)
	h.SetHost(&host{})
	keyed(h, platform.KeyHome, 0)
	if h.Value != 0 {
		t.Errorf("Home on a horizontal slider went to %v, want the start", h.Value)
	}
}

// The tooltip and the accessible value are the reading, not the number.
func TestSliderReadsItsValueTheAppsWay(t *testing.T) {
	s := NewSlider(-12, 12, 4.5, nil)
	s.SetHost(&host{})
	if got := s.Tooltip(); got != "" {
		t.Errorf("a slider with nothing to say has the tooltip %q", got)
	}
	s.Label = "Preamp"
	s.Format = func(v float32) string { return fmt.Sprintf("%+.1f dB", v) }
	if got := s.Tooltip(); got != "Preamp  +4.5 dB" {
		t.Errorf("the tooltip is %q", got)
	}
	var n a11y.Node
	s.Describe(&n)
	if n.Name != "Preamp" || n.Value != "+4.5 dB" {
		t.Errorf("a screen reader hears %q = %q", n.Name, n.Value)
	}
	s.Tip = "Output level"
	if got := s.Tooltip(); got != "Output level" {
		t.Errorf("an explicit tip became %q", got)
	}
}

// The step a reader is offered is the step it gets.
func TestScreenReaderStepsTheSlider(t *testing.T) {
	inputs := 0
	s := NewSlider(0, 100, 50, nil)
	s.SetHost(&host{})
	s.Step = 5
	s.OnInput = func(float32) { inputs++ }
	var n a11y.Node
	s.Describe(&n)
	if !n.Actions.Has(a11y.ActionIncrement) || n.Step != 5 {
		t.Fatalf("the slider offers %v with a step of %v", n.Actions, n.Step)
	}
	if !s.AccessibleAction(0, a11y.ActionIncrement) || s.Value != 55 {
		t.Errorf("an increment left the slider at %v", s.Value)
	}
	if !s.AccessibleAction(0, a11y.ActionDecrement) || s.Value != 50 {
		t.Errorf("a decrement left the slider at %v", s.Value)
	}
	// A reader's step is the user's own, so a view bound to a model hears
	// it (widgets/oninput.go).
	if inputs != 2 {
		t.Errorf("OnInput fired %d times for two accessibility steps", inputs)
	}
}

// The spin button had the same gap: it advertised the step and did not move.
func TestScreenReaderStepsTheNumberField(t *testing.T) {
	inputs := 0
	f := NewNumberField(0, 10, 5, 1, nil)
	f.SetHost(&host{})
	f.OnInput = func(float64) { inputs++ }
	if !f.AccessibleAction(0, a11y.ActionIncrement) || f.Value != 6 {
		t.Errorf("an increment left the spin button at %v", f.Value)
	}
	if !f.AccessibleAction(0, a11y.ActionDecrement) || f.Value != 5 {
		t.Errorf("a decrement left the spin button at %v", f.Value)
	}
	if inputs != 2 {
		t.Errorf("OnInput fired %d times for two accessibility steps", inputs)
	}
}
