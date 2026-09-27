package platform

import (
	"testing"

	"github.com/codemodify/paintengine2d"
)

func TestParseDecorations(t *testing.T) {
	for in, want := range map[string]Decorations{
		"server": DecorationsServer, "System": DecorationsServer, "ssd": DecorationsServer,
		"client": DecorationsClient, "toolkit": DecorationsClient, " CSD ": DecorationsClient,
		"none": DecorationsNone, "frameless": DecorationsNone, "auto": DecorationsAuto,
	} {
		if got, ok := ParseDecorations(in); !ok || got != want {
			t.Errorf("ParseDecorations(%q) = %v,%v want %v", in, got, ok, want)
		}
	}
	if _, ok := ParseDecorations("sideways"); ok {
		t.Error("an unknown mode parsed")
	}
	if _, ok := ParseDecorations(""); ok {
		t.Error("empty is no mode")
	}
	t.Setenv(EnvDecorations, "client")
	if d, ok := DecorationsFromEnv(); !ok || d != DecorationsClient {
		t.Errorf("env client: %v %v", d, ok)
	}
	t.Setenv(EnvDecorations, "auto")
	if _, ok := DecorationsFromEnv(); ok {
		t.Error("auto is no override")
	}
	for _, d := range []Decorations{DecorationsAuto, DecorationsServer, DecorationsClient, DecorationsNone} {
		if back, _ := ParseDecorations(d.String()); back != d {
			t.Errorf("%v does not round-trip", d)
		}
	}
}

func TestEffectiveDecorations(t *testing.T) {
	cases := []struct {
		want, answer, got Decorations
	}{
		{DecorationsServer, DecorationsServer, DecorationsServer},
		{DecorationsClient, DecorationsClient, DecorationsClient},
		// KWin's "no frame" answer to a client request (full screen).
		{DecorationsClient, DecorationsServer, DecorationsServer},
		{DecorationsNone, DecorationsClient, DecorationsNone},
		// GNOME: nobody to ask, the toolkit draws.
		{DecorationsServer, DecorationsAuto, DecorationsClient},
		{DecorationsNone, DecorationsAuto, DecorationsNone},
	}
	for _, c := range cases {
		if got := effectiveDecorations(c.want, c.answer); got != c.got {
			t.Errorf("want %v answer %v: %v, expected %v", c.want, c.answer, got, c.got)
		}
	}
	if requestedDecorations(DecorationsAuto) != DecorationsServer {
		t.Error("a backend handed Auto keeps the desktop's frame")
	}
}

func TestEdges(t *testing.T) {
	valid := []Edges{EdgeTop, EdgeBottom, EdgeLeft, EdgeRight,
		EdgeTop | EdgeLeft, EdgeTop | EdgeRight, EdgeBottom | EdgeLeft, EdgeBottom | EdgeRight}
	for _, e := range valid {
		if !e.Valid() {
			t.Errorf("%v should be a resize edge", e)
		}
		if e.ResizeCursor() == CursorDefault {
			t.Errorf("%v has no resize cursor", e)
		}
	}
	for _, e := range []Edges{0, EdgeTop | EdgeBottom, EdgeLeft | EdgeRight, EdgeTop | EdgeBottom | EdgeLeft, 16} {
		if e.Valid() {
			t.Errorf("%v is not a resize edge", e)
		}
	}
	if (EdgeTop|EdgeLeft).String() != "top-left" || Edges(0).String() != "none" {
		t.Errorf("names %q %q", (EdgeTop | EdgeLeft).String(), Edges(0).String())
	}
	if (EdgeBottom|EdgeRight).ResizeCursor() != CursorResizeSE || EdgeTop.ResizeCursor() != CursorResizeN {
		t.Error("cursor per edge")
	}
}

// xdg_toplevel.resize_edge: top 1, bottom 2, left 4, top_left 5,
// bottom_left 6, right 8, top_right 9, bottom_right 10.
func TestXdgResizeEdge(t *testing.T) {
	want := map[Edges]uint32{
		EdgeTop: 1, EdgeBottom: 2, EdgeLeft: 4, EdgeTop | EdgeLeft: 5,
		EdgeBottom | EdgeLeft: 6, EdgeRight: 8, EdgeTop | EdgeRight: 9, EdgeBottom | EdgeRight: 10,
		0: 0, EdgeTop | EdgeBottom: 0, EdgeLeft | EdgeRight: 0,
	}
	for e, v := range want {
		if got := xdgResizeEdge(e); got != v {
			t.Errorf("%v: %d want %d", e, got, v)
		}
	}
}

