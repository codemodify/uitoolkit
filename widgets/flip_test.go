package widgets

import (
	"github.com/codemodify/uitoolkit/platform"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

const (
	flipW = 400
	flipH = 300
	// flipTestStrips divides flipW exactly, so at rest every column is a
	// whole number of pixels wide and a frame drawn flat is a copy of the
	// face rather than a resample of it (see
	// TestTheFirstFrameOfATurnIsTheFaceItStartedFrom).
	flipTestStrips = 100
)

// flipFaces builds a panel with a button on each side, so each face has
// something that takes focus and answers a hit test.
func flipFaces(t *testing.T) (*FlipPanel, *Button, *Button, *timerHost) {
	t.Helper()
	front := NewButton("front", nil)
	back := NewButton("back", nil)
	f := NewFlipPanel(front, back)
	h := &timerHost{look: style.DarkLook(), now: time.Unix(1000, 0)}
	f.SetHost(h)
	front.SetHost(h)
	back.SetHost(h)
	f.Strips = flipTestStrips
	f.Arrange(paintengine2d.XYWH(0, 0, flipW, flipH))
	return f, front, back, h
}

// useFlipClock runs the panel's clock off the host's hand-driven one.
func useFlipClock(t *testing.T, h *timerHost) {
	t.Helper()
	old := flipNow
	t.Cleanup(func() { flipNow = old })
	flipNow = func() time.Time { return h.now }
}

// paintPanel paints the panel into a fresh image w x h and returns it.
// The image is deliberately larger than the panel in
// TestTheCardNeverLeavesThePanel, which is why the size is a parameter.
func paintPanel(f *FlipPanel, w, h int) *paintengine2d.Image {
	img := paintengine2d.NewImage(w, h)
	ctx := paintengine2d.NewContext(img)
	f.Paint(ctx)
	img.Touch()
	return img
}

// differingPixels counts pixels where a and b differ by more than tol on
// any channel.
func differingPixels(a, b *paintengine2d.Image, tol int) int {
	if a.Width != b.Width || a.Height != b.Height {
		return a.Width * a.Height
	}
	n := 0
	for i := 0; i+3 < len(a.Pix) && i+3 < len(b.Pix); i += 4 {
		for c := 0; c < 4; c++ {
			d := int(a.Pix[i+c]) - int(b.Pix[i+c])
			if d < 0 {
				d = -d
			}
			if d > tol {
				n++
				break
			}
		}
	}
	return n
}

// paintedColumns is the first and last column of img holding anything at
// all, or (-1, -1) for an empty image.
func paintedColumns(img *paintengine2d.Image) (first, last int) {
	first, last = -1, -1
	for x := 0; x < img.Width; x++ {
		for y := 0; y < img.Height; y++ {
			if img.Pix[(y*img.Width+x)*4+3] != 0 {
				if first < 0 {
					first = x
				}
				last = x
				break
			}
		}
	}
	return first, last
}

// mirrorImage is img reflected left to right.
func mirrorImage(img *paintengine2d.Image) *paintengine2d.Image {
	out := paintengine2d.NewImage(img.Width, img.Height)
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			s := (y*img.Width + x) * 4
			d := (y*img.Width + (img.Width - 1 - x)) * 4
			copy(out.Pix[d:d+4], img.Pix[s:s+4])
		}
	}
	out.Touch()
	return out
}

// With motion off the faces swap at once: no turn to run, and the
// announcement comes straight away rather than at the end of an animation
// that never happens.
func TestAFlipWithoutMotionSwapsAtOnce(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "0")
	f, front, back, _ := flipFaces(t)
	var got []bool
	f.OnFlip = func(b bool) { got = append(got, b) }

	f.Flip()
	if f.Turning() {
		t.Error("the panel is turning with motion off")
	}
	if !f.Showing() {
		t.Error("the back is not the face up")
	}
	if front.Visible() || !back.Visible() {
		t.Errorf("front visible=%v back visible=%v", front.Visible(), back.Visible())
	}
	if len(got) != 1 || !got[0] {
		t.Errorf("OnFlip got %v, want one call with true", got)
	}
}

