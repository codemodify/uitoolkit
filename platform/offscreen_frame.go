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
	// AxisMaximizes are the one-axis maximize requests, in order (true:
	// vertically).
	AxisMaximizes []bool
	// Aboves are the keep-above requests, in order, and ShadeHeights the
	// heights a rolled-up window was pinned to (0 unpins).
	Aboves       []bool
	ShadeHeights []int
	// Requests are the decoration modes asked for, in order, and Frames
	// the frames (margin, resize band, corners, alpha) handed over.
	Requests []Decorations
	Frames   []Frame
	// Palettes are the colour schemes named for the desktop's frame, in
	// order (only while the simulated desktop takes one:
	// SimulateDecorationPalette), and Icons the icons given, each as its
	// image sizes.
	Palettes []string
	Icons    [][]int
}

// offscreenFrame is Offscreen's WindowFrame state: a desktop that grants
// whatever it is asked (a client frame included) and whose window menu can be
// switched off to exercise the toolkit's fallback.
type offscreenFrame struct {
	deco         Decorations
	decoSet      bool // always set by NewOffscreen; a zero Offscreen reports Server
	state        WindowState
	caps         FrameCaps
	capsSet      bool // the simulated desktop has said (SimulateCapabilities)
	calls        FrameCalls
	noWindowMenu bool
	noMoveResize bool
	// glass is whether the simulated desktop blurs behind a window
	// (SimulateGlass); off by default, as a plain compositor is.
	glass bool
	// above is whether the simulated desktop can keep a window above the
	// others (SimulateKeepAbove). It is off by default — an offscreen
	// window has no desktop to be stacked in, which is also the Wayland
	// answer — so a test that wants the caption button asks for it.
	above bool
	// shadeH is the height a rolled-up window is pinned to (0: not).
	shadeH int
	// palette is whether the simulated desktop takes a frame palette
	// (SimulateDecorationPalette), and paletteSet the one it was given.
	palette    bool
	paletteSet string
	// sysShadow / sysBand are whether the simulated desktop draws the
	// window's drop shadow itself and runs edge resizing itself
	// (SimulateSystemShadow, SimulateSystemResizeBand). Both off by
	// default: the simulated desktop is a Linux one, where the client
	// owns both. Windows sets the first, macOS sets both.
	sysShadow bool
	sysBand   bool
	// frame is the client frame's margin and regions; geomW / geomH the
	// visible window's size in device pixels, which the pixmap grows past
	// by the margin, exactly as a compositor's surface does, and geomLW /
	// geomLH the same window in the logical pixels window geometry is
	// stated in (the two are equal until SimulateScale).
	frame          Frame
	geomW, geomH   int
	geomLW, geomLH int
	// imeRect is the caret rectangle the app last gave the simulated
	// secureInput and noCapture are the last SetSecureInput and
	// SetExcludeFromCapture requests, so a test can assert a prompt
	// asked for them.
	secureInput bool
	noCapture   bool
	// role, centered and activated are the window-role requests, for the
	// same reason.
	// capsLock and numLock are the simulated lock keys (SetLockKeys);
	// caps, a hand's width up, is the frame capability set.
	capsLock, numLock bool
	role              WindowRole
	// owner is the window this one belongs to and skipTask whether it asked
	// to be left out of the task bar, both recorded so a test can assert
	// what a satellite panel asked for. noOwner and noSkipTask take the two
	// capabilities away, for the test that wants the *absence* of them —
	// which is Wayland's answer about the task bar.
	owner Surface
	// task is the window's place in the simulated window list and how it came
	// to be asked for, the same policy the real backends follow.
	task                taskbarPolicy
	noOwner, noSkipTask bool
	centered            bool
	centerings          int
	wantAbove           bool
	activated           int
	// input method (surface device pixels), imeOn whether it is enabled.
	imeRect [4]int
	imeOn   bool
}

// Decorations is the requested mode; an offscreen window has no desktop
// frame, so Server (and Auto) is reported as asked (WindowFrame).
func (o *Offscreen) Decorations() Decorations {
	if !o.frame.decoSet {
		return DecorationsServer
	}
	return o.frame.deco
}

// RequestDecorations records d and grants it (WindowFrame).
func (o *Offscreen) RequestDecorations(d Decorations) {
	d = requestedDecorations(d)
	o.frame.calls.Requests = append(o.frame.calls.Requests, d)
	if o.frame.decoSet && d == o.frame.deco {
		return
	}
	o.frame.deco, o.frame.decoSet = d, true
	o.queue = append(o.queue, Event{Kind: EventDecorations, Decor: d})
}

