package app

import (
	"math"

	"github.com/codemodify/uitoolkit/platform"
)

// Two things a window's title bar does beyond moving it: rolling the window
// up to the title bar and back down (shading), and keeping the window above
// the others.
//
// **Rolling up is the toolkit's own doing, and only for a frame the toolkit
// draws.** There is no shade request in xdg-shell, and X11's
// _NET_WM_STATE_SHADED is the *window manager's* title bar rolling up —
// useless to a window whose title bar the toolkit draws, because a window
// that asked for no decorations is not shadeable at all. So a rolled-up
// window here is a window that resized itself down to its caption: the
// desktop sees an ordinary resize, and the toolkit remembers the height to
// come back to. Under the desktop's frame ("OS window borders") the wheel
// over the title bar never reaches the client and shading is entirely the
// desktop's business — KWin's own "Mouse wheel on titlebar" setting — which
// is why CanShade is false there rather than the toolkit fighting for it.
//
// **Keeping a window above the others is the desktop's doing, and only X11
// can.** See [platform.AboveSurface]: X11 has _NET_WM_STATE_ABOVE, and
// Wayland has no keep-above a client may ask for on its own surface. The
// caption button follows [Window.CanKeepAbove] and says so rather than
// going quiet.

// CanKeepAbove reports whether the desktop can keep the window above the
// others. X11 window managers that list _NET_WM_STATE_ABOVE in
// _NET_SUPPORTED can; Wayland compositors cannot, whatever desktop they
// are (see [platform.AboveSurface]).
func (w *Window) CanKeepAbove() bool {
	if w == nil || w.Closed() {
		return false
	}
	return platform.SurfaceKeepAboveSupported(w.surf)
}

// KeepAbove reports whether the desktop is keeping the window above the
// others. It is the desktop's answer, not the last request: a window
// manager that refused says so by never setting the state.
func (w *Window) KeepAbove() bool {
	return w != nil && w.state.KeepAbove
}

// SetKeepAbove asks the desktop to keep the window above the others, or to
// stop. It reports whether there was anything to ask — false where the
// desktop cannot, which is every Wayland compositor.
//
// The answer arrives as an ordinary window-state change, so [Window.KeepAbove]
// and the caption button follow what the desktop did rather than what was
// asked.
func (w *Window) SetKeepAbove(on bool) bool {
	if w == nil || w.Closed() {
		return false
	}
	return platform.SetKeepAbove(w.surf, on)
}

// ToggleKeepAbove turns keep-above on or off (the caption button, the
// window menu, the desktop's title-bar action).
func (w *Window) ToggleKeepAbove() bool {
	if w == nil {
		return false
	}
	return w.SetKeepAbove(!w.KeepAbove())
}

// CanShade reports whether the window can be rolled up to its title bar
// now. It needs a title bar of the toolkit's to roll up to, a backend that
// can pin the window's height while it is rolled up
// ([platform.ShadeSurface]), and a window whose height is the toolkit's to
// change: a maximized, tiled or full-screen window's size belongs to the
// desktop, and a window pinned to one size (platform.SizingFixed) has said
// its height is not to be touched.
func (w *Window) CanShade() bool {
	if w == nil || w.Closed() || w.caption == nil || !w.framed() {
		return false
	}
	if w.state.Maximized || w.state.Fullscreen || !w.Resizable() {
		return false
	}
	// A window tiled or held at the top and bottom cannot change height.
	if (w.state.Tiled|w.state.Constrained)&(platform.EdgeTop|platform.EdgeBottom) != 0 {
		return false
	}
	_, ok := w.surf.(platform.ShadeSurface)
	return ok
}

// Shaded reports whether the window is rolled up to its title bar.
func (w *Window) Shaded() bool { return w != nil && w.shaded }

// ToggleShade rolls the window up to its title bar, or a rolled-up one back
// down. It reports whether it did.
func (w *Window) ToggleShade() bool { return w.SetShaded(!w.Shaded()) }

