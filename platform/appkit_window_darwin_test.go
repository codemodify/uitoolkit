//go:build darwin && cgo

package platform

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/codemodify/paintengine2d"
)

// These tests open real windows on the machine that runs them. That is
// allowed here in a way it is not on Linux — tools/testenv.sh exists to
// keep the suite off a desktop somebody is using, and there is no
// macOS equivalent because there is nothing to wrap: no Wayland, no
// X11, no session bus. The Mac these run on is a build machine.
//
// What makes them possible at all is TestMain. AppKit requires the
// process's *first* thread, `go test` runs every test on a goroutine of
// its own, and a window made anywhere else does not draw. So the first
// thread is kept here, the tests run beside it, and a test that needs
// AppKit asks for the thread back with onMain.

var mainQueue = make(chan func())

func TestMain(m *testing.M) {
	done := make(chan int, 1)
	go func() { done <- m.Run() }()
	for {
		select {
		case f := <-mainQueue:
			f()
		case code := <-done:
			os.Exit(code)
		}
	}
}

// onMain runs f on the process's first thread and waits for it.
func onMain(f func()) {
	ret := make(chan struct{})
	mainQueue <- func() {
		defer close(ret)
		f()
	}
	<-ret
}

// akTestWindow is a shown window, closed when the test ends. It must be
// called from inside onMain.
func akTestWindow(t *testing.T, w, h int) *akSurface {
	t.Helper()
	s, err := newAkSurface(WindowOptions{Title: "uitoolkit test", Width: w, Height: h})
	if err != nil {
		t.Fatalf("the window could not be made: %v", err)
	}
	t.Cleanup(func() { onMain(func() { s.Close() }) })
	return s.(*akSurface)
}

// pump turns the run loop for a while, so AppKit gets to order the
// window in, composite it and deliver what was posted.
func pump(s *akSurface, d time.Duration) []Event {
	var out []Event
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		out = append(out, s.Poll()...)
		time.Sleep(5 * time.Millisecond)
	}
	return out
}

// at is the colour of one pixel, as 0-255 RGBA.
func at(img *paintengine2d.Image, x, y int) [4]uint8 {
	i := y*img.Stride + x*4
	return [4]uint8{img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]}
}

// near reports whether two colours are the same to within tol per
// channel: the layer goes through a colour space on its way back, so an
// exact match is too much to ask of a value that was 217.
func near(got, want [4]uint8, tol int) bool {
	for i := range 4 {
		d := int(got[i]) - int(want[i])
		if d < -tol || d > tol {
			return false
		}
	}
	return true
}