// **The face that is down is out of reach.** It is not painted, it does
// not answer a hit test and it is not in the tab order — one hidden flag
// does all three, because widget.Walk stops at a hidden component. A back
// face that could be tabbed into while face down is the defect this
// guards: the settings on the back of the card would take the keyboard
// while the front was showing.
func TestTheFaceThatIsDownIsOutOfReach(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "0")
	f, front, back, _ := flipFaces(t)

	if cands := widget.Focusables(f); len(cands) != 1 || cands[0] != front {
		t.Errorf("face up is the front, tab order is %v, want just the front button", cands)
	}
	if hit := f.HitTest(paintengine2d.Pt(flipW/2, flipH/2)); hit == back {
		t.Error("a hit test reached the back face while it was face down")
	}

	f.Flip()
	if cands := widget.Focusables(f); len(cands) != 1 || cands[0] != back {
		t.Errorf("face up is the back, tab order is %v, want just the back button", cands)
	}
	if hit := f.HitTest(paintengine2d.Pt(flipW/2, flipH/2)); hit == front {
		t.Error("a hit test reached the front face while it was face down")
	}
}

// A turn runs, and ends at rest on the face it was turning to. While it
// runs neither face is live — both are pictures — and the announcement
// comes once, at the end.
func TestATurnEndsAtRestOnTheFaceItTurnedTo(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	style.SetReduceMotion(false)
	f, front, back, h := flipFaces(t)
	useFlipClock(t, h)
	var got []bool
	f.OnFlip = func(b bool) { got = append(got, b) }

	f.Flip()
	if !f.Turning() {
		t.Fatal("the panel is not turning")
	}
	if front.Visible() || back.Visible() {
		t.Error("a face is live while the card is a picture")
	}
	if len(got) != 0 {
		t.Errorf("OnFlip was called %v before the turn ended", got)
	}

	// Paint through the turn the way the window would, a frame at a time.
	for i := 0; i < 40 && f.Turning(); i++ {
		paintPanel(f, flipW, flipH)
		h.advance(flipFrame)
	}
	if f.Turning() {
		t.Fatalf("the turn is still running after %v", 40*flipFrame)
	}
	if !f.Showing() || !back.Visible() || front.Visible() {
		t.Errorf("at rest: showing back=%v, back visible=%v, front visible=%v", f.Showing(), back.Visible(), front.Visible())
	}
	if len(got) != 1 || !got[0] {
		t.Errorf("OnFlip got %v, want one call with true", got)
	}
}

// **The keyboard follows the card.** The face holding the focus is about
// to become a picture, so the focus is parked for the turn and handed to
// the face that comes up. Without this the window is left focused on a
// component nobody can see or reach.
func TestATurnHandsTheKeyboardToTheFaceComingUp(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	style.SetReduceMotion(false)
	f, front, back, h := flipFaces(t)
	useFlipClock(t, h)
	h.RequestFocus(front)

	f.Flip()
	if h.Focus() == front {
		t.Error("the focus is still on the face going down")
	}
	for i := 0; i < 40 && f.Turning(); i++ {
		paintPanel(f, flipW, flipH)
		h.advance(flipFrame)
	}
	if h.Focus() != back {
		t.Errorf("after the turn the focus is %v, want the back button", h.Focus())
	}
}

// A flip the user never had the keyboard in does not take it: a card
// turning in the corner of a window must not steal focus from the field
// someone is typing in.
func TestATurnDoesNotTakeAFocusItNeverHad(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	style.SetReduceMotion(false)
	f, _, back, h := flipFaces(t)
	useFlipClock(t, h)
	elsewhere := NewButton("elsewhere", nil)
	elsewhere.SetHost(h)
	h.RequestFocus(elsewhere)

	f.Flip()
	for i := 0; i < 40 && f.Turning(); i++ {
		paintPanel(f, flipW, flipH)
		h.advance(flipFrame)
	}
	if h.Focus() != elsewhere {
		t.Errorf("the turn moved the focus to %v", h.Focus())
	}
	if h.Focus() == back {
		t.Error("the turn took the keyboard into its own back face")
	}
}

