package widgets

import (
	"strings"
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A two-row text view's scroll bar, in the look the report was written
// against. Plastik asks for three step buttons at one end
// (ArrowsTripleEnd), which a 47-pixel bar has no room for.
func shortArea(t *testing.T, text string, rows int) (*TextArea, style.LookAndFeel) {
	t.Helper()
	pack, ok := style.LoadTheme("plastik")
	if !ok {
		t.Skip("the plastik engine is not in this build")
	}
	lk := pack.Look()
	ta := NewMonoTextView(text, "")
	ta.MinRows = rows
	ta.SetLook(lk)
	ta.SetHost(&host{})
	sz := ta.Measure(layout.Loose(300, 1000))
	ta.Arrange(paintengine2d.XYWH(0, 0, 300, sz.Y))
	return ta, lk
}

// A bar too short for its buttons gives up the buttons, not the track.
//
// It used to shrink them — three 13-pixel arrows in a 47-pixel bar — and what
// was left of the track, 8 pixels, was shorter than the look's smallest
// thumb. The thumb was clamped to the track, filled it, sat in the same place
// at every offset and could not be dragged: the bar said neither that there
// was more nor where the view was. What the person saw was three small arrows,
// one apart from the other two.
func TestAShortScrollBarDropsItsButtonsRatherThanSqueezeThem(t *testing.T) {
	ta, lk := shortArea(t, "one line\ntwo lines\nthree lines\nfour lines", 2)
	sp := ta.vparts()
	if sp.Bar.Empty() {
		t.Fatal("the bar is not there at all")
	}
	st := style.ScrollBarStyleOf(lk)
	// Either the buttons are full size or they are gone.
	for name, b := range map[string]paintengine2d.Rect{"Dec": sp.Dec, "DecEnd": sp.DecEnd, "Inc": sp.Inc} {
		if b.Empty() {
			continue
		}
		if b.Dy() < st.ArrowLen-0.5 {
			t.Errorf("%s is %g tall, squeezed from the look's %g", name, b.Dy(), st.ArrowLen)
		}
	}
	// And the thumb can move: that is what the bar is for.
	top := sp.Thumb
	ta.ScrollTo(ta.MaxOffset())
	bot := ta.vparts().Thumb
	if top.Empty() || bot.Empty() {
		t.Fatalf("there is no thumb (top %v, bottom %v) in a view with more text than room", top, bot)
	}
	if bot.Min.Y <= top.Min.Y+0.5 {
		t.Errorf("the thumb is at %g at the top and %g at the bottom: it cannot move", top.Min.Y, bot.Min.Y)
	}
	if sp.Track.Dy() < st.MinThumb {
		t.Errorf("the track is %g, shorter than the look's smallest thumb %g", sp.Track.Dy(), st.MinThumb)
	}
}

// A bar whose track cannot hold a thumb that moves shows no thumb. One
// clamped to fill its track says nothing — not that there is more, not where
// the view is — and cannot be dragged.
func TestABarTooShortForAThumbShowsNone(t *testing.T) {
	pack, ok := style.LoadTheme("plastik")
	if !ok {
		t.Skip("the plastik engine is not in this build")
	}
	st := style.ScrollBarStyleOf(pack.Look())
	view := paintengine2d.XYWH(0, 0, 200, st.MinThumb-2)
	sp := style.ScrollGeometry(pack.Look(), view, true, 1000, view.Dy(), 10, false)
	if !sp.Thumb.Empty() {
		t.Errorf("a %g-pixel bar has a thumb of %g, which cannot move in a track of %g",
			view.Dy(), sp.Thumb.Dy(), sp.Track.Dy())
	}
}

// No text runs under the bar.
//
// The lines wrapped to the field's whole inner width while the bar was
// painted over its right edge, so the end of every full line was hidden —
// "io.gith" and the rest of a file name — with no way to scroll it clear,
// because the bar was not part of the arithmetic anywhere.
func TestTextDoesNotWrapUnderTheScrollBar(t *testing.T) {
	ta, lk := shortArea(t, strings.Repeat("a sentence long enough to wrap in this field. ", 6), 2)
	sp := ta.vparts()
	if sp.Bar.Empty() {
		t.Skip("there is no bar to run under")
	}
	wrap := ta.wrapWidth()
	if wrap > sp.Bar.Min.X+0.5 {
		t.Errorf("lines wrap to %g, past the bar's left edge at %g", wrap, sp.Bar.Min.X)
	}
	// And the gutter really is the look's.
	in := ta.inner()
	if got, want := in.Dx()-wrap, style.ScrollGutter(lk); got < want-0.5 {
		t.Errorf("the text gives up %g to the bar, less than the look's gutter of %g", got, want)
	}
}

// A field with nothing to scroll gives up nothing: a two-line note in a
// three-row box wraps to the whole width.
func TestTextWithNoBarKeepsTheWholeWidth(t *testing.T) {
	ta, _ := shortArea(t, "one line\ntwo lines", 4)
	if got, want := ta.wrapWidth(), ta.inner().Dx(); got < want-0.5 {
		t.Errorf("a field that does not overflow wraps to %g of %g", got, want)
	}
}