// A window opens, and what was painted into the buffer is what the
// layer ends up showing — in the right corners, in the right channel
// order, the right way up.
//
// This is the test the Windows backend did not have and needed: a
// present that silently puts nothing on the screen looks exactly like a
// present that works, right up until somebody looks at the window. Here
// the window does not even have to be looked at.
func TestAppKitPresentsWhatWasPainted(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		w, h := s.Size()
		if w <= 0 || h <= 0 {
			t.Fatalf("the buffer is %dx%d", w, h)
		}
		img := s.Buffer()
		if img == nil {
			t.Fatal("no buffer")
		}
		red := paintengine2d.RGB(0.85, 0.15, 0.15)
		green := paintengine2d.RGB(0.15, 0.7, 0.2)
		blue := paintengine2d.RGB(0.15, 0.3, 0.9)
		white := paintengine2d.RGB(0.95, 0.95, 0.95)
		ctx := paintengine2d.NewContext(img)
		fw, fh := float32(img.Width), float32(img.Height)
		ctx.DrawRect(paintengine2d.XYWH(0, 0, fw/2, fh/2), paintengine2d.Fill(red))
		ctx.DrawRect(paintengine2d.XYWH(fw/2, 0, fw/2, fh/2), paintengine2d.Fill(green))
		ctx.DrawRect(paintengine2d.XYWH(0, fh/2, fw/2, fh/2), paintengine2d.Fill(blue))
		ctx.DrawRect(paintengine2d.XYWH(fw/2, fh/2, fw/2, fh/2), paintengine2d.Fill(white))
		if err := s.Present(nil); err != nil {
			t.Fatalf("present: %v", err)
		}
		pump(s, 300*time.Millisecond)

		shot := akReadback(s, w, h)
		if shot == nil {
			t.Fatal("the layer could not be read back")
		}
		for _, c := range []struct {
			name string
			x, y int
			want [4]uint8
		}{
			{"top-left red", w / 4, h / 4, [4]uint8{217, 38, 38, 255}},
			{"top-right green", 3 * w / 4, h / 4, [4]uint8{38, 179, 51, 255}},
			{"bottom-left blue", w / 4, 3 * h / 4, [4]uint8{38, 77, 230, 255}},
			{"bottom-right white", 3 * w / 4, 3 * h / 4, [4]uint8{242, 242, 242, 255}},
		} {
			if got := at(shot, c.x, c.y); !near(got, c.want, 6) {
				t.Errorf("%s at %d,%d is rgba %v, want %v", c.name, c.x, c.y, got, c.want)
			}
		}
		// The two ways a present goes wrong while still putting four
		// colours in four corners, told apart — because they need
		// different fixes and the corner check alone names neither.
		tl, tr := at(shot, w/4, h/4), at(shot, 3*w/4, h/4)
		bl := at(shot, w/4, 3*h/4)
		if near(tl, [4]uint8{38, 77, 230, 255}, 6) && near(bl, [4]uint8{217, 38, 38, 255}, 6) {
			t.Error("the window is upside down: the buffer's first row is at the bottom. " +
				"A layer-backed NSView that answers YES to isFlipped does this, " +
				"because geometryFlipped flips the contents image too.")
		}
		if near(tl, [4]uint8{38, 38, 217, 255}, 6) && near(tr, [4]uint8{51, 179, 38, 255}, 6) {
			t.Error("red and blue have traded places: the present's channel order is wrong. " +
				"paintengine2d keeps RGBA and CGBitmapContext must be told " +
				"kCGImageAlphaPremultipliedLast to match it.")
		}
	})
}

// A window at a Retina scale gets a buffer in device pixels, and the
// layer shows it at the size it really is rather than a quarter of it.
func TestAppKitBufferIsDevicePixels(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		w, h := s.Size()
		scale := s.Scale()
		if scale <= 0 {
			t.Fatalf("scale %g", scale)
		}
		if want := DevicePixels(320, scale); w != want {
			t.Errorf("the buffer is %d wide at scale %g, want %d", w, scale, want)
		}
		if want := DevicePixels(200, scale); h != want {
			t.Errorf("the buffer is %d tall at scale %g, want %d", h, scale, want)
		}
		cw, ch := akContentSize(s)
		if cw != w || ch != h {
			t.Errorf("AppKit says the content is %dx%d device pixels, the buffer is %dx%d", cw, ch, w, h)
		}
	})
}

// A press, a move and a release arrive as the events the toolkit
// expects, at the pixel they were aimed at.
func TestAppKitDeliversMouseInput(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		pump(s, 300*time.Millisecond)
		// AppKit's coordinates: points, y up from the bottom. A press
		// 30 points below the top of a 200-point window is at y 170
		// there, and the toolkit should be told y 30 — in device
		// pixels, so 60 at a scale of 2. Posting in AppKit's terms and
		// asserting in the toolkit's is what makes this a test of the
		// conversion rather than of its own inverse.
		akPostMouse(s, EventMouseDown, 50, 170, 1, 0)
		akPostMouse(s, EventMouseUp, 50, 170, 1, 0)
		evs := pump(s, 500*time.Millisecond)

		var down, up *Event
		for i := range evs {
			switch evs[i].Kind {
			case EventMouseDown:
				down = &evs[i]
			case EventMouseUp:
				up = &evs[i]
			}
		}
		if down == nil {
			t.Fatalf("no press arrived; saw %s", kindsOf(evs))
		}
		if up == nil {
			t.Fatalf("no release arrived; saw %s", kindsOf(evs))
		}
		if down.Button != 1 {
			t.Errorf("the press says button %v, want 1", down.Button)
		}
		sc := float64(s.Scale())
		wantX, wantY := 50*sc, 30*sc
		if dx, dy := float64(down.Pos.X)-wantX, float64(down.Pos.Y)-wantY; dx < -1.5 || dx > 1.5 || dy < -1.5 || dy > 1.5 {
			t.Errorf("a press 30 points below the top landed at %v, want %g,%g device pixels; "+
				"a y that is the height less this one means the flip is the wrong way round",
				down.Pos, wantX, wantY)
		}
	})
}

