// Package dock is dockable panels: a central widget with dock areas on its
// four sides, panels that split, stack as tabs, collapse, float in windows
// of their own, and a layout that can be written out and read back.
//
// The pieces:
//
//	Host   — the dock host: a central widget and the four areas around it
//	Panel  — one dockable panel: an id, a title and its content
//	Stack  — panels sharing one box, shown as tabs when there is more than one
//	Split  — panes side by side or one above the other, with sashes between
//
// An area's arrangement is a tree of Splits whose leaves are Stacks. The
// host keeps a fixed skeleton — a vertical split of top area, middle and
// bottom area, the middle a horizontal split of left area, centre and right
// area — so every sash, including the one between an area and the centre,
// is the same Split sash.
//
// Nothing here opens windows on its own: floating panels go through
// [WindowOpener], which app supplies (app.DockWindows). That keeps the
// package free of a dependency on app, and leaves room for a later
// compositor-side tear-off (xdg-toplevel-drag) behind the same interface.
package dock

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// Side is the edge of the host a dock area sits on.
type Side int

const (
	SideLeft Side = iota
	SideRight
	SideTop
	SideBottom
)

// String names the side as the layout format writes it.
func (s Side) String() string {
	switch s {
	case SideRight:
		return "right"
	case SideTop:
		return "top"
	case SideBottom:
		return "bottom"
	default:
		return "left"
	}
}

// parseSide reads a side back; anything else is the left area.
func parseSide(s string) Side {
	switch s {
	case "right":
		return SideRight
	case "top":
		return SideTop
	case "bottom":
		return SideBottom
	default:
		return SideLeft
	}
}

// vertical reports whether an area on this side stacks its panels top to
// bottom (the left and right areas) rather than side by side.
func (s Side) vertical() bool { return s == SideLeft || s == SideRight }

// Node is one element of an area's tree: a [Split] or a [Stack]. The centre
// pane is one too, so the skeleton around it is ordinary Splits.
type Node interface {
	widget.Component

	// MinSize is the smallest box the node can be arranged in.
	MinSize() paintengine2d.Point
	// Empty reports whether the node holds nothing worth space: a stack
	// with no open panel, a split whose children are all empty.
	Empty() bool
	// Fixed reports whether the node keeps one extent whatever space is
	// going (a collapsed stack keeps its header height).
	Fixed(vertical bool) (float32, bool)
	// dockNode keeps the interface closed to this package.
	dockNode()
}

// sashWidth is the room a split leaves between two panes, never thinner
// than a pointer can comfortably catch.
func sashWidth(lk style.LookAndFeel) float32 {
	if lk == nil {
		return 4
	}
	w := lk.Metrics().Splitter
	if min := style.Dip(lk, 4); w < min {
		w = min
	}
	return w
}

// minPanelSize is the floor under a panel's own minimum, so a panel can
// never be squeezed to nothing by its neighbours.
func minPanelSize(lk style.LookAndFeel) paintengine2d.Point {
	return paintengine2d.Pt(style.Dip(lk, 48), style.Dip(lk, 32))
}

// maxPt is the per-axis maximum of two sizes.
func maxPt(a, b paintengine2d.Point) paintengine2d.Point {
	if b.X > a.X {
		a.X = b.X
	}
	if b.Y > a.Y {
		a.Y = b.Y
	}
	return a
}

// axisOf picks the extent of sz along the split's axis.
func axisOf(sz paintengine2d.Point, vertical bool) float32 {
	if vertical {
		return sz.Y
	}
	return sz.X
}

// hostOf is the dock host c belongs to, or nil. A floating panel's chrome
// is in another window, so the walk up the tree never reaches the host:
// the panel's own record of its owner answers for it.
func hostOf(c widget.Component) *Host {
	for n := c; n != nil; n = n.Parent() {
		switch v := n.(type) {
		case *Host:
			return v
		case *Panel:
			if v.owner != nil {
				return v.owner
			}
		case *Stack:
			if cur := v.Current(); cur != nil && cur.owner != nil {
				return cur.owner
			}
		}
	}
	return nil
}
