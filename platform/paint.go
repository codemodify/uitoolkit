package platform

import (
	"os"
	"strings"

	"github.com/codemodify/paintengine2d"
)

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

// softDevice caches the software devices of a surface's buffers. A device
// owns the rasterizer's scratch (edges, coverage rows, stroke outlines);
// making a new one per frame reallocated all of it every frame. Two slots
// cover a double-buffered shm surface.
type softDevice struct {
	img [2]*paintengine2d.Image
	dev [2]*paintengine2d.CPUDevice
}

// of is the device painting img, reused while img is.
func (c *softDevice) of(img *paintengine2d.Image) *paintengine2d.CPUDevice {
	if img == nil {
		return nil
	}
	for i := range c.img {
		if c.img[i] == img && c.dev[i] != nil {
			if i == 1 {
				c.img[0], c.img[1] = c.img[1], c.img[0]
				c.dev[0], c.dev[1] = c.dev[1], c.dev[0]
			}
			return c.dev[0]
		}
	}
	c.img[1], c.dev[1] = c.img[0], c.dev[0]
	c.img[0], c.dev[0] = img, paintengine2d.NewCPUDevice(img)
	return c.dev[0]
}

// SurfaceUsesGPU reports whether s presents through paintengine2d.GPUDevice
// (wl_egl_window / X11 EGL + eglSwapBuffers).
func SurfaceUsesGPU(s Surface) bool {
	g, ok := s.(gpuSurface)
	return ok && g.UsesGPU()
}

// SurfaceDevice is the live paint [paintengine2d.Device] for s.
func SurfaceDevice(s Surface) paintengine2d.Device {
	if s == nil {
		return nil
	}
	if p, ok := s.(paintDevicer); ok {
		if d := p.PaintDevice(); d != nil {
			return d
		}
	}
	img := s.Buffer()
	if img == nil {
		return nil
	}
	return paintengine2d.NewCPUDevice(img)
}

// EnvScene is UITK_SCENE — retained graph (default) or immediate paint.
const EnvScene = "UITK_SCENE"

// WantScene reports whether windows should record a retained scene
// (Qt Quick / GSK model). UITK_SCENE=off|0|immediate uses the v0.5
// immediate Fill path.
func WantScene() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvScene))) {
	case "0", "off", "false", "immediate":
		return false
	default:
		return true
	}
}