// WindowState is the simulated state (WindowFrame).
func (o *Offscreen) WindowState() WindowState { return o.frame.state }

// Caps is what the simulated desktop will do for the window
// (WindowFrame). The offscreen backend is the third implementation of this
// boundary and the one every headless test runs against, so it answers the
// whole of it and answers honestly: everything a test has switched on, and
// nothing it has not.
//
// The switches are the point. A real desktop's capabilities change under a
// running window — a compositor withholds minimize, a window manager is
// replaced, KWin drops blur when effects go off — and the only way to test
// that the layer above degrades instead of drawing a dead control is to be
// able to take a capability away. SimulateCapabilities, SimulateKeepAbove,
// SimulateGlass, SimulateDecorationPalette, SetWindowMenu and SetMoveResize
// each take one away.
func (o *Offscreen) FrameCaps() FrameCaps {
	if o == nil {
		return 0
	}
	// What the simulated desktop grants. Until a test has said, all four,
	// which is what a real backend assumes before its desktop has spoken.
	c := frameDesktopCaps
	if o.frame.capsSet {
		c = o.frame.caps
	}
	if o.frame.noWindowMenu {
		c &^= FrameMenu
	}
	if !o.frame.noMoveResize {
		c |= FrameMove | FrameResize | FrameClientFrame
	}
	// The simulated desktop always stacks, rolls up and dresses windows:
	// it is a desktop with no protocol to be short of, so a test that
	// wants the *absence* of one asks for it.
	c |= FrameLower | FrameMaximizeAxis | FrameShade | FrameIcon
	if o.frame.above {
		c |= FrameKeepAbove
	}
	if o.frame.glass {
		c |= FrameBlurBehind
	}
	if o.frame.palette {
		c |= FramePalette
	}
	if o.frame.sysShadow {
		c |= FrameSystemShadow
	}
	if o.frame.sysBand {
		c |= FrameSystemResizeBand
	}
	if !o.frame.noOwner {
		c |= FrameOwner
	}
	if !o.frame.noSkipTask {
		c |= FrameSkipTaskbar
	}
	return dropResizeCaps(c, o.sizing)
}

// SimulateSystemShadow makes the simulated desktop draw the window's drop
// shadow itself, the way Windows' DWM and macOS's AppKit do
// ([FrameSystemShadow]). A frame the toolkit draws must then reserve no
// margin for one and paint none. Off by default: the simulated desktop is
// a Linux one, where the client owns its shadow.
func (o *Offscreen) SimulateSystemShadow(on bool) {
	if o == nil || o.frame.sysShadow == on {
		return
	}
	o.frame.sysShadow = on
	o.announceCaps()
}

// SimulateSystemResizeBand makes the simulated desktop run edge resizing
// itself ([FrameSystemResizeBand]), the way a resizable NSWindow does, so
// the toolkit reserves no input band. Off by default.
func (o *Offscreen) SimulateSystemResizeBand(on bool) {
	if o == nil || o.frame.sysBand == on {
		return
	}
	o.frame.sysBand = on
	o.announceCaps()
}

// Sizing is the window's resize policy (WindowGeometry).
func (o *Offscreen) Sizing() Sizing { return o.sizing }

// SetSizing changes the policy and re-states the limits (WindowGeometry).
func (o *Offscreen) SetSizing(s Sizing) bool {
	if o == nil || !o.GeometryCaps().Has(GeometrySizeLimits) {
		return false
	}
	if o.sizing == s {
		return true
	}
	o.sizing = s
	o.limits = limitsFor(s, o.opts, o.frame.geomLW, o.frame.geomLH)
	// A fixed window loses maximize, edge-resize and roll-up
	// (dropResizeCaps), so what the caption may draw changed.
	o.announceCaps()
	return true
}

// SizeLimits is what the simulated desktop was told (WindowGeometry), the
// height pin of a rolled-up window included.
func (o *Offscreen) SizeLimits() SizeLimits { return shadePin(o.limits, o.frame.shadeH) }

// StartMove records a move (WindowFrame).
func (o *Offscreen) StartMove() bool {
	if !o.FrameCaps().Has(FrameMove) {
		return false
	}
	o.frame.calls.Moves++
	return true
}

