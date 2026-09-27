//go:build linux && cgo

package platform

// screenRectAt asks whichever display backend this process is on
// ([ScreenRectAt]). The X11 and Wayland answers are in x11_popup_linux.go
// and wayland_layer_linux.go, beside the code that already knows how to
// talk to each.
// It is a type switch and not a test of Name(): these are this package's
// own two Linux backends, so the compiler can check the dispatch, and a
// misspelt string cannot silently fall through to "no screen". This is
// not the boundary branching on a backend — the file is linux && cgo, so
// no other platform reaches it — it is one implementation choosing
// between its own two halves.
func screenRectAt(x, y int) (FrameRect, bool) {
	switch Default(false).(type) {
	case wlBackend:
		return wlScreenRectAt(x, y)
	case x11Backend:
		return x11ScreenRectAt(x, y)
	}
	return FrameRect{}, false
}
