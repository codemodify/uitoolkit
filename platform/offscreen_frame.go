package platform

import "github.com/codemodify/paintengine2d"

// FrameCalls records what an app asked an Offscreen surface's "desktop" to
// do, so tests can check that a caption drag started one move (and a click
// none), which edges a resize started from, where a window menu was asked
// for.
type FrameCalls struct {
	Moves      int
	Resizes    []Edges
	Menus      []paintengine2d.Point
	Minimizes  int
	Maximizes  []bool
	Fullscreen []bool
	Lowers     int
	// Requests are the decoration modes asked for, in order, and Frames
	// the frames (margin, resize band, corners, alpha) handed over.
	Requests []Decorations
	Frames   []Frame
}

// offscreenFrame is Offscreen's FrameSurface state: a desktop that grants
// whatever it is asked (a client frame included) and whose window menu can be
// switched off to exercise the toolkit's fallback.
type offscreenFrame struct {
	deco         Decorations
	decoSet      bool // always set by NewOffscreen; a zero Offscreen reports Server
	state        WindowState
	caps         WMCaps
	calls        FrameCalls
	noWindowMenu bool
	noMoveResize bool
	// glass is whether the simulated desktop blurs behind a window
	// (SimulateGlass); off by default, as a plain compositor is.
	glass bool
	// frame is the client frame's margin and regions; geomW / geomH the
	// visible window's size, which the pixmap grows past by the margin,
	// exactly as a compositor's surface does.
	frame        Frame
	geomW, geomH int
	// imeRect is the caret rectangle the app last gave the simulated
	// input method (surface device pixels), imeOn whether it is enabled.
	imeRect [4]int
	imeOn   bool
}

// Decorations is the requested mode; an offscreen window has no desktop
// frame, so Server (and Auto) is reported as asked (FrameSurface).
func (o *Offscreen) Decorations() Decorations {
	if !o.frame.decoSet {
		return DecorationsServer
	}
	return o.frame.deco
}

// RequestDecorations records d and grants it (FrameSurface).
func (o *Offscreen) RequestDecorations(d Decorations) {
	d = requestedDecorations(d)
	o.frame.calls.Requests = append(o.frame.calls.Requests, d)
	if o.frame.decoSet && d == o.frame.deco {
		return
	}
	o.frame.deco, o.frame.decoSet = d, true
	o.queue = append(o.queue, Event{Kind: EventDecorations, Decor: d})
}

// WindowState is the simulated state (FrameSurface).
func (o *Offscreen) WindowState() WindowState { return o.frame.state }

// Capabilities are the simulated desktop's (FrameSurface).
func (o *Offscreen) Capabilities() WMCaps { return o.frame.caps }

// SuitsClientFrame is true: the simulated desktop moves and resizes.
func (o *Offscreen) SuitsClientFrame() bool { return !o.frame.noMoveResize }

// StartSystemMove records a move (FrameSurface).
func (o *Offscreen) StartSystemMove() bool {
	if o.frame.noMoveResize {
		return false
	}
	o.frame.calls.Moves++
	return true
}

// StartSystemResize records a resize from edges (FrameSurface).
func (o *Offscreen) StartSystemResize(edges Edges) bool {
	if o.frame.noMoveResize || !edges.Valid() {
		return false
	}
	o.frame.calls.Resizes = append(o.frame.calls.Resizes, edges)
	return true
}

// ShowWindowMenu records the request; false when the simulated desktop has
// no window menu (SetWindowMenu).
func (o *Offscreen) ShowWindowMenu(p paintengine2d.Point) bool {
	if o.frame.noWindowMenu || !o.frame.caps.Can(CapWindowMenu) {
		return false
	}
	o.frame.calls.Menus = append(o.frame.calls.Menus, p)
	return true
}

// SetFrame takes the frame the toolkit draws: the pixmap grows by its
// margin at once, so the window can paint the shadow it just asked for
// (FrameSurface).
func (o *Offscreen) SetFrame(f Frame) {
	if o == nil || f.Same(o.frame.frame) {
		return
	}
	o.frame.frame = f
	o.frame.calls.Frames = append(o.frame.calls.Frames, f)
	o.resizeSurface()
}

// Frame is the frame last set (FrameSurface).
func (o *Offscreen) Frame() Frame {
	if o == nil {
		return Frame{}
	}
	return o.frame.frame
}

// resizeSurface sizes the pixmap to the visible window plus the frame's
// margin.
func (o *Offscreen) resizeSurface() {
	if o.frame.geomW < 1 || o.frame.geomH < 1 {
		w, h := o.img.Width, o.img.Height
		m := o.frame.frame.Margin
		o.frame.geomW, o.frame.geomH = max(w-m.Width(), 1), max(h-m.Height(), 1)
	}
	m := o.frame.frame.Margin
	w, h := o.frame.geomW+m.Width(), o.frame.geomH+m.Height()
	if o.img != nil && o.img.Width == w && o.img.Height == h {
		return
	}
	o.img = paintengine2d.NewImage(max(w, 1), max(h, 1))
}

