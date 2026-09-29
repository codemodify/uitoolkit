package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// scaleHost gives a component a look at a chosen display scale.
type scaleHost struct {
	look  style.LookAndFeel
	focus widget.Component
}

func (h *scaleHost) Invalidate(widget.Component, paintengine2d.Rect) {}
func (h *scaleHost) RequestFocus(c widget.Component)                 { h.focus = c }
func (h *scaleHost) Focus() widget.Component                         { return h.focus }
func (h *scaleHost) Scale() float32                                  { return style.LookScale(h.look) }
func (h *scaleHost) RequestLayout()                                  {}
func (h *scaleHost) Look() style.LookAndFeel                         { return h.look }

func atScale(t *testing.T, c widget.Component, scale float32) *scaleHost {
	t.Helper()
	h := &scaleHost{look: style.WithScale(style.DarkLook(), scale)}
	c.SetHost(h)
	return h
}

// 0.22.0 made FlexBox's gap and Grid's row and column gaps 1x design
// lengths and left Pad, Spacer and ProgressBar in device pixels — so a
// form at 2x had scaled gaps between unscaled pads, which is worse than
// either answer on its own.
func TestPadFollowsTheDisplayScale(t *testing.T) {
	loose := layout.Constraints{MaxW: -1, MaxH: -1}
	measure := func(scale float32, raw bool) paintengine2d.Point {
		p := NewPad(10, NewSpacerSize(0, 0))
		p.RawSpacing = raw
		atScale(t, p, scale)
		return p.Measure(loose)
	}

	one := measure(1, false)
	two := measure(2, false)
	if two.X != one.X*2 || two.Y != one.Y*2 {
		t.Errorf("a 10-pixel pad measured %v at 1x and %v at 2x, want twice", one, two)
	}
	// RawSpacing is the way back.
	if got := measure(2, true); got != one {
		t.Errorf("RawSpacing at 2x measured %v, want the 1x %v", got, one)
	}
	// At scale 1 nothing moved, which is why no pinned geometry changed.
	if got := measure(1, true); got != one {
		t.Errorf("RawSpacing at 1x measured %v, want %v", got, one)
	}
}

// The child sits inside the scaled insets, not the raw ones.
func TestPadArrangesInsideTheScaledInsets(t *testing.T) {
	child := NewSpacerSize(0, 0)
	p := NewPad(10, child)
	p.RawSpacing = true // so the child's own size is not scaled too
	atScale(t, p, 2)
	p.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	if got := child.Bounds().Min; got.X != 10 || got.Y != 10 {
		t.Errorf("RawSpacing child at %v, want (10,10)", got)
	}

	p2 := NewPad(10, NewSpacerSize(0, 0))
	atScale(t, p2, 2)
	p2.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	if got := p2.Children()[0].Bounds().Min; got.X != 20 || got.Y != 20 {
		t.Errorf("scaled child at %v, want (20,20)", got)
	}
}

// A fixed gap between two groups is a design length as well.
func TestSpacerFollowsTheDisplayScale(t *testing.T) {
	loose := layout.Constraints{MaxW: -1, MaxH: -1}
	s1 := NewSpacerSize(24, 8)
	atScale(t, s1, 1)
	s2 := NewSpacerSize(24, 8)
	atScale(t, s2, 2)

	a, b := s1.Measure(loose), s2.Measure(loose)
	if b.X != a.X*2 || b.Y != a.Y*2 {
		t.Errorf("a 24x8 spacer measured %v at 1x and %v at 2x", a, b)
	}

	raw := NewSpacerSize(24, 8)
	raw.RawSpacing = true
	atScale(t, raw, 2)
	if got := raw.Measure(loose); got != a {
		t.Errorf("RawSpacing at 2x measured %v, want the 1x %v", got, a)
	}
}

// A spacer with no preferred size is a grower, and still is.
func TestZeroSpacerIsStillAGrower(t *testing.T) {
	s := NewSpacer()
	atScale(t, s, 2)
	got := s.Measure(layout.Constraints{MinW: 5, MinH: 7, MaxW: -1, MaxH: -1})
	if got.X != 5 || got.Y != 7 {
		t.Errorf("an unsized spacer measured %v, want its minimum (5,7)", got)
	}
}

// A progress bar's natural length was 160 device pixels, so it was half
// as long as everything beside it on a 2x display.
func TestProgressBarLengthFollowsTheDisplayScale(t *testing.T) {
	loose := layout.Constraints{MaxW: -1, MaxH: -1}
	p1, p2 := &ProgressBar{}, &ProgressBar{}
	p1.Init(p1)
	p2.Init(p2)
	atScale(t, p1, 1)
	atScale(t, p2, 2)

	a, b := p1.Measure(loose), p2.Measure(loose)
	if b.X != a.X*2 {
		t.Errorf("a bar measured %v wide at 1x and %v at 2x, want twice", a.X, b.X)
	}
}
