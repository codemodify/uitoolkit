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

// OnPaintDeviceClosing is called immediately *before* this window's GPU
// device is closed, while its context is still current.
//
// [Window.OnPaintDeviceChange] is the other half and says what happened;
// this one is the moment to do something about it. An application that
// holds a renderer on the device — libmpv's, whose render API requires
// the context it was created against to be current when the renderer is
// destroyed — has nowhere else to let go: every backend closed the
// device and only then queued the change, so by the time anything heard,
// the context it needed was gone.
//
// It runs on the UI goroutine, from inside the backend's teardown, so it
// must do its own cleanup and return. It must not resize or repaint the
// window, and what it releases it has to make again when
// OnPaintDeviceChange reports whatever [Window.PaintDevice] now answers.
//
// It is not called for a device that is lost rather than closed — a GPU
// that went away mid-run takes its context with it and there is no
// moment to run in. OnPaintDeviceChange still reports that, which is why
// an application needs both.
func (w *Window) OnPaintDeviceClosing(fn func()) {
	if w == nil {
		return
	}
	w.onDeviceClosing = fn
}

// deviceClosing runs the window's before-close callback.
func (w *Window) deviceClosing() {
	if w != nil && w.onDeviceClosing != nil {
		w.onDeviceClosing()
	}
}
