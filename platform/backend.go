package platform

import (
	"os"
	"strings"
)

// Default returns the native window backend when a display is available,
// otherwise offscreen. Headless options always use offscreen.
//
// Auto-select (when UITK_BACKEND is unset):
//  1. Wayland if WAYLAND_DISPLAY is set and a Wayland backend exists
//  2. X11 if DISPLAY is set and CGO+Linux
//  3. offscreen
//
// UITK_BACKEND=x11|wayland|offscreen overrides the auto pick. A missing
// backend falls through so offscreen and X11 keep working when Wayland
// is requested but not built.
func Default(headless bool) Backend {
	if headless {
		return OffscreenBackend{}
	}
	if name := strings.TrimSpace(os.Getenv("UITK_BACKEND")); name != "" {
		return Select(name, false)
	}
	return autoBackend()
}

// Select picks a named backend. Unknown or unavailable names fall back
// to Default (or offscreen when headless).
func Select(name string, headless bool) Backend {
	if headless {
		return OffscreenBackend{}
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "offscreen":
		return OffscreenBackend{}
	case "x11":
		if b := x11Available(); b != nil {
			return b
		}
		return OffscreenBackend{}
	case "wayland":
		if b := waylandBackend(); b != nil {
			return b
		}
		return autoBackend()
	case "", "auto":
		return autoBackend()
	}
	return autoBackend()
}

func autoBackend() Backend {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if b := waylandBackend(); b != nil {
			return b
		}
	}
	if os.Getenv("DISPLAY") != "" {
		if b := x11Available(); b != nil {
			return b
		}
	}
	return OffscreenBackend{}
}
