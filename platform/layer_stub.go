//go:build !linux || !cgo

package platform

// layerSurfacesAvailable is Wayland's answer, and there is no Wayland
// backend in this build: no zwlr_layer_shell_v1 here. Windows, macOS and
// the offscreen desktop place windows outright, so
// [ScreenPlacementAvailable] never consults this.
func layerSurfacesAvailable() bool { return false }

// layerShellVersion is 0 with no Wayland backend.
func layerShellVersion() int { return 0 }
