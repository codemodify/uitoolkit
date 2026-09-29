package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
)

// longLabel is a wrapping label whose height depends entirely on the
// width it is given, which is what every case here turns on.
func longLabel() *Label {
	l := NewLabel("A very long message about a program path that will certainly " +
		"wrap when the card is narrower than it was measured for, several times over.")
	l.Wrap = true
	return l
}

// The card is measured at 0.8 of the overlay and arranged at anything
// from the 280-pixel floor to 0.92 of it, so the two widths routinely
// differ. A card narrowed by that cap wrapped into more lines than were
// measured, and its last lines and its buttons were drawn outside it,
// where clicks on them missed.
func TestOverlayCardIsTallEnoughForTheWidthItGets(t *testing.T) {
	for _, w := range []float32{360, 420, 520, 700, 900} {
		col := NewColumn().WithGap(6)
		col.Add(longLabel())
		btn := NewButton("OK", nil)
		col.Add(btn)

		o := NewOverlay(col)
		atScale(t, o, 1)
		o.Arrange(paintengine2d.XYWH(0, 0, w, 600))

		card := col.Bounds()
		if card.Empty() {
			t.Fatalf("width %v: the card was not arranged", w)
		}
		// The button is the last thing in the card; it has to be inside it.
		b := btn.Bounds()
		if b.Max.Y > card.Dy()+0.5 {
			t.Errorf("width %v: the button ends at %v, past the card's %v",
				w, b.Max.Y, card.Dy())
		}
		// And the card holds what the column needs at the width it got.
		need := col.Measure(layout.Constraints{MaxW: card.Dx(), MaxH: -1})
		if card.Dy()+0.5 < need.Y && card.Dy() < 600*0.88-0.5 {
			t.Errorf("width %v: card %v tall, content needs %v at %v wide",
				w, card.Dy(), need.Y, card.Dx())
		}
	}
}

// A flexible column was frozen at its children's *unbounded* width when
// there was not room for it. That is right for a child that cannot
// shrink — a button is as wide as its label — and wrong for one that
// can: a wrapping label with nothing to constrain it reports the width
// of its whole text on one line, 1453 pixels for a sentence. Its column
// took all 1453 of the 400 there were, so the form ran off its own right
// edge, and its height came back as the single line nobody would ever
// see.
func TestFormFitsAWrappingFieldInTheWidthItIsGiven(t *testing.T) {
	const offered = 400
	f := NewForm()
	lbl := longLabel()
	f.AddRow("To", lbl)
	atScale(t, f, 1)

	got := f.Measure(layout.Constraints{MaxW: offered, MaxH: -1})
	if got.X > offered+0.5 {
		t.Errorf("measured %v wide inside %v", got.X, offered)
	}
	// One line is what it used to report; the wrapped text is taller.
	alone := lbl.Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	if got.Y <= alone.Y+1 {
		t.Errorf("measured %v tall — a single line — for text that must fold at %v wide",
			got.Y, offered)
	}

	f.Arrange(paintengine2d.XYWH(0, 0, offered, got.Y))
	for _, c := range f.Children() {
		if b := c.Bounds(); b.Max.X > offered+0.5 {
			t.Errorf("a field reaches %v, past the form's %v", b.Max.X, offered)
		}
		if b := c.Bounds(); b.Max.Y > got.Y+0.5 {
			t.Errorf("a field reaches %v, past the form's %v tall", b.Max.Y, got.Y)
		}
	}
}

// A child that cannot shrink still keeps its width: the floor is what it
// can be squeezed to, not what the layout would like it to be.
func TestFormDoesNotSqueezeWhatCannotShrink(t *testing.T) {
	f := NewForm()
	btn := NewButton("A button with a long label on it", nil)
	f.AddRow("Action", btn)
	atScale(t, f, 1)

	natural := btn.Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	f.Arrange(paintengine2d.XYWH(0, 0, 120, 200))
	if got := btn.Bounds().Dx(); got < natural.X-1 {
		t.Errorf("the button was squeezed to %v; it cannot go below %v", got, natural.X)
	}
}

// A grid with room to spare still reports its natural width, so it is
// not greedy in a row that would otherwise leave it alone.
func TestGridIsNotGreedy(t *testing.T) {
	f := NewForm()
	f.AddRow("To", NewLabel("ada@example.com"))
	atScale(t, f, 1)

	got := f.Measure(layout.Constraints{MaxW: 2000, MaxH: -1})
	if got.X >= 2000 {
		t.Errorf("a small grid asked for %v of the 2000 it was offered", got.X)
	}
}

// SetBounds snaps a component's bounds to the pixel grid, so a measure
// taken at a fractional width and a layout done at the rounded one
// disagreed: one fits a child on the line the other folds, and the
// difference showed as a line's height of empty space under the last row.
func TestWrapMeasuresAndArrangesTheSameWay(t *testing.T) {
	for _, w := range []float32{300, 300.2, 300.4, 300.5, 300.6, 300.9} {
		wr := NewWrap()
		for i := 0; i < 7; i++ {
			wr.Add(NewButton("Button", nil))
		}
		atScale(t, wr, 1)
		sz := wr.Measure(layout.Constraints{MaxW: w, MaxH: -1})
		wr.Arrange(paintengine2d.XYWH(0, 0, w, sz.Y))

		var lowest float32
		for _, c := range wr.Children() {
			if b := c.Bounds(); b.Max.Y > lowest {
				lowest = b.Max.Y
			}
		}
		if d := sz.Y - lowest; d > 1 || d < -1 {
			t.Errorf("width %v: measured %v tall, laid out to %v (%v adrift)", w, sz.Y, lowest, d)
		}
	}
}
