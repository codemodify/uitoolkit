package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// A component's bounds are snapped to the pixel grid by SetBounds, which
// rounds each edge on its own. A chip that asked for a fractional width
// could therefore be handed back up to a pixel less than it asked for —
// and Paint answers a box a hair too small by eliding the label. So a
// chip cut its own text short, on its own measurement, for want of a
// fraction of a pixel.
func TestTokenMeasureIsWholePixels(t *testing.T) {
	loose := layout.Constraints{MaxW: -1, MaxH: -1}
	for _, text := range []string{
		"a", "ada@example.com", "Ada Lovelace", "grace.hopper@navy.mil",
		"wwwwwwwwww", "iiiiiiiiii", "Ada", "x@y.z",
	} {
		for _, removable := range []bool{true, false} {
			tok := NewToken(text)
			tok.Removable = removable
			sz := tok.Measure(loose)
			if sz.X != float32(int(sz.X)) {
				t.Errorf("%q removable=%v: width %v is not whole", text, removable, sz.X)
			}
		}
	}
}

// The real property: after the box has been through the pixel grid, the
// label still fits in it, so Paint does not elide.
func TestTokenLabelFitsItsOwnBox(t *testing.T) {
	loose := layout.Constraints{MaxW: -1, MaxH: -1}
	// Fractional origins are what a flowed field hands a chip, and they
	// are what used to round the far edge down.
	for _, x := range []float32{0, 0.1, 0.25, 0.4, 0.5, 0.6, 0.75, 0.9} {
		for _, text := range []string{"ada@example.com", "Ada Lovelace", "w", "iiii"} {
			tok := NewToken(text)
			sz := tok.Measure(loose)
			tok.Arrange(paintengine2d.XYWH(x, 0, sz.X, sz.Y))

			b := tok.LocalBounds()
			room := b.Dx() - tok.pad()*2 - tok.crossW()
			if adv := tok.Look().Font().Advance(text); adv > room {
				t.Errorf("x=%v %q: label needs %v, box leaves %v — Paint would elide it",
					x, text, adv, room)
			}
		}
	}
	_ = widget.Component(NewToken("x"))
}

// A chip is the look's shape, not always a capsule. Zero is a radius —
// and the commonest one, since 41 of the packs draw square corners — so
// testing for one above zero read every square pack as having no opinion
// and gave it the rounded shape it never drew.
func TestTokenRadiusFollowsTheLook(t *testing.T) {
	tok := NewToken("ada@example.com")
	loose := layout.Constraints{MaxW: -1, MaxH: -1}
	sz := tok.Measure(loose)
	tok.Arrange(paintengine2d.XYWH(0, 0, sz.X, sz.Y))
	h := tok.LocalBounds().Dy()

	for _, tc := range []struct {
		name   string
		radius float32
		want   float32
	}{
		{"square", 0, 0},
		{"slight", 3, 3},
		{"round", 1e6, h * 0.5}, // never more than a capsule
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tokenRadius(h, tc.radius); got != tc.want {
				t.Errorf("radius %v of a %v-tall chip = %v, want %v",
					tc.radius, h, got, tc.want)
			}
		})
	}
}
