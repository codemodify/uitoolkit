package platform

import (
	"os"
	"strings"
	"sync/atomic"

	"github.com/codemodify/paintengine2d"
)

// EnvPaint is UITK_PAINT — the paintengine2d cpu|gpu|auto selector.
const EnvPaint = paintengine2d.EnvPaint

// savedPaintPref is the preference an application applied from its
// preferences file (look.json "renderer"), normalized, or "" for none.
// UITK_PAINT wins over it; see [PaintPref].
var savedPaintPref atomic.Value // string

// SetPaintPref remembers the cpu|gpu|auto preference the application read
// out of its preferences file. An empty value (or "auto") forgets it.
//
// It changes nothing that is already on screen: a surface binds its paint
// device when it is created, so this decides what the *next* window gets.
// app.New pushes look.json's choice into it and
// app.Application.ApplyAppearance replaces it; an application that wants
// to choose for itself calls app.Application.SetRenderer after New,
// which goes through here.
func SetPaintPref(pref string) {
	savedPaintPref.Store(normalPaintPref(pref))
}

// PaintPrefEnv is the UITK_PAINT value when the environment sets one:
// the normalized choice and whether it was there at all. It is the one
// thing that beats [SetPaintPref], the way UITK_THEME beats the saved
// theme, so a chooser in a settings page can say that the environment is
// in charge here.
func PaintPrefEnv() (string, bool) {
	s := strings.TrimSpace(os.Getenv(EnvPaint))
	if s == "" {
		return "", false
	}
	return paintengine2d.ParsePaintPref(s), true
}

// PaintPref returns the normalized paint choice (cpu, gpu, or auto):
// UITK_PAINT when it is set, otherwise whatever [SetPaintPref] was last
// given, otherwise auto — try GPU, fall back to CPU.
//
// It is read when a surface binds its device, not once at start-up, so a
// preference applied while an application runs reaches the windows it
// opens afterwards and leaves the ones already open alone.
func PaintPref() string {
	if pref, ok := PaintPrefEnv(); ok {
		return pref
	}
	if saved, _ := savedPaintPref.Load().(string); saved != "" {
		return saved
	}
	return paintengine2d.PaintAuto
}

// normalPaintPref is ParsePaintPref with an empty value left empty:
// paintengine2d reads anything it does not know as cpu, and "nobody has
// said" is not "somebody said CPU".
func normalPaintPref(pref string) string {
	if strings.TrimSpace(pref) == "" {
		return ""
	}
	return paintengine2d.ParsePaintPref(pref)
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

// SurfaceBackend is what s is really painting through — "gpu" or "cpu" —
// which is not the same question as [PaintPref]. A preference of gpu or
// auto whose EGL never started is a window on the CPU, and the honest
// answer to "what am I getting" is this one.
func SurfaceBackend(s Surface) string {
	if SurfaceUsesGPU(s) {
		return paintengine2d.PaintGPU
	}
	return paintengine2d.PaintCPU
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
