package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

func wrappingChild() *Label {
	l := NewLabel("A very long message about a program path that will certainly wrap when " +
		"the container is narrower than it was measured for, several times over, and more.")
	l.Wrap = true
	return l
}

// Every container that sizes a child must measure it at the width it
// will give it. One that measures at one width and arranges at another
// leaves the child's last lines — and whatever is under them — outside
// the box, where clicks on them miss.
//
// Two of these were doing it before 0.23: Overlay measured its card at
// 0.8 of itself and arranged it at anything up to 0.92, and Grid
// measured its rows at the columns' natural widths while Arrange shared
// the whole width out. This is the sweep that says the rest do not.
//
// It does not cover a *caller* that measures a container at one width
// and arranges it at another. Nothing inside the container can fix
// that: a column handed a box too short for its content cannot grow
// it.
func TestContainersSizeChildrenAtTheWidthTheyGive(t *testing.T) {
	cases := []struct {
		name string
		make func() (parent widget.Component, deepest widget.Component)
	}{
		{"ScrollView", func() (widget.Component, widget.Component) {
			c := wrappingChild()
			return NewScrollView(NewColumn(c)), c
		}},
		{"Panel", func() (widget.Component, widget.Component) {
			c := wrappingChild()
			return NewPanel("Title", NewColumn(c)), c
		}},
		{"Pad", func() (widget.Component, widget.Component) {
			c := wrappingChild()
			return NewPad(8, c), c
		}},
		{"Overlay", func() (widget.Component, widget.Component) {
			c := wrappingChild()
			return NewOverlay(NewColumn(c)), c
		}},
		{"Form", func() (widget.Component, widget.Component) {
			c := wrappingChild()
			f := NewForm()
			f.AddRow("Label", c)
			return f, c
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parent, child := tc.make()
			atScale(t, parent, 1)
			const w = 340
			sz := parent.Measure(layout.Constraints{MaxW: w, MaxH: -1})
			parent.Arrange(paintengine2d.XYWH(0, 0, w, sz.Y))

			// Where the child actually ended up, in the parent's space.
			o := deviceOriginWithin(parent, child)
			cb := child.Bounds()
			bottom := o + cb.Dy()
			if bottom > sz.Y+1 {
				t.Errorf("%s: measured %v tall, but its child ends at %v",
					tc.name, sz.Y, bottom)
			}
			// And the child got a width it can actually wrap into.
			need := child.Measure(layout.Constraints{MaxW: cb.Dx(), MaxH: -1})
			if need.Y > cb.Dy()+1 {
				t.Errorf("%s: child is %v tall but needs %v at the %v it was given",
					tc.name, cb.Dy(), need.Y, cb.Dx())
			}
		})
	}
}

// deviceOriginWithin is c's y offset from root, summing the chain.
func deviceOriginWithin(root, c widget.Component) float32 {
	var y float32
	for n := c; n != nil && n != root; n = n.Parent() {
		y += n.Bounds().Min.Y
	}
	return y
}