// **The decomposition is checked against the face itself.** The first
// frame of a turn is flat — the projection at 0° is the box exactly — so
// the columns drawn then must add up to the picture the panel shows at
// rest. This is what catches a wrong source-to-destination mapping: a
// reversed column, an off-by-one in the strip bounds or a mirror applied
// at the wrong time all leave the card looking like something other than
// the face it started from.
func TestTheFirstFrameOfATurnIsTheFaceItStartedFrom(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	style.SetReduceMotion(false)
	f, _, _, h := flipFaces(t)
	useFlipClock(t, h)

	atRest := paintPanel(f, flipW, flipH)
	f.Flip()
	firstFrame := paintPanel(f, flipW, flipH)

	if !f.Turning() {
		t.Fatal("the turn ended on its first frame")
	}
	// The face at rest must not be blank, or this test would pass on two
	// empty images.
	if blank := paintengine2d.NewImage(flipW, flipH); differingPixels(atRest, blank, 8) < 100 {
		t.Fatal("the panel paints nothing at rest, so this proves nothing")
	}
	if n := differingPixels(atRest, firstFrame, 8); n > 0 {
		t.Errorf("the first frame of the turn differs from the face at rest in %d pixels", n)
	}
}

// Half way through, the card is visibly a turned face and not the face:
// it differs from flat, and it has pulled in from both edges of its box.
// Measured in pixels rather than by recomputing the angle, which would
// only check the easing against a second copy of itself.
func TestMidTurnTheCardIsTurned(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	style.SetReduceMotion(false)
	f, _, _, h := flipFaces(t)
	useFlipClock(t, h)

	atRest := paintPanel(f, flipW, flipH)
	if l, r := paintedColumns(atRest); l != 0 || r != flipW-1 {
		t.Fatalf("at rest the face spans columns %d..%d, want the whole box", l, r)
	}
	f.Flip()
	h.advance(flipDuration / 4)
	mid := paintPanel(f, flipW, flipH)
	if n := differingPixels(atRest, mid, 8); n < 1000 {
		t.Errorf("a quarter through the turn the card differs from flat in only %d pixels", n)
	}
	l, r := paintedColumns(mid)
	if l <= 0 || r >= flipW-1 {
		t.Errorf("a quarter through the turn the card spans columns %d..%d, still touching the box edges", l, r)
	}
}

// **The far side reads the right way round.** Past a quarter turn the
// geometry mirrors itself, and the back face's columns are sampled in
// reverse to undo that. Get it wrong and the back of the card is its own
// mirror image — legible as a shape, wrong as a page, and invisible to
// any test that only looks at the flat first frame.
func TestTheLastFrameOfATurnIsTheFaceItTurnedTo(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	style.SetReduceMotion(false)
	f, _, _, h := flipFaces(t)
	useFlipClock(t, h)

	// The back face flat, for comparison: turn with motion off, paint, and
	// turn back.
	f.ShowBack()
	for i := 0; i < 40 && f.Turning(); i++ {
		paintPanel(f, flipW, flipH)
		h.advance(flipFrame)
	}
	backFlat := paintPanel(f, flipW, flipH)
	f.ShowFront()
	for i := 0; i < 40 && f.Turning(); i++ {
		paintPanel(f, flipW, flipH)
		h.advance(flipFrame)
	}

	// Now turn to the back and stop a hair before the end, where the card
	// is all but flat and showing its far side.
	f.Flip()
	paintPanel(f, flipW, flipH)
	h.advance(flipDuration - time.Millisecond)
	last := paintPanel(f, flipW, flipH)
	if !f.Turning() {
		t.Fatal("the turn ended before its last frame")
	}

	mirrored := mirrorImage(backFlat)
	if differingPixels(backFlat, mirrored, 8) < 100 {
		t.Fatal("the back face is its own mirror image, so this proves nothing")
	}
	asIs := differingPixels(last, backFlat, 24)
	flipped := differingPixels(last, mirrored, 24)
	if asIs >= flipped {
		t.Errorf("the last frame differs from the back face in %d pixels and from its mirror in %d — the far side is drawn mirrored", asIs, flipped)
	}
}

