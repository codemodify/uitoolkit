package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Splitter is two panes with a draggable divider. Ratio is the first pane share.
type Splitter struct {
	widget.Base
	Vertical bool
	Ratio    float32
	A, B     widget.Component
	drag     bool
	hovered  bool
}

func NewSplitter(vertical bool, a, b widget.Component) *Splitter {
	s := &Splitter{Vertical: vertical, Ratio: 0.4, A: a, B: b}
	s.Init(s)
	if a != nil {
		s.Base.Add(a)
	}
	if b != nil {
		s.Base.Add(b)
	}
	return s
}

func (s *Splitter) Measure(c layout.Constraints) paintengine2d.Point {
	w, h := float32(320), float32(200)
	if c.HasMaxW() {
		w = c.MaxW
	}
	if c.HasMaxH() {
		h = c.MaxH
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (s *Splitter) bar() float32 { return s.Look().Metrics().Splitter }

func (s *Splitter) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	if s.Ratio < 0.08 {
		s.Ratio = 0.08
	}
	if s.Ratio > 0.92 {
		s.Ratio = 0.92
	}
	bar := s.bar()
	if s.Vertical {
		aw := (r.Dx() - bar) * s.Ratio
		if s.A != nil {
			s.A.Arrange(paintengine2d.XYWH(0, 0, aw, r.Dy()))
		}
		if s.B != nil {
			s.B.Arrange(paintengine2d.XYWH(aw+bar, 0, r.Dx()-aw-bar, r.Dy()))
		}
		return
	}
	ah := (r.Dy() - bar) * s.Ratio
	if s.A != nil {
		s.A.Arrange(paintengine2d.XYWH(0, 0, r.Dx(), ah))
	}
	if s.B != nil {
		s.B.Arrange(paintengine2d.XYWH(0, ah+bar, r.Dx(), r.Dy()-ah-bar))
	}
}

func (s *Splitter) divider() paintengine2d.Rect {
	b := s.LocalBounds()
	bar := s.bar()
	if s.Vertical {
		x := (b.Dx() - bar) * s.Ratio
		return paintengine2d.XYWH(x, 0, bar, b.Dy())
	}
	y := (b.Dy() - bar) * s.Ratio
	return paintengine2d.XYWH(0, y, b.Dx(), bar)
}

func (s *Splitter) Paint(ctx *paintengine2d.Context) {
	st := s.State()
	if s.hovered || s.drag {
		st |= style.StateHovered
	}
	if s.drag {
		st |= style.StatePressed
	}
	s.Look().DrawSplitter(ctx, s.divider(), s.Vertical, st)
}

func (s *Splitter) MouseEnter() { s.hovered = true; s.Base.MouseEnter() }
func (s *Splitter) MouseExit()  { s.hovered = false; s.Base.MouseExit() }

func (s *Splitter) MousePress(e widget.MouseEvent) bool {
	if s.divider().Contains(e.Pos) {
		s.drag = true
		s.Invalidate()
		return true
	}
	return false
}

func (s *Splitter) MouseMove(e widget.MouseEvent) bool {
	if !s.drag {
		return false
	}
	b := s.LocalBounds()
	if s.Vertical {
		s.Ratio = e.Pos.X / b.Dx()
	} else {
		s.Ratio = e.Pos.Y / b.Dy()
	}
	s.Arrange(s.Bounds())
	s.Invalidate()
	return true
}

func (s *Splitter) MouseRelease(widget.MouseEvent) bool {
	s.drag = false
	s.Invalidate()
	return true
}