// A key arrives as a key, and the character it types arrives separately
// as text — the split the whole keyboard path is built on.
func TestAppKitDeliversKeyboardInput(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		pump(s, 300*time.Millisecond)
		akPostKey(s, "s", "s", 0, true)
		evs := pump(s, 500*time.Millisecond)

		var key, text *Event
		for i := range evs {
			switch evs[i].Kind {
			case EventKeyDown:
				key = &evs[i]
			case EventText:
				text = &evs[i]
			}
		}
		if key == nil {
			t.Fatalf("no key arrived; saw %s", kindsOf(evs))
		}
		if key.Key != KeyS {
			t.Errorf("the key is %v, want KeyS", key.Key)
		}
		if text == nil {
			t.Fatalf("no text arrived; saw %s", kindsOf(evs))
		}
		if text.Text != "s" {
			t.Errorf("the text is %q, want %q", text.Text, "s")
		}
	})
}

// Command+S is a shortcut: the key arrives, the text does not. A text
// field that inserted an "s" here would be inserting the shortcut.
func TestAppKitCommandIsAShortcutNotText(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		pump(s, 300*time.Millisecond)
		akPostKey(s, "s", "s", ModSuper, true)
		evs := pump(s, 500*time.Millisecond)

		for _, e := range evs {
			if e.Kind == EventText {
				t.Errorf("Command+S inserted %q", e.Text)
			}
			if e.Kind == EventKeyDown && e.Key == KeyS && e.Mods&ModSuper == 0 {
				t.Errorf("Command+S arrived without ModSuper: %v", e.Mods)
			}
		}
	})
}

// Move and Position are the same number: what the toolkit asks for is
// where the window goes, and what it reads back is where it is.
func TestAppKitGeometryRoundTrips(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		pump(s, 300*time.Millisecond)
		if !s.GeometryCaps().Has(GeometryMove | GeometryPosition) {
			t.Fatalf("caps %v", s.GeometryCaps())
		}
		x, y, ok := s.Position()
		if !ok {
			t.Fatal("macOS always knows where a window is, and this one did not")
		}
		if !s.Move(x+40, y+30) {
			t.Fatal("Move refused")
		}
		pump(s, 300*time.Millisecond)
		nx, ny, _ := s.Position()
		if nx != x+40 || ny != y+30 {
			t.Errorf("moved to %d,%d, asked for %d,%d", nx, ny, x+40, y+30)
		}
	})
}

// A fixed window drops the capabilities that are a resize in disguise,
// so a caption bar does not draw a maximize button that cannot work.
func TestAppKitFixedWindowDropsResizeCaps(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		if !s.FrameCaps().Has(FrameMaximize) {
			t.Fatal("a resizable window should offer maximize")
		}
		s.SetSizing(SizingFixed)
		c := s.FrameCaps()
		for _, drop := range []struct {
			cap  FrameCaps
			name string
		}{
			{FrameMaximize, "maximize"},
			{FrameMaximizeAxis, "maximize-axis"},
			{FrameResize, "resize"},
			{FrameShade, "shade"},
		} {
			if c.Has(drop.cap) {
				t.Errorf("a fixed window still offers %s", drop.name)
			}
		}
		if l := s.SizeLimits(); !l.Fixed() {
			t.Errorf("the limits of a fixed window are %+v", l)
		}
	})
}

