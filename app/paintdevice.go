package app

import (
	"github.com/codemodify/paintengine2d"

	"github.com/codemodify/uitoolkit/platform"
)

// OnPaintDeviceChange is called when the window's paint device is replaced or
// goes away, after it has already changed — so the window answers for the new
// one by the time the callback runs.
//
// A window's device is not forever. It is closed and made again when the
// window changes visual (an X11 window that gains a shadow has to be
// re-created on a 32-bit visual), when a resize of it fails, and when the GPU
// is lost mid-run and the window falls back to painting on the CPU. Every one
// of those paths used to drop it in silence.
//
// For most applications that is nothing: the toolkit repaints the window and
// the widgets draw again. It matters for one that holds something *on* the
// device — a [paintengine2d.ForeignTexture] it renders video into, a texture
// it uploaded itself. Those name objects in a GL context that has gone, and
// the application has to make them again against whatever
// [Window.PaintDevice] now answers, or stop drawing them if it answers a CPU
// device.
//
// It is one callback per window, like [Window.OnMove] and
// [Window.OnLockKeys]: the window's own code, not a subscription. It runs on
// the UI goroutine, from the window's event dispatch.
func (w *Window) OnPaintDeviceChange(fn func()) {
	if w == nil {
		return
	}
	w.onPaintDevice = fn
}

// PaintDevice is what this window is painting through now — a GPU device where
// it has one, otherwise a CPU device over its pixmap.
//
// Ask it again after [Window.OnPaintDeviceChange] fires; the answer before and
// after are different objects, and anything built on the old one is void.
// [Window.UsesGPU] is the shorter question when all that matters is whether
// there is a GPU at all.
func (w *Window) PaintDevice() paintengine2d.Device {
	if w == nil || w.surf == nil {
		return nil
	}
	return platform.SurfaceDevice(w.surf)
}

// UsesGPU reports whether this window is painting through a GPU device.
//
// It is the question to ask before building a renderer that needs one: a
// window on the CPU — a remote X session, no EGL, UITK_PAINT=cpu, or a GPU
// that was lost and fell back — cannot take a foreign texture, so an
// application that offers a GPU path needs its CPU one too.
func (w *Window) UsesGPU() bool {
	return w != nil && w.surf != nil && platform.SurfaceUsesGPU(w.surf)
}
