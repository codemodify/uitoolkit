//go:build linux && cgo

package platform

import "os"

func x11Available() Backend {
	if os.Getenv("DISPLAY") == "" {
		return nil
	}
	return X11Backend{}
}

func waylandBackend() Backend {
	if !waylandProbe() {
		return nil
	}
	return WaylandBackend{}
}
