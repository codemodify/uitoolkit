package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// A box holds its child to the height it was given, and hands it the full
// width, whatever the child would rather be.
func TestHeightBoxHoldsItsChild(t *testing.T) {
	child := NewMonoTextView("one\ntwo\nthree\nfour\nfive\nsix\nseven\neight", "")
	box := NewHeightBox(80, child)
	root := NewColumn(box)
	root.Arrange(paintengine2d.XYWH(0, 0, 400, 600))
	got := box.Measure(layout.Loose(400, 600))
	if got.Y != 80 {
		t.Errorf("the box measured %v tall, want 80", got.Y)
	}
	if got.X != 400 {
		t.Errorf("the box measured %v wide, want the full 400", got.X)
	}
	box.Arrange(paintengine2d.XYWH(0, 0, 400, 80))
	if b := child.Bounds(); b.Dy() != 80 || b.Dx() != 400 {
		t.Errorf("the child got %v, want the box's whole 400x80", b)
	}
}

// Share caps the box at a fraction of what it is offered, so a tall box
// does not swallow a short window.
func TestHeightBoxShareGivesTheColumnItsRoomBack(t *testing.T) {
	box := NewHeightBoxShare(400, 0.25, NewLabel("summary"))
	if got := box.Measure(layout.Loose(300, 1000)).Y; got != 250 {
		t.Errorf("in 1000 the box is %v tall, want a quarter", got)
	}
	if got := box.Measure(layout.Loose(300, 200)).Y; got != 50 {
		t.Errorf("in 200 the box is %v tall, want a quarter", got)
	}
	// Under its cap the box is simply its own height.
	if got := box.Measure(layout.Loose(300, 4000)).Y; got != 400 {
		t.Errorf("in 4000 the box is %v tall, want its 400", got)
	}
}

// A height of zero means "leave the child alone", and the child comes
// back unwrapped rather than in a box that does nothing.
func TestHeightBoxZeroIsNoBox(t *testing.T) {
	child := NewLabel("as tall as it likes")
	if got := NewHeightBox(0, child); got != widget.Component(child) {
		t.Errorf("NewHeightBox(0, child) = %T, want the child itself", got)
	}
}
