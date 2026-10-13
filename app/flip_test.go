package app

import (
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
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

// **A window can say its turn carries meaning**, and then it turns
// although the user's preference is to reduce motion — which is what the
// preference is for: the decoration, not the thing itself.
//
// The window is where an application says it once. Its content, the
// dialogs it opens, its cover and its notes are separate trees with this
// window as their host, so there is no component they all share to say it
// on.
func TestAWindowCanSayItsTurnCarriesMeaning(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "")
	a := New(Options{Headless: true, Scale: 1})
	win, err := a.NewWindow(platform.WindowOptions{Width: 520, Height: 360, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(win.Close)
	p := win.SetFlipFaces(widgets.NewButton("front", nil), widgets.NewButton("back", nil))
	p.Strips = 32
	a.PumpOnce()
	win.Capture()
	// After New, which applies the appearance file and would overwrite it.
	style.SetReduceMotion(true)
	t.Cleanup(func() { style.SetReduceMotion(false) })

	win.Flip()
	win.Capture()
	if p.Turning() {
		t.Fatal("the preference stopped being the default for a window that says nothing")
	}
	p.ShowFront()

	win.SetMotion(style.MotionAlways)
	if got := win.Motion(); got != style.MotionAlways {
		t.Fatalf("the window reads back %v", got)
	}
	win.Flip()
	win.Capture()
	if !p.Turning() {
		t.Error("the window said its turn carries meaning and the preference still cut it")
	}
}

// framedFlipWindow is a window wearing the toolkit's own frame whose
// whole content is two faces, in a pack whose caption is unmistakable in
// a picture: Hangar's close button is a block of ember.
func framedFlipWindow(t *testing.T) (*Application, *Window, *widgets.FlipPanel) {
	t.Helper()
	t.Setenv(style.AnimationsEnv, "1")
	t.Setenv(platform.EnvDecorations, "")
	p, ok := style.LoadTheme("hangar")
	if !ok {
		t.Skip("the hangar pack is not in this build")
	}
	a := New(Options{Look: p.Look(), Headless: true, Scale: 1})
	win, err := a.NewWindow(platform.WindowOptions{Title: "Player", Width: 520, Height: 360,
		Headless: true, Decorations: platform.DecorationsClient})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(win.Close)
	fp := win.SetFlipFaces(widgets.NewButton("front", nil), widgets.NewButton("back", nil))
	fp.Strips = 32
	a.PumpOnce()
	win.Capture()
	if !win.geom.framed {
		t.Fatal("the window is not drawing its own frame; this test is not testing what it says")
	}
	return a, win, fp
}

// **A framed window turns over with its frame.**
//
// Reported as a gap: `Window.Flip` worked only on a window with no frame
// of the toolkit's, and the frame is what supplies the resize edges, the
// minimise and the maximise — so an application that wanted the turn had
// to give those up and draw a caption of its own on each face. The panel
// could not do otherwise: it is the window's content, every component is
// clipped to its own box, and the frame is drawn outside that box.
//
// So the window turns as one picture. This asks the two questions that
// says: the caption is *in* the turning card, and the card is all there
// is of the window while it turns.
func TestAFramedWindowTurnsWithItsFrame(t *testing.T) {
	_, win, fp := framedFlipWindow(t)
	accent := win.look.Palette().Accent

	// At rest the caption is there, with its ember close button.
	flat := win.Capture()
	if flat == nil {
		t.Fatal("nothing captured")
	}
	if n := countNear(flat, accent, 0.08); n < 50 {
		t.Fatalf("the window at rest has %d accent pixels; the caption's close button is not in it", n)
	}

	win.Flip()
	if !win.Turning() {
		t.Fatal("the window is not turning")
	}
	// The content under the picture is already the face being turned to,
	// so when the picture goes there is nothing left to swap.
	if !fp.Showing() {
		t.Error("the content did not swap to the back under the picture")
	}
	if fp.Turning() {
		t.Error("the content is turning inside a window that is turning: the card would turn twice")
	}

	// A turn runs on the real clock, so the loop does as little as it
	// can: one capture to draw the frame, the shape's width read off the
	// raster, and exactly one picture kept — the first frame the card
	// has plainly turned in. Counting pixels inside the loop spent the
	// four hundred milliseconds it has on arithmetic instead of frames.
	wide := win.windowBox().Dx()
	var widths []int
	var turned *paintengine2d.Image
	deadline := time.Now().Add(10 * time.Second)
	for win.Turning() {
		if time.Now().After(deadline) {
			t.Fatal("the turn was still running after ten seconds")
		}
		img := win.Capture()
		if img == nil || !win.Turning() {
			break
		}
		r := win.shapeRaster()
		if r == nil {
			t.Fatal("a turning window has no silhouette: the shape is not following the card")
		}
		widths = append(widths, r.Bounds.W)
		if turned == nil && float32(r.Bounds.W) < wide*0.8 {
			turned = img.Clone()
		}
	}
	if len(widths) < 3 {
		t.Fatalf("only %d frames were drawn; the turn did not run", len(widths))
	}
	narrowest := widths[0]
	for _, w := range widths {
		if w > int(wide) {
			t.Fatalf("the silhouette is %d wide on a %v window: it is not the card", w, wide)
		}
		if w < narrowest {
			narrowest = w
		}
	}
	if float32(narrowest) > wide*0.8 {
		t.Errorf("the narrowest frame is %d of %v: the card never turned far enough to be one", narrowest, wide)
	}
	caption := 0
	if turned != nil && countNear(turned, accent, 0.08) > 20 {
		caption++
	}
	if caption == 0 {
		t.Error("no frame of the turn had the caption's ember in it: the frame did not turn with the window")
	}

	// And at the end the window is itself again: frame, caption, buttons.
	win.Capture()
	if win.turn != nil {
		t.Error("the picture is still over the window after the turn")
	}
	if win.flipShaped {
		t.Error("the window kept the card's shape after the turn")
	}
	done := win.Capture()
	if n := countNear(done, accent, 0.08); n < 50 {
		t.Errorf("the window has %d accent pixels after the turn; the caption did not come back", n)
	}
}

// countNear is how many pixels of img are within tol of col on every
// channel, alpha included only as "not transparent".
func countNear(img *paintengine2d.Image, col paintengine2d.Color, tol float32) int {
	if img == nil {
		return 0
	}
	n := 0
	near := func(a, b float32) bool { return a-b < tol && b-a < tol }
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			p := img.NRGBAAt(x, y)
			if p.A < 128 {
				continue
			}
			if near(float32(p.R)/255, col.R) && near(float32(p.G)/255, col.G) && near(float32(p.B)/255, col.B) {
				n++
			}
		}
	}
	return n
}

