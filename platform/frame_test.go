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

func TestWMCaps(t *testing.T) {
	var unknown WMCaps
	if !unknown.Can(CapMinimize) || !unknown.Can(CapWindowMenu) {
		t.Error("an unknown desktop is assumed to do everything")
	}
	c := xdgCapsFromMask(1<<1 | 1<<2)
	if !c.Can(CapWindowMenu) || !c.Can(CapMaximize) || c.Can(CapMinimize) || c.Can(CapFullscreen) {
		t.Errorf("menu+maximize %08b", c)
	}
	// An empty array means "none", not "unknown".
	if c := xdgCapsFromMask(0); c.Can(CapMinimize) || c&CapKnown == 0 {
		t.Errorf("empty caps %08b", c)
	}
	all := xdgCapsFromMask(1<<1 | 1<<2 | 1<<3 | 1<<4)
	if !all.Can(CapWindowMenu | CapMaximize | CapFullscreen | CapMinimize) {
		t.Errorf("all %08b", all)
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
	if !c.Can(CapMinimize) || !c.Can(CapMaximize) || !c.Can(CapWindowMenu) || c.Can(CapFullscreen) {
		t.Errorf("caps %08b", c)
	}
	if c := netAllowedCaps(netActionMaximizeVert, false); c.Can(CapMaximize) || c.Can(CapWindowMenu) {
		t.Errorf("one-way maximize is not maximize %08b", c)
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
		t.Error("the window manager's frame removes the hint")
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

func TestOffscreenFrameSurface(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 100, Height: 80})
	var fs FrameSurface = o
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
	if !fs.StartSystemMove() || !fs.StartSystemResize(EdgeBottom|EdgeRight) || fs.StartSystemResize(EdgeTop|EdgeBottom) {
		t.Fatal("move / resize")
	}
	if !fs.ShowWindowMenu(paintengine2d.Pt(3, 4)) {
		t.Fatal("menu")
	}
	fs.Minimize()
	SetMaximized(o, true)
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
	if fs.ShowWindowMenu(paintengine2d.Pt(0, 0)) {
		t.Fatal("no window menu")
	}
	o.SimulateCapabilities(CapMaximize)
	if fs.Capabilities().Can(CapMinimize) || !fs.Capabilities().Can(CapMaximize) {
		t.Fatalf("caps %08b", fs.Capabilities())
	}
	if pop := NewOffscreen(WindowOptions{Popup: true}); pop.Decorations() != DecorationsNone {
		t.Fatal("a popup has no frame")
	}
	if SurfaceDecorations(NewOffscreen(WindowOptions{Decorations: DecorationsClient})) != DecorationsClient {
		t.Fatal("the option asks at creation")
	}
}
