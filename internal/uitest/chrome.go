package uitest

import (
	"fmt"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// CheckToolBarGaps reports when adjacent labeled tools are closer than ToolItemGap.
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
		gap := t.ItemRect(i).Min.X - t.ItemRect(i-1).Max.X
		if gap < style.ToolItemGap-0.51 {
			return fmt.Errorf("tools %d/%d gap %v < %v", i-1, i, gap, style.ToolItemGap)
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
