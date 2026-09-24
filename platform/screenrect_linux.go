//go:build linux && cgo

package platform

// screenRectAt asks whichever display backend this process is on
// ([ScreenRectAt]). The X11 and Wayland answers are in x11_popup_linux.go
// and wayland_layer_linux.go, beside the code that already knows how to
// talk to each.
func screenRectAt(x, y int) (FrameRect, bool) {
	switch Default(false).Name() {
	case "wayland":
		return wlScreenRectAt(x, y)
	case "x11":
		return x11ScreenRectAt(x, y)
	}
	return FrameRect{}, false
}
