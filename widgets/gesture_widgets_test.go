package widgets

import (
	"errors"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

func zoomPicture(t *testing.T) *Picture {
	t.Helper()
	p := NewPicture(paintengine2d.NewImage(100, 50))
	p.SetHost(&fakeWindow{look: style.DarkLook()})
	p.Zoomable = true
	p.Arrange(paintengine2d.XYWH(0, 0, 200, 100))
	return p
}

func pinch(phase platform.GesturePhase, x, y, scale float32) widget.GestureEvent {
	return widget.GestureEvent{Kind: platform.GesturePinch, Phase: phase, Pos: paintengine2d.Pt(x, y), Fingers: 2, Scale: scale}
}

// A pinch zooms a Zoomable picture about the fingers: the image's point
// under them stays under them, the zoom is the begin's times Scale, it
// stops at 1 (the Fit size) and MaxZoom, and a zoomed picture never shows
// a gap at an edge.
func TestPicturePinchZoomsAboutTheFingers(t *testing.T) {
	p := zoomPicture(t)
	var heard []float32
	p.OnZoom = func(z float32) { heard = append(heard, z) }
	if !p.Gesture(pinch(platform.GestureBegin, 50, 25, 1)) {
		t.Fatal("a zoomable picture refused a pinch")
	}
	p.Gesture(pinch(platform.GestureUpdate, 50, 25, 2))
	if p.Zoom() != 2 {
		t.Fatalf("zoom %v", p.Zoom())
	}
	_, dst := p.shown(p.LocalBounds())
	if dst != paintengine2d.XYWH(-50, -25, 400, 200) {
		t.Fatalf("shown at %v: the point under the fingers moved", dst)
	}
	p.Gesture(pinch(platform.GestureUpdate, 50, 25, 100))
	if p.Zoom() != 8 {
		t.Fatalf("zoom %v, want the cap 8", p.Zoom())
	}
	p.Gesture(pinch(platform.GestureUpdate, 50, 25, 0.01))
	if p.Zoom() != 1 {
		t.Fatalf("zoom %v, want the floor 1", p.Zoom())
	}
	if _, dst := p.shown(p.LocalBounds()); dst != paintengine2d.XYWH(0, 0, 200, 100) {
		t.Fatalf("back at 1 it is at %v", dst)
	}
	p.Gesture(pinch(platform.GestureEnd, 50, 25, 0.01))
	if len(heard) != 3 || heard[0] != 2 || heard[1] != 8 || heard[2] != 1 {
		t.Fatalf("OnZoom heard %v", heard)
	}
	// Zoomed at a corner: pulled back so no gap shows.
	p.Gesture(pinch(platform.GestureBegin, 0, 0, 1))
	p.Gesture(pinch(platform.GestureUpdate, 0, 0, 3))
	_, dst = p.shown(p.LocalBounds())
	if dst.Min.X > 0 || dst.Min.Y > 0 || dst.Max.X < 200 || dst.Max.Y < 100 {
		t.Fatalf("a gap at an edge: %v", dst)
	}
	// Not zoomable, or not a pinch: not taken, so it can bubble.
	q := zoomPicture(t)
	q.Zoomable = false
	if q.Gesture(pinch(platform.GestureBegin, 1, 1, 1)) {
		t.Fatal("a plain picture took a pinch")
	}
	swipe := pinch(platform.GestureBegin, 1, 1, 1)
	swipe.Kind = platform.GestureSwipe
	if p.Gesture(swipe) {
		t.Fatal("a picture took a swipe")
	}
}

// Ctrl and the wheel zoom about the pointer; the wheel alone pans a zoomed
// picture and lets go at the edge (so the page around it scrolls on), and
// does nothing to one at 1.
func TestPictureWheelZoomAndPan(t *testing.T) {
	p := zoomPicture(t)
	if p.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 1)}) {
		t.Fatal("the wheel took an unzoomed picture's scroll")
	}
	p.MouseWheel(widget.MouseEvent{Pos: paintengine2d.Pt(100, 50), Scroll: paintengine2d.Pt(0, -1), Mods: platform.ModCtrl})
	if z := p.Zoom(); z < 1.249 || z > 1.251 {
		t.Fatalf("a notch in with Ctrl: %v", z)
	}
	p.SetZoom(4)
	_, before := p.shown(p.LocalBounds())
	if !p.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 20), Precise: true}) {
		t.Fatal("a zoomed picture did not pan")
	}
	_, after := p.shown(p.LocalBounds())
	if after.Min.Y != before.Min.Y-20 {
		t.Fatalf("panned from %v to %v", before, after)
	}
	for i := 0; i < 100; i++ {
		p.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 50), Precise: true})
	}
	if p.MouseWheel(widget.MouseEvent{Scroll: paintengine2d.Pt(0, 50), Precise: true}) {
		t.Fatal("at the edge the picture still took the scroll")
	}
	p.SetZoom(1)
	if _, dst := p.shown(p.LocalBounds()); dst != paintengine2d.XYWH(0, 0, 200, 100) {
		t.Fatalf("SetZoom(1) left it at %v", dst)
	}
}

// A link opens through the platform's opener, as its window asks; a click
// that ends off it does not; Return does.
func TestLinkButtonOpens(t *testing.T) {
	got := make(chan string, 4)
	old := openURI
	openURI = func(uri string, opts platform.OpenURIOptions) error {
		got <- uri + " " + opts.ParentWindow
		return errors.New("nothing here")
	}
	defer func() { openURI = old }()
	l := NewLinkButton("The page", "https://example.org/")
	l.SetHost(&fakeWindow{look: style.DarkLook()})
	l.Arrange(paintengine2d.XYWH(0, 0, 120, 24))
	failed := make(chan error, 4)
	l.OnError = func(err error) { failed <- err }
	l.MousePress(widget.MouseEvent{Button: platform.ButtonLeft, Pos: paintengine2d.Pt(5, 5)})
	l.MouseRelease(widget.MouseEvent{Button: platform.ButtonLeft, Pos: paintengine2d.Pt(500, 5)})
	l.MousePress(widget.MouseEvent{Button: platform.ButtonRight, Pos: paintengine2d.Pt(5, 5)})
	l.MouseRelease(widget.MouseEvent{Button: platform.ButtonRight, Pos: paintengine2d.Pt(5, 5)})
	select {
	case u := <-got:
		t.Fatalf("opened %q on a click that was not one", u)
	case <-time.After(50 * time.Millisecond):
	}
	l.MousePress(widget.MouseEvent{Button: platform.ButtonLeft, Pos: paintengine2d.Pt(5, 5)})
	l.MouseRelease(widget.MouseEvent{Button: platform.ButtonLeft, Pos: paintengine2d.Pt(6, 5)})
	if u := <-got; u != "https://example.org/ " {
		t.Fatalf("opened %q", u)
	}
	l.KeyPress(widget.KeyEvent{Key: platform.KeyReturn})
	<-got
	if !l.Visited || l.Tooltip() != "https://example.org/" {
		t.Fatal("visited / tooltip")
	}
	// The opener's failure comes back (on its goroutine here: the fake
	// host has no timers).
	select {
	case err := <-failed:
		if err == nil {
			t.Fatal("nil error")
		}
	case <-time.After(time.Second):
		t.Fatal("the failure never came back")
	}
}
