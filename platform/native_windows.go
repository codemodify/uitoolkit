//go:build windows

package platform

// Windows has no X11 and no Wayland; it has Win32, which is a backend of
// its own rather than one of those two answering differently.
func x11Available() Backend   { return nil }
func waylandBackend() Backend { return nil }

// win32Available is the Win32 backend, or nil where a window cannot be
// made at all (a service with no window station).
func win32Available() Backend { return Win32Backend{} }

// nativeDetectScale is 0 here: the scale is per *window* on Windows
// (GetDpiForWindow), not per process, so there is no one number to
// detect. A window answers with its own, and EventScale says when it
// changes — which on a multi-monitor desktop it does.
func nativeDetectScale() float32 { return 0 }

func appkitAvailable() Backend { return nil }
