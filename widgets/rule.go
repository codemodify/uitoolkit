package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Separator is a first-class horizontal (default) or vertical rule.
type Separator struct {
	widget.Base
	Vertical bool
}

// NewSeparator is a horizontal hairline.
func NewSeparator() *Separator {
	s := &Separator{}
	s.Init(s)
	return s
}

// NewVSeparator is a vertical hairline.
func NewVSeparator() *Separator {
	s := &Separator{Vertical: true}
	s.Init(s)
	return s
}

func (s *Separator) Measure(c layout.Constraints) paintengine2d.Point {
	// Intrinsic hairline only. Stretch is the parent's job at Arrange time.
	if s.Vertical {
		return c.Constrain(paintengine2d.Pt(8, 16))
	}
	return c.Constrain(paintengine2d.Pt(16, 8))
}

func (s *Separator) Arrange(r paintengine2d.Rect) { s.SetBounds(r) }

func (s *Separator) Paint(ctx *paintengine2d.Context) {
	s.Look().DrawSeparator(ctx, s.LocalBounds(), s.Vertical)
}

// Spacer is an empty box. Use FlexBox.AddFlex(NewSpacer(), 1) to soak leftover space.
type Spacer struct {
	widget.Base
	// RawSpacing takes [NewSpacerSize]'s width and height as device
	// pixels rather than 1x design lengths. It is the opt-out rather than
	// the default, for the same reason as [FlexBox.RawSpacing].
	RawSpacing bool
}

// NewSpacer is a zero-preferred gap (grows only when flexed or given a min).
func NewSpacer() *Spacer {
	s := &Spacer{}
	s.Init(s)
	return s
}

// NewSpacerSize is a fixed empty box.
func NewSpacerSize(w, h float32) *Spacer {
	s := NewSpacer()
	s.SetPreferred(w, h)
	return s
}

func (s *Spacer) Measure(c layout.Constraints) paintengine2d.Point {
	pref := s.Preferred()
	if pref.X == 0 && pref.Y == 0 {
		return c.Constrain(paintengine2d.Pt(c.MinW, c.MinH))
	}
	// A fixed gap is a 1x design length, like every other length in the
	// toolkit: a 24-pixel space between two groups was 24 pixels on a 2x
	// display too, so it read as half a gap beside controls that had
	// doubled.
	if !s.RawSpacing {
		if k := style.LookScale(s.Look()); k != 1 {
			pref = paintengine2d.Pt(pref.X*k, pref.Y*k)
		}
	}
	return c.Constrain(pref)
}

func (s *Spacer) Arrange(r paintengine2d.Rect) { s.SetBounds(r) }
