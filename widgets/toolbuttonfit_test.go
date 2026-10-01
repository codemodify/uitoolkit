package widgets

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// A tool button given exactly the width it asked for must draw its whole
// label. Measure reserves style.ControlFontOf(lk, RoleTool); the engine
// draws with whatever face it likes and clips to the button's box. Where
// the two disagree the label is cut, and the cut is silent — adwaita
// draws libadwaita's semibold and had declared no control font at all,
// so a Thunderbird-shaped toolbar read "Get Message", "Reply Al",
// "Forwarc", "Delet".
//
// Asking a face how wide it is would only re-ask the question the bug is
// in. So this renders: the button is drawn twice at one width, once with
// the label and once without, and the difference between the two images
// is the label's own ink. Do that at the measured width and again with
// 80px to spare, and compare how wide that ink is. A label that fits
// draws the same glyphs either way — an engine that centres moves the
// ink but does not change its width. One that clips or elides makes it
// narrower.
func TestToolButtonDrawsTheLabelItMeasured(t *testing.T) {
	packs := style.ListThemes()
	if len(packs) == 0 {
		t.Skip("no packs in this build")
	}
	const label = "Get Messages"
	for _, p := range packs {
		lk := style.Appearance{Name: p.Name, Theme: p.Palette}.Look()
		b := NewToolButton(label, style.IconNone, nil)
		b.SetHost(&fakeWindow{look: lk})
		sz := b.Measure(layout.Constraints{MaxW: 1e6, MaxH: 1e6})

		tight := labelInkWidth(lk, label, sz.X, sz.Y)
		roomy := labelInkWidth(lk, label, sz.X+80, sz.Y)
		if tight < 0 || roomy < 0 {
			t.Errorf("%s: the label left no ink at all", p.Name)
			continue
		}
		// One pixel of slack: a centred label lands on a different
		// sub-pixel offset in the two renders, and anti-aliasing puts
		// the edge column either side of the boundary.
		if tight < roomy-1 {
			t.Errorf("%s: measured %.1fpx and drew %dpx of label, but %dpx with room to spare — the engine draws a wider face than ControlFontOf(RoleTool) reports",
				p.Name, sz.X, tight, roomy)
		}
	}
}

// labelInkWidth draws the tool button with and without its label and
// returns how many pixels wide the difference is, or -1 for no ink.
func labelInkWidth(lk style.LookAndFeel, label string, w, h float32) int {
	iw, ih := int(w+2), int(h+2)
	draw := func(text string) *paintengine2d.Image {
		img := paintengine2d.NewImage(iw, ih)
		ctx := paintengine2d.NewContext(img)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(iw), float32(ih)), paintengine2d.Fill(lk.Palette().Background))
		lk.DrawToolButton(ctx, paintengine2d.XYWH(0, 0, w, h), style.StateNone|style.StateAutoRaise, text, style.IconNone)
		img.Touch()
		return img
	}
	with, without := draw(label), draw("")
	lo, hi := -1, -1
	for x := 0; x < iw; x++ {
		for y := 0; y < ih; y++ {
			r1, g1, b1, _ := with.At(x, y).RGBA()
			r2, g2, b2, _ := without.At(x, y).RGBA()
			d := absDiff32(r1, r2) + absDiff32(g1, g2) + absDiff32(b1, b2)
			if d/3/257 > 4 {
				if lo < 0 {
					lo = x
				}
				hi = x
				break
			}
		}
	}
	if lo < 0 {
		return -1
	}
	return hi - lo + 1
}

func absDiff32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
