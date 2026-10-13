package platform

import (
	"sync"

	"github.com/codemodify/paintengine2d"
)

// A window's paint device is not forever.
//
// It is closed and made again when the window changes visual (an X11 window
// that gains a shadow is re-created on a 32-bit visual), when a resize of it
// fails, and when the GPU is lost mid-run and the window falls back to the
// CPU. Every one of those paths used to drop the device in silence.
//
// That is only a leak of information until an application holds something *on*
// the device — a texture it renders video into — and then it is a dangling GL
// name that nothing told anybody about.

// paintDeviceNotice tracks which device a surface last reported, so that
// [EventPaintDevice] is queued when it really changed and not every time a
// backend walks past one of these paths.
//
// It is a value, embedded in each surface, like taskbarPolicy: the rule is one
// rule and three backends should not each have their own version of it.
type paintDeviceNotice struct {
	// last is the device the surface last told anyone about. A surface that
	// has told nobody anything yet reports its first device, which is what an
	// application that asks on its first frame already has — so `seen` keeps
	// the first report from being an event.
	last paintengine2d.Device
	seen bool
}

// changed records dev as the surface's device and reports whether that is news
// worth an event.
//
// The first device is not news: an application reads [SurfaceDevice] when it
// starts and gets that one. Afterwards every change is, including the change
// to nothing — a window that fell back to the CPU has no GPU device, and an
// application holding a texture needs to hear that most of all.
func (n *paintDeviceNotice) changed(dev paintengine2d.Device) bool {
	if !n.seen {
		// The device a surface is born with. An application asks
		// [SurfaceDevice] when it starts and gets exactly this one, so
		// reporting it would be an event about nothing — and would have the
		// application throw away and rebuild a target it had only just made.
		n.last, n.seen = dev, true
		return false
	}
	if dev == n.last {
		return false
	}
	n.last = dev
	return true
}

// paintDeviceEvent is the event, or nothing when the device is the same one.
func (n *paintDeviceNotice) paintDeviceEvent(dev paintengine2d.Device) []Event {
	if !n.changed(dev) {
		return nil
	}
	return []Event{{Kind: EventPaintDevice}}
}

// closingGPU is what to run, on the UI goroutine and with the old
// context still current, immediately before a surface's GPU device is
// closed.
//
// After-the-fact notice is not enough for an application that holds a
// renderer *on* the device. libmpv's render API requires the context it
// was created against to be current when its renderer is destroyed, and
// every backend's closeGPU closes the device before queuing
// [EventPaintDevice] — so by the time anything heard, the context the
// application needed was gone. It had no moment in which to let go.
//
// It is a package-level hook rather than a field on each surface for the
// same reason [SetLoopPoster] is: the app package owns the one UI
// goroutine and knows which window a surface belongs to, and three
// backends should not each grow their own callback plumbing.
var closingGPU func(Surface)

// SetGPUClosing registers that hook, and nil clears it. It is the
// process-wide one, and a window's own ([SetSurfaceGPUClosing]) is
// preferred over it.
func SetGPUClosing(fn func(Surface)) { closingGPU = fn }

// GPUClosing is the hook installed now, for a program that installs one
// of its own and wants to chain to what was there.
func GPUClosing() func(Surface) { return closingGPU }

// surfaceClosing is the callback each surface's owner registered, which
// is where this really belongs.
//
// The process-wide hook above cannot answer for more than one owner: the
// app package installed it when an [app.Application] was made, so
// constructing a second one took the first one's windows off it and
// their closing callbacks stopped firing. media-player-music reproduced
// that against real GPU windows — v0.23.6 answered 1,1 and v0.23.7
// answered 1,0 — and it is the whole reason this map exists. A surface
// has exactly one owner for as long as it lives, so that is what the
// entry is keyed by.
var surfaceClosing sync.Map // Surface -> func()

// SetSurfaceGPUClosing registers what to run immediately before s's GPU
// device is closed, with that device's context current; nil forgets it,
// which a window does when it closes.
func SetSurfaceGPUClosing(s Surface, fn func()) {
	if s == nil {
		return
	}
	if fn == nil {
		surfaceClosing.Delete(s)
		return
	}
	surfaceClosing.Store(s, fn)
}

// NotifyGPUClosing runs the closing callback for s: the surface's own
// ([SetSurfaceGPUClosing]) where it has one, else the process-wide hook.
//
// A backend calls this immediately before it closes a GPU device, with
// that device's context current, and never after. It is exported
// because it is the published half of the seam — [SetSurfaceGPUClosing]
// registers, this delivers — and because the layers above it are then
// testable on a machine with no GPU.
func NotifyGPUClosing(s Surface) { noteGPUClosing(s) }

// noteGPUClosing runs the hook for s. A backend calls it as the last
// thing before it closes a GPU device, never after.
func noteGPUClosing(s Surface) {
	if s == nil {
		return
	}
	if fn, ok := surfaceClosing.Load(s); ok {
		if f, _ := fn.(func()); f != nil {
			f()
			return
		}
	}
	if closingGPU != nil {
		closingGPU(s)
	}
}

// gpuCloser is what a backend's GPU device offers here: closing it, and
// — for a device that has a context to bind — making that context
// current.
type gpuCloser interface{ Close() error }

type gpuBinder interface{ MakeCurrent() error }

// closeGPUDevice binds the device, tells the application and then closes
// it, in that order — which is the whole of this: an application holding
// a renderer on the device gets its moment while the device's own
// context is current.
//
// **The binding is not a formality.** The hook promises the closing
// device's context, and a process with two GPU windows had whatever was
// current from the last thing painted: media-player-video reproduced the
// callback running with the *other* window's context, which is exactly
// the case libmpv's render API refuses — its renderer must be destroyed
// against the context it was made against. Binding first costs one
// eglMakeCurrent on a path that happens once per window.
//
// Both backends go through it so the order is written once. Doing it by
// hand in each was how the order came to be wrong in the first place,
// and nothing about a hand-written sequence can be tested without a GPU.
func closeGPUDevice(s Surface, dev gpuCloser) {
	if b, ok := dev.(gpuBinder); ok {
		// A device that cannot be made current is a device whose context
		// has already gone; the application still gets its notice.
		_ = b.MakeCurrent()
	}
	noteGPUClosing(s)
	if dev != nil {
		_ = dev.Close()
	}
}
