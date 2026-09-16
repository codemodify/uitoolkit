package dock

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// dropKind is what releasing the pointer would do with the panel.
type dropKind uint8

const (
	dropNone dropKind = iota
	// dropArea docks the panel to one side of the whole host.
	dropArea
	// dropSplit puts it beside an existing stack.
	dropSplit
	// dropTab drops it into an existing stack as another tab.
	dropTab
	// dropFloat takes it out into a window of its own.
	dropFloat
)

// dropTarget is where a dragged panel would land, and the box the
// indicator draws to show it.
type dropTarget struct {
	kind  dropKind
	side  Side
	stack *Stack
	// rect is in the host's local coordinates.
	rect paintengine2d.Rect
}

func (t dropTarget) same(o dropTarget) bool {
	return t.kind == o.kind && t.side == o.side && t.stack == o.stack
}

// dragState is a panel being dragged by its title bar or its tab.
type dragState struct {
	panel *Panel
	from  *Stack
	// start is where the press landed, in host-local coordinates.
	start paintengine2d.Point
	// active is set once the pointer has moved past the threshold, which
	// is when the indicator appears — a click that never moves must not
	// start a drag.
	active bool
	target dropTarget
}

// dragThreshold is how far the pointer moves before a press on a title bar
// becomes a drag, matching the toolkit's tab reorder and the desktop's own
// move threshold.
func dragThreshold(lk style.LookAndFeel) float32 { return style.Dip(lk, 8) }

// edgeBand is how close to the host's border the pointer has to be for the
// drop to mean "a new area down this whole side" rather than "beside the
// pane under the pointer".
func edgeBand(lk style.LookAndFeel) float32 { return style.Dip(lk, 22) }

// dragging reports whether a panel is being dragged right now: the press
// has landed and the pointer has moved past the threshold.
func (h *Host) dragging() bool { return h.drag != nil && h.drag.active }

// dragArmed reports whether a press has landed on a title bar or a tab. It
// is true before the threshold too, so the title bar keeps feeding moves in
// until the drag either starts or the button comes up.
func (h *Host) dragArmed() bool { return h.drag != nil }

// toLocal moves a window point into the host's own coordinates.
func (h *Host) toLocal(p paintengine2d.Point) paintengine2d.Point {
	o := widget.DeviceOrigin(h)
	return paintengine2d.Pt(p.X-o.X, p.Y-o.Y)
}

// rectOf is c's box in the host's local coordinates.
func (h *Host) rectOf(c widget.Component) paintengine2d.Rect {
	o := widget.DeviceOrigin(h)
	b := widget.DeviceBounds(c)
	return b.Translate(paintengine2d.Pt(-o.X, -o.Y))
}

// beginDrag arms a drag of p out of stack from. at is in window
// coordinates. Nothing shows until the pointer has moved far enough.
func (h *Host) beginDrag(from *Stack, p *Panel, at paintengine2d.Point) {
	if p == nil || p.features&FeatureMovable == 0 {
		return
	}
	h.drag = &dragState{panel: p, from: from, start: h.toLocal(at)}
}

// dragMove follows the pointer: once past the threshold it works out where
// the panel would land and moves the indicator there.
func (h *Host) dragMove(at paintengine2d.Point) {
	d := h.drag
	if d == nil {
		return
	}
	p := h.toLocal(at)
	if !d.active {
		t := dragThreshold(h.Look())
		if absF(p.X-d.start.X) < t && absF(p.Y-d.start.Y) < t {
			return
		}
		d.active = true
	}
	target := h.targetAt(p)
	if target.same(d.target) && h.ind.Visible() {
		return
	}
	d.target = target
	h.showIndicator(target)
}

// endDrag drops the panel where the indicator says.
func (h *Host) endDrag(at paintengine2d.Point) {
	d := h.drag
	h.drag = nil
	h.hideIndicator()
	if d == nil || !d.active {
		return
	}
	h.applyDrop(d.panel, h.targetAt(h.toLocal(at)))
}

// cancelDrag gives up the drag and leaves the panel where it was (Escape,
// or the host being torn down under it).
func (h *Host) cancelDrag() {
	h.drag = nil
	h.hideIndicator()
}

// applyDrop moves p to where the target says.
func (h *Host) applyDrop(p *Panel, t dropTarget) {
	if p == nil {
		return
	}
	switch t.kind {
	case dropArea:
		if p.Floating() {
			h.dockBack(p, func() { h.Dock(p, t.side) })
			return
		}
		h.Dock(p, t.side)
	case dropTab:
		cur := t.stack.Current()
		if cur == nil || cur == p {
			return
		}
		if p.Floating() {
			h.dockBack(p, func() { h.DockInto(p, cur) })
			return
		}
		h.DockInto(p, cur)
	case dropSplit:
		cur := t.stack.Current()
		if cur == nil || cur == p {
			return
		}
		if p.Floating() {
			h.dockBack(p, func() { h.DockBeside(p, cur, t.side) })
			return
		}
		h.DockBeside(p, cur, t.side)
	case dropFloat:
		if !p.Floating() {
			h.FloatPanel(p, p.FloatGeometry())
		}
	}
}

