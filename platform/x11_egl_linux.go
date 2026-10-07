//go:build linux && cgo

package platform

import (
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

func (s *x11Surface) tryBindGPU() {
	if s == nil || s.gpu != nil || s.conn == nil || s.conn.dpy == nil || s.win == 0 {
		return
	}
	if !WantGPU() {
		return
	}
	w, h := 1, 1
	if s.img != nil {
		w, h = s.img.Width, s.img.Height
	}
	dev, err := paintengine2d.NewGPUDeviceEGL(paintengine2d.EGLNative{
		Display:  uintptr(unsafe.Pointer(s.conn.dpy)),
		Window:   uintptr(s.win),
		Platform: paintengine2d.EGLPlatformX11,
		Width:    w,
		Height:   h,
		// A window on a 32-bit visual must be drawn through a config with
		// an alpha channel, or the buffers it swaps have no alpha for the
		// compositor to read and the window shows through everywhere.
		Alpha: s.argb(),
	})
	if err != nil {
		return
	}
	s.gpu = dev
	s.notePaintDevice()
}

// notePaintDevice queues [EventPaintDevice] when the window's paint device
// really changed, so an application holding a texture on it hears before it
// draws with a name that means nothing.
//
// Called from every path that can replace or drop the device. It is cheap and
// it is idempotent — the notice only fires on a change — so a path that calls
// it needlessly costs nothing, and one that forgets is the bug.
func (s *x11Surface) notePaintDevice() {
	if s == nil || s.conn == nil {
		return
	}
	var dev paintengine2d.Device
	if s.gpu != nil {
		// Not `dev = s.gpu` unconditionally: a nil *GPUDevice in an interface
		// is not a nil interface, and the notice compares identities.
		dev = s.gpu
	}
	if ev := s.paintNotice.paintDeviceEvent(dev); len(ev) > 0 {
		s.conn.queues[s.win] = append(s.conn.queues[s.win], ev...)
	}
}

func (s *x11Surface) closeGPU() {
	if s == nil || s.gpu == nil {
		return
	}
	_ = s.gpu.Close()
	s.gpu = nil
	s.notePaintDevice()
}

func (s *x11Surface) resizeGPU(w, h int) {
	if s == nil || s.gpu == nil {
		return
	}
	if err := s.gpu.Resize(w, h); err != nil {
		s.closeGPU()
	}
}

func (s *x11Surface) presentGPU() error {
	if s == nil || s.gpu == nil {
		return paintengine2d.ErrGPUUnavailable
	}
	return s.gpu.Present()
}

func (s *x11Surface) abandonGPU() {
	if s == nil || s.gpu == nil {
		return
	}
	if snap := s.gpu.Image(); snap != nil {
		s.img = snap.Clone()
	}
	s.closeGPU()
}

func (s *x11Surface) UsesGPU() bool { return s != nil && s.gpu != nil }

func (s *x11Surface) PaintDevice() paintengine2d.Device {
	if s != nil && s.gpu != nil {
		return s.gpu
	}
	if s != nil && s.img != nil {
		return s.soft.of(s.img)
	}
	return nil
}
