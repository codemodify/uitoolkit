package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// lookAt is the default look at a display scale.
func lookAt(scale float32) style.LookAndFeel {
	return style.WithScale(style.DarkLook(), scale)
}

// A column made WithPad(20) has 20 pixels of padding at 1x and 40 at 2x.
//
// They used to be device pixels, so a dialog's margins halved on a HiDPI
// display beside text and controls that had doubled, and every
// application that cared had to write style.Dip around its own numbers.
func TestFlexSpacingFollowsTheScale(t *testing.T) {
	build := func(scale float32) *FlexBox {
		col := NewColumn(newSized(40, 20), newSized(40, 20)).WithGap(10).WithPad(20)
		col.SetLook(lookAt(scale))
		col.SetHost(&host{})
		return col
	}
	one := build(1).Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	two := build(2).Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	// The children do not scale here (they are fixed test widgets), so
	// the whole difference is the padding and the gap: 2*20 + 10 more.
	if want := one.Y + 20*2 + 10; two.Y != want {
		t.Fatalf("at 2x the column is %v tall, want %v (1x was %v)", two.Y, want, one.Y)
	}

	// And the children land inside the scaled padding.
	col := build(2)
	col.Arrange(paintengine2d.XYWH(0, 0, 200, two.Y))
	first := col.Children()[0]
	if got := first.Bounds().Min.Y; got != 40 {
		t.Fatalf("the first child starts at %v, want 40 (20 design pixels at 2x)", got)
	}
}

// RawSpacing is the way back for a caller that means device pixels.
func TestRawSpacingOptsOut(t *testing.T) {
	col := NewColumn(newSized(40, 20), newSized(40, 20)).WithGap(10).WithPad(20)
	col.RawSpacing = true
	col.SetLook(lookAt(2))
	col.SetHost(&host{})
	col.Arrange(paintengine2d.XYWH(0, 0, 200, 200))
	if got := col.Children()[0].Bounds().Min.Y; got != 20 {
		t.Fatalf("RawSpacing put the first child at %v, want 20", got)
	}
}

// The same for a grid, and so for a form.
func TestGridSpacingFollowsTheScale(t *testing.T) {
	build := func(scale float32) *Grid {
		g := NewGrid()
		g.ColGap, g.RowGap = 10, 6
		g.Place(newSized(40, 20), 0, 0)
		g.Place(newSized(40, 20), 0, 1)
		g.Place(newSized(40, 20), 1, 0)
		g.SetLook(lookAt(scale))
		g.SetHost(&host{})
		return g
	}
	one := build(1).Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	two := build(2).Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	if want := one.X + 10; two.X != want {
		t.Fatalf("at 2x the grid is %v wide, want %v (1x was %v)", two.X, want, one.X)
	}
	if want := one.Y + 6; two.Y != want {
		t.Fatalf("at 2x the grid is %v tall, want %v (1x was %v)", two.Y, want, one.Y)
	}
	g := build(2)
	g.RawSpacing = true
	if got := g.Measure(layout.Constraints{MaxW: -1, MaxH: -1}); got.X != one.X {
		t.Fatalf("RawSpacing grid is %v wide, want the 1x %v", got.X, one.X)
	}
}

// Nothing moves at 1x, which is why the toolkit's pinned geometry is
// untouched: the scale is the identity there.
func TestSpacingIsUnchangedAtOneToOne(t *testing.T) {
	raw := NewColumn(newSized(40, 20), newSized(40, 20)).WithGap(10).WithPad(20)
	raw.RawSpacing = true
	raw.SetLook(lookAt(1))
	raw.SetHost(&host{})
	scaled := NewColumn(newSized(40, 20), newSized(40, 20)).WithGap(10).WithPad(20)
	scaled.SetLook(lookAt(1))
	scaled.SetHost(&host{})
	c := layout.Constraints{MaxW: -1, MaxH: -1}
	if raw.Measure(c) != scaled.Measure(c) {
		t.Fatalf("at 1x scaled %v differs from raw %v", scaled.Measure(c), raw.Measure(c))
	}
}
