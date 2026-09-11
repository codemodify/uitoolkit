//go:build linux && cgo

package platform

// nativeBackend is X11 when a display is advertised.
func nativeBackend() Backend {
	if !hasDisplay() {
		return nil
	}
	return X11Backend{}
}
