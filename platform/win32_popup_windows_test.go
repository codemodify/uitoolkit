//go:build windows

package platform

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
)

// winPopupFrom opens a popup under the window, anchored at a box in it.
func winPopupFrom(t *testing.T, s *winSurface, w, h int, anchor FrameRect) *winSurface {
	t.Helper()
	p, err := s.OpenPopup(PopupOptions{
		Parent: s, Role: PopupRoleMenu,
		Placement: PopupPlacement{
			Anchor: anchor, AnchorEdge: EdgeBottom | EdgeLeft, Gravity: EdgeBottom | EdgeRight,
			W: w, H: h, Adjust: AdjustFlipY | AdjustSlideX | AdjustSlideY,
		},
	})
	if err != nil {
		t.Fatalf("OpenPopup: %v", err)
	}
	return p.(*winSurface)
}

// winPump turns the message loop for a while.
func winPump(s *winSurface, d time.Duration) []Event {
	var out []Event
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		out = append(out, s.Poll()...)
		time.Sleep(5 * time.Millisecond)
	}
	return out
}

// A popup is a window of its own with its own buffer, and the toolkit
// is told where it went.
//
// Without this the app draws a popup inside its window, which works
// until the popup is taller than the room under it — a combo box near
// the foot of a window — and then it is clipped or moved somewhere it
// does not belong.
func TestWinOpensRealPopups(t *testing.T) {
	s := testSurface(t, 400, 300)
	winPump(s, 200*time.Millisecond)
	if !s.PopupsSupported() {
		t.Fatal("Win32 says it cannot open popups")
	}
	p := winPopupFrom(t, s, 160, 120, FrameRect{X: 20, Y: 40, W: 80, H: 20})
	defer p.Close()
	winPump(s, 200*time.Millisecond)

	if got := p.Root(); got != Surface(s) {
		t.Errorf("the popup's root is %p, want the window %p", got, s)
	}
	if p.hwnd == 0 {
		t.Fatal("the popup has no window")
	}
	if p.hwnd == s.hwnd {
		t.Error("the popup shares the window's HWND")
	}
	placed := p.Placed()
	if placed.W != 160 || placed.H != 120 {
		t.Errorf("placed %+v, want 160x120", placed)
	}
	if placed.X != 20 || placed.Y != 60 {
		t.Errorf("placed at %d,%d, want 20,60 — under the anchor", placed.X, placed.Y)
	}
	if w, h := p.Size(); w != DevicePixels(160, s.Scale()) || h != DevicePixels(120, s.Scale()) {
		t.Errorf("the popup's buffer is %dx%d at scale %g", w, h, s.Scale())
	}
	if p.Buffer() == nil {
		t.Error("the popup has no buffer to paint")
	}
	if p.Buffer() == s.Buffer() {
		t.Error("the popup shares the window's buffer")
	}
	o := p.Origin()
	if o.X != 20*s.Scale() || o.Y != 60*s.Scale() {
		t.Errorf("origin %v, want %v,%v", o, 20*s.Scale(), 60*s.Scale())
	}
}

// The popup is placed against the monitor's work area rather than
// blindly at the anchor: one that would hang off the bottom goes above
// it. This is SolvePopup, the same function X11 and the headless tests
// use, so the answer is the same everywhere.
func TestWinPopupIsPlacedInTheWorkArea(t *testing.T) {
	s := testSurface(t, 400, 300)
	winPump(s, 200*time.Millisecond)
	area, ok := s.PopupWorkArea()
	if !ok {
		t.Fatal("Windows knows where its windows are and this one did not")
	}
	if area.W < 200 || area.H < 200 {
		t.Fatalf("the work area is %+v, which is not a monitor", area)
	}
	anchorY := area.Y + area.H - 40
	p := winPopupFrom(t, s, 120, 300, FrameRect{X: 10, Y: anchorY, W: 60, H: 20})
	defer p.Close()
	placed := p.Placed()
	if placed.Y+placed.H > area.Y+area.H {
		t.Errorf("the popup runs off the bottom: placed %+v, area %+v", placed, area)
	}
}

