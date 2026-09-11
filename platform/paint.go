package platform

import "github.com/codemodify/paintengine2d"

// EnvPaint is UITK_PAINT — the paintengine2d cpu|gpu|auto selector.
const EnvPaint = paintengine2d.EnvPaint

// PaintPref returns the normalized UITK_PAINT choice (cpu, gpu, or auto).
// An unset or empty value is auto: try GPU, fall back to CPU.
func PaintPref() string {
	return paintengine2d.EnvPaintPref()
}

// WantGPU reports whether the toolkit should try an EGL/GLES paint device.
func WantGPU() bool {
	switch PaintPref() {
	case paintengine2d.PaintGPU, paintengine2d.PaintAuto:
		return true
	default:
		return false
	}
}

type paintDevicer interface {
	PaintDevice() paintengine2d.Device
}

type gpuSurface interface {
	UsesGPU() bool
}

// NewPaintContext draws through the surface's live Device: GPUDevice when
// an EGL window is bound, otherwise the CPU pixmap from Buffer.
func NewPaintContext(s Surface) *paintengine2d.Context {
	if s == nil {
		return nil
	}
	if p, ok := s.(paintDevicer); ok {
		if d := p.PaintDevice(); d != nil {
			return paintengine2d.NewContextDevice(d)
		}
	}
	img := s.Buffer()
	if img == nil {
		return nil
	}
	return paintengine2d.NewContext(img)
}

// SurfaceUsesGPU reports whether s presents through paintengine2d.GPUDevice
// (wl_egl_window / X11 EGL + eglSwapBuffers).
func SurfaceUsesGPU(s Surface) bool {
	g, ok := s.(gpuSurface)
	return ok && g.UsesGPU()
}