// _NET_WM_MOVERESIZE: SIZE_TOPLEFT 0 … SIZE_LEFT 7, clockwise.
func TestNetMoveResizeDirection(t *testing.T) {
	want := map[Edges]uint32{
		EdgeTop | EdgeLeft: 0, EdgeTop: 1, EdgeTop | EdgeRight: 2, EdgeRight: 3,
		EdgeBottom | EdgeRight: 4, EdgeBottom: 5, EdgeBottom | EdgeLeft: 6, EdgeLeft: 7,
	}
	for e, v := range want {
		if got, ok := netMoveResizeDirection(e); !ok || got != v {
			t.Errorf("%v: %d,%v want %d", e, got, ok, v)
		}
	}
	if _, ok := netMoveResizeDirection(EdgeTop | EdgeBottom); ok {
		t.Error("opposite edges have no direction")
	}
	if netMoveResizeMove != 8 || netMoveResizeCancel != 11 {
		t.Error("move / cancel")
	}
}

func TestXdgStateFromMask(t *testing.T) {
	bit := func(v ...uint) uint32 {
		var m uint32
		for _, x := range v {
			m |= 1 << x
		}
		return m
	}
	if st := xdgStateFromMask(0); st != (WindowState{}) {
		t.Errorf("empty %+v", st)
	}
	st := xdgStateFromMask(bit(1, 4))
	if !st.Maximized || !st.Activated || st.Fullscreen || st.Resizing || st.Tiled != 0 {
		t.Errorf("maximized+activated %+v", st)
	}
	// A KWin quick tile on the left: tiled left, top and bottom.
	st = xdgStateFromMask(bit(4, 5, 7, 8))
	if st.Tiled != EdgeLeft|EdgeTop|EdgeBottom || st.Maximized {
		t.Errorf("left tile %+v", st)
	}
	st = xdgStateFromMask(bit(2, 3, 9))
	if !st.Fullscreen || !st.Resizing || !st.Suspended || st.Activated {
		t.Errorf("fullscreen resizing suspended %+v", st)
	}
	st = xdgStateFromMask(bit(6, 10, 11, 12, 13))
	if st.Tiled != EdgeRight || st.Constrained != EdgeLeft|EdgeRight|EdgeTop|EdgeBottom {
		t.Errorf("constrained %+v", st)
	}
	// Unknown future states are ignored.
	if st := xdgStateFromMask(bit(20)); st != (WindowState{}) {
		t.Errorf("unknown state %+v", st)
	}
}

func TestFrameCaps(t *testing.T) {
	c := xdgFrameCapsFromMask(1<<1 | 1<<2)
	if !c.Has(FrameMenu|FrameMaximize) || c.Has(FrameMinimize) || c.Has(FrameFullscreen) {
		t.Errorf("menu+maximize %v", c)
	}
	// An empty array means "none". "Nobody has said yet" is not a value
	// the boundary carries any more: the backend resolves it (see
	// frameDesktopCaps), so a reader never has to hold a tri-state.
	if c := xdgFrameCapsFromMask(0); c != 0 {
		t.Errorf("empty caps %v", c)
	}
	all := xdgFrameCapsFromMask(1<<1 | 1<<2 | 1<<3 | 1<<4)
	if all != frameDesktopCaps {
		t.Errorf("all %v, want %v", all, frameDesktopCaps)
	}
	if got := (FrameMove | FrameKeepAbove).String(); got != "move keep-above" {
		t.Errorf("String %q", got)
	}
	if FrameCaps(0).String() != "none" {
		t.Error("no capability at all")
	}
	// Has asks for every bit, not any.
	if (FrameMove).Has(FrameMove | FrameResize) {
		t.Error("Has is all of them")
	}
}

