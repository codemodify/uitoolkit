//go:build linux && cgo

package platform

import "os"

func x11Available() Backend {
	if os.Getenv("DISPLAY") == "" {
		return nil
	}
	return x11Backend{}
}

func waylandBackend() Backend {
	if !waylandProbe() {
		return nil
	}
	return wlBackend{}
}

// win32Available is nil on Linux: Win32 is a Windows backend.
func win32Available() Backend { return nil }

func appkitAvailable() Backend { return nil }