// **A maximized window turns against its own ground, not against itself.**
//
// A silhouette is dropped where the window is maximized or tiled — it
// fills a box the desktop chose and shares its edges with the screen —
// so the card cannot be cut out of the window there. Were the window
// still drawn under its own picture it would stand flat behind the
// turning one: two windows, one of them rotating.
//
// What says so is the strip beside the card. Drawn under its picture,
// that strip is the window, pixel for pixel what it looks like at rest;
// with nothing under the picture it is the ground the window paints and
// nothing else.
func TestAMaximizedWindowDoesNotStandBehindItsOwnTurn(t *testing.T) {
	for _, scene := range []string{"1", "0"} {
		t.Run("scene="+scene, func(t *testing.T) {
			t.Setenv(platform.EnvScene, scene)
			maximizedTurnStandsAlone(t)
		})
	}
}

// maximizedTurnStandsAlone is the body, run once against each painter:
// the recorded one and the immediate one. A layer the window draws has
// to be in both, and this one was added to both.
func maximizedTurnStandsAlone(t *testing.T) {
	a, win, _ := framedFlipWindow(t)
	o := win.Surface().(*platform.Offscreen)
	o.SimulateWindowState(platform.WindowState{Activated: true, Maximized: true})
	a.PumpOnce()
	win.Capture()
	if !win.state.Maximized {
		t.Fatal("the window did not maximize")
	}

	win.Flip()
	if !win.Turning() {
		t.Fatal("a maximized window did not turn")
	}
	if win.shapeRaster() != nil {
		t.Fatal("a maximized window took a shape; this test is about the case where it cannot")
	}
	// The first frame whose card leaves a wide strip of window beside it.
	var mid *paintengine2d.Image
	var cardRight float32
	deadline := time.Now().Add(10 * time.Second)
	for win.Turning() && mid == nil {
		if time.Now().After(deadline) {
			t.Fatal("the turn was still running after ten seconds")
		}
		img := win.Capture()
		if img == nil || !win.Turning() {
			break
		}
		p := win.turn
		if p == nil || p.CardOutline() == nil {
			continue
		}
		b := p.CardOutline().Bounds()
		if b.Dx() < win.windowBox().Dx()*0.7 {
			mid, cardRight = img.Clone(), b.Max.X
		}
	}
	if mid == nil {
		t.Skip("the turn never reached a frame narrow enough to read; the clock beat the painter")
	}
	for win.Turning() {
		win.Capture()
	}
	flat := win.Capture()
	if flat == nil || flat.Width != mid.Width || flat.Height != mid.Height {
		t.Fatal("the window at rest is not the same size as the window turning")
	}

	// Beside the card, down the whole height: identical to the window at
	// rest means the window is being drawn there.
	x0 := int(cardRight) + 8
	if x0 >= mid.Width-4 {
		t.Skipf("the card reaches %v of %d: no strip beside it to read", cardRight, mid.Width)
	}
	same, total := 0, 0
	for y := 0; y < mid.Height; y += 3 {
		for x := x0; x < mid.Width; x += 3 {
			total++
			if mid.NRGBAAt(x, y) == flat.NRGBAAt(x, y) {
				same++
			}
		}
	}
	if total == 0 {
		t.Fatal("no pixels beside the card to compare")
	}
	if float32(same) > float32(total)*0.9 {
		t.Errorf("%d of %d pixels beside the turning card are the window at rest, pixel for pixel: it is standing behind its own turn",
			same, total)
	}
}