func TestNetWMState(t *testing.T) {
	st := netWMState(netStateMaxVert|netStateMaxHorz|netStateFocused, true, false)
	if !st.Maximized || !st.Activated || st.Tiled != 0 {
		t.Errorf("maximized focused %+v", st)
	}
	st = netWMState(netStateMaxVert, true, true)
	if st.Maximized || st.Tiled != EdgeTop|EdgeBottom || st.Activated {
		t.Errorf("vertical only %+v", st)
	}
	if st := netWMState(netStateMaxHorz|netStateHidden|netStateFullscreen, true, false); st.Tiled != EdgeLeft|EdgeRight || !st.Minimized || !st.Fullscreen {
		t.Errorf("horizontal hidden fullscreen %+v", st)
	}
	// Without _NET_WM_STATE_FOCUSED the focus events decide.
	if st := netWMState(0, false, true); !st.Activated {
		t.Error("focus fallback")
	}
	if st := netWMState(netStateFocused, false, false); st.Activated {
		t.Error("an unsupported FOCUSED bit is ignored")
	}
}

func TestNetAllowedCaps(t *testing.T) {
	c := netAllowedCaps(netActionMinimize|netActionMaximizeHorz|netActionMaximizeVert, true)
	if !c.Has(FrameMinimize|FrameMaximize|FrameMenu|FrameMaximizeAxis) || c.Has(FrameFullscreen) {
		t.Errorf("caps %v", c)
	}
	// One way only is not maximize, but it is maximize-on-one-axis, which
	// X11 has and xdg-shell does not.
	c = netAllowedCaps(netActionMaximizeVert, false)
	if c.Has(FrameMaximize) || c.Has(FrameMenu) || !c.Has(FrameMaximizeAxis) {
		t.Errorf("one-way maximize is not maximize %v", c)
	}
}

func TestMotifHints(t *testing.T) {
	h, ok := motifHints(DecorationsClient)
	// flags = MWM_HINTS_DECORATIONS (1<<1), decorations = 0: GTK's value.
	if !ok || h != [5]uint32{2, 0, 0, 0, 0} {
		t.Errorf("client %v %v", h, ok)
	}
	if h, ok := motifHints(DecorationsNone); !ok || h[0] != 2 || h[2] != 0 {
		t.Errorf("none %v %v", h, ok)
	}
	if _, ok := motifHints(DecorationsServer); ok {
		t.Error("a window that never asked keeps no hint")
	}
	// A window that asked for no frame asks for all of it back:
	// flags = MWM_HINTS_DECORATIONS, decorations = MWM_DECOR_ALL (1).
	if motifDecorateAll != [5]uint32{2, 0, 1, 0, 0} {
		t.Errorf("decorate all %v", motifDecorateAll)
	}
}

func TestTilingWM(t *testing.T) {
	for _, n := range []string{"i3", "xmonad", "awesome", " bspwm "} {
		if !tilingWM(n) {
			t.Errorf("%q tiles", n)
		}
	}
	for _, n := range []string{"KWin", "GNOME Shell", "Xfwm4", "Openbox", ""} {
		if tilingWM(n) {
			t.Errorf("%q stacks", n)
		}
	}
}

func TestResizeCursorsOnEveryBackend(t *testing.T) {
	seen := map[uint32]Cursor{}
	for c := CursorResizeN; c <= CursorGrabbing; c++ {
		if c.String() == "unknown" {
			t.Errorf("cursor %d has no name", c)
		}
		shape := waylandCursorShape(c)
		if shape == wlShapeDefault {
			t.Errorf("%v falls back to the default shape", c)
		}
		if prev, dup := seen[shape]; dup {
			t.Errorf("%v and %v share shape %d", c, prev, shape)
		}
		seen[shape] = c
		if x11FontCursorShape(c) == 68 {
			t.Errorf("%v falls back to left_ptr on X11", c)
		}
		if len(waylandThemeCursorNames(c)) == 0 || len(x11ThemeCursorNames(c)) == 0 {
			t.Errorf("%v has no theme names", c)
		}
		if win32CursorID(c) == winIDCArrow {
			t.Errorf("%v is an arrow on Win32", c)
		}
	}
	if waylandCursorShape(CursorResizeNE) != 20 || waylandCursorShape(CursorMove) != 13 {
		t.Error("cursor-shape-v1 values")
	}
	if x11ThemeCursorNames(CursorResizeN)[0] != "top_side" || waylandThemeCursorNames(CursorResizeN)[0] != "n-resize" {
		t.Error("theme name order")
	}
}

