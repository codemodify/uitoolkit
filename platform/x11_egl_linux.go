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
	})
	if err != nil {
		return
	}
	s.gpu = dev
}

func (s *x11Surface) closeGPU() {
	if s == nil || s.gpu == nil {
		return
	}
	_ = s.gpu.Close()
	s.gpu = nil
}

func (s *x11Surface) resizeGPU(w, h int) {
	if s == nil || s.gpu == nil {
		return
	}
	if err := s.gpu.Resize(w, h); err != nil {
		s.closeGPU()
	}
}

func (s *x11Surface) presentGPU(dirty []paintengine2d.Rect) error {
	if s == nil || s.gpu == nil {
		return paintengine2d.ErrGPUUnavailable
	}
	if p, ok := any(s.gpu).(interface {
		PresentRects([]paintengine2d.Rect) error
	}); ok && len(dirty) > 0 {
		return p.PresentRects(dirty)
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
		return paintengine2d.NewCPUDevice(s.img)
	}
	return nil
}
