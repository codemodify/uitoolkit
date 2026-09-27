package platform

import "strings"

// BackendCaps is what a whole backend will do, as opposed to what it will
// do for one window ([FrameCaps]).
//
// It exists because four places used to branch on the backend's *name*:
//
//	a.backend.Name() == "offscreen"   // app/atspi_linux.go, desktopprefs.go, frame.go
//	Default(false).Name() != "wayland" // platform/placement.go
//
// A name is not a capability. It says which backend this is, not what it
// can do, so every reader had to know the answer for every backend — and
// a fifth backend is wrong at all four sites until somebody remembers to
// add it to each list. Worse, the sense is inverted: those tests name the
// backends that *cannot*, so a new backend is silently assumed able. A
// Win32 backend would have been treated as a full desktop by all three
// "offscreen" tests, which is right, and as an absolute placer by the
// fourth, which is also right — but by luck, not because it said so.
//
// A capability says it. [Name] stays, for diagnostics and for the tour
// sample to print; nothing branches on it.
type BackendCaps uint32

const (
	// BackendDesktop: there is a real desktop session behind this backend
	// — a window manager or compositor, desktop preferences to read and
	// follow, assistive technology that may be listening. The offscreen
	// backend has none of that: nothing is on a screen, no preference is
	// anybody's, and no screen reader is watching.
	//
	// It is one bit and not three because the three go together: they are
	// all "is anybody there". A backend that had a window manager but no
	// preferences would be a new thing, and can have a new bit then.
	BackendDesktop BackendCaps = 1 << iota
	// BackendScreenPlace: a window opened with Place [PlaceAtScreen] is
	// really put where it asks.
	//
	// X11 places windows and so would Win32 (SetWindowPos) and AppKit
	// (setFrameOrigin:); the offscreen desktop places them because it is
	// pretending. Wayland cannot: a plain xdg_toplevel has no position at
	// all, and only zwlr_layer_shell_v1 gives one — so this bit is false
	// on GNOME/Mutter, which has declined that protocol, and true on KDE,
	// sway, Hyprland and wayfire, which offer it. That makes it the one
	// capability here that changes with the compositor rather than with
	// the backend, which is why Caps is a method and its answer is never
	// cached.
	BackendScreenPlace
)

// Has reports whether every capability in x is present.
func (c BackendCaps) Has(x BackendCaps) bool { return c&x == x }

var backendCapNames = []string{"desktop", "screen-place"}

// String lists the capabilities present, in bit order, space separated
// ("none" for none).
func (c BackendCaps) String() string {
	if c == 0 {
		return "none"
	}
	out := make([]string, 0, len(backendCapNames))
	for i, name := range backendCapNames {
		if c&(1<<uint(i)) != 0 {
			out = append(out, name)
		}
	}
	return strings.Join(out, " ")
}

// BackendCapsOf is what b will do, for a backend that may be nil (no
// backend does nothing, which is what a headless application without one
// should be treated as).
func BackendCapsOf(b Backend) BackendCaps {
	if b == nil {
		return 0
	}
	return b.Caps()
}
