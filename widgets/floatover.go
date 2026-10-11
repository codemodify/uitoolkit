package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// FloatOver puts chrome over content instead of beside it: a tool bar
// that a list scrolls *under*, a status strip the document runs beneath.
// The content keeps the whole box and the floating pieces are laid on
// top of it, each against an edge and only as tall as it needs.
//
// A column puts a bar above a list and the list begins where the bar
// ends; nothing of the list is ever behind the bar. That is the right
// answer for an opaque bar and the wrong one for a translucent bar,
// because a bar with nothing behind it has nothing to be translucent
// *of*: glass over a flat window background looks exactly like paint,
// however it is drawn. Chrome that floats is what makes a material read
// as a material, and it is a layout decision rather than a palette one.
//
// Clicks fall through. Each floating piece is an ordinary child with its
// own small box, so a press lands on it only where it actually is and
// reaches the content everywhere else — the hit test walks the children
// backwards and skips the ones whose box does not hold the point, which
// is all this needs from it.
//
// A floating bar hides the first part of the content under it, so the
// content usually wants a matching inset of its own at that edge
// ([Pad], or a first row that is the bar's height). This container does
// not add one: how much of itself a view wants to keep clear is the
// view's business, and a list that means to scroll its first row under
// the bar should be allowed to.
type FloatOver struct {
	widget.Base
	content     widget.Component
	top, bottom widget.Component
	// Inset is the gap between a floating piece and the box's edge, as a
	// 1x design length. Zero puts the chrome flush against the edge.
	Inset float32
}

// NewFloatOver floats chrome over content. Content may be nil, which
// leaves the floating pieces over nothing.
func NewFloatOver(content widget.Component) *FloatOver {
	f := &FloatOver{}
	f.Init(f)
	f.SetContent(content)
	return f
}

// SetContent replaces what the chrome floats over.
func (f *FloatOver) SetContent(c widget.Component) { f.set(&f.content, c) }

// SetTop floats a piece against the top edge.
func (f *FloatOver) SetTop(c widget.Component) { f.set(&f.top, c) }

// SetBottom floats a piece against the bottom edge.
func (f *FloatOver) SetBottom(c widget.Component) { f.set(&f.bottom, c) }

// Content, Top and Bottom are what is in each slot.
func (f *FloatOver) Content() widget.Component { return f.content }
func (f *FloatOver) Top() widget.Component     { return f.top }
func (f *FloatOver) Bottom() widget.Component  { return f.bottom }

// set swaps one slot and rebuilds the child order, which decides both
// what paints over what and what a press is offered first.
func (f *FloatOver) set(slot *widget.Component, c widget.Component) {
	if *slot == c {
		return
	}
	*slot = c
	f.ClearChildren()
	// The content first, so the chrome paints over it; the walker paints
	// children in order and hit-tests them backwards.
	for _, ch := range []widget.Component{f.content, f.top, f.bottom} {
		if ch != nil {
			f.Add(ch)
		}
	}
	f.Invalidate()
}

// Measure is the content's own: floating chrome does not make a view
// want to be bigger, which is the point of it floating.
func (f *FloatOver) Measure(c layout.Constraints) paintengine2d.Point {
	if f.content == nil {
		return c.Constrain(paintengine2d.Point{})
	}
	return c.Constrain(f.content.Measure(c))
}

// MinWidth is the widest of everything in it: the chrome has to fit even
// though it does not decide the height.
func (f *FloatOver) MinWidth() float32 {
	var w float32
	for _, ch := range []widget.Component{f.content, f.top, f.bottom} {
		if ch == nil {
			continue
		}
		if m := widget.MinWidthOf(ch); m > w {
			w = m
		}
	}
	return w
}

func (f *FloatOver) Arrange(r paintengine2d.Rect) {
	f.SetBounds(r)
	box := paintengine2d.XYWH(0, 0, r.Dx(), r.Dy())
	if f.content != nil {
		f.content.Arrange(box)
	}
	pad := style.Dip(f.Look(), f.Inset)
	inner := box.Inset(pad)
	if inner.Dx() <= 0 {
		inner = box
	}
	bar := func(c widget.Component, atTop bool) {
		if c == nil {
			return
		}
		h := c.Measure(layout.Loose(inner.Dx(), inner.Dy())).Y
		if h <= 0 || h > inner.Dy() {
			h = min(max(h, 0), inner.Dy())
		}
		y := inner.Min.Y
		if !atTop {
			y = inner.Max.Y - h
		}
		c.Arrange(paintengine2d.XYWH(inner.Min.X, y, inner.Dx(), h))
	}
	bar(f.top, true)
	bar(f.bottom, false)
}
