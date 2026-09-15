package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// sized is a leaf with a fixed natural size.
type sized struct {
	widget.Base
	w, h float32
}

func newSized(w, h float32) *sized {
	s := &sized{w: w, h: h}
	s.Init(s)
	return s
}

func (s *sized) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(s.w, s.h))
}
func (s *sized) Arrange(r paintengine2d.Rect) { s.SetBounds(r) }

// Auto columns fit their widest child, fixed ones keep their length, and a
// flexible column takes what is left; rows fit their tallest child, and a
// spanning child grows the flexible track it covers.
func TestGridTracks(t *testing.T) {
	g := NewGrid()
	g.ColGap, g.RowGap = 10, 5
	g.Cols = []Track{Auto(), Px(50), Flex(1)}
	a, b, c := newSized(80, 20), newSized(30, 30), newSized(40, 24)
	g.Place(a, 0, 0)
	g.Place(b, 0, 1)
	g.Place(c, 1, 2)
	wide := newSized(300, 18)
	g.PlaceSpan(wide, 2, 0, 1, 3)
	g.SetHost(&fakeWindow{look: style.DarkLook()})

	// Naturally: 80 + 50 + max(40, wide needs 300-80-50-20=150) + gaps.
	sz := g.Measure(layout.Unbounded())
	if sz.X != 300 {
		t.Fatalf("natural width %v, want 300 (the spanning child)", sz.X)
	}
	if want := float32(30 + 5 + 24 + 5 + 18); sz.Y != want {
		t.Fatalf("natural height %v, want %v", sz.Y, want)
	}

	g.Arrange(paintengine2d.XYWH(0, 0, 500, 200))
	if r := c.Bounds(); r.Min.X != 150 || r.Dx() != 350 {
		t.Fatalf("flex column child at %v, want x 150 width 350", r)
	}
	if r := b.Bounds(); r.Min.X != 90 || r.Dx() != 50 {
		t.Fatalf("fixed column child at %v, want x 90 width 50", r)
	}
	// Vertically centred in its 30px row.
	if r := a.Bounds(); r.Min.Y != 5 || r.Dy() != 20 {
		t.Fatalf("auto child at %v, want centred at y 5, 20 tall", r)
	}
	if r := wide.Bounds(); r.Dx() != 500 {
		t.Fatalf("spanning child width %v, want the full 500", r.Dx())
	}
}

// Forms right-align their labels in Mac looks and left-align them
// elsewhere; fields take the rest of the width.
func TestFormLabelAlignmentFollowsLook(t *testing.T) {
	for _, tc := range []struct {
		pack string
		want style.Align
	}{{"aqua", style.AlignEnd}, {"win95", style.AlignStart}} {
		p, ok := style.LoadTheme(tc.pack)
		if !ok {
			t.Fatalf("pack %s", tc.pack)
		}
		f := NewForm()
		field := newSized(120, 24)
		l := f.AddRow("Name", field)
		f.AddRow("Longer label", newSized(120, 24))
		f.SetHost(&fakeWindow{look: p.Look()})
		f.Arrange(paintengine2d.XYWH(0, 0, 400, 100))
		if l.Align != tc.want {
			t.Fatalf("%s: label align %v, want %v", tc.pack, l.Align, tc.want)
		}
		if r := field.Bounds(); r.Max.X != 400 {
			t.Fatalf("%s: field %v should reach the right edge", tc.pack, r)
		}
	}
}
