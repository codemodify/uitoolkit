package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
)

// CaptionHitTester is implemented by components that can be title bar. At
// local point p, true means "this is caption: pressing here drags the
// window, double-clicking maximizes it"; false means "this is a control".
// Containers, spacers and labels answer true; a tool bar answers true
// between its buttons, a tab bar after its last tab. A component that does
// not implement it is a control.
type CaptionHitTester interface {
	CaptionAt(p paintengine2d.Point) bool
}

// IsCaption reports whether window point p (device pixels), whose deepest
// hit component is hit, is caption space of the title bar titleBar. Every
// component from hit up to titleBar must say so, so a label inside a button
// is not caption (the nearest-ancestor rule would make it one).
func IsCaption(hit, titleBar Component, p paintengine2d.Point) bool {
	if hit == nil || titleBar == nil {
		return false
	}
	for c := hit; c != nil; c = c.Parent() {
		ct, ok := c.(CaptionHitTester)
		if !ok {
			return false
		}
		o := DeviceOrigin(c)
		if !ct.CaptionAt(paintengine2d.Pt(p.X-o.X, p.Y-o.Y)) {
			return false
		}
		if c == titleBar {
			return true
		}
	}
	return false
}

// CaptionMenuer is implemented by title-bar components with a context menu
// of their own for their caption space (Chromium's tab-strip menu): a
// right-click there runs CaptionMenu (p in window device pixels) before
// the header bar's OnContextMenu and the window menu, and it reports
// whether it showed a menu.
type CaptionMenuer interface {
	CaptionMenu(p paintengine2d.Point) bool
}

// FrameHost is implemented by app.Window: what a window's caption controls
// (widgets.WindowControls) show and ask for.
type FrameHost interface {
	// Title is the window's title.
	Title() string
	// WindowState is what the desktop says about the window (maximized
	// shows the restore glyph).
	WindowState() platform.WindowState
	// FrameCaps is what the window system will do for the window's frame
	// now: caption buttons for anything it will not do are hidden, rather
	// than drawn and dead.
	FrameCaps() platform.FrameCaps
	// Active reports whether the window paints as active.
	Active() bool
	// Minimize and ToggleMaximize are commands, not questions: unlike the
	// requests on [platform.WindowFrame] they answer nothing, because a
	// caption never asks one it has not already drawn — FrameCaps decided
	// that, and an MDI child window (widgets.mdi) always can. What the
	// desktop did comes back through WindowState.
	Minimize()
	ToggleMaximize()
	// RequestClose asks the window to close, as the desktop's own close
	// button does (an app may keep the window: close-to-tray).
	RequestClose()
	// ShowWindowMenu shows the desktop's window menu at p (window device
	// pixels), or the toolkit's own where the desktop has none.
	ShowWindowMenu(p paintengine2d.Point)
}

// FrameAbove is what a FrameHost implements to carry the keep-above
// *state*: whether the desktop is holding the window above the others, and
// the toggle. app.Window implements it.
//
// Whether the desktop *can* is not here — it is
// [platform.FrameKeepAbove] in FrameCaps, like every other thing the
// window system may or may not do, so the caption asks one question in one
// place. No Wayland compositor can (there is no keep-above request a
// client may make on its own surface); X11 window managers that list
// _NET_WM_STATE_ABOVE in _NET_SUPPORTED can. A caption button that cannot
// work is not drawn, exactly as minimize is not where the desktop cannot
// minimize.
type FrameAbove interface {
	// KeepAbove is whether the desktop is keeping the window above the
	// others — its answer, which is what the button paints.
	KeepAbove() bool
	// ToggleKeepAbove turns it on or off.
	ToggleKeepAbove() bool
}

// FrameRequests is what a FrameHost implements to ask its look for a frame
// of its own: the role the app gave the window, which a look that dresses
// windows differently chooses a frame by, and a caption height the app
// asked for (0: the look's). app.Window implements it.
type FrameRequests interface {
	FrameRole() string
	CaptionHeight() float32
	// Borderless reports that the window's chrome runs to its edges, so
	// the frame draws no line down the sides or along the bottom.
	Borderless() bool
}