// **The photographs of a framed window are wiped when its turn ends.**
//
// A framed window turns as two pictures of itself, so the turn holds
// two copies of whatever the window showed — for secretvault, two
// copies of a password on screen (item 78). The panel wipes its own
// rasterised copies; these are the window's to wipe.
func TestTheWindowsPhotographsAreWipedWhenTheTurnEnds(t *testing.T) {
	_, win, _ := framedFlipWindow(t)
	win.Flip()
	if win.turn == nil {
		t.Fatal("the window is not turning as a picture")
	}
	var shots []*paintengine2d.Image
	for _, c := range []widget.Component{win.turn.Front(), win.turn.Back()} {
		s, ok := c.(*flipShot)
		if !ok || s.img == nil {
			t.Fatal("a face of the turn is not a photograph")
		}
		shots = append(shots, s.img)
	}
	set := 0
	for _, im := range shots {
		for _, b := range im.Pix {
			if b != 0 {
				set++
			}
		}
	}
	if set == 0 {
		t.Fatal("the photographs are blank; this test cannot tell a wipe from nothing")
	}

	deadline := time.Now().Add(10 * time.Second)
	for win.Turning() {
		if time.Now().After(deadline) {
			t.Fatal("the turn was still running after ten seconds")
		}
		win.Capture()
	}
	left := 0
	for _, im := range shots {
		for _, b := range im.Pix {
			if b != 0 {
				left++
			}
		}
	}
	if left != 0 {
		t.Errorf("%d bytes of the window's two photographs are still set after the turn", left)
	}
}

// **A framed window comes back the way it went.**
//
// A card turns one way to its back and the other way home, so it
// retraces its turn rather than spinning on through a second half-turn
// — the panel's own `dir`. The window's stand-in panel was built fresh
// each time with the picture before as its front and `Flip` called on
// it, so every turn went round the same way (secretvault item 91).
//
// The direction shows in where the card sits part way through: turning
// to the back, the near edge is on one side; coming home, the other.
func TestAFramedWindowComesBackTheWayItWent(t *testing.T) {
	_, win, _ := framedFlipWindow(t)

	// Part way through a turn, which side of the window the card's
	// bounding box is pushed towards.
	side := func() float32 {
		deadline := time.Now().Add(10 * time.Second)
		var at float32 = -1
		for win.Turning() {
			if time.Now().After(deadline) {
				t.Fatal("the turn was still running after ten seconds")
			}
			win.Capture()
			p := win.turn
			if p == nil || p.CardOutline() == nil {
				continue
			}
			b := p.CardOutline().Bounds()
			if w := win.windowBox().Dx(); at < 0 && b.Dx() < w*0.85 && b.Dx() > w*0.4 {
				// The middle of the card against the middle of the window.
				at = (b.Min.X + b.Max.X) / 2 / w
			}
		}
		return at
	}

	win.Flip()
	toBack := side()
	win.Capture()
	if !win.ShowingBack() {
		t.Fatal("the window did not turn to its back")
	}
	win.Flip()
	home := side()
	win.Capture()
	if win.ShowingBack() {
		t.Fatal("the window did not come home")
	}
	if toBack < 0 || home < 0 {
		t.Skip("neither turn was caught part way; the clock beat the painter")
	}
	// Turning one way leans the card to one side of the window and the
	// other way to the other, so the two are on opposite sides of the
	// middle. The same way twice puts them on the same side.
	if (toBack-0.5)*(home-0.5) > 0 {
		t.Errorf("both turns leaned the same way (%.3f and %.3f of the window): the card does not retrace", toBack, home)
	}
}
