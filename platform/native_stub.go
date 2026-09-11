//go:build !linux || !cgo

package platform

// nativeBackend is nil on this GOOS / CGO combination (Wayland / Win / macOS
// are stubs in v0.1). Apps still run headless / offscreen.
func nativeBackend() Backend { return nil }
