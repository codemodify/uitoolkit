package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// ScrollView clips a larger child and paints a vertical scrollbar.
type ScrollView struct {
	widget.Base
	OffsetY float32
	child   widget.Component
	content paintengine2d.Point
	drag    bool
	hovered bool
}

func NewScrollView(child widget.Component) *ScrollView {
	s := &ScrollView{}
	s.Init(s)
	s.SetManagesChildren(true)
	if child != nil {
		s.SetChild(child)
	}
	return s
}

func (s *ScrollView) SetChild(c widget.Component) {
	s.ClearChildren()
	s.child = c
	if c != nil {
		s.Base.Add(c)
	}
}

func (s *ScrollView) Measure(c layout.Constraints) paintengine2d.Point {
	if s.child != nil {
		s.content = s.child.Measure(layout.Constraints{MaxW: c.MaxW, MaxH: -1})
	}
	w, h := s.content.X+12, float32(160)
	if c.HasMaxH() {
		h = c.MaxH
	}
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (s *ScrollView) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	if s.child == nil {
		return
	}
	bar := s.Look().Metrics().Scroll
	cw := r.Dx() - bar
	if cw < 0 {
		cw = 0
	}
	s.content = s.child.Measure(layout.Constraints{MinW: cw, MaxW: cw, MaxH: -1})
	if s.content.Y < r.Dy() {
		s.content.Y = r.Dy()
	}
	s.clamp()
	s.child.Arrange(paintengine2d.XYWH(0, -s.OffsetY, cw, s.content.Y))
}

func (s *ScrollView) maxOff() float32 {
	m := s.LocalBounds().Dy()
	if s.content.Y <= m {
		return 0
	}
	return s.content.Y - m
}

func (s *ScrollView) clamp() {
	if s.OffsetY < 0 {
		s.OffsetY = 0
	}
	if mx := s.maxOff(); s.OffsetY > mx {
		s.OffsetY = mx
	}
}

func (s *ScrollView) thumb() (track, thumb paintengine2d.Rect) {
	b := s.LocalBounds()
	w := s.Look().Metrics().Scroll
	track = paintengine2d.XYWH(b.Max.X-w-2, 4, w, b.Dy()-8)
	if s.content.Y <= b.Dy() {
		return track, paintengine2d.Rect{}
	}
	frac := b.Dy() / s.content.Y
	th := track.Dy() * frac
	if th < 24 {
		th = 24
	}
	ty := track.Min.Y
	if mx := s.maxOff(); mx > 0 {
		ty += (track.Dy() - th) * (s.OffsetY / mx)
	}
	thumb = paintengine2d.XYWH(track.Min.X, ty, track.Dx(), th)
	return
}

func (s *ScrollView) Paint(ctx *paintengine2d.Context) {
	b := s.LocalBounds()
	lk := s.Look()
	ctx.DrawRect(b, paintengine2d.Fill(lk.Palette().Background.WithAlpha(0.15)))
	ctx.Save()
	ctx.ClipRect(b)
	if s.child != nil {
		widget.PaintTree(s.child, ctx, nil)
	}
	ctx.Restore()
	track, thumb := s.thumb()
	if !thumb.Empty() {
		st := style.StateNone
		if s.hovered {
			st |= style.StateHovered
		}
		if s.drag {
			st |= style.StatePressed
		}
		lk.DrawScrollBar(ctx, track, thumb, st)
	}
}

func (s *ScrollView) HitTest(local paintengine2d.Point) widget.Component {
	if !s.Visible() || !s.LocalBounds().Contains(local) {
		return nil
	}
	_, thumb := s.thumb()
	if !thumb.Empty() && thumb.Contains(local) {
		return s
	}
	if s.child != nil {
		cb := s.child.Bounds()
		lp := paintengine2d.Pt(local.X-cb.Min.X, local.Y-cb.Min.Y)
		if hit := s.child.HitTest(lp); hit != nil {
			return hit
		}
	}
	return s
}

func (s *ScrollView) MouseWheel(e widget.MouseEvent) bool {
	s.OffsetY += e.Scroll.Y
	s.clamp()
	s.Arrange(s.Bounds())
	s.Invalidate()
	return true
}

func (s *ScrollView) MousePress(e widget.MouseEvent) bool {
	_, thumb := s.thumb()
	if thumb.Contains(e.Pos) {
		s.drag = true
		s.Invalidate()
		return true
	}
	return false
}

func (s *ScrollView) MouseMove(e widget.MouseEvent) bool {
	if !s.drag {
		return false
	}
	track, thumb := s.thumb()
	span := track.Dy() - thumb.Dy()
	if span <= 0 {
		return true
	}
	ty := e.Pos.Y - thumb.Dy()*0.5
	t := (ty - track.Min.Y) / span
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	s.OffsetY = t * s.maxOff()
	s.Arrange(s.Bounds())
	s.Invalidate()
	return true
}

func (s *ScrollView) MouseRelease(widget.MouseEvent) bool {
	s.drag = false
	s.Invalidate()
	return true
}
