package style

import "github.com/codemodify/paintengine2d"

// The BeOS R5 window frame (see engine_beos.go): a stacked frame — the
// yellow tab above the window's five-pixel border. The tab is #FFC800 with
// #FFE400 along its top and left and #AF7B00 and #606060 down its right, the
// close box four pixels in from its left corner, the bold black title after
// it and the zoom box (two overlapping squares) four pixels in from its
// right; a backdrop window's tab goes grey and its title with it. The border
// is five pixels read from the outside in — #989898, white, the panel,
// #888888, #989898 along the top and left, and the reverse down the bottom
// and right — and its last row is the tab's own bottom line. Nothing is
// rounded and nothing casts a shadow: the app_server drew the tab straight
// onto the desktop.
//
// The tab is the window's real outline, which is the thing about a BeOS
// window everybody remembers. BeOS sized it to its contents — close box,
// title, zoom box — and left the rest of the top edge to the desktop, so a
// stack of windows showed a row of tabs. That needs two halves, and this
// engine asks for both: [DecorationSpec.CaptionFits], so the toolkit gives
// the caption band the width of its own contents instead of the window's,
// and [WindowShapeEngine], so the window is cut to the tab above that band
// and to its full width below it. Both are dropped in the states that drop
// every other silhouette — maximized and tiled — and there the tab is the
// full-width band it was before, which is what a window filling a box the
// desktop chose has to be.
//
// Colours and lengths are engine_beos.go's, measured off BeOS R5
// screenshots at 1:1.

// beTabMin is the least width a tab may be fitted to, in cells: the width
// engine_beos.go already demands before it will draw a close box at all.
const beTabMin = 40

func (beosEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	u := rpU(l)
	th := beTabH(l)
	return DecorationSpec{
		Stacked: true,
		// The tab stands on the window's corner, not inside its frame: the
		// caption runs from the window's left edge, which puts the close
		// box four pixels in from the tab's corner, and only the content
		// is inset by the five-pixel border.
		Border:        Insets{Bottom: beFrame * u},
		ContentBorder: Insets{Right: beFrame * u, Bottom: beFrame * u, Left: beFrame * u},
		Split:         true,
		// The tab, whose last row is the frame's top line, and the frame.
		Caption:   float32(th-1+beFrame) * u,
		Button:    paintengine2d.Pt(14*u, 14*u),
		ButtonGap: 3 * u,
		ButtonPad: Insets{Top: float32(beBoxY(l)) * u, Right: 4 * u, Left: 4 * u},
		// R5's tab: the title 19 pixels after the close box and 17 before
		// the zoom box, 10 and 12 from the tab's ends where a box is missing
		// (DrawWindowFrame's own layout).
		TitleRoom: TitleRoom{Lead: 19 * u, Trail: 17 * u, LeadEdge: 10 * u, TrailEdge: 12 * u},
		Layout:    "close:maximize",
		// The tab is as wide as its contents; DecorationOf drops this
		// wherever it drops the silhouette that goes with it.
		CaptionFits: true,
	}
}

// beTabCells is how many cells wide the tab is on grid g.
//
// It is the caption band's own width wherever the toolkit fitted that band
// to its contents, and the whole window otherwise — and it asks DecorationOf
// which of the two this state is, so the tab that is painted and the tab
// that takes the pointer can never disagree about a maximized window.
func beTabCells(l *Classic, g rpGrid, f DecorationFrame, st DecorationState) int {
	if f.Caption.Empty() || !DecorationOf(l, st).CaptionFits {
		return g.w
	}
	n := int((f.Caption.Max.X-f.Window.Min.X)/g.u + 0.5)
	return min(max(n, min(beTabMin, g.w)), g.w)
}

// WindowShape is the tab and the body under it: the window's top edge is the
// window only as far as the tab runs, and the desktop shows through the rest
// of it. A tab that fills the width is no silhouette at all and says so —
// the window is then the rectangle it has always been, and costs what one
// costs.
func (beosEngine) WindowShape(l *Classic, f DecorationFrame, st DecorationState) *Silhouette {
	u := rpU(l)
	g := rpGridOf(f.Window, u)
	th := beTabH(l)
	if g.w < 2*beFrame+8 || g.h < th+2 {
		return nil
	}
	cells := beTabCells(l, g, f, st)
	if cells >= g.w {
		return nil
	}
	// The tab's last row *is* the frame's top line, so the body starts
	// there: one rectangle's bottom edge and the other's top edge are the
	// same row of pixels, and the union has no seam in it.
	top := float32(th-1) * u
	return SilhouetteOfRects(
		SilhouetteRect{Rect: paintengine2d.XYWH(f.Window.Min.X, f.Window.Min.Y, float32(cells)*u, top)},
		SilhouetteRect{Rect: paintengine2d.XYWH(f.Window.Min.X, f.Window.Min.Y+top, f.Window.Dx(), max(f.Window.Dy()-top, 0))},
	)
}

