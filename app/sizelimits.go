package app

import "github.com/codemodify/uitoolkit/platform"

// SetMinSize states the smallest the window may be made, in logical
// pixels, after it has been created.
//
// [platform.WindowOptions].MinWidth and MinHeight are given as the window
// is made, which is before the content that decides them exists: a program
// that wants a window no smaller than its content can fit in has nothing
// to measure yet, and was left to hard-code a constant it found by
// experiment. With the widgets in place [widget.MinWidthOf] can answer,
// and this is where to put the answer:
//
//	win.SetMinSize(widget.MinWidthOf(content)/win.Scale(), 0)
//
// Zero leaves that dimension to whatever the window already had, so a
// width may be stated without inventing a height. The maximum, and the
// resize policy, are untouched.
//
// What the window system does about a window that is already smaller
// differs: X11 and Wayland leave it to the window manager or compositor,
// Windows applies it at the next resize, macOS resizes at once. Nothing
// here resizes the window itself.
//
// It reports whether the backend took it. An offscreen surface does;
// a backend without [platform.SizeLimitSurface] does not.
func (w *Window) SetMinSize(minW, minH float32) bool {
	if w == nil || w.surf == nil {
		return false
	}
	s, ok := w.surf.(platform.SizeLimitSurface)
	if !ok {
		return false
	}
	g, hasG := w.surf.(interface{ SizeLimits() platform.SizeLimits })
	var l platform.SizeLimits
	if hasG {
		l = g.SizeLimits()
	}
	if minW > 0 {
		l.MinWidth = int(minW + 0.5)
	}
	if minH > 0 {
		l.MinHeight = int(minH + 0.5)
	}
	// A minimum above the maximum is a window nothing can satisfy; the
	// larger of the two is what the caller just asked for.
	if l.MaxWidth > 0 && l.MinWidth > l.MaxWidth {
		l.MaxWidth = l.MinWidth
	}
	if l.MaxHeight > 0 && l.MinHeight > l.MaxHeight {
		l.MaxHeight = l.MinHeight
	}
	s.SetSizeLimits(l)
	return true
}

// MinSize is the smallest the window may be made, in logical pixels, as
// the backend last had it (see [Window.SetMinSize]).
func (w *Window) MinSize() (minW, minH float32) {
	if w == nil || w.surf == nil {
		return 0, 0
	}
	if g, ok := w.surf.(interface{ SizeLimits() platform.SizeLimits }); ok {
		l := g.SizeLimits()
		return float32(l.MinWidth), float32(l.MinHeight)
	}
	return 0, 0
}
