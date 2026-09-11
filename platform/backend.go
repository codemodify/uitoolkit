package platform

import "os"

// Default returns the native window backend when a display is available,
// otherwise offscreen. Headless options always use offscreen.
func Default(headless bool) Backend {
	if headless {
		return OffscreenBackend{}
	}
	if native := nativeBackend(); native != nil {
		return native
	}
	return OffscreenBackend{}
}

func hasDisplay() bool {
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}
