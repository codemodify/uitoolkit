package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A label can carry a mark, and the mark is drawn rather than written.
//
// The toolkit's own rule is that arrows, checks and chevrons are icons,
// because no bundled face has them — a status bar that wrote "↑2 ↓0"
// drew two tofu boxes. Before Label.Icon the only widget that could put
// a mark on screen was a button, which is wrong where nothing is
// clickable.
func TestLabelIconIsMeasuredAndDrawn(t *testing.T) {
	lk := style.DarkLook()
	plain, marked := NewLabel("2"), NewIconLabel(style.IconArrowUp, "2")
	for _, l := range []*Label{plain, marked} {
		l.SetHost(&fakeWindow{look: lk})
	}
	bare := plain.Measure(layout.Unbounded())
	wide := marked.Measure(layout.Unbounded())
	if wide.X <= bare.X {
		t.Fatalf("an icon added no width: %.1f with, %.1f without", wide.X, bare.X)
	}
	if wide.Y != bare.Y {
		t.Errorf("an icon changed the line height: %.1f vs %.1f", wide.Y, bare.Y)
	}

	// The mark is ink, not reserved space: draw both and require the
	// icon's own column to differ.
	draw := func(l *Label) *paintengine2d.Image {
		img := paintengine2d.NewImage(int(wide.X)+4, int(wide.Y)+4)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(img.Width), float32(img.Height)), paintengine2d.Fill(lk.Palette().Background))
		l.Arrange(paintengine2d.XYWH(0, 0, wide.X, wide.Y))
		l.Paint(ctx)
		img.Touch()
		return img
	}
	if n := visiblyDifferentPx(draw(plain), draw(marked)); n < 8 {
		t.Errorf("the icon drew %d pixels — nothing visible", n)
	}

	// A label with a mark and no text is just the mark, and still has
	// width: a status bar puts one there on its own.
	only := NewIconLabel(style.IconArrowDown, "")
	only.SetHost(&fakeWindow{look: lk})
	if sz := only.Measure(layout.Unbounded()); sz.X < 4 {
		t.Errorf("a mark on its own measured %.1f wide", sz.X)
	}
}

func visiblyDifferentPx(a, b *paintengine2d.Image) int {
	n := 0
	for y := 0; y < a.Height && y < b.Height; y++ {
		for x := 0; x < a.Width && x < b.Width; x++ {
			r1, g1, b1, _ := a.At(x, y).RGBA()
			r2, g2, b2, _ := b.At(x, y).RGBA()
			if (absDiff32(r1, r2)+absDiff32(g1, g2)+absDiff32(b1, b2))/3/257 > 4 {
				n++
			}
		}
	}
	return n
}

// A wrapping label with a mark is measured at the width its text will have.
//
// Measure wrapped at the whole width and added the mark's room afterwards;
// Paint took the mark's room off first and wrapped at what was left. So the
// text wrapped to more lines than the box was measured for, and the lines,
// centred in it, lost the first and the last — a sender warning, a
// signed-and-encrypted line, a "Not available: ..." — at ordinary widths.
func TestAWrappingIconLabelIsMeasuredAtTheWidthItsTextGets(t *testing.T) {
	const text = "This message came from a server that is not the one its sender claims, " +
		"which is usually a mistake and occasionally not."
	lk := style.DarkLook()
	for _, w := range []float32{180, 240, 320, 460} {
		l := NewIconLabel(style.IconWarning, text)
		l.Wrap = true
		l.SetHost(&fakeWindow{look: lk})
		got := l.Measure(layout.Loose(w, 1000))
		l.Arrange(paintengine2d.XYWH(0, 0, got.X, got.Y))

		// The invariant, asked of the box Measure actually returned rather
		// than of a width the test works out for itself: the lines Paint
		// wraps to must fit in it. Either half of the bug — wrapping at a
		// width the text will not get, or losing the last fraction of a
		// pixel to layout rounding — shows up as lines that do not.
		f := l.font()
		painted := len(l.layoutLines(f, l.wrapWidth(f, l.LocalBounds().Dx())))
		need := f.Height()*float32(painted) + 2
		if got.Y < need-0.01 {
			t.Errorf("at %.0f wide: measured a box %.2f tall, paints %d lines needing %.2f",
				w, got.Y, painted, need)
		}
	}
}

// And the mark still gets its room: the text is not simply given the whole
// width back.
func TestAWrappingIconLabelKeepsRoomForItsMark(t *testing.T) {
	lk := style.DarkLook()
	const text = "a warning that runs on for long enough to wrap at any sensible width"
	plain, marked := NewLabel(text), NewIconLabel(style.IconWarning, text)
	for _, l := range []*Label{plain, marked} {
		l.Wrap = true
		l.SetHost(&fakeWindow{look: lk})
	}
	bare := plain.Measure(layout.Loose(240, 1000))
	wide := marked.Measure(layout.Loose(240, 1000))
	if wide.Y < bare.Y {
		t.Errorf("the marked label is shorter (%.0f) than the plain one (%.0f): the mark took no room from the text", wide.Y, bare.Y)
	}
}
