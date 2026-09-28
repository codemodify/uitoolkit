//go:build darwin && cgo

package platform

// On macOS there is one window system and it is always there.
func x11Available() Backend   { return nil }
func waylandBackend() Backend { return nil }
func win32Available() Backend { return nil }

// appkitAvailable is the AppKit backend. A macOS process that can make a
// window always can, so there is nothing to probe — the same reasoning
// as Win32, and unlike X11 and Wayland which need a display to connect to.
func appkitAvailable() Backend { return AppKitBackend{} }

// nativeDetectScale is 0: the scale is per *window* on macOS, from the
// screen it is on (NSWindow.backingScaleFactor), and there is no one
// answer for the process.
func nativeDetectScale() float32 { return 0 }
