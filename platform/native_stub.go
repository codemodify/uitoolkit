//go:build !linux || !cgo

package platform

// Win32 / AppKit / Wayland-without-CGO stay stubs. Apps still run
// headless / offscreen. nativeDetectScale is 0 so DetectScale uses env or 1.
func x11Available() Backend      { return nil }
func waylandBackend() Backend    { return nil }
func nativeDetectScale() float32 { return 0 }
