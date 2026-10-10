package app

import (
	"testing"
	"time"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// runWindowTurn paints frames until the turn ends, recording the card's
// outline width at each, and says so if it never does.
//
// These windows run on the real clock — the panel's is not reachable from
// here — so the loop waits for the turn to be over instead of assuming a
// count of frames is long enough. A fixed count passed on a loaded
// machine and failed on an idle one, which is the third timing
// assumption of its kind in this repo and the same remedy as the others.
func runWindowTurn(t *testing.T, win *Window, p *widgets.FlipPanel) []float32 {
	t.Helper()
	var widths []float32
	deadline := time.Now().Add(10 * time.Second)
	for p.Turning() {
		if time.Now().After(deadline) {
			t.Fatal("the turn was still running after ten seconds")
		}
		win.Capture()
		if !p.Turning() {
			break
		}
		if out := p.CardOutline(); out != nil {
			widths = append(widths, out.Bounds().Dx())
		}
	}
	return widths
}

// flipWindow is a headless window whose whole content is two faces.
func flipWindow(t *testing.T) (*Application, *Window, *widgets.FlipPanel) {
	t.Helper()
	t.Setenv(style.AnimationsEnv, "1")
	a := New(Options{Headless: true, Scale: 1})
	win, err := a.NewWindow(platform.WindowOptions{Width: 520, Height: 360, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	p := win.SetFlipFaces(widgets.NewButton("front", nil), widgets.NewButton("back", nil))
	p.Strips = 32
	a.PumpOnce()
	win.Capture()
	return a, win, p
}

// **The window follows the card.** A toplevel cannot be rotated by the
// program that owns it, so the illusion is the window taking the turning
// card's silhouette as its shape: the desktop shows through everywhere
// the card is not, and what appears to rotate is the window. The card
// narrows through the turn, so the shape must narrow with it rather than
// being set once.
func TestAWindowTakesTheTurningCardsShape(t *testing.T) {
	_, win, p := flipWindow(t)
	if win.flipShaped {
		t.Error("the window is shaped before anything has turned")
	}

	win.Flip()
	if !win.flipShaped {
		// The shape goes on at the first frame, which Capture draws.
		win.Capture()
	}
	if !win.flipShaped && p.Turning() {
		t.Fatal("the window is not wearing the card's shape while it turns")
	}
	widths := runWindowTurn(t, win, p)
	if len(widths) < 4 {
		t.Fatalf("only %d frames were drawn; the turn did not run", len(widths))
	}
	// It is a turn, not a fixed cut-out: the card is narrower part way
	// through than it was flat.
	var narrowest float32 = 1e9
	for _, w := range widths {
		if w < narrowest {
			narrowest = w
		}
	}
	if narrowest >= widths[0] {
		t.Errorf("the card never narrowed: widths %v", widths)
	}
	if win.flipShaped {
		t.Error("the window kept the card's shape after the turn ended")
	}
}

// A window told not to take the shape stays the rectangle it was, and
// turns its content inside it — which is what a window wearing the
// desktop's own frame has to do, since that frame cannot turn with it.
func TestAWindowCanTurnWithoutTakingTheShape(t *testing.T) {
	_, win, p := flipWindow(t)
	win.SetFlipShapesWindow(false)

	win.Flip()
	deadline := time.Now().Add(10 * time.Second)
	for p.Turning() {
		if time.Now().After(deadline) {
			t.Fatal("the turn was still running after ten seconds")
		}
		win.Capture()
		if win.flipShaped {
			t.Fatal("the window took the card's shape after being told not to")
		}
	}
	if !win.ShowingBack() {
		t.Error("the window did not turn over")
	}
}

// The window's own flip answers for itself: which face is up, and the
// panel for an application that wants to configure the projection.
func TestAWindowSaysWhichFaceIsUp(t *testing.T) {
	_, win, p := flipWindow(t)
	if win.ShowingBack() {
		t.Error("a new window starts on its back")
	}
	if win.FlipPanel() != p {
		t.Error("the window does not hand back the panel it built")
	}
	win.Flip()
	runWindowTurn(t, win, p)
	if !win.ShowingBack() {
		t.Error("the back is not up after a turn")
	}
}

// A window with no faces is not broken by the flip calls.
func TestAWindowWithNoFacesIgnoresTheFlip(t *testing.T) {
	a := New(Options{Headless: true, Scale: 1})
	win, err := a.NewWindow(platform.WindowOptions{Width: 200, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	win.Flip()
	if win.ShowingBack() || win.FlipPanel() != nil {
		t.Error("a window with no faces claims to have some")
	}
}