// A popup's input is the window's: every event on it arrives on the
// root, moved by the popup's origin, because as far as the application
// is concerned the menu is part of the window.
func TestWinPopupInputGoesToTheRoot(t *testing.T) {
	s := testSurface(t, 400, 300)
	winPump(s, 200*time.Millisecond)
	p := winPopupFrom(t, s, 160, 120, FrameRect{X: 20, Y: 40, W: 80, H: 20})
	defer p.Close()
	winPump(s, 200*time.Millisecond)

	// Queued on the popup as its own window procedure would, and it
	// must come out on the root translated rather than on the popup.
	p.push(Event{Kind: EventMouseDown, Pos: paintengine2d.Pt(10, 10), Button: ButtonLeft})
	for _, e := range p.Poll() {
		if e.Kind == EventMouseDown {
			t.Error("the press stayed on the popup's own queue")
		}
	}
	var down *Event
	for _, e := range s.Poll() {
		if e.Kind == EventMouseDown {
			ev := e
			down = &ev
		}
	}
	if down == nil {
		t.Fatal("a press on the popup never reached the window")
	}
	o := p.Origin()
	if down.Pos.X != o.X+10 || down.Pos.Y != o.Y+10 {
		t.Errorf("the press arrived at %v, want %v,%v (the popup's origin plus 10,10)",
			down.Pos, o.X+10, o.Y+10)
	}
	// A resize is the popup's own business and stays on it.
	p.push(Event{Kind: EventResize, Width: 1, Height: 2})
	var sawResize bool
	for _, e := range p.Poll() {
		if e.Kind == EventResize {
			sawResize = true
		}
	}
	if !sawResize {
		t.Error("the popup's own resize was passed to the root")
	}
}

// Closing the window takes its popups with it, and a submenu goes with
// its menu: Windows destroys an owner's owned windows, so a surface
// left behind would be holding an HWND that no longer exists.
func TestWinPopupsCloseWithTheirParent(t *testing.T) {
	s := testSurface(t, 400, 300)
	winPump(s, 200*time.Millisecond)
	menu := winPopupFrom(t, s, 160, 120, FrameRect{X: 20, Y: 40, W: 80, H: 20})
	sub := winPopupFrom(t, menu, 100, 80, FrameRect{X: 10, Y: 10, W: 40, H: 16})
	if len(s.kids) != 1 || len(menu.kids) != 1 {
		t.Fatalf("the popup chain is %d and %d deep", len(s.kids), len(menu.kids))
	}
	menu.Close()
	if !sub.Closed() {
		t.Error("closing a menu left its submenu open")
	}
	if sub.hwnd != 0 {
		t.Error("the submenu still holds an HWND")
	}
	if len(s.kids) != 0 {
		t.Errorf("the window still lists %d popups", len(s.kids))
	}
}

// Reposition moves an open popup — a combo list that grew, a menu whose
// anchor moved — rather than the app closing and reopening it.
func TestWinPopupReposition(t *testing.T) {
	s := testSurface(t, 400, 300)
	winPump(s, 200*time.Millisecond)
	p := winPopupFrom(t, s, 160, 120, FrameRect{X: 20, Y: 40, W: 80, H: 20})
	defer p.Close()
	before := p.Placed()
	if !p.Reposition(PopupPlacement{
		Anchor:     FrameRect{X: 100, Y: 40, W: 80, H: 20},
		AnchorEdge: EdgeBottom | EdgeLeft, Gravity: EdgeBottom | EdgeRight,
		W: 160, H: 120, Adjust: AdjustFlipY | AdjustSlideX,
	}) {
		t.Fatal("Reposition refused")
	}
	after := p.Placed()
	if after.X == before.X {
		t.Errorf("the popup did not move: %+v then %+v", before, after)
	}
	if after.X != 100 {
		t.Errorf("repositioned to %+v, want x 100", after)
	}
}

// The caret in a preedit is a byte offset into the UTF-8, and Windows
// counts UTF-16 units. The two part company on the first character
// outside the basic plane — which for an input method is not a corner
// case, because that is where the rarer CJK characters live.
func TestWinIMECaretIsAByteOffset(t *testing.T) {
	for _, c := range []struct {
		name  string
		text  string
		units int
		want  int
	}{
		{"nothing", "", 0, 0},
		{"the start", "にほん", 0, 0},
		{"one BMP character is three bytes", "にほん", 1, 3},
		{"two", "にほん", 2, 6},
		{"the end", "にほん", 3, 9},
		{"past the end is the end", "にほん", 99, 9},
		{"ASCII is one for one", "abc", 2, 2},
		// U+20BB7, a surrogate pair: two UTF-16 units, four UTF-8 bytes.
		{"before a pair", "a\U00020BB7b", 1, 1},
		{"a caret inside a pair belongs before it", "a\U00020BB7b", 2, 1},
		{"after a pair", "a\U00020BB7b", 3, 5},
		{"negative is the start", "にほん", -1, 0},
	} {
		if got := utf16ByteOffset(c.text, c.units); got != c.want {
			t.Errorf("%s: offset %d of %q is %d, want %d", c.name, c.units, c.text, got, c.want)
		}
	}
}

// The window carries the IME seam, and turning it on and off is safe
// whatever state it is in.
func TestWinIsAnIMESurface(t *testing.T) {
	s := testSurface(t, 320, 200)
	var _ IMESurface = s
	s.SetIMEEnabled(true)
	s.SetIMECursor(40, 60, 2, 18)
	s.SetIMEEnabled(false)
	// Off twice, and on again: the context is set aside rather than
	// destroyed, so this has to be idempotent.
	s.SetIMEEnabled(false)
	s.SetIMEEnabled(true)
	if !s.imeOn {
		t.Error("the input method did not come back on")
	}
}
