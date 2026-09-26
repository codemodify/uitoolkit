package app

import (
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// RendererEnv is UITK_PAINT, the variable that overrides the saved
// renderer wherever it is set — as UITK_THEME overrides the saved theme.
// The file is never rewritten to match it.
const RendererEnv = platform.EnvPaint

// RendererOverride is the UITK_PAINT choice when the environment sets
// one, and whether it does. While it does, [Application.SetRenderer]
// still records what was applied and still writes nothing to the screen:
// the environment is what new surfaces follow.
func RendererOverride() (style.RendererPref, bool) {
	pref, ok := platform.PaintPrefEnv()
	if !ok {
		return style.RendererAuto, false
	}
	return style.ParseRendererPref(pref), true
}

// SetRenderer asks for a paint device for the windows opened from now
// on: [style.RendererAuto] (EGL/GLES where it starts, the CPU rasterizer
// where it does not), [style.RendererGPU] or [style.RendererCPU].
//
// It changes nothing that is on screen. A surface binds its device when
// it is created — wl_egl_window / EGL for the GPU, wl_shm or XPutImage
// for the CPU — and swapping that under a mapped window means tearing
// the context down and building it again, with a frame of nothing in
// between. So the windows already open keep what they have, and the
// application says so rather than looking inert:
// cmd/uitoolkit-settings/settingsapp puts it on the chooser.
//
// UITK_PAINT, if it is set, goes on winning over this.
func (a *Application) SetRenderer(p style.RendererPref) {
	if a == nil {
		return
	}
	a.renderPref = style.ParseRendererPref(string(p))
	platform.SetPaintPref(string(a.renderPref))
}

// Renderer is the paint device this application asks new windows for:
// what look.json said at start-up, or what [Application.SetRenderer]
// applied since. It is a request, not an answer — [Application.PaintBackend]
// is what the windows actually got.
func (a *Application) Renderer() style.RendererPref {
	if a == nil {
		return style.RendererAuto
	}
	return a.renderPref
}

// PaintBackend is what this application is really painting through right
// now: [style.RendererGPU] if any of its open windows presents through
// EGL/GLES, [style.RendererCPU] otherwise — and CPU is the honest answer
// for a process whose EGL never started, whatever it asked for.
//
// An application with no window open yet has nothing to have painted
// with, and reports the CPU: nothing has been presented.
func (a *Application) PaintBackend() style.RendererPref {
	if a == nil {
		return style.RendererCPU
	}
	for _, w := range a.Windows() {
		if w.PaintBackend() == style.RendererGPU {
			return style.RendererGPU
		}
	}
	return style.RendererCPU
}

// PaintBackend is what this one window presents through — the GPU only
// if its surface bound an EGL device. Two windows of one application can
// differ: the preference is read when each surface is created, and a
// window opened before it changed keeps what it had.
func (w *Window) PaintBackend() style.RendererPref {
	if w == nil || w.surf == nil {
		return style.RendererCPU
	}
	return style.ParseRendererPref(platform.SurfaceBackend(w.surf))
}
