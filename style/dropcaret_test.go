package style

import (
	"math"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// caretLook is a pack that paints the insertion caret itself.
type caretLook struct {
	LookAndFeel
	got      paintengine2d.Rect
	vertical bool
	calls    int
}

func (c *caretLook) DrawDropCaret(ctx *paintengine2d.Context, b paintengine2d.Rect, vertical bool) {
	c.got, c.vertical, c.calls = b, vertical, c.calls+1
}

// A pack that says how it wants the caret drawn is asked; one that says
// nothing gets the default, built from its own palette.
func TestDropCaretLookIsAsked(t *testing.T) {
	img := paintengine2d.NewImage(40, 40)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Transparent)
	box := paintengine2d.XYWH(4, 10, 32, 2)

	own := &caretLook{LookAndFeel: DarkLook()}
	DrawDropCaretOf(own, ctx, box, false)
	if own.calls != 1 || own.got != box || own.vertical {
		t.Fatalf("the pack's own caret was called %d times with %v (vertical=%v)", own.calls, own.got, own.vertical)
	}

	// The default paints the caret in the pack's accent.
	DrawDropCaretOf(DarkLook(), ctx, box, false)
	want := DarkLook().Palette().Accent.NRGBA()
	wr, wg, wb, _ := want.RGBA()
	gr, gg, gb, ga := img.At(20, 11).RGBA()
	if ga == 0 || gr != wr || gg != wg || gb != wb {
		t.Fatalf("the default caret is %v,%v,%v,%v, want the accent %v,%v,%v", gr, gg, gb, ga, wr, wg, wb)
	}

	// Nothing is drawn for an empty box, or with no context at all.
	DrawDropCaretOf(own, ctx, paintengine2d.Rect{}, false)
	DrawDropCaretOf(own, nil, box, false)
	if own.calls != 1 {
		t.Fatalf("an empty caret was still painted (%d calls)", own.calls)
	}
}

// The caret is a whole number of device pixels thick, centred on the gap
// and snapped to the pixel grid, at every scale a display can have.
func TestDropCaretRectSnapsToWholePixels(t *testing.T) {
	for _, scale := range []float32{1, 1.25, 1.5, 1.75, 2} {
		lk := themePack(t, "breeze").Look().setScale(scale)
		w := DropCaretThickness(lk)
		if w < 1 {
			t.Fatalf("scale %g: a %g px caret is thinner than a device pixel", scale, w)
		}
		for _, at := range []float32{0, 10, 10.5, 23.25} {
			h := DropCaretRect(lk, at, 0, 60, false)
			if h.Dy() != w || h.Dx() != 60 {
				t.Fatalf("scale %g at %g: caret %v, want %g tall and 60 wide", scale, at, h, w)
			}
			if h.Min.Y != float32(math.Round(float64(h.Min.Y))) {
				t.Fatalf("scale %g at %g: caret starts at %g, off the pixel grid", scale, at, h.Min.Y)
			}
			if d := (h.Min.Y+h.Max.Y)*0.5 - at; d > 0.5 || d < -0.5 {
				t.Fatalf("scale %g: the caret for the gap at %g is centred on %g", scale, at, (h.Min.Y+h.Max.Y)*0.5)
			}
			// The vertical caret is the same bar turned on its side.
			v := DropCaretRect(lk, at, 0, 60, true)
			if v.Dx() != w || v.Dy() != 60 {
				t.Fatalf("scale %g at %g: vertical caret %v, want %g wide and 60 tall", scale, at, v, w)
			}
		}
		// A band with no length at all is no caret.
		if !DropCaretRect(lk, 4, 10, 10, false).Empty() {
			t.Fatalf("scale %g: an empty band still produced a caret", scale)
		}
	}
}
