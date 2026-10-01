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
