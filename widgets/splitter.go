package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"math"
)

// SplitAxis says how the two panes sit. It describes the panes rather than
// the divider between them: the old boolean named the divider, so
// NewSplitter(true, …) produced a side-by-side split, and every reader of
// the call had to stop and work that out. Two applications made the same
// mistake, which is how this came to be a named type.
type SplitAxis uint8

const (
	// SplitColumns puts the panes side by side, with a divider that runs
	// up and down between them.
	SplitColumns SplitAxis = iota
	// SplitRows stacks the panes, one above the other.
	SplitRows
)

func (a SplitAxis) String() string {
	if a == SplitRows {
		return "rows"
	}
	return "columns"
}

// Splitter is two panes with a draggable divider. Ratio is the first pane share
// of the space beside the sash. Arranged panes are exclusive; each is clipped
// to its rect on paint and hit-test so children cannot overlap the sibling.
type Splitter struct {
	widget.Base
	Axis  SplitAxis
	Ratio float32
	A, B  widget.Component
	// MinA and MinB are the least each pane may be given, as 1x design
	// lengths. Zero takes the pane's own minimum — what
	// [widget.MinWidthOf] says it needs — which is what a splitter
	// should do and what Qt's QSplitter does: a pane is never dragged
	// narrower than the things inside it, so a sidebar cannot be
	// squeezed until its buttons run off the edge.
	//
	// Set one only to override that: a pane that may be collapsed to
	// nothing takes 1, a pane with a floor of its own takes that.
	// AllowCollapse lets a drag go all the way to zero from either.
	MinA, MinB float32
	// AllowCollapse lets a pane be dragged to nothing, as a collapsible
	// sidebar is. Without it a drag stops at the minimum.
	AllowCollapse bool
	// OnRatioChanged is called while the divider is dragged, with the
	// first pane's width in device pixels and the new ratio.
	//
	// It fires on the **drag**, not on the layout, and that is the whole
	// point: anything that has to line up with the divider — a header
	// bar's start section, a footer, a second tool bar — needs the
	// number *before* the next layout begins. A watcher that measured
	// the pane after a layout would always be a frame behind, and on a
	// drag that is a frame the user can see.
	OnRatioChanged func(paneA, ratio float32)
	drag           bool
	hovered        bool
}

// sideBySide is what the look's painters and the hit tests want: whether the
// divider itself runs up and down.
func (s *Splitter) sideBySide() bool { return s.Axis == SplitColumns }

func NewSplitter(axis SplitAxis, a, b widget.Component) *Splitter {
	s := &Splitter{Axis: axis, Ratio: 0.4, A: a, B: b}
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
	// A pane that may be collapsed may be collapsed all the way; one
	// that may not keeps a sliver, so a divider dragged to the edge can
	// always be found and dragged back.
	lo, hi := float32(0.08), float32(0.92)
	if s.AllowCollapse {
		lo, hi = 0, 1
	}
	if s.Ratio < lo {
		s.Ratio = lo
	}
	if s.Ratio > hi {
		s.Ratio = hi
	}
}

// minFor is the least a pane may be given along the split axis: what the
// caller set, or what the pane says it needs.
//
// A splitter that divides by ratio alone lets a pane be dragged — or
// opened — narrower than the things inside it, and they run off the
// edge. Qt's QSplitter takes the children's minimum sizes; so does this,
// now that a component can be asked for one.
//
// Only the horizontal split asks: MinWidthOf is a width, and nothing in
// the toolkit answers the same question about height. A row-split pane
// therefore keeps whatever the caller set and nothing more.
func (s *Splitter) minFor(c widget.Component, set float32) float32 {
	if set > 0 {
		return style.Dip(s.Look(), set)
	}
	if s.AllowCollapse || !s.sideBySide() {
		return 0
	}
	return widget.MinWidthOf(c)
}

