package style

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// An application that asks for its chrome to run to the window's edges
// gets that from every pack, not from the one that happened to be tested.
//
// SetBorderless zeroed the *content* insets, so content ran to the outer
// edge while the frame went on painting a border underneath it. You saw
// that border wherever nothing opaque covered it — the tab strip and the
// page — and not where the tool bar did, which read as the tool bar
// overstepping the window. Nothing was overstepping; the border should
// not have been there.
//
// Nineteen call sites draw that ring, each behind its own guard. One
// guard per engine is one drift per engine, so the decision is in the
// helper they all call and this holds every pack to it.
func TestBorderlessWindowsHaveNoBorderInAnyPack(t *testing.T) {
	packs := ListThemes()
	if len(packs) == 0 {
		t.Skip("no packs in this build")
	}
	const w, h = 160, 120
	for _, p := range packs {
		c := Appearance{Name: p.Name, Theme: p.Palette}.Look()
		// A skin's frame is its art — the sprite sheet *is* the window —
		// so there is no border in it to drop. That is the third way to
		// build with this toolkit and it answers this question
		// differently on purpose (docs/building.md).
		if _, isSkin := c.eng().(skinEngine); isSkin {
			continue
		}
		// Two eras paint a structural frame they do not declare as a
		// border: the Amiga's drag bar sits in a plate, and BeOS's window
		// is a tab on a box whose sides are part of the box. Written down
		// rather than filtered by a rule, because each is a decision.
		switch p.Name {
		case "amiga13", "beos":
			continue
		}
		// Only looks whose frame is a *line*. An era whose window is a
		// sculpted bevel — Motif, System 7, the Amiga, NeXT — is drawing
		// its frame, not a border on one, and a window of that era
		// without it would be shapeless. SetBorderless drops a border;
		// it does not dismantle a frame.
		if b := DecorationOf(c, DecorationState{Active: true}).Border; b.Left > 2 || b.Right > 2 {
			continue
		}
		draw := func(noBorder bool) *paintengine2d.Image {
			img := paintengine2d.NewImage(w, h)
			ctx := paintengine2d.NewContext(img)
			// A ground no frame paints, so anything at the edge is the
			// frame's doing and not the test's.
			ctx.DrawRect(paintengine2d.XYWH(0, 0, w, h), paintengine2d.Fill(paintengine2d.RGB(0, 1, 0)))
			st := DecorationState{Active: true, Custom: true, NoBorder: noBorder}
			f := DecorationFrame{
				Window:  paintengine2d.XYWH(0, 0, w, h),
				Caption: paintengine2d.XYWH(0, 0, w, 28),
			}
			DrawDecorationOf(c, ctx, f, st)
			img.Touch()
			return img
		}
		framed, bare := draw(false), draw(true)

		// Down the left and right edges, below the caption: a bordered
		// frame paints something there and a borderless one leaves the
		// ground showing.
		edge := func(img *paintengine2d.Image) int {
			n := 0
			for y := 40; y < h-4; y++ {
				for _, x := range []int{0, w - 1} {
					r, g, b, a := img.At(x, y).RGBA()
					if a > 0x2000 && !(r>>8 == 0 && g>>8 == 255 && b>>8 == 0) {
						n++
					}
				}
			}
			return n
		}
		if got := edge(bare); got > 0 {
			// Only a complaint where the pack draws one at all: a pack
			// with no border has nothing to drop.
			if edge(framed) > 0 {
				t.Errorf("%s: a borderless window still paints %d pixels down its sides", p.Name, got)
			}
		}
	}
}
