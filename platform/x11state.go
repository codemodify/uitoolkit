package platform

import "strings"

// Display-independent parts of the X11 backend's frame support: decoding the
// window manager's _NET_WM_STATE and _NET_WM_ALLOWED_ACTIONS, packing
// _MOTIF_WM_HINTS, and telling tiling window managers apart. No cgo, no
// build tag: tested headless.

// _NET_WM_STATE atoms the backend follows, as bits of the set it hands to
// netWMState.
const (
	netStateMaxVert uint32 = 1 << iota
	netStateMaxHorz
	netStateFullscreen
	netStateHidden
	netStateFocused
)

// netWMState decodes a window's _NET_WM_STATE. EWMH has no tiled states;
// like GTK and Chromium, a window maximized one way only counts as tiled on
// those edges. focusKnown is false when the window manager does not
// maintain _NET_WM_STATE_FOCUSED (EWMH 1.5): Activated then comes from
// focus events instead.
func netWMState(bits uint32, focusKnown, focused bool) WindowState {
	st := WindowState{
		Fullscreen: bits&netStateFullscreen != 0,
		Minimized:  bits&netStateHidden != 0,
	}
	vert, horz := bits&netStateMaxVert != 0, bits&netStateMaxHorz != 0
	switch {
	case vert && horz:
		st.Maximized = true
	case vert:
		st.Tiled = EdgeTop | EdgeBottom
	case horz:
		st.Tiled = EdgeLeft | EdgeRight
	}
	if focusKnown {
		st.Activated = bits&netStateFocused != 0
	} else {
		st.Activated = focused
	}
	return st
}

// _NET_WM_ALLOWED_ACTIONS atoms the backend follows, as bits of the set it
// hands to netAllowedCaps.
const (
	netActionMinimize uint32 = 1 << iota
	netActionMaximizeHorz
	netActionMaximizeVert
	netActionFullscreen
)

// netAllowedCaps decodes _NET_WM_ALLOWED_ACTIONS; windowMenu is whether the
// window manager lists _GTK_SHOW_WINDOW_MENU in _NET_SUPPORTED.
func netAllowedCaps(bits uint32, windowMenu bool) WMCaps {
	c := CapKnown
	if bits&netActionMinimize != 0 {
		c |= CapMinimize
	}
	if bits&netActionMaximizeHorz != 0 && bits&netActionMaximizeVert != 0 {
		c |= CapMaximize
	}
	if bits&netActionFullscreen != 0 {
		c |= CapFullscreen
	}
	if windowMenu {
		c |= CapWindowMenu
	}
	return c
}

// _MOTIF_WM_HINTS flags and decorations (the five-CARD32 layout every X11
// window manager reads: flags, functions, decorations, input mode, status).
const (
	motifHintsDecorations = 1 << 1
	motifDecorAll         = 1 << 0
)

// motifDecorateAll is _MOTIF_WM_HINTS asking for every decoration: what a
// window that asked for none says to have them back. Removing the
// property says nothing to a window manager that has already read it —
// KWin re-reads the hints when they change and acts only on hints that
// state decorations, so a window whose "no frame" property was deleted kept
// no frame at all.
var motifDecorateAll = [5]uint32{motifHintsDecorations, 0, motifDecorAll, 0, 0}

// motifHints is the _MOTIF_WM_HINTS value asking the window manager for no
// frame (a client-side frame, or none at all). ok is false for a window
// that keeps the window manager's frame: the property is removed then, so
// the manager decorates by its own rules.
func motifHints(d Decorations) (hints [5]uint32, ok bool) {
	switch d {
	case DecorationsClient, DecorationsNone:
		return [5]uint32{motifHintsDecorations, 0, 0, 0, 0}, true
	}
	return hints, false
}

// tilingWM reports whether the window manager called name (its
// _NET_SUPPORTING_WM_CHECK window's _NET_WM_NAME) tiles windows: there a
// window keeps the manager's frame unless the app insists (Chromium's rule).
func tilingWM(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "i3", "awesome", "xmonad", "qtile", "ratpoison", "stumpwm", "wmii",
		"ion3", "notion", "dwm", "bspwm", "herbstluftwm", "spectrwm", "leftwm":
		return true
	}
	return false
}