// MinWidth is how narrow the splitter itself may be: side by side, both
// panes' floors and the bar between them; split in rows, the wider of the
// two, because then they share the whole width rather than dividing it.
//
// Without this a window sized from its content got the probe's guess for
// anything made of splitters, which is what an application assembling its
// layout out of them is most likely to ask about. A collapsible pane
// contributes nothing, since it is allowed to close.
func (s *Splitter) MinWidth() float32 {
	a, b := s.minFor(s.A, s.MinA), s.minFor(s.B, s.MinB)
	if !s.sideBySide() {
		// minFor answers 0 for a row split (MinWidthOf is a width, and the
		// panes are stacked), so ask the panes directly.
		a, b = widget.MinWidthOf(s.A), widget.MinWidthOf(s.B)
		return max(a, b)
	}
	return a + b + max(s.bar(), 1)
}

// mins are the two floors, already reduced to fit if together they are// mins are the two floors, already reduced to fit if together they are
// wider than there is: a splitter too narrow for both panes shares what
// it has in proportion rather than giving the first one everything.
func (s *Splitter) mins(avail float32) (a, b float32) {
	a, b = s.minFor(s.A, s.MinA), s.minFor(s.B, s.MinB)
	if total := a + b; total > avail && total > 0 {
		k := avail / total
		a, b = a*k, b*k
	}
	return a, b
}

// panes returns exclusive A / sash / B rects in local coordinates for box.
func (s *Splitter) panes(box paintengine2d.Rect) (a, div, b paintengine2d.Rect) {
	s.clampRatio()
	bar := s.bar()
	if bar < 1 {
		bar = 1
	}
	w, h := box.Dx(), box.Dy()
	if s.sideBySide() {
		if w < bar {
			bar = w
		}
		avail := w - bar
		if avail < 0 {
			avail = 0
		}
		// Whole pixels, because Arrange snaps a child's bounds to the
		// grid and PaneA must be the rect the child actually got.
		aw := float32(math.Round(float64(avail * s.Ratio)))
		// Neither pane below what it holds.
		if minA, minB := s.mins(avail); minA > 0 || minB > 0 {
			if aw < minA {
				aw = minA
			}
			if avail-aw < minB {
				aw = avail - minB
			}
			if aw < 0 {
				aw = 0
			}
		}
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
	if !widget.Recording(ctx) {
		a, _, b := s.panes(s.LocalBounds())
		s.paintPane(ctx, s.A, a)
		s.paintPane(ctx, s.B, b)
	}
	st := s.State()
	if s.hovered || s.drag {
		st |= style.StateHovered
	}
	if s.drag {
		st |= style.StatePressed
	}
	s.Look().DrawSplitter(ctx, s.divider(), s.sideBySide(), st)
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
	if s.sideBySide() {
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

func (s *Splitter) MouseEnter() { s.hovered = true; s.Base.MouseEnter() }

func (s *Splitter) MouseExit() {
	s.hovered = false
	if !s.drag {
		widget.ApplyCursor(s.Host(), platform.CursorDefault)
	}
	s.Base.MouseExit()
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
		s.Invalidate()
	}
	if !s.drag {
		s.applyCursor(e.Pos)
		return over
	}
	b := s.LocalBounds()
	bar := s.bar()
	if s.sideBySide() {
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
	s.Arrange(s.Bounds())
	s.notifyRatio()
	s.RequestLayout()
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

// notifyRatio tells a watcher where the divider is now, with the pane's
// width already resolved — so a caller does not have to work the ratio
// back into pixels itself, and gets the same number the pane actually
// got after the minimums were applied.
func (s *Splitter) notifyRatio() {
	if s.OnRatioChanged == nil {
		return
	}
	s.OnRatioChanged(s.PaneA().Dx(), s.Ratio)
}

// SetRatio moves the divider and reports it, as a drag does. It is how
// an application restores a saved layout without the watcher missing it.
func (s *Splitter) SetRatio(r float32) {
	s.Ratio = r
	s.clampRatio()
	if !s.Bounds().Empty() {
		s.Arrange(s.Bounds())
	}
	s.notifyRatio()
	s.Invalidate()
}
