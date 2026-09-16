//go:build linux && cgo

package platform

/*
#cgo linux pkg-config: wayland-egl wayland-client egl
#include <wayland-client.h>
#include <wayland-egl.h>
#include <EGL/egl.h>

static struct wl_egl_window *ui_wl_egl_create(struct wl_surface *s, int w, int h) {
	if (!s || w < 1 || h < 1) return NULL;
	return wl_egl_window_create(s, w, h);
}

// eglSwapBuffers with the default swap interval of 1 blocks inside Mesa
// until the compositor sends a frame callback. An occluded or throttled
// window then stalls the whole run loop -- tray, timers and posted work
// included. Interval 0 returns immediately; pacing is done with
// wl_surface.frame instead.
static void ui_wl_egl_no_vsync(uintptr_t dpy) {
	if (!dpy) return;
	eglSwapInterval((EGLDisplay)dpy, 0);
}
static void ui_wl_egl_resize(struct wl_egl_window *win, int w, int h) {
	if (win) wl_egl_window_resize(win, w, h, 0, 0);
}
static void ui_wl_egl_destroy(struct wl_egl_window *win) {
	if (win) wl_egl_window_destroy(win);
}
*/
import "C"

import (
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

func (s *wlSurface) tryBindGPU() {
	if s == nil || s.gpu != nil || s.surf == nil || s.conn == nil || s.conn.dpy == nil {
		return
	}
	if !WantGPU() {
		return
	}
	bw, bh := s.bufferWH()
	win := C.ui_wl_egl_create(s.surf, C.int(bw), C.int(bh))
	if win == nil {
		return
	}
	dev, err := paintengine2d.NewGPUDeviceEGL(paintengine2d.EGLNative{
		Display:  uintptr(unsafe.Pointer(s.conn.dpy)),
		Window:   uintptr(unsafe.Pointer(win)),
		Platform: paintengine2d.EGLPlatformWayland,
		Width:    bw,
		Height:   bh,
		// An alpha visual only where the frame needs one: an opaque window
		// on an ARGB config that ever left alpha short of 1 would present
		// see-through.
		Alpha: s.frame.Alpha,
	})
	if err != nil {
		C.ui_wl_egl_destroy(win)
		return
	}
	s.eglWin = unsafe.Pointer(win)
	s.gpu = dev
	s.gpuAlpha = s.frame.Alpha
	// The buffer is this size now: a configure that arrived since the
	// surface was made (configure_bounds, a compositor's size) changed it,
	// and a present that thought otherwise never resized the EGL window.
	s.bufW, s.bufH = bw, bh
	if egl, _, _ := dev.EGLHandles(); egl != 0 {
		C.ui_wl_egl_no_vsync(C.uintptr_t(egl))
	}
}

// rebindGPUAlpha swaps the GPU device for one whose EGL config matches the
// frame's need for an alpha channel. The new device is made before the old
// one goes, so the EGL display stays initialised (tearing it down and back
// up costs a driver reload); the app repaints everything after a frame
// change anyway, so nothing is copied over.
func (s *wlSurface) rebindGPUAlpha(alpha bool) {
	if s == nil || s.gpu == nil || s.gpuAlpha == alpha {
		return
	}
	old, oldWin := s.gpu, s.eglWin
	s.gpu, s.eglWin = nil, nil
	s.tryBindGPU()
	if s.gpu == nil {
		// Nothing better to fall back on: keep the device we had.
		s.gpu, s.eglWin = old, oldWin
		return
	}
	_ = old.Close()
	if oldWin != nil {
		C.ui_wl_egl_destroy((*C.struct_wl_egl_window)(oldWin))
	}
}

func (s *wlSurface) closeGPU() {
	if s == nil {
		return
	}
	if s.gpu != nil {
		_ = s.gpu.Close()
		s.gpu = nil
	}
	if s.eglWin != nil {
		C.ui_wl_egl_destroy((*C.struct_wl_egl_window)(s.eglWin))
		s.eglWin = nil
	}
}

func (s *wlSurface) resizeGPU(w, h int) {
	if s == nil || s.gpu == nil {
		return
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if s.eglWin != nil {
		C.ui_wl_egl_resize((*C.struct_wl_egl_window)(s.eglWin), C.int(w), C.int(h))
	}
	if err := s.gpu.Resize(w, h); err != nil {
		s.closeGPU()
	}
}

func (s *wlSurface) presentGPU() error {
	if s == nil || s.gpu == nil {
		return paintengine2d.ErrGPUUnavailable
	}
	return s.gpu.Present()
}

func (s *wlSurface) abandonGPU() {
	if s == nil || s.gpu == nil {
		return
	}
	if snap := s.gpu.Image(); snap != nil {
		s.img = snap.Clone()
	}
	s.closeGPU()
	if s.img == nil {
		// The GPU painted the window and left no frame behind: the shm
		// path needs a pixmap of the buffer's size.
		s.img = paintengine2d.NewImage(max(s.bufW, 1), max(s.bufH, 1))
	}
}

func (s *wlSurface) UsesGPU() bool { return s != nil && s.gpu != nil }

func (s *wlSurface) PaintDevice() paintengine2d.Device {
	if s != nil && s.gpu != nil {
		return s.gpu
	}
	if s != nil && s.img != nil {
		return s.soft.of(s.img)
	}
	return nil
}
