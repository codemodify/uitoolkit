package app

import (
	"testing"

	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
)

// tearRig is a window whose box tears a "tab" out of itself, the window
// that tear-off opens, and what the tear-off was told.
type tearRig struct {
	app  *Application
	win  *Window
	box  *dragBox
	off  *platform.Offscreen
	torn *Window
	// opens counts the windows the tear-off asked for, res what it was
	// told in the end and ended whether it was told at all.
	opens int
	res   widget.TearResult
	ended bool
}

// newTearRig makes the source window and the drag that tears off. open
// says whether a window opens at all (a source that changes its mind
// returns nil).
func newTearRig(t *testing.T, open bool) *tearRig {
	t.Helper()
	// The private type first and the document's own after it, as a
	// torn-off tab offers them: a strip of ours takes the tab whole, and
	// anything else takes what it knows.
	d := widget.NewDrag([]string{"application/x-uitoolkit-tab", "text/uri-list"},
		map[string][]byte{
			"application/x-uitoolkit-tab": []byte("Home"),
			"text/uri-list":               []byte("file:///home/ada\r\n"),
		})
	d.Actions, d.Preferred = platform.DragMove, platform.DragMove
	d.Payload = "Home"
	a, w, box, off := dragWindow(t, d)
	r := &tearRig{app: a, win: w, box: box, off: off}
	tear := &widget.TearOff{
		Offset: paintengine2d.Pt(30, 12),
		Open: func() widget.TearOffWindow {
			r.opens++
			if !open {
				return nil
			}
			win, err := a.NewWindow(platform.WindowOptions{Width: 240, Height: 160, Headless: true})
			if err != nil {
				t.Fatal(err)
			}
			win.SetContent(widgets.NewColumn())
			r.torn = win
			return win
		},
		Done: func(res widget.TearResult, win widget.TearOffWindow) {
			r.res, r.ended = res, true
			if win != nil && res != widget.TearKept {
				win.Close()
			}
		},
	}
	if !w.StartTearOff(d, tear) {
		t.Fatal("StartTearOff")
	}
	return r
}

// A tear-off on a desktop that carries windows opens its window at once
// and hands it to the drag, under the pointer where it was grabbed.
func TestTearOffCarriesItsWindow(t *testing.T) {
	r := newTearRig(t, true)
	defer r.win.Close()

	if !r.win.DragsWindows() {
		t.Fatal("the offscreen desktop carries windows")
	}
	if r.opens != 1 || r.torn == nil {
		t.Fatalf("the window opens as the drag starts: opens=%d", r.opens)
	}
	got, dx, dy := r.off.DragToplevel()
	if got != r.torn.Surface() || dx != 30 || dy != 12 {
		t.Fatalf("attached %v at %d,%d", got, dx, dy)
	}
	p, running := r.off.DragOffer()
	if !running || !p.Toplevel {
		t.Fatalf("the drag must say it may carry a window: %+v", p)
	}
	// The compositor moves it with the pointer.
	r.off.Move(100, 50)
	r.off.SimulateDragOver(paintengine2d.Pt(200, 120))
	if x, y, ok := r.torn.Position(); !ok || x != 100+200-30 || y != 50+120-12 {
		t.Fatalf("the window follows the pointer: %d,%d (ok=%v)", x, y, ok)
	}
	r.off.SimulateDragRelease()
	r.app.PumpOnce()
	if !r.ended || r.res != widget.TearKept {
		t.Fatalf("a drop over nothing keeps the window: res=%v ended=%v", r.res, r.ended)
	}
	if r.torn.Closed() {
		t.Fatal("the window the user dropped on the desktop must stay")
	}
	r.torn.Close()
}

