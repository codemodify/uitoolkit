package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// StatusBar is a bottom chrome strip with one or more text sections.
type StatusBar struct {
	widget.Base
	parts []string
}

// NewStatusBar constructs a status bar from left-to-right sections.
func NewStatusBar(parts ...string) *StatusBar {
	s := &StatusBar{parts: parts}
	s.Init(s)
	return s
}

// Parts returns the current sections.
func (s *StatusBar) Parts() []string { return s.parts }

// Set replaces section i.
func (s *StatusBar) Set(i int, text string) {
	if i < 0 {
		return
	}
	for len(s.parts) <= i {
		s.parts = append(s.parts, "")
	}
	if s.parts[i] == text {
		return
	}
	s.parts[i] = text
	s.Invalidate()
}

// SetParts replaces every section.
func (s *StatusBar) SetParts(parts ...string) {
	s.parts = parts
	s.Invalidate()
}

func (s *StatusBar) barH() float32 {
	h := s.Look().Metrics().StatusBarH
	if h <= 0 {
		h = 26
	}
	return h
}

func (s *StatusBar) Measure(c layout.Constraints) paintengine2d.Point {
	w := style.Dip(s.Look(), 200)
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, s.barH()))
}

func (s *StatusBar) Arrange(r paintengine2d.Rect) { s.SetBounds(r) }

func (s *StatusBar) Paint(ctx *paintengine2d.Context) {
	s.Look().DrawStatusBar(ctx, s.LocalBounds(), s.parts)
}
