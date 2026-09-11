//go:build linux && cgo

package platform

import "os"

func x11Available() Backend {
	if os.Getenv("DISPLAY") == "" {
		return nil
	}
	return X11Backend{}
}

// waylandBackend is provided by wayland_linux.go when the Wayland CGO
// backend is compiled; this stub keeps X11+offscreen working until then.
func waylandBackend() Backend { return nil }