// A drop a target takes merges: whoever took it holds the content, so the
// window that carried it goes.
func TestTearOffMergedClosesItsWindow(t *testing.T) {
	r := newTearRig(t, true)
	defer r.win.Close()
	r.box.takes = true
	r.box.allow = platform.DragMove

	r.off.SimulateDragOver(paintengine2d.Pt(20, 20))
	r.app.PumpOnce()
	r.off.SimulateDragDrop(paintengine2d.Pt(20, 20))
	r.app.PumpOnce()
	if r.box.drops != 1 {
		t.Fatalf("the box took %d drops", r.box.drops)
	}
	if r.res != widget.TearMerged {
		t.Fatalf("res=%v", r.res)
	}
	if !r.torn.Closed() {
		t.Fatal("a merged tear-off closes the window it was carried in")
	}
}

// Escape (the desktop cancelling the drag) undoes the tear-off: the
// window goes and the source is told to take its content back.
func TestTearOffCancelledClosesItsWindow(t *testing.T) {
	r := newTearRig(t, true)
	defer r.win.Close()

	r.off.CancelDrag()
	r.app.PumpOnce()
	if r.res != widget.TearCancelled {
		t.Fatalf("res=%v", r.res)
	}
	if !r.torn.Closed() {
		t.Fatal("a cancelled tear-off closes the window it made")
	}
}

// Without the protocol the drag carries only its picture and the window
// is made at the drop — and never at all when the drag is cancelled.
func TestTearOffWithoutToplevelDragOpensAtTheDrop(t *testing.T) {
	for _, tc := range []struct {
		name   string
		end    func(r *tearRig)
		opens  int
		result widget.TearResult
	}{
		{"dropped on the desktop", func(r *tearRig) { r.off.SimulateDragRelease() }, 1, widget.TearKept},
		{"cancelled", func(r *tearRig) { r.off.CancelDrag() }, 0, widget.TearCancelled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := widget.NewDrag([]string{"application/x-uitoolkit-tab"}, nil)
			a, w, _, off := dragWindow(t, d)
			defer w.Close()
			off.SetDragsToplevels(false)
			r := &tearRig{app: a, win: w, off: off}
			if w.DragsWindows() {
				t.Fatal("this desktop cannot carry a window")
			}
			ok := w.StartTearOff(d, &widget.TearOff{
				Open: func() widget.TearOffWindow {
					r.opens++
					win, err := a.NewWindow(platform.WindowOptions{Width: 240, Height: 160, Headless: true})
					if err != nil {
						t.Fatal(err)
					}
					r.torn = win
					return win
				},
				Done: func(res widget.TearResult, win widget.TearOffWindow) {
					r.res, r.ended = res, true
					if win != nil && res != widget.TearKept {
						win.Close()
					}
				},
			})
			if !ok {
				t.Fatal("StartTearOff")
			}
			if r.opens != 0 {
				t.Fatal("nothing opens while the drag runs: the source still shows the tab")
			}
			tc.end(r)
			a.PumpOnce()
			if r.opens != tc.opens || r.res != tc.result {
				t.Fatalf("opens=%d res=%v", r.opens, r.res)
			}
			if r.torn != nil {
				r.torn.Close()
			}
		})
	}
}

// A source that opens no window has torn nothing off: a drop over nothing
// ends as a cancel, so the content goes back where it was.
func TestTearOffWithNoWindowCancels(t *testing.T) {
	r := newTearRig(t, false)
	defer r.win.Close()

	if r.opens != 1 || r.torn != nil {
		t.Fatalf("opens=%d", r.opens)
	}
	if _, _, dy := r.off.DragToplevel(); dy != 0 {
		t.Fatal("nothing to attach")
	}
	r.off.SimulateDragRelease()
	r.app.PumpOnce()
	if r.res != widget.TearCancelled {
		t.Fatalf("res=%v", r.res)
	}
}

// A window on Wayland is not told where it is, and says so rather than
// inventing a position.
func TestWindowPositionFollowsTheBackend(t *testing.T) {
	a, w, _, off := dragWindow(t, widget.DragText("x"))
	defer w.Close()
	off.Move(64, 48)
	if x, y, ok := w.Position(); !ok || x != 64 || y != 48 {
		t.Fatalf("got %d,%d (ok=%v)", x, y, ok)
	}
	w.Close()
	a.PumpOnce()
	if _, _, ok := w.Position(); ok {
		t.Fatal("a closed window has no position")
	}
}
