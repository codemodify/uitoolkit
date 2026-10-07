package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// Every engine draws a push button's mark, and draws its label in the box left
// beside it.
//
// DrawButton used to take the label alone, so a widget that wanted a mark had
// to reserve a strip at *each* end and let the engine's centred label land
// between them — about 64 device pixels a button, half of it symmetry. The
// engine is handed the mark now (as DrawToolButton always was), which is the
// only way a pack can say where its own mark goes and in what ink.
//
// This is the matrix: every pack, not a sample, because a seam every engine
// implements is one every engine can forget.

// buttonShot draws one button and reports the image.
func buttonShot(t *testing.T, lk LookAndFeel, d ButtonDraw, w, h int) *paintengine2d.Image {
	t.Helper()
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(lk.Palette().Background)
	lk.DrawButton(ctx, paintengine2d.XYWH(0, 0, float32(w), float32(h)), StateNone, d)
	img.Touch()
	return img
}

// inkIn counts pixels inside r that differ from the same pixel of base.
func inkIn(a, base *paintengine2d.Image, r paintengine2d.Rect) int {
	n := 0
	for y := int(r.Min.Y); y < int(r.Max.Y) && y < a.Height; y++ {
		for x := int(r.Min.X); x < int(r.Max.X) && x < a.Width; x++ {
			r1, g1, b1, _ := a.PremulAt(x, y)
			r2, g2, b2, _ := base.PremulAt(x, y)
			if r1 != r2 || g1 != g2 || b1 != b2 {
				n++
			}
		}
	}
	return n
}

func TestEveryEngineDrawsAButtonsMark(t *testing.T) {
	const w, h = 160, 32
	var missing, unmoved []string
	for _, p := range ListThemes() {
		lk := p.Look()
		if lk == nil {
			continue
		}
		plain := buttonShot(t, lk, ButtonDraw{Label: "Save"}, w, h)
		marked := buttonShot(t, lk, ButtonDraw{Label: "Save", Icon: IconSave}, w, h)

		box := paintengine2d.XYWH(0, 0, w, h)
		ib := ButtonIconBox(lk, box, ButtonDraw{Label: "Save", Icon: IconSave})
		if ib.Empty() {
			continue
		}
		// The mark is drawn: the strip it goes in differs from the plain
		// button, whose strip holds only face.
		if inkIn(marked, plain, ib) < 8 {
			missing = append(missing, p.Name)
			continue
		}
		// And the label moved out of the way: the whole picture differs
		// somewhere to the right of the strip, because a label centred in the
		// button and one centred beside the mark are not in the same place.
		rest := paintengine2d.XYWH(ib.Max.X, 0, float32(w)-ib.Max.X, h)
		if inkIn(marked, plain, rest) < 4 {
			unmoved = append(unmoved, p.Name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d packs drew no mark: %v", len(missing), trim(missing))
	}
	if len(unmoved) > 0 {
		t.Errorf("%d packs left the label where it was, so the mark overlaps it or the width is wasted: %v",
			len(unmoved), trim(unmoved))
	}
}

func trim(v []string) []string {
	if len(v) > 8 {
		return append(v[:8:8], "...")
	}
	return v
}