// What macOS cannot do, it says it cannot do. A capability claimed and
// then not delivered is a control that does nothing.
func TestAppKitRefusesWhatItCannotDo(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		if s.StartResize(EdgeBottom | EdgeRight) {
			t.Error("StartResize claimed to start one: macOS has no API for it")
		}
		if s.ShowMenu(paintengine2d.Pt(1, 1)) {
			t.Error("ShowMenu claimed a window menu: macOS has none")
		}
		if s.SetShadedHeight(30) {
			t.Error("SetShadedHeight claimed a shade: macOS has none")
		}
		if s.SetPalette("/tmp/x.colors") {
			t.Error("SetPalette took a KDE colour scheme")
		}
		if s.MaximizeAxis(true) {
			t.Error("MaximizeAxis claimed one axis: zoom is both or neither")
		}
		c := s.FrameCaps()
		for _, no := range []struct {
			cap  FrameCaps
			name string
		}{
			{FrameResize, "resize"},
			{FrameMenu, "menu"},
			{FrameMaximizeAxis, "maximize-axis"},
			{FrameShade, "shade"},
			{FramePalette, "palette"},
		} {
			if c.Has(no.cap) {
				t.Errorf("the caps claim %s", no.name)
			}
		}
		// And the two it must claim, because the toolkit draws no
		// shadow and reserves no resize band when they are set.
		for _, yes := range []struct {
			cap  FrameCaps
			name string
		}{
			{FrameSystemShadow, "system-shadow"},
			{FrameSystemResizeBand, "system-resize-band"},
		} {
			if !c.Has(yes.cap) {
				t.Errorf("the caps do not claim %s, so the toolkit would draw its own", yes.name)
			}
		}
	})
}

// Maximizing really maximizes, and the state says so afterwards.
func TestAppKitMaximizeChangesTheState(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		pump(s, 300*time.Millisecond)
		if s.WindowState().Maximized {
			t.Fatal("a new window is already maximized")
		}
		if !s.SetMaximized(true) {
			t.Fatal("SetMaximized refused")
		}
		pump(s, 400*time.Millisecond)
		if !s.WindowState().Maximized {
			t.Error("SetMaximized(true) answered yes and the window is not maximized")
		}
		s.SetMaximized(false)
		pump(s, 400*time.Millisecond)
		if s.WindowState().Maximized {
			t.Error("SetMaximized(false) answered yes and the window is still maximized")
		}
	})
}

// Asking for a client frame gets one, and says so.
func TestAppKitDecorationsCanBeTurnedOff(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		pump(s, 200*time.Millisecond)
		if got := s.Decorations(); got != DecorationsServer {
			t.Fatalf("a new window has %v decorations, want server", got)
		}
		s.RequestDecorations(DecorationsClient)
		evs := pump(s, 300*time.Millisecond)
		if got := s.Decorations(); got != DecorationsClient {
			t.Errorf("after asking for a client frame: %v", got)
		}
		var said bool
		for _, e := range evs {
			if e.Kind == EventDecorations && e.Decor == DecorationsClient {
				said = true
			}
		}
		if !said {
			t.Errorf("no EventDecorations; saw %s", kindsOf(evs))
		}
		// And the window still takes keys: changing the style mask
		// drops the first responder, and a window that quietly stopped
		// hearing the keyboard would be a nasty one to find.
		s.RequestDecorations(DecorationsServer)
		pump(s, 200*time.Millisecond)
		akPostKey(s, "s", "s", 0, true)
		for _, e := range pump(s, 500*time.Millisecond) {
			if e.Kind == EventKeyDown {
				return
			}
		}
		t.Error("no key arrived after the decorations changed back")
	})
}

// kindsOf names the events a pump collected, for a failure message
// that says what did arrive rather than only what did not.
func kindsOf(evs []Event) string {
	if len(evs) == 0 {
		return "nothing"
	}
	out := ""
	for i, e := range evs {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprint(e.Kind)
	}
	return out
}

// The pasteboard round-trips, and a read finds what another process
// would have put there.
//
// This one writes to the *real* pasteboard of the machine it runs on,
// which is a side effect a test should own up to: the Mac these run on
// is a build machine, and what was on its clipboard is gone. There is
// no per-process pasteboard to use instead — NSPasteboard general is
// the system's one — and a test against a named pasteboard of our own
// would not be testing the code that ships.
func TestAppKitClipboardRoundTrips(t *testing.T) {
	const want = "uitoolkit pasteboard ✓ 日本語"
	ClipboardSet(want)
	got, ok := clipboardNativeGet()
	if !ok {
		t.Fatal("the pasteboard holds no string after one was set")
	}
	if got != want {
		t.Errorf("the pasteboard holds %q, want %q", got, want)
	}
	if got := ClipboardGet(); got != want {
		t.Errorf("ClipboardGet is %q, want %q", got, want)
	}
	// No PRIMARY here, so it answers from the same place rather than
	// from a second selection that does not exist.
	if got := ClipboardPrimaryGet(); got != want {
		t.Errorf("ClipboardPrimaryGet is %q, want %q", got, want)
	}
}

