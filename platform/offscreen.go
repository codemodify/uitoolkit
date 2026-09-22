package platform

import (
	"sync/atomic"
	"time"

	"github.com/codemodify/paintengine2d"
)

// Offscreen is a pixmap surface with no OS window. Used for tests,
// screenshots, and Application.Headless.
type Offscreen struct {
	soft   softDevice // software devices of the buffers (no GPU)
	title  string
	img    *paintengine2d.Image
	closed bool
	hidden bool
	queue  []Event
	cursor Cursor
	wake   chan struct{}
	wakes  atomic.Int32
	// posX, posY are where this window sits on the offscreen desktop, so
	// a test can place windows and read back where a drag carried one.
	// They are the desktop's device pixels, as an X11 root window's are;
	// Move and Position speak logical ones and convert at the scale.
	posX, posY int
	// drop is a simulated drop's data by type (SimulateDrop); dragOut is
	// a drag started *from* this surface (offscreen_drag.go).
	drop      map[string][]byte
	dropTaken *bool
	dragOut   offscreenDrag
	// frame is the simulated desktop behind FrameSurface (offscreen_frame.go).
	frame offscreenFrame
	// opts are the options the window was made with, sizing its resize
	// policy and limits what the simulated desktop was told about how
	// large the window may be (platform.SizingSurface).
	opts   WindowOptions
	sizing Sizing
	limits SizeLimits
	// scale is the simulated display scale (SimulateScale), 1 unless a
	// test sets one: the window's size is logical pixels and the pixmap
	// behind it is that many times this.
	scale float32
	// pops is the simulated popup support (offscreen_popup.go): on a top
	// level, whether popups open as surfaces and the work area they are
	// placed in; on a popup, what it hangs from and where it went.
	pops offscreenPopups
}

// NewOffscreen allocates a CPU pixmap of the requested size.
func NewOffscreen(opts WindowOptions) *Offscreen {
	lw, lh := opts.Width, opts.Height
	if lw < 1 {
		lw = 1
	}
	if lh < 1 {
		lh = 1
	}
	scale := opts.Scale
	if scale <= 0 {
		scale = 1
	}
	o := &Offscreen{
		title: opts.Title,
		img:   paintengine2d.NewImage(DevicePixels(lw, scale), DevicePixels(lh, scale)),
		wake:  make(chan struct{}, 1),
		posX:  DevicePosition(opts.X, scale), posY: DevicePosition(opts.Y, scale),
		scale: scale,
	}
	// An offscreen desktop can carry a window under a drag, the way an
	// X11 one does: a test turns it off to take the fallback path.
	o.dragOut.carries = true
	// The requested size is the window's, in logical pixels; the pixmap is
	// that times the display scale, plus a frame's margin around it
	// (offscreen_frame.go).
	o.frame.geomLW, o.frame.geomLH = lw, lh
	o.frame.geomW, o.frame.geomH = DevicePixels(lw, scale), DevicePixels(lh, scale)
	o.opts = opts
	o.sizing = opts.Sizing
	o.limits = limitsFor(o.sizing, opts, lw, lh)
	o.frame.deco, o.frame.decoSet = requestedDecorations(opts.Decorations), true
	if opts.Popup {
		o.frame.deco = DecorationsNone
	}
	return o
}

func (o *Offscreen) Title() string                { return o.title }
func (o *Offscreen) SetTitle(title string)        { o.title = title }
func (o *Offscreen) Size() (w, h int)             { return o.img.Width, o.img.Height }
func (o *Offscreen) Buffer() *paintengine2d.Image { return o.img }
func (o *Offscreen) Closed() bool                 { return o.closed }
func (o *Offscreen) SetCursor(c Cursor)           { o.cursor = c }
func (o *Offscreen) Cursor() Cursor               { return o.cursor }

// Scale is the simulated display scale (1 unless SimulateScale set one).
func (o *Offscreen) Scale() float32 {
	if o == nil || o.scale <= 0 {
		return 1
	}
	return o.scale
}

// SimulateScale gives the offscreen desktop a display scale, so a test can
// see what a window asked for in logical pixels becomes in device ones
// without a compositor. The window keeps its logical size and the pixmap
// is rebuilt around it, which is what a real backend does when a window
// moves to a monitor of another scale.
func (o *Offscreen) SimulateScale(s float32) {
	if o == nil || s <= 0 || o.scale == s {
		return
	}
	o.scale = s
	o.frame.geomW = DevicePixels(o.frame.geomLW, s)
	o.frame.geomH = DevicePixels(o.frame.geomLH, s)
	o.resizeSurface()
}

