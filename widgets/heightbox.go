package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// HeightBox holds one child to a height given in 1x pixels, scaled with
// the look's display scale. It is what a view that would otherwise grow
// without bound needs when it sits in a column that scrolls: a list as
// tall as all its rows, a tree as tall as its nodes, a summary as long as
// its text. The child gets the box's full width.
//
// Share caps the height at that fraction of the height the parent offers,
// so a box does not swallow a short window; 0 never caps.
type HeightBox struct {
	widget.Base
	Height float32
	Share  float32
}

// NewHeightBox is child held to h 1x pixels tall. h of 0 or less leaves
// child its own height and returns it unwrapped.
func NewHeightBox(h float32, child widget.Component) widget.Component {
	return NewHeightBoxShare(h, 0, child)
}

// NewHeightBoxShare is [NewHeightBox] that also never takes more than
// share (0..1) of the height its parent offers.
func NewHeightBoxShare(h, share float32, child widget.Component) widget.Component {
	if h <= 0 {
		return child
	}
	b := &HeightBox{Height: h, Share: share}
	b.Init(b)
	if child != nil {
		b.Add(child)
	}
	return b
}

func (b *HeightBox) Measure(c layout.Constraints) paintengine2d.Point {
	h := style.Dip(b.Look(), b.Height)
	if b.Share > 0 && c.HasMaxH() && h > c.MaxH*b.Share {
		h = c.MaxH * b.Share
	}
	var w float32
	if kids := b.Children(); len(kids) > 0 {
		w = kids[0].Measure(layout.Constraints{MinW: c.MinW, MaxW: c.MaxW, MaxH: h}).X
	}
	if c.HasMaxW() {
		w = c.MaxW
	}
	return c.Constrain(paintengine2d.Pt(w, h))
}

func (b *HeightBox) Arrange(r paintengine2d.Rect) {
	b.SetBounds(r)
	if kids := b.Children(); len(kids) > 0 {
		kids[0].Arrange(paintengine2d.XYWH(0, 0, r.Dx(), r.Dy()))
	}
}
