package platform

import "testing"

// A menu opened at a point is placed beside the point, never on it: the
// tray hands over a point and never the icon's rectangle, and a menu
// whose corner is on the point hides the icon it came from.
func TestSolveScreenMenuNeverCoversThePoint(t *testing.T) {
	screen := FrameRect{W: 1920, H: 1080}
	for _, p := range []struct{ x, y int }{{10, 10}, {960, 540}, {1900, 20}, {30, 1070}, {1910, 1075}} {
		r := SolveScreenMenu(p.x, p.y, 200, 300, screen, ScreenMenuPointerGap)
		if p.x >= r.X && p.x < r.X+r.W && p.y >= r.Y && p.y < r.Y+r.H {
			t.Fatalf("menu %+v covers the point %d,%d", r, p.x, p.y)
		}
		if r.X < screen.X || r.Y < screen.Y || r.X+r.W > screen.W || r.Y+r.H > screen.H {
			t.Fatalf("menu %+v for point %d,%d is off the %+v screen", r, p.x, p.y, screen)
		}
	}
}

// No room to the right: the menu flips to the other side of the point
// rather than hanging off the edge. That is the whole of the submenu bug
// — the surface is measured to hold the parent *and* its cascade, so it
// is the wide rectangle that has to be flipped.
func TestSolveScreenMenuFlipsAtTheRightEdge(t *testing.T) {
	screen := FrameRect{W: 1000, H: 800}
	r := SolveScreenMenu(900, 400, 400, 200, screen, 10)
	if r.X != 900-10-400 {
		t.Fatalf("menu %+v did not flip left of 900", r)
	}
	if r.W != 400 || r.H != 200 {
		t.Fatalf("menu %+v was resized where a flip was enough", r)
	}
}

// No room below: the menu flips above the point — the rule a context menu
// follows on a bottom panel.
func TestSolveScreenMenuFlipsAtTheBottomEdge(t *testing.T) {
	screen := FrameRect{W: 1000, H: 800}
	r := SolveScreenMenu(400, 780, 200, 300, screen, 10)
	if r.Y != 780-10-300 {
		t.Fatalf("menu %+v did not flip above 780", r)
	}
}

// Neither side has room: the menu slides back onto the screen, keeping
// its size.
func TestSolveScreenMenuSlidesWhenNeitherSideFits(t *testing.T) {
	screen := FrameRect{W: 1000, H: 800}
	r := SolveScreenMenu(500, 400, 900, 200, screen, 10)
	if r.X != 100 || r.W != 900 {
		t.Fatalf("menu %+v should have slid to x=100 at its full width", r)
	}
	if r.X+r.W > screen.W {
		t.Fatalf("menu %+v still runs off the screen", r)
	}
}

// A menu larger than the screen is clamped to the screen, not pushed to a
// negative corner (a negative position is unplaceable on every backend
// and was what the old "subtract the height" arithmetic could produce).
func TestSolveScreenMenuLargerThanTheScreen(t *testing.T) {
	screen := FrameRect{X: 0, Y: 0, W: 800, H: 600}
	r := SolveScreenMenu(100, 100, 3000, 2000, screen, ScreenMenuPointerGap)
	if r.X < 0 || r.Y < 0 {
		t.Fatalf("menu %+v has a negative corner", r)
	}
	if r.W > screen.W || r.H > screen.H {
		t.Fatalf("menu %+v is larger than the %+v screen", r, screen)
	}
	if r.W < 1 || r.H < 1 {
		t.Fatalf("menu %+v was shrunk to nothing", r)
	}
}

// On a second monitor the screen rectangle does not start at the origin,
// and the menu is constrained to that rectangle rather than to a screen
// the point is not on.
func TestSolveScreenMenuOnAnOffsetScreen(t *testing.T) {
	screen := FrameRect{X: 1920, Y: 0, W: 1280, H: 1024}
	r := SolveScreenMenu(3180, 40, 300, 200, screen, 10)
	if r.X < screen.X || r.X+r.W > screen.X+screen.W {
		t.Fatalf("menu %+v is outside %+v", r, screen)
	}
}

// With no screen to speak of — a backend that cannot say where its
// monitors are — the menu is still placed beside the point and is not
// moved at all, which is what the toolkit did before it could ask.
func TestSolveScreenMenuWithoutAScreen(t *testing.T) {
	r := SolveScreenMenu(100, 200, 300, 400, FrameRect{}, 10)
	if r.X != 110 || r.Y != 210 || r.W != 300 || r.H != 400 {
		t.Fatalf("unconstrained menu %+v", r)
	}
}

// The numbers from the KDE Wayland session that reported both faults: a
// 2880x1800 panel at 175% (a 1645x1029 logical desktop), the tray icon at
// device 2305,28, a menu surface of 988x221 device — 565x126 logical.
// Before this the menu was placed at 1317,16 and ran 237 logical pixels
// off the right edge, submenus and all.
func TestSolveScreenMenuKDETrayReport(t *testing.T) {
	const (
		screenW, screenH = 1645, 1029
		scale            = 1.75
	)
	screen := FrameRect{W: screenW, H: screenH}
	x, y := LogicalPosition(2305, scale), LogicalPosition(28, scale)
	w, h := LogicalPixels(988, scale), LogicalPixels(221, scale)
	gap := LogicalPixels(ScreenMenuPointerGap, scale)
	r := SolveScreenMenu(x, y, w, h, screen, gap)
	if r.X+r.W > screenW {
		t.Fatalf("menu %+v runs %d px off the right edge of %d", r, r.X+r.W-screenW, screenW)
	}
	if r.Y+r.H > screenH || r.X < 0 || r.Y < 0 {
		t.Fatalf("menu %+v is not inside the %dx%d screen", r, screenW, screenH)
	}
	if x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
		t.Fatalf("menu %+v covers the tray point %d,%d", r, x, y)
	}
	if r.W != w || r.H != h {
		t.Fatalf("menu %+v was shrunk; %dx%d fits this screen whole", r, w, h)
	}
}
