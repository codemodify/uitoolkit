package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// FlexBox is a row or column of children with optional grow weights.
type FlexBox struct {
	widget.Base
	Spec  layout.Flex
	items []layout.Item
}

// NewColumn stacks children top-to-bottom.
func NewColumn(children ...widget.Component) *FlexBox {
	f := newFlex(layout.AxisVertical)
	for _, ch := range children {
		f.Add(ch)
	}
	return f
}

// NewRow places children left-to-right.
func NewRow(children ...widget.Component) *FlexBox {
	f := newFlex(layout.AxisHorizontal)
	for _, ch := range children {
		f.Add(ch)
	}
	return f
}

func newFlex(axis layout.Axis) *FlexBox {
	f := &FlexBox{Spec: layout.Flex{Axis: axis, Gap: 8, Align: layout.AlignStretch}}
	f.Init(f)
	return f
}

// WithGap sets the main-axis gap.
func (f *FlexBox) WithGap(g float32) *FlexBox { f.Spec.Gap = g; return f }

// WithPad sets uniform padding.
func (f *FlexBox) WithPad(v float32) *FlexBox {
	f.Spec.PadL, f.Spec.PadT, f.Spec.PadR, f.Spec.PadB = v, v, v, v
	return f
}

// WithPadding sets LTRB padding.
func (f *FlexBox) WithPadding(l, t, r, b float32) *FlexBox {
	f.Spec.PadL, f.Spec.PadT, f.Spec.PadR, f.Spec.PadB = l, t, r, b
	return f
}

// WithAlign sets cross-axis alignment.
func (f *FlexBox) WithAlign(a layout.Align) *FlexBox { f.Spec.Align = a; return f }

// WithJustify sets main-axis packing.
func (f *FlexBox) WithJustify(j layout.Justify) *FlexBox { f.Spec.Justify = j; return f }

func (f *FlexBox) Add(child widget.Component) {
	if child == nil {
		return
	}
	f.Base.Add(child)
	f.items = append(f.items, layout.Item{Node: wrap(child)})
}

// Remove keeps the flex item list in step with the children. Without this the
// next syncItems saw a length mismatch and rebuilt every item with weight 0,
// so removing one child silently dropped every sibling's grow weight.
func (f *FlexBox) Remove(child widget.Component) {
	f.Base.Remove(child)
	f.syncItems()
}

// ClearChildren drops the children and their flex weights together.
func (f *FlexBox) ClearChildren() {
	f.Base.ClearChildren()
	f.items = f.items[:0]
}

func (f *FlexBox) AddFlex(child widget.Component, weight float32) {
	if child == nil {
		return
	}
	for i, ch := range f.Children() {
		if ch == child {
			f.syncItems()
			if i < len(f.items) {
				f.items[i].Flex = weight
			}
			return
		}
	}
	f.Base.Add(child)
	f.items = append(f.items, layout.Item{Node: wrap(child), Flex: weight})
}

func (f *FlexBox) Measure(c layout.Constraints) paintengine2d.Point {
	return f.Spec.Measure(c, f.visibleItems(false))
}

func (f *FlexBox) Arrange(r paintengine2d.Rect) {
	f.SetBounds(r)
	f.Spec.Arrange(r, f.visibleItems(true))
}

func (f *FlexBox) visibleItems(collapseHidden bool) []layout.Item {
	f.syncItems()
	chs := f.Children()
	var out []layout.Item
	for i, ch := range chs {
		if ch == nil || !ch.Visible() {
			if collapseHidden && ch != nil {
				ch.Arrange(paintengine2d.Rect{})
			}
			continue
		}
		if i < len(f.items) {
			out = append(out, f.items[i])
		}
	}
	return out
}

// syncItems reconciles items with the child list, carrying grow weights across
// by component identity so add / remove / re-parent cannot silently reset them.
func (f *FlexBox) syncItems() {
	chs := f.Children()
	if len(f.items) == len(chs) {
		aligned := true
		for i, ch := range chs {
			if n, ok := f.items[i].Node.(node); !ok || n.c != ch {
				aligned = false
				break
			}
		}
		if aligned {
			return
		}
	}
	var weights map[widget.Component]float32
	for _, it := range f.items {
		n, ok := it.Node.(node)
		if !ok || n.c == nil || it.Flex == 0 {
			continue
		}
		if weights == nil {
			weights = make(map[widget.Component]float32, len(f.items))
		}
		weights[n.c] = it.Flex
	}
	f.items = f.items[:0]
	for _, ch := range chs {
		f.items = append(f.items, layout.Item{Node: wrap(ch), Flex: weights[ch]})
	}
}

type node struct{ c widget.Component }

func wrap(c widget.Component) layout.Child { return node{c} }

func (n node) Measure(c layout.Constraints) paintengine2d.Point { return n.c.Measure(c) }
func (n node) Arrange(r paintengine2d.Rect)                     { n.c.Arrange(r) }

// Stack overlays children; each is arranged to the same box.
type Stack struct {
	widget.Base
	Pad float32
}

func NewStack(children ...widget.Component) *Stack {
	s := &Stack{}
	s.Init(s)
	for _, ch := range children {
		s.Add(ch)
	}
	return s
}

func (s *Stack) Measure(c layout.Constraints) paintengine2d.Point {
	var w, h float32
	inner := c.Inset(s.Pad*2, s.Pad*2)
	for _, ch := range s.Children() {
		sz := ch.Measure(inner)
		if sz.X > w {
			w = sz.X
		}
		if sz.Y > h {
			h = sz.Y
		}
	}
	return c.Constrain(paintengine2d.Pt(w+s.Pad*2, h+s.Pad*2))
}

func (s *Stack) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	inner := r.Inset(s.Pad)
	// Convert to local
	local := paintengine2d.XYWH(s.Pad, s.Pad, inner.Dx(), inner.Dy())
	for _, ch := range s.Children() {
		ch.Arrange(local)
	}
}

// Pad insets a single child.
type Pad struct {
	widget.Base
	L, T, R, B float32
}

func NewPad(v float32, child widget.Component) *Pad {
	p := &Pad{L: v, T: v, R: v, B: v}
	p.Init(p)
	if child != nil {
		p.Add(child)
	}
	return p
}

func (p *Pad) Measure(c layout.Constraints) paintengine2d.Point {
	inner := c.Inset(p.L+p.R, p.T+p.B)
	var sz paintengine2d.Point
	if len(p.Children()) > 0 {
		sz = p.Children()[0].Measure(inner)
	}
	return c.Constrain(paintengine2d.Pt(sz.X+p.L+p.R, sz.Y+p.T+p.B))
}

func (p *Pad) Arrange(r paintengine2d.Rect) {
	p.SetBounds(r)
	if len(p.Children()) == 0 {
		return
	}
	p.Children()[0].Arrange(paintengine2d.XYWH(p.L, p.T, r.Dx()-p.L-p.R, r.Dy()-p.T-p.B))
}
