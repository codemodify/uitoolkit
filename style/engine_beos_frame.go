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
// One departure, and it is the tab's whole point: BeOS sized the tab to its
// contents — close box, title, zoom box — and left the rest of the window's
// top edge to the desktop. A uitoolkit caption is a band as wide as the
// window, whose ends are where the buttons go, so the tab is that band: the
// same yellow, the same bevel, the same boxes at its two ends in the order
// BeOS put them (the packs ask for close at the left and zoom at the right),
// stretched to the window's width.
//
// Colours and lengths are engine_beos.go's, measured off BeOS R5
// screenshots at 1:1.

func (beosEngine) Decoration(l *Classic, st DecorationState) DecorationSpec {
	u := rpU(l)
	th := beTabH(l)
	return DecorationSpec{
		Stacked: true,
		Border:  Insets{Right: beFrame * u, Bottom: beFrame * u, Left: beFrame * u},
		// The tab, whose last row is the frame's top line, and the frame.
		Caption:   float32(th-1+beFrame) * u,
		Button:    paintengine2d.Pt(14*u, 14*u),
		ButtonGap: 3 * u,
		ButtonPad: Insets{Top: float32(beBoxY(l)) * u, Right: 4 * u, Left: 4 * u},
		Layout:    "close:maximize",
	}
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
	w := g.w
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
	if g.w < 12 {
		return
	}
	// The title stands clear of the close box, as it did on the tab.
	l.drawFittedText(ctx, l.BoldFont(), title, g.at(10, 1, g.w-14, min(th-1, g.h-1)), col, AlignStart, 0)
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