// StartResize records a resize from edges (WindowFrame). A fixed window
// refuses one, as a desktop does.
func (o *Offscreen) StartResize(edges Edges) bool {
	if !o.FrameCaps().Has(FrameResize) || !edges.Valid() {
		return false
	}
	o.frame.calls.Resizes = append(o.frame.calls.Resizes, edges)
	return true
}

// ShowMenu records the request; false when the simulated desktop has no
// window menu (SetWindowMenu).
func (o *Offscreen) ShowMenu(p paintengine2d.Point) bool {
	if !o.FrameCaps().Has(FrameMenu) {
		return false
	}
	o.frame.calls.Menus = append(o.frame.calls.Menus, p)
	return true
}

// SetFrame takes the frame the toolkit draws: the pixmap grows by its
// margin at once, so the window can paint the shadow it just asked for
// (WindowFrame).
func (o *Offscreen) SetFrame(f Frame) {
	if o == nil || f.Same(o.frame.frame) {
		return
	}
	o.frame.frame = f
	o.frame.calls.Frames = append(o.frame.calls.Frames, f)
	o.resizeSurface()
}

// Frame is the frame last set (WindowFrame).
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
		o.frame.geomLW = LogicalPixels(o.frame.geomW, o.Scale())
		o.frame.geomLH = LogicalPixels(o.frame.geomH, o.Scale())
	}
	m := o.frame.frame.Margin
	w, h := o.frame.geomW+m.Width(), o.frame.geomH+m.Height()
	if o.img != nil && o.img.Width == w && o.img.Height == h {
		return
	}
	o.img = paintengine2d.NewImage(max(w, 1), max(h, 1))
}

// Minimize records the request (WindowFrame).
func (o *Offscreen) Minimize() bool {
	if !o.FrameCaps().Has(FrameMinimize) {
		return false
	}
	o.frame.calls.Minimizes++
	return true
}

// Lower records the request (WindowFrame).
func (o *Offscreen) Lower() bool {
	if !o.FrameCaps().Has(FrameLower) {
		return false
	}
	o.frame.calls.Lowers++
	return true
}

// SetMaximized records the request and, like a compositor, answers with
// the new state (WindowFrame).
func (o *Offscreen) SetMaximized(on bool) bool {
	if !o.FrameCaps().Has(FrameMaximize) {
		return false
	}
	o.frame.calls.Maximizes = append(o.frame.calls.Maximizes, on)
	st := o.frame.state
	st.Maximized = on
	o.SimulateWindowState(st)
	return true
}

// MaximizeAxis maximizes or restores one way only (WindowFrame). The
// simulated desktop answers with the state, as it does for SetMaximized:
// a window maximized both ways is Maximized, one way is not.
func (o *Offscreen) MaximizeAxis(vertical bool) bool {
	if !o.FrameCaps().Has(FrameMaximizeAxis) {
		return false
	}
	o.frame.calls.AxisMaximizes = append(o.frame.calls.AxisMaximizes, vertical)
	return true
}

// SetFullscreen records the request and answers with the new state
// (WindowFrame).
func (o *Offscreen) SetFullscreen(on bool) bool {
	if !o.FrameCaps().Has(FrameFullscreen) {
		return false
	}
	o.frame.calls.Fullscreen = append(o.frame.calls.Fullscreen, on)
	st := o.frame.state
	st.Fullscreen = on
	o.SimulateWindowState(st)
	return true
}

// SetKeepAbove records the request and, like a window manager that grants
// it, answers with the new state (WindowFrame).
func (o *Offscreen) SetKeepAbove(on bool) bool {
	if !o.FrameCaps().Has(FrameKeepAbove) {
		return false
	}
	o.frame.calls.Aboves = append(o.frame.calls.Aboves, on)
	st := o.frame.state
	st.KeepAbove = on
	o.SimulateWindowState(st)
	return true
}

// SimulateKeepAbove switches whether the simulated desktop can keep a
// window above the others, so a test can run both the X11 path and the
// Wayland one, where the caption button has to say it cannot (tests).
func (o *Offscreen) SimulateKeepAbove(can bool) {
	if o == nil || o.frame.above == can {
		return
	}
	o.frame.above = can
	o.announceCaps()
}

// announceCaps queues EventCapabilities with what the simulated desktop
// will do now.
//
// Every capability here can change under a running window — that is the
// whole reason Caps is asked afresh rather than cached — and a change
// nobody is told about is a caption button that stays drawn after it has
// stopped working, or stays hidden after it has started. The real
// backends send this event; the simulated desktop has to send it too, or
// the tests are kinder than the world.
func (o *Offscreen) announceCaps() {
	o.queue = append(o.queue, Event{Kind: EventCapabilities, Caps: o.FrameCaps()})
}

