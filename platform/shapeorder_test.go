package platform

import "testing"

// X11's SHAPE extension takes the caller's word for the order of the
// rectangles it is given, and answers a false claim with BadMatch and no
// shape at all. These are the orders it may be told about.
func TestYXBanded(t *testing.T) {
	for _, tc := range []struct {
		name  string
		rects []FrameRect
		want  bool
	}{
		{"nothing", nil, true},
		{"one", []FrameRect{{0, 0, 10, 1}}, true},
		{"a rasteriser's scanlines", []FrameRect{
			{2, 0, 6, 1}, {1, 1, 8, 1}, {0, 2, 10, 1},
		}, true},
		{"two in a band, left to right", []FrameRect{
			{0, 0, 4, 2}, {6, 0, 4, 2}, {0, 2, 10, 2},
		}, true},
		{"bands that touch", []FrameRect{{0, 0, 10, 5}, {0, 5, 10, 5}}, true},
		// The bug: the resize bands are appended after the silhouette, so
		// they start again at the top of the window.
		{"a band that goes back up", []FrameRect{
			{0, 0, 10, 20}, {0, 0, 10, 4},
		}, false},
		{"bands that overlap", []FrameRect{{0, 0, 10, 6}, {0, 4, 10, 6}}, false},
		{"one band out of order across", []FrameRect{
			{6, 0, 4, 2}, {0, 0, 4, 2},
		}, false},
		{"two in a band that touch", []FrameRect{
			{0, 0, 5, 2}, {4, 0, 5, 2},
		}, false},
		{"the same y, different heights", []FrameRect{
			{0, 0, 4, 2}, {6, 0, 4, 3},
		}, false},
	} {
		if got := yxBanded(tc.rects); got != tc.want {
			t.Errorf("%s: yxBanded = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// A rectangle with no area names no pixels, and is no part of the order: it
// has to go before the order is judged, because the list that is sent is the
// list that was judged.
func TestNonEmptyRects(t *testing.T) {
	in := []FrameRect{{0, 0, 10, 2}, {0, 2, 0, 2}, {0, 2, 10, 0}, {0, 2, 10, 2}}
	got := nonEmptyRects(in)
	if len(got) != 2 || got[0] != in[0] || got[1] != in[3] {
		t.Fatalf("nonEmptyRects = %v, want the two with area", got)
	}
	// The input is untouched: a Frame's own slice is not this function's.
	if len(in) != 4 {
		t.Errorf("the caller's slice was rewritten: %v", in)
	}
	// A list with nothing to drop is handed straight back.
	full := []FrameRect{{0, 0, 1, 1}, {0, 1, 1, 1}}
	if out := nonEmptyRects(full); len(out) != 2 || &out[0] != &full[0] {
		t.Error("a list with no empty rectangles was copied for nothing")
	}
	// And one whose only rectangles are empty comes back empty, which is the
	// documented way to say "no pixels at all".
	if out := nonEmptyRects([]FrameRect{{0, 0, 0, 0}}); len(out) != 0 {
		t.Errorf("nonEmptyRects of nothing but empties = %v", out)
	}
}