func TestOffscreenWindowFrame(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 100, Height: 80})
	fs := FrameOf(o)
	if fs.Decorations() != DecorationsServer {
		t.Fatalf("default %v", fs.Decorations())
	}
	fs.RequestDecorations(DecorationsClient)
	if fs.Decorations() != DecorationsClient {
		t.Fatalf("client %v", fs.Decorations())
	}
	evs := o.Poll()
	if len(evs) != 1 || evs[0].Kind != EventDecorations || evs[0].Decor != DecorationsClient {
		t.Fatalf("decoration event %+v", evs)
	}
	if !fs.StartMove() || !fs.StartResize(EdgeBottom|EdgeRight) || fs.StartResize(EdgeTop|EdgeBottom) {
		t.Fatal("move / resize")
	}
	if !fs.ShowMenu(paintengine2d.Pt(3, 4)) {
		t.Fatal("menu")
	}
	if !fs.Minimize() || !fs.SetMaximized(true) {
		t.Fatal("minimize / maximize")
	}
	calls := o.FrameCalls()
	if calls.Moves != 1 || len(calls.Resizes) != 1 || calls.Resizes[0] != EdgeBottom|EdgeRight ||
		len(calls.Menus) != 1 || calls.Minimizes != 1 || len(calls.Maximizes) != 1 {
		t.Fatalf("calls %+v", calls)
	}
	if !fs.WindowState().Maximized {
		t.Fatal("maximize answered with the state")
	}
	evs = o.Poll()
	if len(evs) != 1 || evs[0].Kind != EventWindowState || !evs[0].State.Maximized {
		t.Fatalf("state event %+v", evs)
	}
	o.SetWindowMenu(false)
	if fs.ShowMenu(paintengine2d.Pt(0, 0)) {
		t.Fatal("no window menu")
	}
	if fs.Caps().Has(FrameMenu) {
		t.Fatal("a desktop with no window menu says so")
	}
	o.SimulateCapabilities(FrameMaximize)
	if fs.Caps().Has(FrameMinimize) || !fs.Caps().Has(FrameMaximize) {
		t.Fatalf("caps %v", fs.Caps())
	}
	// A request the desktop will not grant is refused, not silently
	// dropped: that is the whole point of the capability being data.
	if fs.Minimize() {
		t.Fatal("minimize was granted by a desktop that cannot")
	}
	if pop := NewOffscreen(WindowOptions{Popup: true}); pop.Decorations() != DecorationsNone {
		t.Fatal("a popup has no frame")
	}
	if FrameOf(NewOffscreen(WindowOptions{Decorations: DecorationsClient})).Decorations() != DecorationsClient {
		t.Fatal("the option asks at creation")
	}
}

// A surface with no frame seam answers for everything rather than being
// absent: FrameOf never returns nil, so a caller has one thing to ask.
func TestFrameOfSurfaceWithoutAFrame(t *testing.T) {
	f := FrameOf(nil)
	if f == nil {
		t.Fatal("FrameOf is never nil")
	}
	if f.Caps() != 0 || f.Decorations() != DecorationsServer || f.WindowState() != (WindowState{}) {
		t.Errorf("caps %v deco %v state %+v", f.Caps(), f.Decorations(), f.WindowState())
	}
	if f.StartMove() || f.StartResize(EdgeTop) || f.ShowMenu(paintengine2d.Pt(0, 0)) ||
		f.Minimize() || f.SetMaximized(true) || f.MaximizeAxis(true) || f.SetFullscreen(true) ||
		f.SetKeepAbove(true) || f.Lower() || f.SetShadedHeight(10) ||
		f.SetPalette("/x.colors") || f.SetIcon(nil) {
		t.Error("a surface with no frame granted something")
	}
	f.SetFrame(Frame{Alpha: true})
	if !f.Frame().Zero() {
		t.Error("nothing was kept")
	}
	f.RequestDecorations(DecorationsClient)
	if f.Decorations() != DecorationsServer {
		t.Error("nothing was negotiated")
	}
}

