package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Splitter is two panes with a draggable divider. Ratio is the first pane share
// of the space beside the sash. Arranged panes are exclusive; each is clipped
// to its rect on paint and hit-test so children cannot overlap the sibling.
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
	s.SetManagesChildren(true)
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

func (s *Splitter) clampRatio() {
	if s.Ratio < 0.08 {
		s.Ratio = 0.08
	}
	if s.Ratio > 0.92 {
		s.Ratio = 0.92
	}
}

// panes returns exclusive A / sash / B rects in local coordinates for box.
func (s *Splitter) panes(box paintengine2d.Rect) (a, div, b paintengine2d.Rect) {
	s.clampRatio()
	bar := s.bar()
	if bar < 1 {
		bar = 1
	}
	w, h := box.Dx(), box.Dy()
	if s.Vertical {
		if w < bar {
			bar = w
		}
		avail := w - bar
		if avail < 0 {
			avail = 0
		}
		aw := avail * s.Ratio
		if aw < 0 {
			aw = 0
		}
		if aw > avail {
			aw = avail
		}
		bw := avail - aw
		a = paintengine2d.XYWH(0, 0, aw, h)
		div = paintengine2d.XYWH(aw, 0, bar, h)
		b = paintengine2d.XYWH(aw+bar, 0, bw, h)
		return
	}
	if h < bar {
		bar = h
	}
	avail := h - bar
	if avail < 0 {
		avail = 0
	}
	ah := avail * s.Ratio
	if ah < 0 {
		ah = 0
	}
	if ah > avail {
		ah = avail
	}
	bh := avail - ah
	a = paintengine2d.XYWH(0, 0, w, ah)
	div = paintengine2d.XYWH(0, ah, w, bar)
	b = paintengine2d.XYWH(0, ah+bar, w, bh)
	return
}

// PaneA is the first pane in local coordinates (after the last Arrange).
func (s *Splitter) PaneA() paintengine2d.Rect {
	a, _, _ := s.panes(s.LocalBounds())
	return a
}

// PaneB is the second pane in local coordinates (after the last Arrange).
func (s *Splitter) PaneB() paintengine2d.Rect {
	_, _, b := s.panes(s.LocalBounds())
	return b
}

func (s *Splitter) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	a, _, b := s.panes(s.LocalBounds())
	if s.A != nil {
		s.A.Arrange(a)
	}
	if s.B != nil {
		s.B.Arrange(b)
	}
}

func (s *Splitter) divider() paintengine2d.Rect {
	_, d, _ := s.panes(s.LocalBounds())
	return d
}

func (s *Splitter) paintPane(ctx *paintengine2d.Context, child widget.Component, pane paintengine2d.Rect) {
	if ctx == nil || child == nil || !child.Visible() || pane.Empty() {
		return
	}
	ctx.Save()
	ctx.ClipRect(pane)
	if !ctx.ClipEmpty() {
		widget.PaintTree(child, ctx, nil)
	}
	ctx.Restore()
}

func (s *Splitter) Paint(ctx *paintengine2d.Context) {
	a, _, b := s.panes(s.LocalBounds())
	clip := ctx.LocalClipBounds()
	if clip.Empty() || clip.Overlaps(a) {
		s.paintPane(ctx, s.A, a)
	}
	if clip.Empty() || clip.Overlaps(b) {
		s.paintPane(ctx, s.B, b)
	}
	st := s.State()
	if s.hovered || s.drag {
		st |= style.StateHovered
	}
	if s.drag {
		st |= style.StatePressed
	}
	s.Look().DrawSplitter(ctx, s.divider(), s.Vertical, st)
}

func (s *Splitter) HitTest(local paintengine2d.Point) widget.Component {
	if !s.Visible() {
		return nil
	}
	lb := s.LocalBounds()
	if lb.Empty() || !lb.Contains(local) {
		return nil
	}
	if s.divider().Contains(local) {
		return s
	}
	a, _, b := s.panes(lb)
	try := func(ch widget.Component, pane paintengine2d.Rect) widget.Component {
		if ch == nil || !ch.Visible() || pane.Empty() || !pane.Contains(local) {
			return nil
		}
		cb := ch.Bounds()
		hitPane := pane
		if !cb.Empty() {
			hitPane = pane.Intersect(cb)
		}
		if hitPane.Empty() || !hitPane.Contains(local) {
			return nil
		}
		lp := paintengine2d.Pt(local.X-cb.Min.X, local.Y-cb.Min.Y)
		return ch.HitTest(lp)
	}
	if hit := try(s.B, b); hit != nil {
		return hit
	}
	if hit := try(s.A, a); hit != nil {
		return hit
	}
	return s
}

func (s *Splitter) resizeCursor() platform.Cursor {
	if s.Vertical {
		return platform.CursorColResize
	}
	return platform.CursorRowResize
}

// CursorAt is col-resize / row-resize on the sash (and while dragging).
func (s *Splitter) CursorAt(local paintengine2d.Point) platform.Cursor {
	if s.drag || s.divider().Contains(local) {
		return s.resizeCursor()
	}
	return platform.CursorDefault
}

func (s *Splitter) applyCursor(local paintengine2d.Point) {
	widget.ApplyCursor(s.Host(), s.CursorAt(local))
}

func (s *Splitter) invalidateSash() {
	s.InvalidateRect(s.divider().Inset(-2))
}

func (s *Splitter) MouseEnter() {
	s.hovered = true
	s.SetHovered(true)
	s.invalidateSash()
}

func (s *Splitter) MouseExit() {
	s.hovered = false
	s.SetHovered(false)
	if !s.drag {
		widget.ApplyCursor(s.Host(), platform.CursorDefault)
	}
	s.invalidateSash()
}

func (s *Splitter) MousePress(e widget.MouseEvent) bool {
	if s.divider().Contains(e.Pos) {
		s.drag = true
		s.applyCursor(e.Pos)
		s.Invalidate()
		return true
	}
	return false
}

func (s *Splitter) MouseMove(e widget.MouseEvent) bool {
	over := s.divider().Contains(e.Pos)
	if over != s.hovered {
		s.hovered = over
		s.invalidateSash()
	}
	if !s.drag {
		s.applyCursor(e.Pos)
		return over
	}
	b := s.LocalBounds()
	bar := s.bar()
	if s.Vertical {
		usable := b.Dx() - bar
		if usable > 0 {
			s.Ratio = (e.Pos.X - bar*0.5) / usable
		}
	} else {
		usable := b.Dy() - bar
		if usable > 0 {
			s.Ratio = (e.Pos.Y - bar*0.5) / usable
		}
	}
	s.clampRatio()
	// Arrange this splitter only. RequestLayout full-Measures the window
	// and full-invalidates every pixel of the drag.
	s.Arrange(s.Bounds())
	s.applyCursor(e.Pos)
	s.Invalidate()
	return true
}

func (s *Splitter) MouseRelease(e widget.MouseEvent) bool {
	s.drag = false
	s.applyCursor(e.Pos)
	s.Invalidate()
	return true
}