// SetShadedHeight records the height a rolled-up window is pinned to and
// states it as a limit, exactly as the real backends do (FrameShade).
func (o *Offscreen) SetShadedHeight(h int) bool {
	if o == nil {
		return false
	}
	h = max(h, 0)
	// Unpinning always works: a window must never be left rolled up
	// because the capability went away under it.
	if h > 0 && !o.FrameCaps().Has(FrameShade) {
		return false
	}
	if h == o.frame.shadeH {
		return true
	}
	o.frame.shadeH = h
	o.frame.calls.ShadeHeights = append(o.frame.calls.ShadeHeights, h)
	o.limits = limitsFor(o.sizing, o.opts, o.frame.geomLW, o.frame.geomLH)
	return true
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

// SimulateCapabilities sets which of the four per-window capabilities the
// simulated desktop grants — the window menu, minimize, maximize, full
// screen — and queues EventCapabilities (tests). Anything else in c is
// ignored: the rest of [FrameCaps] is what the *backend* can do, and the
// offscreen backend's own switches say that.
func (o *Offscreen) SimulateCapabilities(c FrameCaps) {
	c &= frameDesktopCaps
	if o.frame.capsSet && c == o.frame.caps {
		return
	}
	o.frame.caps, o.frame.capsSet = c, true
	o.queue = append(o.queue, Event{Kind: EventCapabilities, Caps: o.FrameCaps()})
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

// SimulateGlass switches the simulated desktop's blur-behind on or off, so
// a test can run both the real-glass path and the painted fallback
// (FrameBlurBehind). It takes effect for the next frame the window asks for.
func (o *Offscreen) SimulateGlass(on bool) {
	if o == nil || o.frame.glass == on {
		return
	}
	o.frame.glass = on
	o.announceCaps()
}

// SetWindowMenu switches the simulated desktop's window menu on or off
// (off exercises the toolkit's own menu).
func (o *Offscreen) SetWindowMenu(on bool) { o.frame.noWindowMenu = !on }

// SetMoveResize switches the simulated desktop's interactive move and
// resize on or off.
func (o *Offscreen) SetMoveResize(on bool) { o.frame.noMoveResize = !on }

// Offscreen is a full WindowFrame: a compile-time check, and the reason
// the check is worth anything. The frame seam is one interface a backend
// implements whole, so a capability the boundary grows cannot quietly stop
// being implemented here — the headless tests would stop compiling rather
// than stop covering it.
var _ WindowFrame = (*Offscreen)(nil)

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

// SimulateDecorationPalette makes the simulated desktop take a colour
// scheme for its frame, as KWin does, or stop taking one.
func (o *Offscreen) SimulateDecorationPalette(on bool) {
	if o == nil || o.frame.palette == on {
		return
	}
	o.frame.palette = on
	o.announceCaps()
}

// SetPalette records the colour scheme when it changes (WindowFrame).
func (o *Offscreen) SetPalette(path string) bool {
	if !o.FrameCaps().Has(FramePalette) {
		return false
	}
	if path == o.frame.paletteSet {
		return true
	}
	o.frame.paletteSet = path
	o.frame.calls.Palettes = append(o.frame.calls.Palettes, path)
	return true
}

// DecorationPalette is the palette the simulated desktop was last given.
func (o *Offscreen) DecorationPalette() string { return o.frame.paletteSet }

// SetIcon records the icon's sizes (WindowFrame).
func (o *Offscreen) SetIcon(images []*paintengine2d.Image) bool {
	if !o.FrameCaps().Has(FrameIcon) {
		return false
	}
	var sizes []int
	for _, im := range iconImages(images) {
		sizes = append(sizes, im.Width)
	}
	o.frame.calls.Icons = append(o.frame.calls.Icons, sizes)
	return true
}

// SetSecureInput records the request ([SecureInputSurface]). Offscreen
// has no keyboard to take, so it answers yes and remembers, which is
// what a test asserting that a prompt asked for it needs.
func (o *Offscreen) SetSecureInput(on bool) bool {
	if o == nil {
		return false
	}
	o.frame.secureInput = on
	return true
}

// SetExcludeFromCapture records the request ([CaptureExcludeSurface]).
func (o *Offscreen) SetExcludeFromCapture(on bool) bool {
	if o == nil {
		return false
	}
	o.frame.noCapture = on
	return true
}

// SecureInput and ExcludeFromCapture are what the app last asked for.
func (o *Offscreen) SecureInput() bool {
	return o != nil && o.frame.secureInput
}

func (o *Offscreen) ExcludeFromCapture() bool {
	return o != nil && o.frame.noCapture
}

// SetWindowRole records the request ([RoleSurface]).
func (o *Offscreen) SetWindowRole(r WindowRole) bool {
	if o == nil {
		return false
	}
	was := o.frame.role
	o.frame.role = r
	o.frame.task.roleChanged(was, r)
	return true
}

// SetOwner records which window this one belongs to ([OwnedSurface]), and
// refuses the two owners no backend accepts: this window itself, and one of
// another kind.
func (o *Offscreen) SetOwner(owner Surface) bool {
	if o == nil || !o.FrameCaps().Has(FrameOwner) {
		return false
	}
	if owner == nil {
		o.frame.owner = nil
		return true
	}
	if ow, ok := owner.(*Offscreen); !ok || ow == o {
		return false
	}
	o.frame.owner = owner
	return true
}

// SetSkipTaskbar records the request ([TaskbarSurface]).
func (o *Offscreen) SetSkipTaskbar(skip bool) bool {
	if o == nil || !o.FrameCaps().Has(FrameSkipTaskbar) {
		return false
	}
	o.frame.task.ask(skip)
	return true
}

// Owner and SkipsTaskbar are what the app asked for.
func (o *Offscreen) Owner() Surface {
	if o == nil {
		return nil
	}
	return o.frame.owner
}

func (o *Offscreen) SkipsTaskbar() bool { return o != nil && o.frame.task.skipping() }

// SimulateOwner and SimulateSkipTaskbar switch whether the simulated desktop
// can keep a window with the one it belongs to and out of the task bar. Both
// are on by default — the simulated desktop is a desktop with no protocol to
// be short of — and a test that wants the absence of one turns it off. The
// task bar is the one that matters: Wayland has no way for a client to ask.
func (o *Offscreen) SimulateOwner(can bool) {
	if o != nil {
		o.frame.noOwner = !can
	}
}

func (o *Offscreen) SimulateSkipTaskbar(can bool) {
	if o != nil {
		o.frame.noSkipTask = !can
	}
}

// Center and Activate record the request: offscreen has no desktop to be
// centred on and no focus to take, and a test asserting that a prompt
// asked for either is what these are for.
func (o *Offscreen) Center() bool {
	if o == nil {
		return false
	}
	o.frame.centered = true
	o.frame.centerings++
	return true
}

func (o *Offscreen) Activate() bool {
	if o == nil {
		return false
	}
	o.frame.activated++
	return true
}

// WindowRole, Centered and Activations are what the app asked for.
func (o *Offscreen) WindowRole() WindowRole {
	if o == nil {
		return RoleNormal
	}
	return o.frame.role
}

func (o *Offscreen) Centered() bool { return o != nil && o.frame.centered }

// Centerings is how many times the window was put in the middle. A fit
// that changes the size has to centre again — a resize keeps the
// top-left corner, so a dialog centred at the size it was made with is
// off-centre at the size it is fitted to.
func (o *Offscreen) Centerings() int {
	if o == nil {
		return 0
	}
	return o.frame.centerings
}

// KeepAboveRequested is what WindowOptions asked for, which is not the
// same fact as WindowState().KeepAbove — that is what the desktop did.
func (o *Offscreen) KeepAboveRequested() bool { return o != nil && o.frame.wantAbove }

func (o *Offscreen) Activations() int {
	if o == nil {
		return 0
	}
	return o.frame.activated
}

var (
	_ RoleSurface     = (*Offscreen)(nil)
	_ CenterSurface   = (*Offscreen)(nil)
	_ ActivateSurface = (*Offscreen)(nil)
	_ OwnedSurface    = (*Offscreen)(nil)
	_ TaskbarSurface  = (*Offscreen)(nil)
)

// SetLockKeys simulates the keyboard's lock state, so a test can open a
// window with Caps Lock already on ([LockKeysSurface]).
func (o *Offscreen) SetLockKeys(caps, num bool) {
	if o != nil {
		o.frame.capsLock, o.frame.numLock = caps, num
	}
}

// LockKeys is what SetLockKeys was told ([LockKeysSurface]).
func (o *Offscreen) LockKeys() (caps, num bool) {
	if o == nil {
		return false, false
	}
	return o.frame.capsLock, o.frame.numLock
}

var _ LockKeysSurface = (*Offscreen)(nil)
