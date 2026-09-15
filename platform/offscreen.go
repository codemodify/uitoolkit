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
	// drop is a simulated drop's data by type (SimulateDrop).
	drop      map[string][]byte
	dropTaken *bool
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
	return &Offscreen{
		title: opts.Title,
		img:   paintengine2d.NewImage(w, h),
		wake:  make(chan struct{}, 1),
	}
}

func (o *Offscreen) Title() string                { return o.title }
func (o *Offscreen) SetTitle(title string)        { o.title = title }
func (o *Offscreen) Size() (w, h int)             { return o.img.Width, o.img.Height }
func (o *Offscreen) Buffer() *paintengine2d.Image { return o.img }
func (o *Offscreen) Closed() bool                 { return o.closed }
func (o *Offscreen) Scale() float32               { return 1 }
func (o *Offscreen) SetCursor(c Cursor)           { o.cursor = c }
func (o *Offscreen) Cursor() Cursor               { return o.cursor }

func (o *Offscreen) Resize(w, h int) error {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if o.img.Width == w && o.img.Height == h {
		return nil
	}
	o.img = paintengine2d.NewImage(w, h)
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
func (o *Offscreen) Move(x, y int) {
	_, _ = x, y
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