// **The card never leaves the panel.** Painted into an image larger than
// the panel, with no clip to save it, nothing lands outside the box the
// panel was given. This is the normalisation in flipGeomAt doing its job
// where it matters — through the whole turn, in pixels.
func TestTheCardNeverLeavesThePanel(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	style.SetReduceMotion(false)
	f, _, _, h := flipFaces(t)
	useFlipClock(t, h)
	const padW, padH = flipW + 60, flipH + 60

	f.Flip()
	for i := 0; i < 40 && f.Turning(); i++ {
		img := paintPanel(f, padW, padH)
		for y := 0; y < padH; y++ {
			for x := 0; x < padW; x++ {
				if x < flipW && y < flipH {
					continue
				}
				if a := img.Pix[(y*padW+x)*4+3]; a != 0 {
					t.Fatalf("frame %d painted outside the panel at (%d,%d), alpha %d", i, x, y, a)
				}
			}
		}
		h.advance(flipFrame)
	}
}

// The card is one size whichever way up it is, so a flip never moves the
// layout around it.
func TestTheCardIsTheLargerOfItsTwoFaces(t *testing.T) {
	front := NewHeightBox(50, NewLabel("front"))
	back := NewHeightBox(120, NewLabel("back"))
	f := NewFlipPanel(front, back)
	f.SetHost(&host{})
	sz := f.Measure(layout.Unbounded())
	if sz.Y < 120 {
		t.Errorf("the card measures %v high, want at least the taller face's 120", sz.Y)
	}
	f.ShowBack()
	if sz2 := f.Measure(layout.Unbounded()); sz2 != sz {
		t.Errorf("the card measures %v face down and %v face up", sz2, sz)
	}
}

// A press while the card is turning goes nowhere: it is a picture, not a
// target, and letting it bubble would hand the press to whatever is
// behind the panel.
func TestAPressIsSwallowedWhileTurning(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	style.SetReduceMotion(false)
	f, _, _, h := flipFaces(t)
	useFlipClock(t, h)

	f.Flip()
	if !f.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(flipW/2, flipH/2)}) {
		t.Error("a press during the turn was not taken")
	}
	for i := 0; i < 40 && f.Turning(); i++ {
		paintPanel(f, flipW, flipH)
		h.advance(flipFrame)
	}
	if f.MousePress(widget.MouseEvent{Pos: paintengine2d.Pt(1, 1)}) {
		t.Error("a press at rest was swallowed")
	}
}

// A one-sided panel turns to a blank back rather than refusing to turn.
func TestAPanelWithOneFaceStillTurns(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	style.SetReduceMotion(false)
	front := NewButton("front", nil)
	f := NewFlipPanel(front, nil)
	h := &timerHost{look: style.DarkLook(), now: time.Unix(1000, 0)}
	f.SetHost(h)
	front.SetHost(h)
	f.Arrange(paintengine2d.XYWH(0, 0, flipW, flipH))
	useFlipClock(t, h)

	f.Flip()
	for i := 0; i < 40 && f.Turning(); i++ {
		paintPanel(f, flipW, flipH)
		h.advance(flipFrame)
	}
	if !f.Showing() {
		t.Error("the panel did not turn to its blank back")
	}
	f.Flip()
	for i := 0; i < 40 && f.Turning(); i++ {
		paintPanel(f, flipW, flipH)
		h.advance(flipFrame)
	}
	if f.Showing() {
		t.Error("the panel did not turn back to its front")
	}
}

// runTurn paints a turn out frame by frame, recording the card's outline
// at each one.
func runTurn(f *FlipPanel, h *timerHost) []paintengine2d.Rect {
	var out []paintengine2d.Rect
	for i := 0; i < 60 && f.Turning(); i++ {
		paintPanel(f, flipW, flipH)
		if !f.Turning() {
			break
		}
		if o := f.CardOutline(); o != nil {
			out = append(out, o.Bounds())
		}
		h.advance(flipFrame)
	}
	return out
}