func (beosEngine) DrawDecoration(l *Classic, ctx *paintengine2d.Context, f DecorationFrame, st DecorationState) {
	c := beColorsOf(l)
	u := rpU(l)
	th := beTabH(l)
	g := rpGridAt(ctx, f.Window, u)
	if g.w < 2*beFrame+8 || g.h < th+2 {
		return
	}
	ctx.DrawRect(paintengine2d.Rect{Min: f.Window.Min, Max: paintengine2d.Pt(f.Window.Max.X, f.Caption.Max.Y)}, paintengine2d.Fill(c.panel))
	var kTab, kHi, kLo, kD2, kD4, kWhite, kMid rpInk
	w := beTabCells(l, g, f, st)
	kTab.cells(g, 2, 2, w-4, th-3)
	kD2.cells(g, 0, 0, w-1, 1)
	kD2.cells(g, 0, 1, 1, th-2)
	kD4.cells(g, w-1, 0, 1, th-1)
	kHi.cells(g, 1, 1, w-3, 1)
	kHi.cells(g, 1, 2, 1, th-3)
	kLo.cells(g, w-2, 1, 1, th-2)
	if !st.Maximized {
		// The five-pixel border, read from the outside in; the tab's last row
		// is its top line.
		fy, fh := th-1, g.h-th+1
		tl := [beFrame]*rpInk{&kD2, &kWhite, nil, &kMid, &kD2}
		br := [beFrame]*rpInk{&kD4, &kMid, nil, &kWhite, &kD2}
		for i := 0; i < beFrame; i++ {
			rpEdge(tl[i], br[i], g, i, fy+i, g.w-2*i, fh-2*i, 1)
		}
	}
	fill, hi, lo := c.tab, c.tabHi, c.tabLo
	if !st.Active {
		fill, hi, lo = c.tabOff, c.white, c.d2
	}
	kTab.fill(ctx, fill)
	kHi.fill(ctx, hi)
	kLo.fill(ctx, lo)
	kWhite.fill(ctx, c.white)
	kMid.fill(ctx, c.mid)
	kD2.fill(ctx, c.d2)
	kD4.fill(ctx, c.d4)
}

func (beosEngine) DrawCaptionTitle(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, title string, st DecorationState) {
	c := beColorsOf(l)
	u := rpU(l)
	th := beTabH(l)
	col := c.text
	if !st.Active {
		col = c.tabOffText
	}
	g := rpGridAt(ctx, b, u)
	if g.w < 4 {
		return
	}
	// b is already clear of the boxes by R5's gaps (DecorationSpec.TitleRoom).
	// The title is measured in whole cells, as DrawWindowFrame measures it:
	// a fitted tab's band is sized in device pixels, and at a fractional
	// scale its box can fall a cell short of a title that fits, with 17
	// cells of gap still clear of the zoom box after it.
	f := l.BoldFont()
	tw := min(int(f.Advance(title)/u+0.99), g.w+1)
	l.drawFittedText(ctx, f, title, g.at(0, 1, max(tw, g.w), min(th-1, g.h-1)), col, AlignStart, 0)
}

func (beosEngine) DrawCaptionButton(l *Classic, ctx *paintengine2d.Context, b paintengine2d.Rect, k CaptionButton, cs ControlState, st DecorationState) {
	c := beColorsOf(l)
	u := rpU(l)
	g := rpGridAt(ctx, b, u)
	if g.w < 9 || g.h < 9 {
		return
	}
	if k == CaptionMaximize {
		c.zoomBox(ctx, g, 0, 0, st.Active)
		return
	}
	// Close, and the buttons BeOS kept in the window menu rather than the
	// tab, drawn as its box with the plainest glyph that fits.
	c.closeBox(ctx, g, 0, 0, st.Active, cs.Pressed())
	ink := c.text
	if !st.Active {
		ink = c.tabOffText
	}
	var k2 rpInk
	switch k {
	case CaptionMinimize:
		k2.cells(g, 4, 7, 6, 1)
	case CaptionMenu:
		k2.frame(g, 4, 4, 6, 6, 1)
		k2.cells(g, 4, 4, 6, 2)
	}
	k2.fill(ctx, ink)
}
