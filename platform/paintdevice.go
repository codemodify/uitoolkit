package platform

import "github.com/codemodify/paintengine2d"

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