// targetAt is where a panel dropped at host-local point p would land.
//
// The order matters and follows Qt Creator and VS Code: a pointer off the
// host floats the panel; one in the thin band along the host's border
// makes a new area down that whole side; otherwise the pane under the
// pointer decides — its middle takes the panel as another tab, its edges
// split it.
func (h *Host) targetAt(p paintengine2d.Point) dropTarget {
	b := h.LocalBounds()
	if b.Empty() {
		return dropTarget{}
	}
	if !b.Contains(p) {
		return dropTarget{kind: dropFloat}
	}
	band := edgeBand(h.Look())
	switch {
	case p.X-b.Min.X < band:
		return dropTarget{kind: dropArea, side: SideLeft, rect: areaPreview(b, SideLeft)}
	case b.Max.X-p.X < band:
		return dropTarget{kind: dropArea, side: SideRight, rect: areaPreview(b, SideRight)}
	case p.Y-b.Min.Y < band:
		return dropTarget{kind: dropArea, side: SideTop, rect: areaPreview(b, SideTop)}
	case b.Max.Y-p.Y < band:
		return dropTarget{kind: dropArea, side: SideBottom, rect: areaPreview(b, SideBottom)}
	}
	st := h.stackAt(p)
	if st == nil {
		return dropTarget{}
	}
	r := h.rectOf(st)
	if r.Empty() {
		return dropTarget{}
	}
	side, ok := edgeZone(r, p)
	if !ok {
		return dropTarget{kind: dropTab, stack: st, rect: r}
	}
	return dropTarget{kind: dropSplit, side: side, stack: st, rect: halfOf(r, side)}
}

// stackAt is the stack whose box holds host-local point p, or nil.
func (h *Host) stackAt(p paintengine2d.Point) *Stack {
	for _, st := range h.stacks() {
		if st.Empty() {
			continue
		}
		if h.rectOf(st).Contains(p) {
			return st
		}
	}
	return nil
}

// edgeZone is the quarter of r that p is in — the side the panel would
// split off — and false when p is in the middle, which means "as a tab".
func edgeZone(r paintengine2d.Rect, p paintengine2d.Point) (Side, bool) {
	// A quarter each side leaves the middle half for tabbing, which is the
	// easiest target to hit and the most common thing to want.
	qx, qy := r.Dx()*0.25, r.Dy()*0.25
	dl, dr := p.X-r.Min.X, r.Max.X-p.X
	dt, db := p.Y-r.Min.Y, r.Max.Y-p.Y
	switch {
	case dl < qx && dl <= dt && dl <= db:
		return SideLeft, true
	case dr < qx && dr <= dt && dr <= db:
		return SideRight, true
	case dt < qy:
		return SideTop, true
	case db < qy:
		return SideBottom, true
	}
	return SideLeft, false
}

// halfOf is the half of r on the given side, which is where a split drop
// would put the panel.
func halfOf(r paintengine2d.Rect, s Side) paintengine2d.Rect {
	switch s {
	case SideLeft:
		return paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx()*0.5, r.Dy())
	case SideRight:
		return paintengine2d.XYWH(r.Min.X+r.Dx()*0.5, r.Min.Y, r.Dx()*0.5, r.Dy())
	case SideTop:
		return paintengine2d.XYWH(r.Min.X, r.Min.Y, r.Dx(), r.Dy()*0.5)
	default:
		return paintengine2d.XYWH(r.Min.X, r.Min.Y+r.Dy()*0.5, r.Dx(), r.Dy()*0.5)
	}
}

// areaPreview is the strip a new area down one side of the host would
// take: a quarter of the host, which is close to what the area gets.
func areaPreview(b paintengine2d.Rect, s Side) paintengine2d.Rect {
	switch s {
	case SideLeft:
		return paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx()*0.25, b.Dy())
	case SideRight:
		return paintengine2d.XYWH(b.Max.X-b.Dx()*0.25, b.Min.Y, b.Dx()*0.25, b.Dy())
	case SideTop:
		return paintengine2d.XYWH(b.Min.X, b.Min.Y, b.Dx(), b.Dy()*0.25)
	default:
		return paintengine2d.XYWH(b.Min.X, b.Max.Y-b.Dy()*0.25, b.Dx(), b.Dy()*0.25)
	}
}

// showIndicator moves the drop indicator over the target, or hides it when
// there is nothing to show.
func (h *Host) showIndicator(t dropTarget) {
	if t.kind == dropNone || t.kind == dropFloat || t.rect.Empty() {
		h.hideIndicator()
		return
	}
	h.ind.show(t)
	h.Invalidate()
}

// hideIndicator takes the drop indicator away.
func (h *Host) hideIndicator() {
	if h.ind.hide() {
		h.Invalidate()
	}
}

// ---- the indicator ------------------------------------------------------

// dropIndicator is the highlight that says where a dragged panel would
// land. It is the host's last child, so it paints over the panels, and it
// takes no hits of its own.
type dropIndicator struct {
	widget.Base
	target dropTarget
}

func newDropIndicator() *dropIndicator {
	d := &dropIndicator{}
	d.Init(d)
	d.SetVisible(false)
	return d
}

func (d *dropIndicator) show(t dropTarget) {
	d.target = t
	d.SetVisible(true)
	d.Invalidate()
}

// hide takes the indicator away and reports whether it had been showing.
func (d *dropIndicator) hide() bool {
	if !d.Visible() {
		return false
	}
	d.SetVisible(false)
	return true
}

func (d *dropIndicator) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Point{})
}

func (d *dropIndicator) Arrange(r paintengine2d.Rect) { d.SetBounds(r) }

// HitTest is nil: the indicator is a mark on the glass, never a target.
func (d *dropIndicator) HitTest(paintengine2d.Point) widget.Component { return nil }

func (d *dropIndicator) Paint(ctx *paintengine2d.Context) {
	if !d.Visible() || d.target.rect.Empty() {
		return
	}
	style.DrawDropIndicatorOf(d.Look(), ctx, d.target.rect, d.target.kind == dropTab)
}

func absF(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