// Every request the frame seam offers is gated by the capability that
// names it, both ways: with the capability the request is made, without it
// the request is refused rather than quietly dropped.
//
// This is the property the whole shape exists for. Under the old nine
// interfaces a caller that forgot a type assertion, or forgot the
// *Supported() predicate inside the interface it had just found, got
// silence — a caption button that did nothing, which is a bug this repo
// has shipped more than once. There is nowhere to forget now: the
// capability is data on one object, and the request answers for itself.
func TestFrameCapsGateEveryRequest(t *testing.T) {
	// off switches the capability off on a fresh offscreen window, on
	// switches it on; do makes the request. A nil off means the offscreen
	// desktop can always do it and only a fixed window cannot.
	cases := []struct {
		name string
		cap  FrameCaps
		off  func(*Offscreen)
		on   func(*Offscreen)
		do   func(WindowFrame) bool
	}{
		{"move", FrameMove,
			func(o *Offscreen) { o.SetMoveResize(false) }, nil,
			func(f WindowFrame) bool { return f.StartMove() }},
		{"resize", FrameResize,
			func(o *Offscreen) { o.SetMoveResize(false) }, nil,
			func(f WindowFrame) bool { return f.StartResize(EdgeBottom | EdgeRight) }},
		{"menu", FrameMenu,
			func(o *Offscreen) { o.SetWindowMenu(false) }, nil,
			func(f WindowFrame) bool { return f.ShowMenu(paintengine2d.Pt(1, 2)) }},
		{"minimize", FrameMinimize,
			func(o *Offscreen) { o.SimulateCapabilities(0) }, nil,
			func(f WindowFrame) bool { return f.Minimize() }},
		{"maximize", FrameMaximize,
			func(o *Offscreen) { o.SimulateCapabilities(0) }, nil,
			func(f WindowFrame) bool { return f.SetMaximized(true) }},
		{"fullscreen", FrameFullscreen,
			func(o *Offscreen) { o.SimulateCapabilities(0) }, nil,
			func(f WindowFrame) bool { return f.SetFullscreen(true) }},
		{"maximize-axis", FrameMaximizeAxis,
			func(o *Offscreen) { o.SetSizing(SizingFixed) }, nil,
			func(f WindowFrame) bool { return f.MaximizeAxis(true) }},
		{"shade", FrameShade,
			func(o *Offscreen) { o.SetSizing(SizingFixed) }, nil,
			func(f WindowFrame) bool { return f.SetShadedHeight(30) }},
		{"keep-above", FrameKeepAbove,
			nil, func(o *Offscreen) { o.SimulateKeepAbove(true) },
			func(f WindowFrame) bool { return f.SetKeepAbove(true) }},
		{"palette", FramePalette,
			nil, func(o *Offscreen) { o.SimulateDecorationPalette(true) },
			func(f WindowFrame) bool { return f.SetPalette("/cache/a.colors") }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Without the capability: not in Caps, and refused.
			o := NewOffscreen(WindowOptions{Width: 200, Height: 140})
			if c.off != nil {
				c.off(o)
			}
			if f := FrameOf(o); f.Caps().Has(c.cap) {
				t.Fatalf("%v is still in %v", c.cap, f.Caps())
			} else if c.do(f) {
				t.Fatalf("%v was granted by a desktop that says it cannot", c.cap)
			}
			// With it: in Caps, and granted.
			o = NewOffscreen(WindowOptions{Width: 200, Height: 140})
			if c.on != nil {
				c.on(o)
			}
			if f := FrameOf(o); !f.Caps().Has(c.cap) {
				t.Fatalf("%v missing from %v", c.cap, f.Caps())
			} else if !c.do(f) {
				t.Fatalf("%v was refused by a desktop that says it can", c.cap)
			}
		})
	}
}

// Unpinning a rolled-up window always works, even where the pin itself
// cannot be taken. A window must never be stuck rolled up because the
// capability went away under it (app.Window.unshadeIfStuck relies on this).
func TestShadeUnpinAlwaysWorks(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 200, Height: 140})
	o.SetSizing(SizingFixed)
	if !FrameOf(o).SetShadedHeight(0) {
		t.Fatal("unpinning was refused")
	}
}
