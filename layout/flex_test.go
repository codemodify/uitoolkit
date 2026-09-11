package layout

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

type box struct {
	min paintengine2d.Point
	got paintengine2d.Rect
}

func (b *box) Measure(c Constraints) paintengine2d.Point { return c.Constrain(b.min) }
func (b *box) Arrange(r paintengine2d.Rect)              { b.got = r }

func TestColumnStacks(t *testing.T) {
	a := &box{min: paintengine2d.Pt(40, 10)}
	b := &box{min: paintengine2d.Pt(20, 20)}
	f := Flex{Axis: AxisVertical, Gap: 4, Align: AlignStretch}
	items := []Item{{Node: a}, {Node: b}}
	sz := f.Measure(Loose(100, 200), items)
	if sz.Y < 34 {
		t.Fatalf("height %v", sz)
	}
	f.Arrange(paintengine2d.XYWH(0, 0, 80, 80), items)
	if a.got.Min.Y != 0 || b.got.Min.Y < 13 {
		t.Fatalf("stack %+v %+v", a.got, b.got)
	}
	if a.got.Dx() != 80 {
		t.Fatalf("stretch width %v", a.got.Dx())
	}
}

func TestRowFlexGrow(t *testing.T) {
	a := &box{min: paintengine2d.Pt(10, 10)}
	b := &box{min: paintengine2d.Pt(10, 10)}
	f := Flex{Axis: AxisHorizontal, Gap: 0}
	items := []Item{{Node: a, Flex: 1}, {Node: b, Flex: 3}}
	f.Arrange(paintengine2d.XYWH(0, 0, 80, 20), items)
	if a.got.Dx() < 18 || a.got.Dx() > 22 {
		t.Fatalf("flex1 width %v", a.got.Dx())
	}
	if b.got.Dx() < 58 || b.got.Dx() > 62 {
		t.Fatalf("flex3 width %v", b.got.Dx())
	}
}

func TestArrangeIsLocal(t *testing.T) {
	a := &box{min: paintengine2d.Pt(20, 10)}
	f := Flex{Axis: AxisVertical, Gap: 0}
	items := []Item{{Node: a}}
	f.Arrange(paintengine2d.XYWH(40, 50, 80, 40), items)
	if a.got.Min.X != 0 || a.got.Min.Y != 0 {
		t.Fatalf("child must be parent-local, got %+v", a.got)
	}
}

func TestConstraintsConstrain(t *testing.T) {
	c := Constraints{MinW: 10, MaxW: 30, MinH: 5, MaxH: 20}
	sz := c.Constrain(paintengine2d.Pt(100, 1))
	if sz.X != 30 || sz.Y != 5 {
		t.Fatalf("%v", sz)
	}
}