// SetShaded rolls the window up to its title bar, or back down to the
// height it had. It reports whether it did: rolling up needs CanShade, and
// rolling one back down always works.
//
// The height to come back to is remembered here, not by the desktop, which
// never learns that the window is rolled up at all. A compositor-driven
// resize while the window is rolled up changes its width and leaves the
// remembered height alone, so unrolling gives the window back the height it
// went up with.
func (w *Window) SetShaded(on bool) bool {
	if w == nil || w.Closed() {
		return false
	}
	if on == w.shaded {
		return false
	}
	if !on {
		w.unshade()
		return true
	}
	if !w.CanShade() {
		return false
	}
	h := w.shadedHeight()
	if h < 1 {
		return false
	}
	lw, lh := w.Size()
	if h >= lh {
		// The window is already no taller than its title bar.
		return false
	}
	w.shaded, w.shadeRestore = true, lh
	// Pin the height first: a window manager clamps a resize to the
	// window's stated minimum, so unpinned the roll-up would spring open.
	platform.SetShadedHeight(w.surf, h)
	w.SetSize(lw, h)
	w.shadeChanged()
	return true
}

// unshade gives the window its height back. It is also the way out of every
// corner a rolled-up window could be cornered in (see unshadeIfStuck): it
// never refuses.
func (w *Window) unshade() {
	if !w.shaded {
		return
	}
	h := w.shadeRestore
	w.shaded, w.shadeRestore = false, 0
	// Unpin before resizing, or the pin clamps the window back to its
	// rolled-up height.
	platform.SetShadedHeight(w.surf, 0)
	if h > 0 {
		lw, _ := w.Size()
		w.SetSize(lw, h)
	}
	w.shadeChanged()
}

// unshadeIfStuck rolls the window back down when it can no longer be rolled
// up — the desktop took the frame back ("OS window borders"), the window was
// maximized, tiled or put full screen, or it was pinned to one size. A
// window must never be left rolled up with no title bar to roll it back
// down with.
func (w *Window) unshadeIfStuck() {
	if w != nil && w.shaded && !w.CanShade() {
		w.unshade()
	}
}

// shadeChanged re-lays the window out for its new height: with no room
// under the caption the content measures to nothing, so nothing else has to
// know the window is rolled up.
func (w *Window) shadeChanged() {
	w.band = nil
	w.laid = false
	w.dropScene()
	w.fullInvalidate()
	if w.caption != nil {
		w.caption.Invalidate()
	}
}

// shadedHeight is how tall the window is once it is rolled up, in the
// logical pixels [Window.SetSize] speaks: the look's top border, the
// caption band, and the bottom border, so the frame still closes under the
// title bar. It is 0 when there is no caption to roll up to.
func (w *Window) shadedHeight() int {
	g := w.geom
	if w.caption == nil || g.caption.Empty() {
		return 0
	}
	px := g.caption.Max.Y - g.window.Min.Y + g.border.Bottom
	if px < 1 {
		return 0
	}
	return platform.LogicalPixels(int(math.Ceil(float64(px))), w.Scale())
}

// captionWheel runs the desktop's wheel-over-the-title-bar action for a
// scroll that landed on caption space. It reports whether it took the
// scroll; when it does not, the scroll goes on to the title bar's own
// contents, so a window with a title bar of its own (Window.SetTitleBar — a
// tab strip, a scrolling breadcrumb) keeps its wheel.
//
// Only caption space itself scrolls the window: a press there moves the
// window, so a turn there rolls it up. Anything the app put in the title bar
// is the app's, buttons and widgets alike.
func (w *Window) captionWheel(ev platform.Event) bool {
	if w == nil || w.caption == nil || !w.geom.framed {
		return false
	}
	if w.app.TitleBarPrefs().Wheel != platform.TitleWheelShade {
		return false
	}
	if r, _ := w.NonClientHit(ev.Pos); r != RegionCaption {
		return false
	}
	// Wheel up rolls the window up, wheel down rolls it back down, which is
	// the way round KWin's Shade/Unshade turns and the way the window
	// visibly moves.
	switch {
	case ev.Scroll.Y < 0:
		return w.SetShaded(true)
	case ev.Scroll.Y > 0:
		return w.SetShaded(false)
	}
	return false
}
