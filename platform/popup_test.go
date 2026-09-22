package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

// SolvePopup is xdg_positioner's placement, so the X11 and offscreen
// backends place a popup where a Wayland compositor would.
func TestSolvePopup(t *testing.T) {
	area := FrameRect{X: 0, Y: 0, W: 1000, H: 700}
	below := PopupPlacement{
		Anchor:     FrameRect{X: 100, Y: 100, W: 80, H: 24},
		AnchorEdge: EdgeBottom | EdgeLeft, Gravity: EdgeBottom | EdgeRight,
		W: 200, H: 300,
		Adjust: AdjustSlideX | AdjustFlipY | AdjustResizeY,
	}
	cases := []struct {
		name string
		p    func(p PopupPlacement) PopupPlacement
		want FrameRect
	}{
		{"fits", func(p PopupPlacement) PopupPlacement { return p }, FrameRect{100, 124, 200, 300}},
		{"flips above", func(p PopupPlacement) PopupPlacement {
			p.Anchor.Y = 600
			return p
		}, FrameRect{100, 300, 200, 300}},
		{"slides left", func(p PopupPlacement) PopupPlacement {
			p.Anchor.X = 900
			return p
		}, FrameRect{800, 124, 200, 300}},
		{"shrinks when neither side fits", func(p PopupPlacement) PopupPlacement {
			p.Anchor.Y = 350
			p.H = 500
			return p
		}, FrameRect{100, 374, 200, 326}},
		{"no flip allowed", func(p PopupPlacement) PopupPlacement {
			p.Anchor.Y = 600
			p.Adjust = AdjustSlideY
			return p
		}, FrameRect{100, 400, 200, 300}},
		{"submenu flips left", func(p PopupPlacement) PopupPlacement {
			p.Anchor = FrameRect{X: 700, Y: 200, W: 200, H: 24}
			p.AnchorEdge, p.Gravity = EdgeTop|EdgeRight, EdgeBottom|EdgeRight
			p.Adjust = AdjustFlipX | AdjustSlideY
			return p
		}, FrameRect{500, 200, 200, 300}},
		{"centred gravity", func(p PopupPlacement) PopupPlacement {
			p.AnchorEdge, p.Gravity = 0, 0
			return p
		}, FrameRect{40, -38, 200, 300}},
	}
	for _, c := range cases {
		p := c.p(below)
		got := SolvePopup(p, area)
		if c.name == "centred gravity" {
			// Nothing to adjust vertically: the popup runs above the area.
			got = SolvePopup(PopupPlacement{Anchor: p.Anchor, W: p.W, H: p.H}, area)
		}
		if got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
	if got := SolvePopup(below, FrameRect{}); got != (FrameRect{100, 124, 200, 300}) {
		t.Errorf("no area: %+v", got)
	}
}

func TestPopupOriginRoundsToTheRootGrid(t *testing.T) {
	win := paintengine2d.Pt(21, 21)
	o := PopupOrigin(win, FrameRect{X: 10, Y: 3}, FrameInsets{Left: 14, Top: 7}, 1.75)
	// 10 logical at 1.75 is 17.5, rounded to 18; less the popup's margin.
	if o != paintengine2d.Pt(21+18-14, 21+5-7) {
		t.Fatalf("origin %v", o)
	}
}

// A popup's input arrives on its root window, moved by the popup's origin.
func TestOffscreenPopupRoutesEventsToItsRoot(t *testing.T) {
	root := NewOffscreen(WindowOptions{Width: 200, Height: 100, X: 50, Y: 60})
	if root.PopupsSupported() {
		t.Fatal("popups on by default headless")
	}
	if _, err := root.OpenPopup(PopupOptions{}); err == nil {
		t.Fatal("opened a popup with the simulation off")
	}
	root.SimulatePopups(FrameRect{W: 800, H: 600})
	area, ok := root.PopupWorkArea()
	if !ok || area != (FrameRect{X: -50, Y: -60, W: 800, H: 600}) {
		t.Fatalf("work area %+v %v", area, ok)
	}
	p, err := root.OpenPopup(PopupOptions{Parent: root, Placement: PopupPlacement{
		Anchor: FrameRect{X: 10, Y: 10, W: 1, H: 1}, AnchorEdge: EdgeBottom | EdgeRight,
		Gravity: EdgeBottom | EdgeRight, W: 120, H: 400, Adjust: AdjustSlideY,
	}, Frame: Frame{Margin: FrameInsets{Left: 5, Top: 5, Right: 5, Bottom: 5}}})
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Placed(); got != (FrameRect{X: 11, Y: 11, W: 120, H: 400}) {
		t.Fatalf("placed %+v", got)
	}
	if w, h := p.Size(); w != 130 || h != 410 {
		t.Fatalf("popup buffer %dx%d: the visible box plus its margin", w, h)
	}
	p.(*Offscreen).Inject(Event{Kind: EventMouseDown, Pos: paintengine2d.Pt(5, 5)})
	evs := root.Poll()
	if len(evs) != 1 || evs[0].Pos != paintengine2d.Pt(11, 11) {
		t.Fatalf("root heard %+v", evs)
	}
	sub, err := p.(PopupOpener).OpenPopup(PopupOptions{Parent: p, Placement: PopupPlacement{W: 10, H: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if sub.Root() != Surface(root) || len(root.Popups()) != 1 || len(p.(*Offscreen).Popups()) != 1 {
		t.Fatal("nesting")
	}
	// Closing the menu takes its submenu with it, the submenu first.
	_ = p.Close()
	if !sub.Closed() || len(root.Popups()) != 0 {
		t.Fatal("closing a popup left its submenu or its place")
	}
}
