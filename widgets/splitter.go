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
	paneA    *paintengine2d.GroupNode
	paneB    *paintengine2d.GroupNode
	bakedA   paintengine2d.Rect
	bakedB   paintengine2d.Rect
}

var _ widget.SceneBaker = (*Splitter)(nil)

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

func (s *Splitter) paneIDs() (idA, idB uint64) {
	return s.ID() ^ (1 << 32), s.ID() ^ (2 << 32)
}

// HasBakedPanes reports whether both occupied panes have a BakeGroup layer
// (splitter drag blits these instead of re-walking children).
func (s *Splitter) HasBakedPanes() bool { return s.canBlitPanes() }

// BakedPaneLayers is the last BakeGroup pixmap for each pane (tests).
func (s *Splitter) BakedPaneLayers() (a, b *paintengine2d.Image) {
	if s.paneA != nil {
		a = s.paneA.Layer
	}
	if s.paneB != nil {
		b = s.paneB.Layer
	}
	return a, b
}

func (s *Splitter) canBlitPanes() bool {
	if s.A != nil && (s.paneA == nil || !s.paneA.HasLayer()) {
		return false
	}
	if s.B != nil && (s.paneB == nil || !s.paneB.HasLayer()) {
		return false
	}
	return s.paneA != nil || s.paneB != nil
}

func (s *Splitter) applyPaneXforms() {
	a, _, b := s.panes(s.LocalBounds())
	if s.paneA != nil {
		s.paneA.Xform = paintengine2d.Translation(a.Min.X-s.bakedA.Min.X, a.Min.Y-s.bakedA.Min.Y)
	}
	if s.paneB != nil {
		s.paneB.Xform = paintengine2d.Translation(b.Min.X-s.bakedB.Min.X, b.Min.Y-s.bakedB.Min.Y)
	}
}

func (s *Splitter) dropBakedPanes() {
	if s.paneA != nil {
		s.paneA.InvalidateLayer()
	}
	if s.paneB != nil {
		s.paneB.InvalidateLayer()
	}
	s.paneA, s.paneB = nil, nil
	s.bakedA, s.bakedB = paintengine2d.Rect{}, paintengine2d.Rect{}
}

func (s *Splitter) recordPane(rec *paintengine2d.Recorder, ctx *paintengine2d.Context, cache *widget.SceneCache, child widget.Component, pane paintengine2d.Rect, id uint64) *paintengine2d.GroupNode {
	if rec == nil || child == nil || !child.Visible() || pane.Empty() {
		return nil
	}
	g := rec.BeginGroup(id, paintengine2d.Identity())
	ctx.Save()
	ctx.ClipRect(pane)
	widget.RecordSubtree(child, rec, ctx, cache)
	ctx.Restore()
	rec.EndGroup()
	return g
}

// RecordBaked implements widget.SceneBaker. Drag frames Attach baked
// pane groups and only change Xform; the first drag frame BakeGroups.
func (s *Splitter) RecordBaked(rec *paintengine2d.Recorder, ctx *paintengine2d.Context, cache *widget.SceneCache) {
	if rec == nil || ctx == nil {
		return
	}
	a, _, b := s.panes(s.LocalBounds())
	idA, idB := s.paneIDs()
	if s.drag && s.canBlitPanes() {
		s.applyPaneXforms()
		if s.paneA != nil {
			rec.Attach(s.paneA)
		}
		if s.paneB != nil {
			rec.Attach(s.paneB)
		}
		return
	}
	s.paneA = s.recordPane(rec, ctx, cache, s.A, a, idA)
	s.paneB = s.recordPane(rec, ctx, cache, s.B, b, idB)
	s.bakedA, s.bakedB = a, b
	if s.drag {
		if s.paneA != nil {
			paintengine2d.BakeGroup(s.paneA)
		}
		if s.paneB != nil {
			paintengine2d.BakeGroup(s.paneB)
		}
	}
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
	if !widget.Recording(ctx) {
		a, _, b := s.panes(s.LocalBounds())
		clip := ctx.LocalClipBounds()
		if clip.Empty() || clip.Overlaps(a) {
			s.paintPane(ctx, s.A, a)
		}
		if clip.Empty() || clip.Overlaps(b) {
			s.paintPane(ctx, s.B, b)
		}
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
		s.dropBakedPanes()
		s.applyCursor(e.Pos)
		s.invalidateSash()
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
	oldA, _, _ := s.panes(s.LocalBounds())
	box := s.LocalBounds()
	bar := s.bar()
	if s.Vertical {
		usable := box.Dx() - bar
		if usable > 0 {
			s.Ratio = (e.Pos.X - bar*0.5) / usable
		}
	} else {
		usable := box.Dy() - bar
		if usable > 0 {
			s.Ratio = (e.Pos.Y - bar*0.5) / usable
		}
	}
	s.clampRatio()
	// Arrange this splitter only. RequestLayout full-Measures the window
	// and full-invalidates every pixel of the drag. Baked panes keep
	// their layers; only Xform and the sash/sliver are dirtied.
	s.Arrange(s.Bounds())
	s.applyPaneXforms()
	s.applyCursor(e.Pos)
	s.invalidateSash()
	a, _, _ := s.panes(s.LocalBounds())
	if s.Vertical {
		x0, x1 := oldA.Max.X, a.Max.X
		if x1 < x0 {
			x0, x1 = x1, x0
		}
		if x1 > x0 {
			s.InvalidateRect(paintengine2d.XYWH(x0, 0, x1-x0, box.Dy()).Inset(-1))
		}
	} else {
		y0, y1 := oldA.Max.Y, a.Max.Y
		if y1 < y0 {
			y0, y1 = y1, y0
		}
		if y1 > y0 {
			s.InvalidateRect(paintengine2d.XYWH(0, y0, box.Dx(), y1-y0).Inset(-1))
		}
	}
	return true
}

func (s *Splitter) MouseRelease(e widget.MouseEvent) bool {
	s.drag = false
	s.dropBakedPanes()
	s.applyCursor(e.Pos)
	s.Invalidate()
	return true
}
