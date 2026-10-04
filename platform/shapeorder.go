package platform

// yxBanded reports whether rects are in the order X11's SHAPE extension
// calls YXBanded, which [x11Surface.applyShapeLocked] may only declare when
// they are.
//
// YXBanded is the order a region's own rasteriser produces: rectangles sorted
// down the window, those at the same height sorted across it, none of them
// overlapping, and no band starting above where the one before it ended.
// Saying so lets the server install a shape of hundreds of rectangles without
// sorting them.
//
// It was said unconditionally, and a shaped window that can be *resized* does
// not satisfy it: its resize bands are appended after its silhouette, so they
// return to the top of the window and overlap what is already there. X answers
// a false declaration with BadMatch and installs no shape at all — so a
// rounded window came out square, with the default full-window input region,
// and nothing in the application could tell. The honest answer is to say
// Unsorted for a list like that and let the server do the work.
func yxBanded(rects []FrameRect) bool {
	for i := 1; i < len(rects); i++ {
		p, r := rects[i-1], rects[i]
		// The same band, further across it, not touching what is there.
		if r.Y == p.Y && r.H == p.H && r.X >= p.X+p.W {
			continue
		}
		// Or a new band, starting at or below the foot of the last one.
		if r.Y >= p.Y+p.H {
			continue
		}
		return false
	}
	return true
}

// nonEmptyRects drops the rectangles with no area, which name no pixels and
// are no part of the order above.
func nonEmptyRects(rects []FrameRect) []FrameRect {
	out := rects
	for i, r := range rects {
		if r.W > 0 && r.H > 0 {
			continue
		}
		// Only now is a copy needed.
		out = make([]FrameRect, 0, len(rects)-1)
		out = append(out, rects[:i]...)
		for _, r := range rects[i+1:] {
			if r.W > 0 && r.H > 0 {
				out = append(out, r)
			}
		}
		break
	}
	return out
}
