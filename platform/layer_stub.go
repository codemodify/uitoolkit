//go:build !linux || !cgo

package platform

// LayerSurfacesAvailable is Wayland's answer, and there is no Wayland
// backend in this build: no zwlr_layer_shell_v1 here. Windows, macOS and
// the offscreen desktop place windows outright, so
// [ScreenPlacementAvailable] never consults this.
func LayerSurfacesAvailable() bool { return false }

// LayerShellVersion is 0 with no Wayland backend.
func LayerShellVersion() int { return 0 }
