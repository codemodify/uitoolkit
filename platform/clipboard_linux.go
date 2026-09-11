//go:build linux && cgo

package platform

import "os"

func clipboardNativeSet(s string) {
	wl := waylandLive() || (os.Getenv("WAYLAND_DISPLAY") != "" && waylandProbe())
	x11 := x11Live() || os.Getenv("DISPLAY") != ""
	if wl {
		wlClipSet(s)
	}
	if x11 {
		x11ClipSet(s)
	}
}

func clipboardNativeGet() (string, bool) {
	return clipboardReadNative(false)
}

func clipboardNativePrimaryGet() (string, bool) {
	return clipboardReadNative(true)
}

func clipboardReadNative(primary bool) (string, bool) {
	if waylandLive() || (os.Getenv("WAYLAND_DISPLAY") != "" && waylandProbe()) {
		if s, ok := wlClipGet(primary); ok {
			return s, true
		}
	}
	if os.Getenv("DISPLAY") != "" {
		return x11ClipGet(primary)
	}
	return "", false
}
