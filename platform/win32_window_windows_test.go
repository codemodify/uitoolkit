//go:build windows

package platform

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// These make real windows, so they need an interactive desktop — which is
// the point. Every bug they cover compiled, vetted and passed the whole
// Linux suite, because a file behind //go:build windows is not even
// compiled there. See docs/windows.md.

var procGetWindowLongT = user32.NewProc("GetWindowLongPtrW")

func testSurfaceWith(t *testing.T, opts WindowOptions) *winSurface {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	s, err := newWinSurface(opts)
	if err != nil {
		t.Skipf("no window could be made (no interactive desktop?): %v", err)
	}
	ws := s.(*winSurface)
	t.Cleanup(func() { ws.Close() })
	return ws
}

func testSurface(t *testing.T, w, h int) *winSurface {
	t.Helper()
	// The same rule the backend is built on, which a test has to keep for
	// itself: Win32 delivers a window's messages to the thread that
	// created it, so that thread must stay put. platform's init locks the
	// *main* goroutine, and `go test` runs every test on a goroutine of
	// its own — so without this the test goroutine wanders off the thread
	// that made the window and ShowWindow blocks for ever, which is
	// precisely the bug below. Observed: this test hung on its own
	// second run.
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
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

// Nothing but WM_ACTIVATE says whether the frame is painted active or
// backdrop. Left unhandled, Activated stayed false for the window's whole
// life, so the first EventWindowState anything pushed — SetKeepAbove's —
// told the application the window had just gone inactive, and pressing
// "keep above" made the title bar lose its colour.
func TestActivationIsReportedSoTheTitleBarKeepsItsColour(t *testing.T) {
	s := testSurface(t, 300, 200)
	last := func() (WindowState, bool) {
		var st WindowState
		var got bool
		for _, ev := range s.Poll() {
			if ev.Kind == EventWindowState {
				st, got = ev.State, true
			}
		}
		return st, got
	}
	s.Poll() // drain whatever opening the window produced

	const waActive, waInactive = 1, 0
	procSendMessage.Call(s.hwnd, wmActivate, waActive, 0)
	if st, ok := last(); !ok || !st.Activated {
		t.Fatalf("after WA_ACTIVE: state %+v, reported %v; want Activated", st, ok)
	}
	procSendMessage.Call(s.hwnd, wmActivate, waInactive, 0)
	if st, ok := last(); !ok || st.Activated {
		t.Fatalf("after WA_INACTIVE: state %+v, reported %v; want not Activated", st, ok)
	}

	// And the state a frame call reports carries it too, which is the path
	// that actually went wrong: SetKeepAbove pushes s.state.
	procSendMessage.Call(s.hwnd, wmActivate, waActive, 0)
	s.Poll()
	FrameOf(s).SetKeepAbove(true)
	if st, ok := last(); !ok || !st.Activated {
		t.Errorf("keep-above reported %+v: it must not say the window went inactive", st)
	}
	FrameOf(s).SetKeepAbove(false)
}

// The wheel messages were declared and never handled, so nothing scrolled
// anywhere on Windows — over a list, a text area or the title bar alike.
func TestTheWheelArrivesAsAScroll(t *testing.T) {
	s := testSurface(t, 300, 200)
	s.Poll()
	scroll := func(msg, delta uintptr) paintengine2d.Point {
		procSendMessage.Call(s.hwnd, msg, delta<<16, 0)
		for _, ev := range s.Poll() {
			if ev.Kind == EventScroll {
				return ev.Scroll
			}
		}
		t.Fatalf("message %#x with delta %d produced no EventScroll", msg, int16(delta))
		return paintengine2d.Point{}
	}
	const up, down = 120, 0x10000 - 120 // WHEEL_DELTA, and -WHEEL_DELTA as a word
	// Away from the user is up, and up is negative — the sign X11 gives
	// button 4, so a scroll means one thing above the boundary.
	if got := scroll(wmMouseWheel, up); got.Y != -1 || got.X != 0 {
		t.Errorf("a notch away from the user gave %v, want (0,-1)", got)
	}
	if got := scroll(wmMouseWheel, down); got.Y != 1 || got.X != 0 {
		t.Errorf("a notch towards the user gave %v, want (0,1)", got)
	}
	if got := scroll(wmMouseHWheel, up); got.X != 1 || got.Y != 0 {
		t.Errorf("a notch of the horizontal wheel gave %v, want (1,0)", got)
	}
}

// Rolling a window up to its title bar. The claim FrameShade makes here is
// that a programmatic resize is not clamped by the minimum tracking size a
// window states — that governs a resize the user drags — so the roll-up
// holds instead of springing back open.
func TestAWindowRollsUpToItsTitleBar(t *testing.T) {
	s := testSurface(t, 400, 300)
	f := FrameOf(s)
	if !f.FrameCaps().Has(FrameShade) {
		t.Fatal("FrameShade is not advertised, so Window.CanShade refuses and nothing can roll up")
	}
	if !f.SetShadedHeight(24) {
		t.Fatal("SetShadedHeight refused")
	}
	if err := s.Resize(400, 24); err != nil {
		t.Fatalf("Resize: %v", err)
	}
	s.Poll()
	c, _ := rects(t, s)
	// Resize speaks logical pixels and the client rectangle is device
	// pixels, which are not the same thing anywhere but a 96-DPI display.
	if h, want := int(c.Bottom-c.Top), DevicePixels(24, s.Scale()); h != want {
		t.Errorf("rolled up to %d device pixels at scale %v, asked for 24 logical (%d): the resize was clamped, so a shaded window springs open",
			h, s.Scale(), want)
	}
}

// askMinMax is what Windows would be told if a resize started now.
func askMinMax(s *winSurface) winMinMaxInfo {
	var mmi winMinMaxInfo
	procSendMessage.Call(s.hwnd, wmGetMinMax, 0, uintptr(unsafe.Pointer(&mmi)))
	runtime.KeepAlive(&mmi)
	return mmi
}

// The bug this is about: SizeLimits answered and GeometrySizeLimits was
// advertised, while nothing was ever said to Windows — WM_GETMINMAXINFO
// went unhandled — so a window could be dragged to any size whatever it
// stated. A capability that lies.
func TestSizeLimitsAreToldToWindows(t *testing.T) {
	s := testSurfaceWith(t, WindowOptions{
		Title: "uitoolkit limits", Width: 400, Height: 300,
		MinWidth: 220, MinHeight: 160, MaxWidth: 900, MaxHeight: 700,
	})
	if !GeometryOf(s).GeometryCaps().Has(GeometrySizeLimits) {
		t.Fatal("GeometrySizeLimits is not advertised")
	}
	sc := s.deviceScale()
	mmi := askMinMax(s)
	// The window is at least as big as its client minimum: the frame is
	// added on top, so greater is right and smaller is the bug.
	if got, want := mmi.MinTrackSize.X, int32(DevicePixels(220, sc)); got < want {
		t.Errorf("minimum width %d, want at least the %d the window states", got, want)
	}
	if got, want := mmi.MinTrackSize.Y, int32(DevicePixels(160, sc)); got < want {
		t.Errorf("minimum height %d, want at least %d", got, want)
	}
	if got, want := mmi.MaxTrackSize.X, int32(DevicePixels(900, sc)); got < want {
		t.Errorf("maximum width %d, want at least %d", got, want)
	}
	if mmi.MinTrackSize.X >= mmi.MaxTrackSize.X || mmi.MinTrackSize.Y >= mmi.MaxTrackSize.Y {
		t.Errorf("a resizable window given min %v and max %v cannot be resized at all",
			mmi.MinTrackSize, mmi.MaxTrackSize)
	}
}

// A fixed window is one Windows must not let the user resize: its
// minimum and its maximum are the same, which is also how a desktop
// knows to drop the resize edges.
func TestAFixedWindowCannotBeDragged(t *testing.T) {
	s := testSurface(t, 360, 240)
	if !GeometryOf(s).SetSizing(SizingFixed) {
		t.Fatal("SetSizing(SizingFixed) refused")
	}
	mmi := askMinMax(s)
	if mmi.MinTrackSize != mmi.MaxTrackSize {
		t.Errorf("fixed window may still be dragged: min %v, max %v", mmi.MinTrackSize, mmi.MaxTrackSize)
	}
}

// The other half of shading. Rolling up works without telling Windows
// anything, because SetWindowPos is not clamped by the minimum tracking
// size — but a drag is a resize the user makes and is, so without this a
// rolled-up window can simply be dragged open again.
//
// The window is given real limits first: with none, Windows is told
// nothing either way and the two tracking sizes are both zero, which
// would make "pinned" and "not pinned" indistinguishable and the test
// worthless. It said so for a while.
func TestARolledUpWindowStaysRolledUp(t *testing.T) {
	s := testSurfaceWith(t, WindowOptions{
		Title: "uitoolkit shade", Width: 400, Height: 300,
		MinWidth: 200, MinHeight: 150, MaxWidth: 900, MaxHeight: 700,
	})
	f := FrameOf(s)
	base := askMinMax(s)
	if base.MinTrackSize.Y >= base.MaxTrackSize.Y {
		t.Fatalf("unshaded, the window should have room to resize: %+v", base)
	}

	if !f.SetShadedHeight(26) {
		t.Fatal("SetShadedHeight refused")
	}
	sh := askMinMax(s)
	if sh.MinTrackSize.Y != sh.MaxTrackSize.Y {
		t.Errorf("rolled up, the window can still be dragged to another height: min %d, max %d",
			sh.MinTrackSize.Y, sh.MaxTrackSize.Y)
	}
	if sh.MaxTrackSize.Y >= base.MaxTrackSize.Y {
		t.Errorf("rolled up, the height was pinned at %d, no shorter than the %d it could already be",
			sh.MaxTrackSize.Y, base.MaxTrackSize.Y)
	}
	// Only the height: a rolled-up window is still as wide as it was.
	if sh.MinTrackSize.X != base.MinTrackSize.X || sh.MaxTrackSize.X != base.MaxTrackSize.X {
		t.Errorf("shading changed the width limits: %v then %v", base.MinTrackSize, sh.MinTrackSize)
	}

	f.SetShadedHeight(0)
	if back := askMinMax(s); back != base {
		t.Errorf("unshaded, the limits did not come back: %+v, want %+v", back, base)
	}
}

var procGetDpiForWindowT = user32.NewProc("GetDpiForWindow")

// The window's scale, and the size that follows from it. This passes
// trivially on a 96-DPI display and is the only thing that catches a
// backend that never declared itself DPI-aware: Windows then answers 96
// however the display is set, bitmap-scales the window, and every
// assertion here still holds at the wrong size — so run it on a guest
// set to something other than 100% (docs/windows.md).
func TestTheWindowIsSizedForItsDisplaysDPI(t *testing.T) {
	s := testSurface(t, 400, 300)
	dpi, _, _ := procGetDpiForWindowT.Call(s.hwnd)
	if dpi < 48 {
		t.Skipf("GetDpiForWindow answered %d: too old a Windows to ask", dpi)
	}
	want := float32(dpi) / 96
	if got := s.Scale(); got != want {
		t.Errorf("Scale() = %v, want %v for %d DPI", got, want, dpi)
	}
	// The client area is the logical size times the scale, in device
	// pixels — not the logical size in device pixels, which is what a
	// window that is not DPI-aware gets and then has stretched for it.
	c, _ := rects(t, s)
	if got, want := int(c.Right-c.Left), DevicePixels(400, s.Scale()); got != want {
		t.Errorf("client is %d device pixels wide at scale %v, want %d", got, s.Scale(), want)
	}
	if got, want := int(c.Bottom-c.Top), DevicePixels(300, s.Scale()); got != want {
		t.Errorf("client is %d device pixels tall at scale %v, want %d", got, s.Scale(), want)
	}
	// And the buffer is that size, or the toolkit paints at the wrong one.
	if bw, bh := s.Size(); bw != DevicePixels(400, s.Scale()) || bh != DevicePixels(300, s.Scale()) {
		t.Errorf("buffer is %dx%d, want %dx%d", bw, bh,
			DevicePixels(400, s.Scale()), DevicePixels(300, s.Scale()))
	}
	t.Logf("display is %d DPI, scale %v, client %dx%d", dpi, s.Scale(), c.Right-c.Left, c.Bottom-c.Top)
}

// Fullscreen is three steps an application does for itself on Windows —
// remember the placement, take the frame off the style, cover the
// monitor — because Windows has no fullscreen state to ask for. It used
// to record the state and do none of them.
func TestFullscreenCoversTheMonitorAndComesBack(t *testing.T) {
	s := testSurface(t, 500, 380)
	f := FrameOf(s)
	before, _ := rects(t, s)
	style := winStyleOf(s)

	if !f.SetFullscreen(true) {
		t.Fatal("SetFullscreen(true) refused")
	}
	const wsCaption, wsThickFrame = 0x00C00000, 0x00040000
	if winStyleOf(s)&(wsCaption|wsThickFrame) != 0 {
		t.Errorf("fullscreen style %#x still has a caption or a sizing border", winStyleOf(s))
	}
	full, _ := rects(t, s)
	if full.Right-full.Left <= before.Right-before.Left ||
		full.Bottom-full.Top <= before.Bottom-before.Top {
		t.Errorf("fullscreen client %dx%d is no bigger than the %dx%d it was",
			full.Right-full.Left, full.Bottom-full.Top,
			before.Right-before.Left, before.Bottom-before.Top)
	}
	if !f.WindowState().Fullscreen {
		t.Error("the window does not say it is fullscreen")
	}

	if !f.SetFullscreen(false) {
		t.Fatal("SetFullscreen(false) refused")
	}
	if got := winStyleOf(s); got != style {
		t.Errorf("style came back as %#x, was %#x", got, style)
	}
	if back, _ := rects(t, s); back != before {
		t.Errorf("client came back %v, was %v", back, before)
	}
	if f.WindowState().Fullscreen {
		t.Error("the window still says it is fullscreen")
	}
}

// SetIcon answered false and did nothing. A window's icon is what the
// caption, the task bar and Alt+Tab show.
func TestSetIconGivesTheWindowAnIcon(t *testing.T) {
	s := testSurface(t, 300, 220)
	f := FrameOf(s)
	if !f.FrameCaps().Has(FrameIcon) {
		t.Skip("this window says it cannot carry an icon")
	}
	paint := func(side int, c paintengine2d.Color) *paintengine2d.Image {
		im := paintengine2d.NewImage(side, side)
		ctx := paintengine2d.NewContext(im)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, float32(side), float32(side)), paintengine2d.Fill(c))
		return im
	}
	icons := []*paintengine2d.Image{
		paint(16, paintengine2d.RGB(0.9, 0.2, 0.2)),
		paint(32, paintengine2d.RGB(0.2, 0.7, 0.3)),
		paint(48, paintengine2d.RGB(0.2, 0.3, 0.9)),
	}
	if !f.SetIcon(icons) {
		t.Fatal("SetIcon refused")
	}
	// WM_GETICON (0x007F): ask the window back for what it now has.
	for _, c := range []struct {
		which uintptr
		name  string
	}{{iconSmall, "small"}, {iconBig, "big"}} {
		got, _, _ := procSendMessage.Call(s.hwnd, 0x007F, c.which, 0)
		if got == 0 {
			t.Errorf("the window has no %s icon after SetIcon", c.name)
		}
	}
	// And an empty list takes them away again.
	if !f.SetIcon(nil) {
		t.Fatal("SetIcon(nil) refused")
	}
	if got, _, _ := procSendMessage.Call(s.hwnd, 0x007F, iconSmall, 0); got != 0 {
		t.Errorf("the small icon survived SetIcon(nil): %#x", got)
	}
}

// pickIcon chooses per size rather than stretching one for both.
func TestPickIconTakesTheNearestAtOrAbove(t *testing.T) {
	im := func(side int) *paintengine2d.Image { return paintengine2d.NewImage(side, side) }
	set := []*paintengine2d.Image{im(16), im(32), im(64)}
	for _, c := range []struct{ want, side int }{
		{16, 16}, {32, 17}, {32, 32}, {64, 33}, {64, 64}, {64, 256},
	} {
		if got := pickIcon(set, c.side); got == nil || got.Width != c.want {
			t.Errorf("for %d pixels picked %v, want the %d", c.side, got, c.want)
		}
	}
	if pickIcon(nil, 16) != nil {
		t.Error("an empty set should pick nothing")
	}
}
