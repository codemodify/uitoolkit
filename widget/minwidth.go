package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
)

// How narrow something may usefully be.
//
// A layout that runs out of room stops at a floor rather than squeezing
// children to nothing: a label folds, so its column can go down to its
// longest word, while a button, a text field and a check box have no
// narrower form and keep what they ask for. When those floors together
// are wider than the space, the row runs over the edge. That is
// deliberate — Qt and GTK stop the same way — but it left a program with
// no way to find out *where* the floor is, so the only way to meet it
// was to hit it.
//
// [MinWidthOf] answers it. A program sizes its window from its own
// content rather than from a guess, and a test can assert that a form
// still fits the narrowest window it claims to support — which is the
// thing that catches a button growing an icon before a user does.

// MinWidther is a component that knows how narrow it may usefully be.
//
// Containers implement it, because they are the ones that can answer:
// asking a container to measure itself in one pixel gives back the one
// pixel it was constrained to, not the width below which its children
// stop fitting. A leaf widget usually needs nothing — [MinWidthOf] works
// it out — and implements this only when probing would be wrong or
// expensive.
type MinWidther interface {
	// MinWidth is the narrowest this component can be laid out at
	// without its contents leaving its box, in device pixels.
	MinWidth() float32
}

// MinWidthOf is how narrow c may usefully be.
//
// A component that implements [MinWidther] is asked. Anything else is
// worked out: it is measured unbounded, then again at a quarter of that
// width, and the **height** decides. What folds gets taller when it is
// narrowed, and the narrow width is real; what cannot fold keeps its
// height, and its natural width is the floor.
//
// The height is what decides because the width cannot. A child that
// cannot fold is clipped to a narrow constraint exactly as one that can,
// and both report back narrow — a button asked to fit in a pixel says
// 65, and a layout that believed it would draw a button 65 pixels wide
// with its label hanging out of both ends.
func MinWidthOf(c Component) float32 {
	if c == nil {
		return 0
	}
	if m, ok := c.(MinWidther); ok {
		return m.MinWidth()
	}
	natural := c.Measure(layout.Unbounded())
	w := MinWidthByProbe(c, natural)
	// A floor the probe cannot see past. The probe reads only the
	// component's own measurements, and a wrapper that passes everything
	// through to one child — a key handler, a drop target, a watcher, the
	// commonest thing an application writes — measures as whatever that
	// child measures. A Splitter unbounded measures a fixed 320x200, so a
	// wrapper round one answered 320 however much its panes needed: a
	// window sized from that line let itself shrink to a third of what
	// its content could live with.
	//
	// Only for a component with exactly one child, which is what a
	// pass-through wrapper is: it holds one thing and gives it everything
	// it has, so it cannot be narrower than that thing. A container with
	// several is not safe to treat this way — it may lay them out in a
	// way that is narrower than the widest of them, and the containers
	// that know (Column, Grid, Splitter, ScrollView, Wrap) answer for
	// themselves above and never reach here.
	if kids := c.Children(); len(kids) == 1 {
		if ch := MinWidthOf(kids[0]); ch > w {
			w = ch
		}
	}
	return w
}

// MinWidthByProbe is [MinWidthOf]'s fallback, exported for a container
// that wants it for a child it is already measuring.
//
// natural is c's unbounded measurement, which the caller usually has
// to hand; passing it avoids measuring twice.
func MinWidthByProbe(c Component, natural paintengine2d.Point) float32 {
	if c == nil || natural.X <= 0 {
		return 0
	}
	probe := natural.X * 0.25
	if probe < 24 {
		probe = 24
	}
	if probe >= natural.X {
		return ceilPx(natural.X)
	}
	got := c.Measure(layout.Constraints{MaxW: probe, MaxH: -1})
	if got.Y <= natural.Y {
		// It did not get taller, so it did not fold: this is as narrow
		// as it goes.
		return ceilPx(natural.X)
	}
	if w := ceilPx(min(got.X, probe)); w >= 1 {
		return w
	}
	return 1
}

// MinWidthOfChildren is the widest minimum among cs, which is the
// minimum of any container that stacks its children rather than placing
// them side by side — a column, a wrap, a scroll view.
func MinWidthOfChildren(cs []Component) float32 {
	var w float32
	for _, c := range cs {
		if c == nil || !c.Visible() {
			continue
		}
		if m := MinWidthOf(c); m > w {
			w = m
		}
	}
	return w
}

// SumMinWidths is the total minimum of cs laid side by side with gap
// between them, which is the minimum of a row.
func SumMinWidths(cs []Component, gap float32) float32 {
	var w float32
	n := 0
	for _, c := range cs {
		if c == nil || !c.Visible() {
			continue
		}
		w += MinWidthOf(c)
		n++
	}
	if n > 1 {
		w += gap * float32(n-1)
	}
	return w
}

func ceilPx(v float32) float32 {
	i := float32(int(v))
	if v > i {
		return i + 1
	}
	return i
}