// Resize sizes the *window*, in logical pixels — the pixmap is that times
// the display scale, plus the frame's margin, as a compositor's surface is
// (a window without a frame of the toolkit's has no margin, so the pixmap
// is only the scaled size).
func (o *Offscreen) Resize(lw, lh int) error {
	if lw < 1 {
		lw = 1
	}
	if lh < 1 {
		lh = 1
	}
	if o.frame.geomLW == lw && o.frame.geomLH == lh {
		return nil
	}
	o.frame.geomLW, o.frame.geomLH = lw, lh
	o.frame.geomW = DevicePixels(lw, o.Scale())
	o.frame.geomH = DevicePixels(lh, o.Scale())
	// A fixed window's limits are its size: the app moving it moves them.
	o.limits = limitsFor(o.sizing, o.opts, lw, lh)
	o.resizeSurface()
	o.queue = append(o.queue, Event{Kind: EventResize, Width: lw, Height: lh})
	return nil
}

func (o *Offscreen) PaintDevice() paintengine2d.Device {
	if o == nil || o.img == nil {
		return nil
	}
	return o.soft.of(o.img)
}

func (o *Offscreen) UsesGPU() bool { return false }

func (o *Offscreen) Present(dirty []paintengine2d.Rect) error {
	_ = dirty
	return nil
}

func (o *Offscreen) Raise() { o.hidden = false }
func (o *Offscreen) Show()  { o.hidden = false }
func (o *Offscreen) Hide()  { o.hidden = true }

// Move implements [HostMover]: the offscreen desktop puts the window
// where it is asked, as an X11 one does. x, y are logical pixels.
func (o *Offscreen) Move(x, y int) {
	o.posX, o.posY = DevicePosition(x, o.Scale()), DevicePosition(y, o.Scale())
}

// Position implements [HostPositioner]: where the window is on the
// offscreen desktop. It is always known — this desktop has no compositor
// keeping it a secret.
func (o *Offscreen) Position() (int, int, bool) {
	if o == nil || o.closed {
		return 0, 0, false
	}
	return LogicalPosition(o.posX, o.Scale()), LogicalPosition(o.posY, o.Scale()), true
}
func (o *Offscreen) Wake() {
	if o == nil {
		return
	}
	o.wakes.Add(1)
	if o.wake == nil {
		return
	}
	select {
	case o.wake <- struct{}{}:
	default:
	}
}

// Wakes is how many times [Offscreen.Wake] ran (tests).
func (o *Offscreen) Wakes() int {
	if o == nil {
		return 0
	}
	return int(o.wakes.Load())
}
func (o *Offscreen) Visible() bool {
	return o != nil && !o.closed && !o.hidden
}

func (o *Offscreen) Poll() []Event {
	ev := o.queue
	o.queue = nil
	return ev
}

func (o *Offscreen) Wait(timeout time.Duration) bool {
	if o != nil && len(o.queue) > 0 {
		return true
	}
	if timeout == 0 {
		return false
	}
	if timeout < 0 || timeout > 50*time.Millisecond {
		timeout = 50 * time.Millisecond
	}
	if o == nil || o.wake == nil {
		time.Sleep(timeout)
		return false
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-o.wake:
		return true
	case <-timer.C:
		return len(o.queue) > 0
	}
}

func (o *Offscreen) Close() error {
	if o.closed {
		return nil
	}
	o.closePopups()
	o.closed = true
	return nil
}

// Inject appends a synthetic event (tests and scripted screenshots). An
// event injected into a popup arrives on its root window, translated, as a
// real one does.
func (o *Offscreen) Inject(ev Event) {
	if p := o.pops.popup; p != nil && p.root != nil {
		if popupInput(ev.Kind) {
			p.root.queue = append(p.root.queue, popupEvent(ev, o.Origin()))
		}
		return
	}
	o.queue = append(o.queue, ev)
}

// OffscreenBackend always succeeds and never talks to a display.
type OffscreenBackend struct{}

func (OffscreenBackend) Name() string { return "offscreen" }

func (OffscreenBackend) NewSurface(opts WindowOptions) (Surface, error) {
	return NewOffscreen(opts), nil
}
