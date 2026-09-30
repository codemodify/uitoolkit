package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// The floor a layout stops at has to be knowable, or the only way to
// meet it is to hit it: a form runs over the edge of a pane and the
// program has no way to have known it would.
//
// The property that matters is not the number but what it promises —
// laid out at its minimum, nothing leaves the box; laid out below it,
// something does.
func TestMinWidthIsTheWidthEverythingStillFits(t *testing.T) {
	build := func() widget.Component {
		g := NewGrid()
		g.Cols = []Track{Auto(), Flex(1), Auto()}
		g.ColGap, g.RowGap = 10, 8
		g.Place(NewLabel("Key"), 0, 0)
		g.Place(NewSecretField("Passphrase"), 0, 1)
		g.Place(NewButton("Remove", nil), 0, 2)
		return g
	}
	c := build()
	atScale(t, c, 1)
	min := widget.MinWidthOf(c)
	if min <= 0 {
		t.Fatal("no minimum reported")
	}

	fits := func(w float32) bool {
		g := build()
		atScale(t, g, 1)
		sz := g.Measure(layout.Constraints{MaxW: w, MaxH: -1})
		g.Arrange(paintengine2d.XYWH(0, 0, w, sz.Y))
		for _, ch := range g.Children() {
			if ch.Bounds().Max.X > w+0.5 {
				return false
			}
		}
		return true
	}
	if !fits(min) {
		t.Errorf("at its own minimum of %v, something still leaves the box", min)
	}
	if fits(min - 12) {
		t.Errorf("%v is not the minimum — everything still fits 12 pixels below it", min)
	}
}

// A container cannot answer this by measuring itself: asked to fit in a
// pixel it reports the pixel it was constrained to. That is why
// MinWidther exists and why probing a container is wrong.
func TestMinWidthIsNotJustAMeasurement(t *testing.T) {
	g := NewGrid()
	g.Cols = []Track{Auto(), Flex(1)}
	g.Place(NewLabel("Key"), 0, 0)
	g.Place(NewButton("A button with a long label", nil), 0, 1)
	atScale(t, g, 1)

	measured := g.Measure(layout.Constraints{MaxW: 1, MaxH: -1}).X
	min := widget.MinWidthOf(g)
	if measured >= min {
		t.Errorf("measuring at 1px gave %v and the real minimum is %v — the measurement is the lie this exists to avoid",
			measured, min)
	}
}

// A button cannot fold, so its minimum is its whole width. A wrapping
// label can, so its minimum is far below the width it would like.
func TestMinWidthTellsFoldingFromNotFolding(t *testing.T) {
	b := NewButton("A button with a long label on it", nil)
	atScale(t, b, 1)
	natural := b.Measure(layout.Unbounded()).X
	if got := widget.MinWidthOf(b); got < natural-1 {
		t.Errorf("a button's minimum is %v, below its %v — it has no narrower form", got, natural)
	}

	l := NewLabel("a sentence with a good many words in it that can be folded up small")
	l.Wrap = true
	atScale(t, l, 1)
	wide := l.Measure(layout.Unbounded()).X
	if got := widget.MinWidthOf(l); got >= wide {
		t.Errorf("a wrapping label's minimum is %v, no less than its natural %v", got, wide)
	}
}

// A row's minimum is its children side by side; a column's is its widest
// child; a wrap's is its widest child, because it folds.
func TestMinWidthOfRowsColumnsAndWraps(t *testing.T) {
	kids := func() []widget.Component {
		return []widget.Component{
			NewButton("One", nil), NewButton("Two", nil), NewButton("Three", nil),
		}
	}
	row := NewRow(kids()...).WithGap(8)
	col := NewColumn(kids()...).WithGap(8)
	wr := NewWrap(kids()...)
	for _, c := range []widget.Component{row, col, wr} {
		atScale(t, c, 1)
	}

	rowMin := widget.MinWidthOf(row)
	colMin := widget.MinWidthOf(col)
	wrapMin := widget.MinWidthOf(wr)

	if rowMin <= colMin {
		t.Errorf("a row of three (%v) should need more than a column of three (%v)", rowMin, colMin)
	}
	if wrapMin != colMin {
		t.Errorf("a wrap folds, so its minimum (%v) should match a column's (%v)", wrapMin, colMin)
	}
}

// Pad does not shrink, so its insets are part of the floor.
func TestMinWidthIncludesPadding(t *testing.T) {
	inner := NewButton("Go", nil)
	p := NewPad(20, inner)
	p.RawSpacing = true
	atScale(t, p, 1)
	bare := NewButton("Go", nil)
	atScale(t, bare, 1)

	if got, want := widget.MinWidthOf(p), widget.MinWidthOf(bare)+40; got != want {
		t.Errorf("a padded button's minimum is %v, want %v", got, want)
	}
}
