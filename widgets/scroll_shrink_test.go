package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A scroll view takes all the height it is offered, which is the whole
// point of one — until ShrinkToContent, which measures to what is inside.
func TestScrollViewShrinksToContent(t *testing.T) {
	s := NewScrollView(newSized(200, 90))
	s.SetLook(style.DarkLook())
	s.SetHost(&host{})
	if got := s.Measure(layout.Loose(300, 500)).Y; got != 500 {
		t.Fatalf("by default a scroll view asked for %v of the 500 offered", got)
	}
	s.ShrinkToContent = true
	if got := s.Measure(layout.Loose(300, 500)).Y; got != 90 {
		t.Fatalf("shrinking, it asked for %v, want the content's 90", got)
	}
}

// MaxHeight is the cap the header case needs: as tall as what is in it,
// up to N, scrolling past that.
func TestScrollViewMaxHeightCaps(t *testing.T) {
	s := NewScrollView(newSized(200, 400))
	s.SetLook(style.DarkLook())
	s.SetHost(&host{})
	s.ShrinkToContent = true
	s.MaxHeight = 120
	got := s.Measure(layout.Loose(300, 500)).Y
	if want := style.Dip(s.Look(), 120); got != want {
		t.Fatalf("capped height %v, want %v", got, want)
	}
	// And it scrolls past the cap: the content is taller than the box.
	s.Arrange(paintengine2d.XYWH(0, 0, 300, got))
	if s.MaxOffset() <= 0 {
		t.Fatalf("nothing to scroll: content %v in %v", s.ContentHeight(), got)
	}
	// Short content stays short under the same cap.
	s.SetChild(newSized(200, 40))
	if h := s.Measure(layout.Loose(300, 500)).Y; h != 40 {
		t.Fatalf("short content asked for %v, want 40", h)
	}
}

// The cap is this view's own; a parent offering less still wins, and
// MinHeight keeps an empty one from collapsing.
func TestScrollViewMinAndOuterBoxWin(t *testing.T) {
	s := NewScrollView(newSized(200, 400))
	s.SetLook(style.DarkLook())
	s.SetHost(&host{})
	s.ShrinkToContent = true
	s.MaxHeight = 300
	if got := s.Measure(layout.Loose(300, 80)).Y; got != 80 {
		t.Fatalf("asked for %v inside a box of 80", got)
	}
	s.SetChild(newSized(200, 0))
	s.MinHeight = 60
	if got, want := s.Measure(layout.Loose(300, 500)).Y, style.Dip(s.Look(), 60); got != want {
		t.Fatalf("empty and floored: %v, want %v", got, want)
	}
}

// MaxHeight without ShrinkToContent is a plain cap on a view that would
// otherwise fill its box.
func TestScrollViewCapWithoutShrinking(t *testing.T) {
	s := NewScrollView(newSized(200, 1000))
	s.SetLook(style.DarkLook())
	s.SetHost(&host{})
	s.MaxHeight = 150
	if got, want := s.Measure(layout.Loose(300, 900)).Y, style.Dip(s.Look(), 150); got != want {
		t.Fatalf("capped %v, want %v", got, want)
	}
}
