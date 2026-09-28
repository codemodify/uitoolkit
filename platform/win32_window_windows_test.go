//go:build windows

package platform

import (
	"testing"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// These make real windows, so they need an interactive desktop — which is
// the point. Every bug they cover compiled, vetted and passed the whole
// Linux suite, because a file behind //go:build windows is not even
// compiled there. See docs/windows.md.

var procGetWindowLongT = user32.NewProc("GetWindowLongPtrW")

func testSurface(t *testing.T, w, h int) *winSurface {
	t.Helper()
	s, err := newWinSurface(WindowOptions{Title: "uitoolkit test", Width: w, Height: h})
	if err != nil {
		t.Skipf("no window could be made (no interactive desktop?): %v", err)
	}
	ws := s.(*winSurface)
	t.Cleanup(func() { ws.Close() })
	return ws
}

func rects(t *testing.T, s *winSurface) (client, window winRect) {
	t.Helper()
	procGetClientRect.Call(s.hwnd, uintptr(unsafe.Pointer(&client)))
	procGetWindowRect.Call(s.hwnd, uintptr(unsafe.Pointer(&window)))
	return
}

func winStyleOf(s *winSurface) uintptr {
	r, _, _ := procGetWindowLongT.Call(s.hwnd, gwlStyle)
	return r
}

// The bug this is about: RequestDecorations recorded the mode and did
// nothing to the window, so the toolkit drew its own title bar while the
// desktop's stayed above it. Two title bars, and an "OS borders" switch
// that did nothing to the borders.
func TestClientDecorationsGiveTheWholeWindowToTheClient(t *testing.T) {
	s := testSurface(t, 400, 300)
	f := FrameOf(s)

	f.RequestDecorations(DecorationsServer)
	c, w := rects(t, s)
	if c.Bottom-c.Top >= w.Bottom-w.Top {
		t.Fatalf("with the desktop's frame the client (%d tall) should be shorter than the window (%d)",
			c.Bottom-c.Top, w.Bottom-w.Top)
	}

	f.RequestDecorations(DecorationsClient)
	c, w = rects(t, s)
	if c.Right-c.Left != w.Right-w.Left || c.Bottom-c.Top != w.Bottom-w.Top {
		t.Errorf("with the toolkit's frame the client is %dx%d and the window %dx%d: "+
			"WM_NCCALCSIZE is not giving the whole window to the client, so the desktop's "+
			"caption is still drawn above ours",
			c.Right-c.Left, c.Bottom-c.Top, w.Right-w.Left, w.Bottom-w.Top)
	}
	// And the style must keep WS_CAPTION even so: DWM hangs the drop
	// shadow, the snap animation and Windows 11's rounded corners on it.
	// Dropping it is the obvious way to lose the title bar and it costs
	// all three.
	const wsCaption = 0x00C00000
	if winStyleOf(s)&wsCaption != wsCaption {
		t.Errorf("style %#x has lost WS_CAPTION: the window keeps it and answers "+
			"WM_NCCALCSIZE instead, or it loses DWM's shadow", winStyleOf(s))
	}
}

// The client area must not change size when the frame does — the drawable
// area used to jump by the height of a caption.
func TestDecorationChangeKeepsTheClientSize(t *testing.T) {
	s := testSurface(t, 480, 360)
	f := FrameOf(s)
	f.RequestDecorations(DecorationsClient)
	before, _ := rects(t, s)
	f.RequestDecorations(DecorationsServer)
	f.RequestDecorations(DecorationsClient)
	after, _ := rects(t, s)
	if before != after {
		t.Errorf("client area was %v and is %v after a round trip through the desktop's frame", before, after)
	}
}

// The bug this is about: SetDIBitsToDevice answered "0 scan lines set" on
// every frame with GetLastError reporting success, the result was thrown
// away, and the window came up white and said nothing.
func TestPresentReportsWhetherThePixelsWent(t *testing.T) {
	s := testSurface(t, 320, 240)
	img := s.Buffer()
	if img == nil {
		t.Fatal("a surface with no buffer")
	}
	ctx := paintengine2d.NewContext(img)
	ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(img.Width), float32(img.Height)),
		paintengine2d.Fill(paintengine2d.RGB(0.85, 0.15, 0.15)))
	if err := s.Present(nil); err != nil {
		t.Fatalf("Present: %v", err)
	}
	if s.dib.mdc == 0 || s.bgra == nil {
		t.Fatal("the surface has no DIB section: Present cannot be handing GDI its own memory")
	}
	// Red, as a 32-bit BI_RGB DIB wants it. Red and blue trading places is
	// the failure this backend is most likely to have, and it is invisible
	// in a screenshot of a grey window.
	r, g, b := img.Pix[0], img.Pix[1], img.Pix[2]
	if s.bgra[0] != b || s.bgra[1] != g || s.bgra[2] != r || s.bgra[3] != 0xff {
		t.Errorf("swizzle put RGBA %d,%d,%d into the DIB as %d,%d,%d,%d; a BI_RGB DIB is BGRX",
			r, g, b, s.bgra[0], s.bgra[1], s.bgra[2], s.bgra[3])
	}
}

// The bug this is about: without the main OS thread pinned, Go moved the
// goroutine between CreateWindowExW and ShowWindow, ShowWindow became a
// cross-thread SendMessage to a thread nobody was pumping, and it blocked
// for ever. Four runs in five. A hang fails this by timing the test out
// rather than by an assertion, which is the only way a deadlock can fail.
func TestWindowsOpenRepeatedlyWithoutWedging(t *testing.T) {
	for i := 0; i < 5; i++ {
		s := testSurface(t, 200, 150)
		s.Poll()
		if _, _, ok := GeometryOf(s).Position(); !ok {
			t.Fatalf("window %d has no position, which on Windows is a bug", i)
		}
		s.Close()
	}
}
