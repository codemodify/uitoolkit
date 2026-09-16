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
	posX, posY int
	// drop is a simulated drop's data by type (SimulateDrop); dragOut is
	// a drag started *from* this surface (offscreen_drag.go).
	drop      map[string][]byte
	dropTaken *bool
	dragOut   offscreenDrag
	// frame is the simulated desktop behind FrameSurface (offscreen_frame.go).
	frame offscreenFrame
}

// NewOffscreen allocates a CPU pixmap of the requested size.
func NewOffscreen(opts WindowOptions) *Offscreen {
	w, h := opts.Width, opts.Height
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	o := &Offscreen{
		title: opts.Title,
		img:   paintengine2d.NewImage(w, h),
		wake:  make(chan struct{}, 1),
		posX:  opts.X, posY: opts.Y,
	}
	// An offscreen desktop can carry a window under a drag, the way an
	// X11 one does: a test turns it off to take the fallback path.
	o.dragOut.carries = true
	// The requested size is the window's; a frame's margin is added to the
	// pixmap around it (offscreen_frame.go).
	o.frame.geomW, o.frame.geomH = w, h
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
func (o *Offscreen) Scale() float32               { return 1 }
func (o *Offscreen) SetCursor(c Cursor)           { o.cursor = c }
func (o *Offscreen) Cursor() Cursor               { return o.cursor }

// Resize sizes the *window* — the pixmap is that plus the frame's margin,
// as a compositor's surface is (a window without a frame of the toolkit's
// has no margin, so the two are the same).
func (o *Offscreen) Resize(w, h int) error {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if o.frame.geomW == w && o.frame.geomH == h {
		return nil
	}
	o.frame.geomW, o.frame.geomH = w, h
	o.resizeSurface()
	o.queue = append(o.queue, Event{Kind: EventResize, Width: w, Height: h})
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
// where it is asked, as an X11 one does.
func (o *Offscreen) Move(x, y int) { o.posX, o.posY = x, y }

// Position implements [HostPositioner]: where the window is on the
// offscreen desktop. It is always known — this desktop has no compositor
// keeping it a secret.
func (o *Offscreen) Position() (int, int, bool) {
	if o == nil || o.closed {
		return 0, 0, false
	}
	return o.posX, o.posY, true
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
	o.closed = true
	return nil
}

// Inject appends a synthetic event (tests and scripted screenshots).
func (o *Offscreen) Inject(ev Event) { o.queue = append(o.queue, ev) }

// OffscreenBackend always succeeds and never talks to a display.
type OffscreenBackend struct{}

func (OffscreenBackend) Name() string { return "offscreen" }

func (OffscreenBackend) NewSurface(opts WindowOptions) (Surface, error) {
	return NewOffscreen(opts), nil
}