// **A card comes back the way it went.** Turning to the back and turning
// to the front are opposite rotations, so the second is the mirror image
// of the first frame for frame — the near edge on the other side, every
// column travelling the other way. Without a direction the card turned
// the same way twice, which reads as it spinning on through a second
// half-turn rather than being put back.
func TestACardTurnsBackTheWayItCame(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	style.SetReduceMotion(false)
	f, _, _, h := flipFaces(t)
	useFlipClock(t, h)

	f.Flip() // to the back
	there := runTurn(f, h)
	if f.Turning() || !f.Showing() {
		t.Fatalf("the first turn did not finish on the back (turning=%v showing=%v)", f.Turning(), f.Showing())
	}
	f.Flip() // and back to the front
	back := runTurn(f, h)
	if f.Turning() || f.Showing() {
		t.Fatalf("the second turn did not finish on the front (turning=%v showing=%v)", f.Turning(), f.Showing())
	}

	if len(there) < 4 || len(back) < 4 {
		t.Fatalf("the turns drew %d and %d frames", len(there), len(back))
	}
	if len(there) != len(back) {
		t.Fatalf("the turns drew %d frames and %d; they are not the same turn", len(there), len(back))
	}

	// The card has to be off-centre at some point, or "mirrored" would be
	// satisfied by a turn that is symmetric and says nothing.
	off := false
	for _, r := range there {
		if absF(r.Min.X-(flipW-r.Max.X)) > 5 {
			off = true
			break
		}
	}
	if !off {
		t.Fatal("the card is centred at every frame, so a mirror image proves nothing")
	}

	for i := range there {
		a, b := there[i], back[i]
		// Mirrored about the middle of the box: a's left edge is as far
		// from the left as b's right edge is from the right.
		if d := absF(a.Min.X - (flipW - b.Max.X)); d > 1 {
			t.Errorf("frame %d: there spans %.1f..%.1f, back spans %.1f..%.1f — not a mirror image (%.2f out)",
				i, a.Min.X, a.Max.X, b.Min.X, b.Max.X, d)
		}
		if d := absF(a.Dx() - b.Dx()); d > 1 {
			t.Errorf("frame %d: there is %.1f wide and back is %.1f", i, a.Dx(), b.Dx())
		}
	}
}

// **A turn the application says is the thing itself runs under the
// reduce-motion preference.**
//
// Reported from an application whose window would not turn: the user's
// look.json had `reduceMotion: true`, and that preference is the right
// default for a hover fade and the wrong one for a card turning over to
// show it has a back — switched off, the back simply appears and the one
// cue that said "this is the same object, seen from behind" is gone. The
// application had nothing it could say about it.
//
// Now it has: style.MotionAlways on the panel, or on any ancestor of it,
// or on the window.
func TestAPanelThatSaysItsTurnCarriesMeaningTurnsUnderReducedMotion(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "")
	style.SetReduceMotion(true)
	t.Cleanup(func() { style.SetReduceMotion(false) })
	f, _, _, h := flipFaces(t)
	useFlipClock(t, h)

	f.Flip()
	if f.Turning() {
		t.Fatal("the preference stopped being the default; a panel that says nothing should swap at once")
	}
	f.ShowFront()

	f.SetMotion(style.MotionAlways)
	f.Flip()
	if !f.Turning() {
		t.Error("the panel said its turn carries meaning and the preference still cut it")
	}
	// And the other way: a panel told to stay still does, with motion on.
	t.Setenv(style.AnimationsEnv, "")
	style.SetReduceMotion(false)
	f.end()
	f.ShowFront()
	f.SetMotion(style.MotionNever)
	f.Flip()
	if f.Turning() {
		t.Error("a panel told to stay still turned anyway")
	}
}