// Minimize records the request (FrameSurface).
func (o *Offscreen) Minimize() { o.frame.calls.Minimizes++ }

// Lower records the request (Lowerer).
func (o *Offscreen) Lower() { o.frame.calls.Lowers++ }

// SetMaximized records the request and, like a compositor, answers with
// the new state (DesktopSurface).
func (o *Offscreen) SetMaximized(on bool) {
	o.frame.calls.Maximizes = append(o.frame.calls.Maximizes, on)
	st := o.frame.state
	st.Maximized = on
	o.SimulateWindowState(st)
}

// SetFullscreen records the request and answers with the new state
// (DesktopSurface).
func (o *Offscreen) SetFullscreen(on bool) {
	o.frame.calls.Fullscreen = append(o.frame.calls.Fullscreen, on)
	st := o.frame.state
	st.Fullscreen = on
	o.SimulateWindowState(st)
}

// FrameCalls returns what the app asked the simulated desktop to do.
func (o *Offscreen) FrameCalls() FrameCalls { return o.frame.calls }

// SimulateWindowState sets the window's state as a compositor would and
// queues EventWindowState when it changed (tests).
func (o *Offscreen) SimulateWindowState(st WindowState) {
	if st == o.frame.state {
		return
	}
	o.frame.state = st
	o.queue = append(o.queue, Event{Kind: EventWindowState, State: st})
}

// SimulateCapabilities sets what the simulated desktop can do and queues
// EventCapabilities (tests).
func (o *Offscreen) SimulateCapabilities(c WMCaps) {
	c |= CapKnown
	if c == o.frame.caps {
		return
	}
	o.frame.caps = c
	o.queue = append(o.queue, Event{Kind: EventCapabilities, Caps: c})
}

// SimulateDecorations answers with mode d as a compositor may on its own
// (KWin drops its frame for a full-screen window), queuing EventDecorations.
func (o *Offscreen) SimulateDecorations(d Decorations) {
	if o.frame.decoSet && d == o.frame.deco {
		return
	}
	o.frame.deco, o.frame.decoSet = d, true
	o.queue = append(o.queue, Event{Kind: EventDecorations, Decor: d})
}

// SimulateCompositing tells the window whether the simulated desktop
// composites it: without compositing a client frame goes solid (square,
// shadowless), as on an X11 screen with no compositing manager (tests).
func (o *Offscreen) SimulateCompositing(on bool) {
	st := o.frame.state
	st.Solid = !on
	o.SimulateWindowState(st)
}

// BlurBehindSupported is whether the simulated desktop blurs behind a
// window (GlassSurface; SimulateGlass switches it).
func (o *Offscreen) BlurBehindSupported() bool { return o != nil && o.frame.glass }

// SimulateGlass switches the simulated desktop's blur-behind on or off, so
// a test can run both the real-glass path and the painted fallback
// (GlassSurface). It takes effect for the next frame the window asks for.
func (o *Offscreen) SimulateGlass(on bool) {
	if o != nil {
		o.frame.glass = on
	}
}

// SetWindowMenu switches the simulated desktop's window menu on or off
// (off exercises the toolkit's own menu).
func (o *Offscreen) SetWindowMenu(on bool) { o.frame.noWindowMenu = !on }

// SetMoveResize switches the simulated desktop's interactive move and
// resize on or off.
func (o *Offscreen) SetMoveResize(on bool) { o.frame.noMoveResize = !on }

// Offscreen is a full FrameSurface: a compile-time check, so a capability
// the backends grow does not silently stop being one here. It answers for
// glass too, so the whole shaped-and-glassy path is testable headless.
var _ FrameSurface = (*Offscreen)(nil)
var _ GlassSurface = (*Offscreen)(nil)

// ---- the simulated input method ------------------------------------------

// SetIMECursor records where the app says the caret is, in surface device
// pixels (IMESurface). A frame's margin is part of those coordinates —
// what a compositor's candidate window is placed against — so tests can
// check that the toolkit offsets them.
func (o *Offscreen) SetIMECursor(x, y, w, h int) {
	if o == nil {
		return
	}
	o.frame.imeRect = [4]int{x, y, w, h}
}

// SetIMEEnabled records the app's request (IMESurface).
func (o *Offscreen) SetIMEEnabled(on bool) {
	if o != nil {
		o.frame.imeOn = on
	}
}

// IMECursor is the caret rectangle last handed to the input method, and
// whether the input method is on.
func (o *Offscreen) IMECursor() (x, y, w, h int, on bool) {
	if o == nil {
		return 0, 0, 0, 0, false
	}
	r := o.frame.imeRect
	return r[0], r[1], r[2], r[3], o.frame.imeOn
}

// Offscreen drives a simulated input method too (compile-time check).
var _ IMESurface = (*Offscreen)(nil)
