package uitest

import (
	"fmt"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// CheckToolBarGaps reports when adjacent labeled tools are closer than
// ToolItemGap — or, after a word naming the control beside it, than the
// half gap that ties the two together ([widgets.ToolLabel]).
func CheckToolBarGaps(t *widgets.ToolBar) error {
	if t == nil {
		return nil
	}
	items := t.Items()
	for i := 1; i < len(items); i++ {
		a, b := items[i-1], items[i]
		if a == nil || b == nil || a.Sep || b.Sep {
			continue
		}
		ra, rb := t.ItemRect(i-1), t.ItemRect(i)
		if ra.Empty() || rb.Empty() {
			continue // dropped: the bar is too narrow to carry it
		}
		want := style.ToolItemGap
		if a.Label {
			want *= 0.5
		}
		if gap := rb.Min.X - ra.Max.X; gap < want-0.51 {
			return fmt.Errorf("tools %d/%d gap %v < %v", i-1, i, gap, want)
		}
	}
	return nil
}

// CheckInkInsideBounds paints c and reports ink outside its arranged box
// (plus slop for focus rings that intentionally outset).
func CheckInkInsideBounds(s *Session, c widget.Component, slop int) error {
	if s == nil || c == nil {
		return nil
	}
	img := s.Paint()
	box := widget.DeviceBounds(c)
	x0 := int(box.Min.X) - slop
	y0 := int(box.Min.Y) - slop
	x1 := int(box.Max.X) + slop
	y1 := int(box.Max.Y) + slop
	n := 0
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			if x >= x0 && x < x1 && y >= y0 && y < y1 {
				continue
			}
			_, _, _, a := img.PremulAt(x, y)
			if a > 40 {
				n++
			}
		}
	}
	if n > 8 {
		return fmt.Errorf("ink escaped bounds %+v (outside=%d)", box, n)
	}
	return nil
}

// ColorDiff counts pixels whose RGBA differs by more than slop.
func ColorDiff(a, b *paintengine2d.Image, slop int) int {
	if a == nil || b == nil {
		return 0
	}
	h, w := a.Height, a.Width
	if b.Height < h {
		h = b.Height
	}
	if b.Width < w {
		w = b.Width
	}
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			ar, ag, ab, aa := a.PremulAt(x, y)
			br, bg, bb, ba := b.PremulAt(x, y)
			if abs(int(ar)-int(br))+abs(int(ag)-int(bg))+abs(int(ab)-int(bb))+abs(int(aa)-int(ba)) > slop {
				n++
			}
		}
	}
	return n
}

// CheckMenuHoverBordered reports when a hovered menu row is only inverted
// text instead of an Office XP bordered fill (gutter + label).
func CheckMenuHoverBordered(img *paintengine2d.Image, row paintengine2d.Rect, lk style.LookAndFeel) error {
	if img == nil || lk == nil || row.Empty() {
		return fmt.Errorf("empty menu hover sample")
	}
	p := style.ResolveMenuChrome(lk.Palette())
	inner := row.Inset(3)
	if inner.Empty() {
		inner = row.Inset(1)
	}
	hr, hg, hb := RGB8(p.MenuHover)
	br, bg, bb := RGB8(p.MenuHoverBorder)
	fill := countNearRGB(img, inner, hr, hg, hb, 48)
	if fill < 8 {
		return fmt.Errorf("hovered row %+v lacks MenuHover fill (near=%d)", row, fill)
	}
	border := countNearRGB(img, row, br, bg, bb, 42)
	if border < 4 {
		return fmt.Errorf("hovered row %+v lacks MenuHoverBorder (near=%d)", row, border)
	}
	// A text-invert highlight would not paint the icon gutter.
	// MenuBar titles have no gutter; classic-3d uses a solid select, so
	// sample the row center. Luna / flat packs still check the icon column.
	ch := style.MenuChromeFor(lk)
	gx := int(row.Min.X + ch.CheckCol()*0.45)
	gy := int((row.Min.Y + row.Max.Y) * 0.5)
	if tok := style.LookTokens(lk); tok.Bevel == style.BevelClassic3D {
		// Solid navy select + inverted label: avoid sampling glyph ink.
		gx = int(row.Min.X + 4)
	}
	if ColorDist(img, gx, gy, p.MenuHover) >= ColorDist(img, gx, gy, p.MenuGutter) {
		return fmt.Errorf("icon gutter %d,%d is not MenuHover fill", gx, gy)
	}
	return nil
}

// CountFocusPixels counts pixels near Palette.Focus in r.
func CountFocusPixels(img *paintengine2d.Image, r paintengine2d.Rect, lk style.LookAndFeel) int {
	if img == nil || lk == nil {
		return 0
	}
	fr, fg, fb := RGB8(lk.Palette().Focus)
	return countNearRGB(img, r, fr, fg, fb, 48)
}

func countNearRGB(img *paintengine2d.Image, r paintengine2d.Rect, wr, wg, wb, slop int) int {
	x0 := clampInt(int(r.Min.X), 0, img.Width)
	y0 := clampInt(int(r.Min.Y), 0, img.Height)
	x1 := clampInt(int(r.Max.X), 0, img.Width)
	y1 := clampInt(int(r.Max.Y), 0, img.Height)
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			cr, cg, cb, a := img.PremulAt(x, y)
			if a < 20 {
				continue
			}
			if abs(int(cr)-wr) <= slop && abs(int(cg)-wg) <= slop && abs(int(cb)-wb) <= slop {
				n++
			}
		}
	}
	return n
}