// Setting the cursor is remembered and answered, and does not need the
// pointer to be over the window for the shape to be recorded.
func TestAppKitCursorIsRemembered(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		if got := s.Cursor(); got != CursorDefault {
			t.Errorf("a new window starts with cursor %v, want the default", got)
		}
		s.SetCursor(CursorText)
		if got := s.Cursor(); got != CursorText {
			t.Errorf("after asking for a text cursor: %v", got)
		}
		s.SetCursor(CursorGrabbing)
		if got := s.Cursor(); got != CursorGrabbing {
			t.Errorf("after asking for a grabbing cursor: %v", got)
		}
	})
}

// Every shape the toolkit names maps to an NSCursor, and the ones that
// should differ do. A mapping that quietly sent everything to the
// arrow would pass a "does SetCursor work" test and be useless.
func TestAppKitCursorShapesAreDistinct(t *testing.T) {
	distinct := map[int][]Cursor{}
	for _, c := range []Cursor{
		CursorDefault, CursorColResize, CursorRowResize, CursorText,
		CursorGrab, CursorGrabbing, CursorDragCopy, CursorDragLink, CursorNoDrop,
	} {
		k := darwinCursorKind(c)
		distinct[k] = append(distinct[k], c)
	}
	if len(distinct) != 9 {
		t.Errorf("nine shapes that should differ map to %d NSCursors: %v", len(distinct), distinct)
	}
	if darwinCursorKind(CursorResizeE) != darwinCursorKind(CursorColResize) {
		t.Error("an east resize and a column resize are the same left-right pointer")
	}
	if darwinCursorKind(CursorResizeN) != darwinCursorKind(CursorRowResize) {
		t.Error("a north resize and a row resize are the same up-down pointer")
	}
}

// Keep-above is reported back in the window's state, not only applied.
//
// This is the test that was missing. SetKeepAbove worked — the window
// really did float — but WindowState never carried KeepAbove, and the
// toolkit's toggle is SetKeepAbove(!WindowState().KeepAbove). With the
// state stuck at false every press meant "on": the caption's pin never
// lit up, and a window once pinned could not be released. A test that
// only asserted SetKeepAbove returns true passed throughout.
func TestAppKitKeepAboveIsReportedInTheState(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		pump(s, 200*time.Millisecond)
		if s.WindowState().KeepAbove {
			t.Fatal("a new window is already kept above")
		}
		if !s.SetKeepAbove(true) {
			t.Fatal("SetKeepAbove refused")
		}
		pump(s, 200*time.Millisecond)
		if !s.WindowState().KeepAbove {
			t.Error("after SetKeepAbove(true) the state still says no: " +
				"the caption cannot draw the pin, and the toggle can never turn it off")
		}
		if !s.SetKeepAbove(false) {
			t.Fatal("SetKeepAbove(false) refused")
		}
		pump(s, 200*time.Millisecond)
		if s.WindowState().KeepAbove {
			t.Error("after SetKeepAbove(false) the window is still kept above")
		}
	})
}

// And the change is announced, so a caption drawn from the state
// repaints without anything having to poll it.
func TestAppKitKeepAboveAnnouncesTheChange(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		pump(s, 200*time.Millisecond)
		s.SetKeepAbove(true)
		var said bool
		for _, e := range pump(s, 300*time.Millisecond) {
			if e.Kind == EventWindowState && e.State.KeepAbove {
				said = true
			}
		}
		if !said {
			t.Error("no EventWindowState carrying KeepAbove after it was turned on")
		}
	})
}

// The toggle a caption button actually performs, end to end: press,
// press again, and the window is back where it started.
func TestAppKitKeepAboveTogglesBothWays(t *testing.T) {
	onMain(func() {
		s := akTestWindow(t, 320, 200)
		pump(s, 200*time.Millisecond)
		toggle := func() { s.SetKeepAbove(!s.WindowState().KeepAbove) }
		toggle()
		pump(s, 200*time.Millisecond)
		if !s.WindowState().KeepAbove {
			t.Fatal("the first press did not pin the window")
		}
		toggle()
		pump(s, 200*time.Millisecond)
		if s.WindowState().KeepAbove {
			t.Fatal("the second press did not release it — this is the bug a user hits")
		}
	})
}
