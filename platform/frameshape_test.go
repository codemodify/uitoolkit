package platform

import (
	"math"
	"testing"

	"github.com/codemodify/paintengine2d"
)

// The geometry a translucent frame hands the window system: the margin
// around the visible window, the resize band inside it, the opaque region
// with the rounded corners left out, and the alpha the buffers carry.

func TestFrameMarginIsWholeLogicalPixels(t *testing.T) {
	// A margin has to be a whole number of logical pixels (Wayland states
	// window geometry in them) and land on a whole device pixel too, or
	// the visible window starts half a pixel off the compositor's grid.
	for _, sc := range []float32{1, 1.25, 1.5, 1.75, 2, 2.5} {
		got := FrameMargin(10, sc)
		if got <= 0 {
			t.Fatalf("scale %v: margin %d", sc, got)
		}
		if float32(got) < 10 {
			t.Fatalf("scale %v: margin %d is below the %v device pixels asked for", sc, got, 10)
		}
		logical := float64(got) / float64(sc)
		if math.Abs(logical-math.Round(logical)) > 1e-3 {
			t.Fatalf("scale %v: margin %d is %.3f logical pixels", sc, got, logical)
		}
		// And not wastefully larger than what was asked for.
		if float32(got) > 10+4*sc {
			t.Fatalf("scale %v: margin %d is far past the 10 device pixels asked for", sc, got)
		}
	}
	if FrameMargin(0, 1.75) != 0 {
		t.Fatal("no shadow, no margin")
	}
}

func TestInputRectIsTheWindowPlusTheBand(t *testing.T) {
	win := FrameRect{X: 20, Y: 16, W: 600, H: 400}
	m := FrameInsets{Top: 16, Right: 20, Bottom: 28, Left: 20}
	in := FrameInsets{Top: 10, Right: 10, Bottom: 10, Left: 10}
	got := InputRect(win, m, in)
	want := FrameRect{X: 10, Y: 6, W: 620, H: 420}
	if got != want {
		t.Fatalf("input %+v want %+v", got, want)
	}
	// The band never reaches past the margin: a thin margin caps it.
	thin := FrameInsets{Top: 4, Right: 4, Bottom: 4, Left: 4}
	got = InputRect(FrameRect{X: 4, Y: 4, W: 100, H: 80}, thin, in)
	if got != (FrameRect{X: 0, Y: 0, W: 108, H: 88}) {
		t.Fatalf("thin margin: %+v", got)
	}
	// No margin at all: the window is the whole of it.
	if got := InputRect(FrameRect{W: 100, H: 80}, FrameInsets{}, in); got != (FrameRect{W: 100, H: 80}) {
		t.Fatalf("no margin: %+v", got)
	}
}

func TestOpaqueRectsLeaveTheRoundedCornersOut(t *testing.T) {
	win := FrameRect{X: 20, Y: 16, W: 600, H: 400}
	// Square: one rect, the whole window.
	if got := OpaqueRects(win, [4]float32{}, 1); len(got) != 1 || got[0] != win {
		t.Fatalf("square window: %+v", got)
	}
	// Rounded all round: a middle band plus the strips between the corners.
	got := OpaqueRects(win, [4]float32{8, 8, 8, 8}, 1)
	if len(got) != 3 {
		t.Fatalf("rounded window: %+v", got)
	}
	if got[0] != (FrameRect{X: 20, Y: 24, W: 600, H: 384}) {
		t.Fatalf("middle band %+v", got[0])
	}
	if got[1] != (FrameRect{X: 28, Y: 16, W: 584, H: 8}) {
		t.Fatalf("top strip %+v", got[1])
	}
	if got[2] != (FrameRect{X: 28, Y: 408, W: 584, H: 8}) {
		t.Fatalf("bottom strip %+v", got[2])
	}
	// No corner of any rect may sit inside a corner's square.
	for _, r := range got {
		for _, c := range [][2]int{{win.X, win.Y}, {win.X + win.W - 1, win.Y}, {win.X, win.Y + win.H - 1}, {win.X + win.W - 1, win.Y + win.H - 1}} {
			if c[0] >= r.X && c[0] < r.X+r.W && c[1] >= r.Y && c[1] < r.Y+r.H {
				t.Fatalf("rect %+v claims the rounded corner %v as opaque", r, c)
			}
		}
	}
	// Device radii become logical ones, rounded up so nothing translucent
	// is ever claimed opaque.
	got = OpaqueRects(FrameRect{W: 100, H: 100}, [4]float32{14, 14, 0, 0}, 1.75)
	if len(got) != 2 || got[0].Y != 8 || got[1].H != 8 {
		t.Fatalf("scaled radius: %+v", got)
	}
	// Only the top rounded: no bottom strip.
	got = OpaqueRects(win, [4]float32{8, 8, 0, 0}, 1)
	if len(got) != 2 || got[0].H != 392 {
		t.Fatalf("top-rounded window: %+v", got)
	}
}