// **A turn's pictures are zeroed when it ends.**
//
// A picture of a face is a copy of what the face showed. secretvault
// turns its window over with an item open and its password on screen,
// and wipes every copy it makes of one; these it could not reach — the
// panel dropped them unzeroed and they sat in memory until the
// collector happened to run. Reported as item 78 against v0.23.6.
func TestATurnsPicturesAreWipedWhenItEnds(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	f, _, _, h := flipFaces(t)
	useFlipClock(t, h)

	f.Flip()
	paintPanel(f, flipW, flipH)
	from, to := f.turn.from, f.turn.to
	if from == nil || to == nil {
		t.Fatal("the turn rasterised no pictures to wipe")
	}
	if ink(from)+ink(to) == 0 {
		t.Fatal("the pictures are blank before the turn ends; this test cannot tell a wipe from nothing")
	}

	h.now = h.now.Add(2 * flipDuration)
	paintPanel(f, flipW, flipH)
	if f.Turning() {
		t.Fatal("the turn did not end")
	}
	if n := ink(from) + ink(to); n != 0 {
		t.Errorf("%d bytes of the two pictures are still set after the turn: a copy of whatever the faces showed", n)
	}
	if f.turn.from != nil || f.turn.to != nil {
		t.Error("the panel is still holding the pictures")
	}
}

// And a panel taken down mid-turn wipes them too: a turn that never
// reaches its end would otherwise leave both copies behind.
func TestAPanelTakenDownMidTurnWipesItsPictures(t *testing.T) {
	t.Setenv(style.AnimationsEnv, "1")
	f, _, _, h := flipFaces(t)
	useFlipClock(t, h)

	f.Flip()
	paintPanel(f, flipW, flipH)
	from, to := f.turn.from, f.turn.to
	if from == nil || to == nil || ink(from)+ink(to) == 0 {
		t.Fatal("the turn rasterised nothing to wipe")
	}
	f.Dismissed()
	if n := ink(from) + ink(to); n != 0 {
		t.Errorf("%d bytes are still set after the panel was taken down mid-turn", n)
	}
	if f.Turning() {
		t.Error("the turn is still running after the panel was taken down")
	}
}

// ink is how many bytes of an image are not zero.
func ink(img *paintengine2d.Image) int {
	n := 0
	for _, b := range img.Pix {
		if b != 0 {
			n++
		}
	}
	return n
}

// **A key offered to the panel reaches the face that is up.**
//
// A window offers its own keys to its content root, which for a window
// that turns over is the flip panel. The panel had no KeyPress, so an
// application's shortcuts — the ones that work with the keyboard on the
// title bar or on nothing — reached neither face, and stopped working
// the day the turn became the toolkit's (secretvault item 90).
func TestAKeyReachesTheFaceThatIsUp(t *testing.T) {
	front, back := newKeyBox(), newKeyBox()
	f := NewFlipPanel(front, back)
	h := &timerHost{look: style.DarkLook(), now: time.Unix(1000, 0)}
	f.SetHost(h)
	f.Arrange(paintengine2d.XYWH(0, 0, flipW, flipH))

	if !f.KeyPress(widget.KeyEvent{Key: platform.KeyF5}) {
		t.Error("the face that is up did not take the key")
	}
	if front.keys != 1 || back.keys != 0 {
		t.Errorf("the key went front=%d back=%d; it belongs to the face that is up", front.keys, back.keys)
	}

	f.ShowNow(true)
	if !f.KeyPress(widget.KeyEvent{Key: platform.KeyF5}) {
		t.Error("after the turn the back did not take the key")
	}
	if front.keys != 1 || back.keys != 1 {
		t.Errorf("after turning, front=%d back=%d", front.keys, back.keys)
	}

	// A key nothing in the face took is not claimed by the panel.
	front.take, back.take = false, false
	if f.KeyPress(widget.KeyEvent{Key: platform.KeyF5}) {
		t.Error("the panel claimed a key its face refused")
	}
}

// keyBox counts the keys it is offered and says whether it took them.
type keyBox struct {
	widget.Base
	keys int
	take bool
}

func newKeyBox() *keyBox {
	b := &keyBox{take: true}
	b.Init(b)
	return b
}

func (b *keyBox) KeyPress(widget.KeyEvent) bool {
	if !b.take {
		return false
	}
	b.keys++
	return true
}
