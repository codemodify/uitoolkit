package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
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
	grab    float32
	overBar bool
}

func NewScrollView(child widget.Component) *ScrollView {
	s := &ScrollView{}
	s.Init(s)
	s.SetManagesChildren(true)
	s.SetWantsFocus(true)
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

// MaxOffset is the largest legal OffsetY.
func (s *ScrollView) MaxOffset() float32 { return s.maxOff() }

// ScrollTo sets OffsetY (clamped) and relayouts the child.
func (s *ScrollView) ScrollTo(y float32) {
	s.OffsetY = y
	s.clamp()
	if !s.Bounds().Empty() {
		s.Arrange(s.Bounds())
	}
	s.Invalidate()
}

// ScrollBy adds dy pixels to the offset.
func (s *ScrollView) ScrollBy(dy float32) { s.ScrollTo(s.OffsetY + dy) }

func (s *ScrollView) Measure(c layout.Constraints) paintengine2d.Point {
	bar, gap := s.barGap()
	if s.child != nil {
		cw := c.MaxW
		if c.HasMaxW() {
			cw = c.MaxW - bar - gap
			if cw < 0 {
				cw = 0
			}
		}
		s.content = s.child.Measure(layout.Constraints{MaxW: cw, MaxH: -1})
	}
	w, h := s.content.X+bar+gap, float32(160)
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
	bar, gap := s.barGap()
	cw := r.Dx() - bar - gap
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

func (s *ScrollView) barGap() (bar, gap float32) {
	bar = s.Look().Metrics().Scroll
	return bar, 2
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

func (s *ScrollView) lineStep() float32 {
	h := s.Look().Font().Height()
	if h < 12 {
		h = 18
	}
	return h + 4
}

func (s *ScrollView) pageStep() float32 {
	h := s.LocalBounds().Dy() * 0.9
	if h < 24 {
		h = 24
	}
	return h
}

func (s *ScrollView) thumb() (track, thumb paintengine2d.Rect) {
	b := s.LocalBounds()
	bar, gap := s.barGap()
	track = paintengine2d.XYWH(b.Max.X-bar-gap, 4, bar, b.Dy()-8)
	if s.content.Y <= b.Dy() {
		return track, paintengine2d.Rect{}
	}
	frac := b.Dy() / s.content.Y
	th := track.Dy() * frac
	if th < 24 {
		th = 24
	}
	if th > track.Dy() {
		th = track.Dy()
	}
	ty := track.Min.Y
	if mx := s.maxOff(); mx > 0 && track.Dy() > th {
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
		st := s.State()
		if s.overBar {
			st |= style.StateHovered
		}
		if s.drag {
			st |= style.StatePressed
		}
		lk.DrawScrollBar(ctx, track, thumb, st)
	}
	if s.Focused() {
		lk.DrawFocusRing(ctx, b.Inset(-2))
	}
}

func (s *ScrollView) HitTest(local paintengine2d.Point) widget.Component {
	if !s.Visible() || !s.LocalBounds().Contains(local) {
		return nil
	}
	track, _ := s.thumb()
	if !track.Empty() && track.Contains(local) {
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
	dy := e.Scroll.Y
	if dy == 0 && e.Scroll.X == 0 {
		return false
	}
	// Notch-sized deltas (typical ±1) become a few lines.
	if dy > -8 && dy < 8 && dy != 0 {
		dy *= s.lineStep() * 3
	}
	s.ScrollBy(dy)
	return true
}

func (s *ScrollView) MousePress(e widget.MouseEvent) bool {
	if !s.Enabled() {
		return false
	}
	track, thumb := s.thumb()
	if thumb.Contains(e.Pos) {
		s.RequestFocus()
		s.drag = true
		s.grab = e.Pos.Y - thumb.Min.Y
		s.Invalidate()
		return true
	}
	if track.Contains(e.Pos) {
		s.RequestFocus()
		page := s.pageStep()
		if e.Pos.Y < thumb.Min.Y {
			s.ScrollBy(-page)
		} else {
			s.ScrollBy(page)
		}
		return true
	}
	return false
}

func (s *ScrollView) MouseMove(e widget.MouseEvent) bool {
	track, thumb := s.thumb()
	over := track.Contains(e.Pos)
	if over != s.overBar {
		s.overBar = over
		s.Invalidate()
	}
	if !s.drag {
		return over
	}
	span := track.Dy() - thumb.Dy()
	if span <= 0 {
		return true
	}
	ty := e.Pos.Y - s.grab
	t := (ty - track.Min.Y) / span
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	s.ScrollTo(t * s.maxOff())
	return true
}

func (s *ScrollView) MouseRelease(widget.MouseEvent) bool {
	if !s.drag {
		return false
	}
	s.drag = false
	s.Invalidate()
	return true
}

func (s *ScrollView) MouseExit() {
	s.overBar = false
	s.Base.MouseExit()
}

func (s *ScrollView) KeyPress(e widget.KeyEvent) bool {
	if !s.Enabled() {
		return false
	}
	switch e.Key {
	case platform.KeyDown:
		s.ScrollBy(s.lineStep())
		return true
	case platform.KeyUp:
		s.ScrollBy(-s.lineStep())
		return true
	case platform.KeyPageDown:
		s.ScrollBy(s.pageStep())
		return true
	case platform.KeyPageUp:
		s.ScrollBy(-s.pageStep())
		return true
	case platform.KeyHome:
		s.ScrollTo(0)
		return true
	case platform.KeyEnd:
		s.ScrollTo(s.maxOff())
		return true
	}
	return false
}