// The shm conversion keeps the frame's alpha: opaque inside the window,
// see-through in the margin. Every other window forces the alpha byte
// opaque, so an opaque UI can never present as a transparent one.
func TestPresentCopyKeepsFrameAlpha(t *testing.T) {
	img := paintengine2d.NewImage(20, 20)
	ctx := paintengine2d.NewContext(img)
	ctx.Clear(paintengine2d.Transparent)
	ctx.DrawRect(paintengine2d.XYWH(4, 4, 12, 12), paintengine2d.Fill(paintengine2d.RGB(0.2, 0.4, 0.8)))
	dst := make([]byte, 20*20*4)
	copyImageRect(dst, 20*4, img, paintengine2d.XYWH(0, 0, 20, 20), true, false)
	at := func(x, y int) byte { return dst[(y*20+x)*4+3] }
	if a := at(10, 10); a != 255 {
		t.Fatalf("inside the window alpha=%d, want 255", a)
	}
	if a := at(1, 1); a != 0 {
		t.Fatalf("in the margin alpha=%d, want 0", a)
	}
	for i := range dst {
		dst[i] = 0
	}
	copyImageRect(dst, 20*4, img, paintengine2d.XYWH(0, 0, 20, 20), true, true)
	if a := at(1, 1); a != 255 {
		t.Fatalf("an opaque window's buffer must be opaque everywhere, alpha=%d", a)
	}
}

// Offscreen keeps a compositor's bargain: the window stays the size the
// app asked for and the surface grows by the frame's margin.
func TestOffscreenSurfaceGrowsByTheMargin(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 600, Height: 400})
	if w, h := o.Size(); w != 600 || h != 400 {
		t.Fatalf("no frame: %dx%d", w, h)
	}
	f := Frame{
		Margin: FrameInsets{Top: 10, Right: 16, Bottom: 24, Left: 16},
		Input:  FrameInsets{Top: 10, Right: 10, Bottom: 10, Left: 10},
		Radius: [4]float32{8, 8, 8, 8},
		Alpha:  true,
	}
	o.SetFrame(f)
	if w, h := o.Size(); w != 600+32 || h != 400+34 {
		t.Fatalf("with a margin: %dx%d", w, h)
	}
	if !o.Frame().Same(f) {
		t.Fatalf("frame %+v", o.Frame())
	}
	if got := o.FrameCalls().Frames; len(got) != 1 || !got[0].Same(f) {
		t.Fatalf("frames %+v", got)
	}
	// A resize is of the window; the margin rides along.
	_ = o.Resize(500, 300)
	if w, h := o.Size(); w != 532 || h != 334 {
		t.Fatalf("after resize: %dx%d", w, h)
	}
	// And the margin goes when the frame does.
	o.SetFrame(Frame{})
	if w, h := o.Size(); w != 500 || h != 300 {
		t.Fatalf("frame dropped: %dx%d", w, h)
	}
}

func TestWindowStateSolidWithoutCompositing(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 100, Height: 80})
	if o.WindowState().Solid {
		t.Fatal("a composited desktop by default")
	}
	o.SimulateCompositing(false)
	if !o.WindowState().Solid {
		t.Fatal("without compositing the window is solid")
	}
	evs := o.Poll()
	if len(evs) != 1 || evs[0].Kind != EventWindowState || !evs[0].State.Solid {
		t.Fatalf("events %+v", evs)
	}
}
