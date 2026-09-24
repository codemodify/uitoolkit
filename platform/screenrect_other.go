//go:build !linux || !cgo

package platform

// screenRectAt: Windows, macOS and the offscreen desktop are not asked
// where their monitors are yet — no window here is placed against the
// screen's edges (see [ScreenRectAt]). A caller that gets false
// constrains against nothing.
func screenRectAt(x, y int) (FrameRect, bool) {
	_, _ = x, y
	return FrameRect{}, false
}
