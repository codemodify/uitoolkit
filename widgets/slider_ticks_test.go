package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Pressing the thumb takes hold of it where it was pressed (it does not
// jump); pressing the track moves it there. Ticks add their rows.
func TestSliderGrabAndTicks(t *testing.T) {
	s := NewSlider(0, 100, 50, nil)
	s.SetLook(style.DarkLook())
	s.SetHost(&host{})
	plain := s.Measure(layout.Unbounded()).Y
	s.Ticks, s.TickInterval = TicksBoth, 25
	if got := s.Measure(layout.Unbounded()).Y; got <= plain {
		t.Fatalf("ticks on both sides: %v, plain %v", got, plain)
	}
	s.Arrange(paintengine2d.XYWH(0, 0, 200, s.Measure(layout.Unbounded()).Y))
	x0, x1 := style.SliderTravelOf(s.Look(), s.track())
	if xs := s.tickXs(x0, x1); len(xs) != 5 || xs[0] != x0 || xs[4] != x1 {
		t.Fatalf("ticks %v over %v..%v", xs, x0, x1)
	}
	thumb := x0 + (x1-x0)*0.5
	// Off-centre on the thumb: no jump.
	s.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(thumb+3, 10)})
	if s.Value != 50 {
		t.Fatalf("pressing the thumb moved it to %v", s.Value)
	}
	s.MouseMove(widget.MouseEvent{Pos: paintengine2d.Pt(thumb+3+(x1-x0)*0.1, 10)})
	if s.Value < 59 || s.Value > 61 {
		t.Fatalf("dragging a tenth of the travel: %v", s.Value)
	}
	s.MouseRelease(widget.MouseEvent{})
	// On the track: jump.
	s.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(x0, 10)})
	if s.Value != 0 {
		t.Fatalf("pressing the track's start: %v", s.Value)
	}
	img := paintengine2d.NewImage(220, 60)
	s.Paint(paintengine2d.NewContext(img))
}

// Progress text sits in a bar tall enough for it, beside a thin one.
func TestProgressText(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, pack := range []string{"win95", "fluent"} {
		p, _ := style.LoadTheme(pack)
		bar := NewProgressBar(0.42)
		bar.SetLook(p.Look())
		bar.SetHost(&host{})
		plain := bar.Measure(layout.Unbounded())
		bar.ShowText = true
		if bar.text() != "42%" {
			t.Fatalf("text %q", bar.text())
		}
		got := bar.Measure(layout.Unbounded())
		if bar.textInside() != (got.X == plain.X) {
			t.Fatalf("%s: inside %v, width %v vs %v", pack, bar.textInside(), got.X, plain.X)
		}
		bar.Arrange(paintengine2d.XYWH(0, 0, got.X, got.Y))
		bar.Paint(paintengine2d.NewContext(paintengine2d.NewImage(int(got.X)+4, int(got.Y)+4)))
	}
}
