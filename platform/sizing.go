package platform

// Whether a window may be resized, and between which limits.
//
// The desktop does the refusing: a window states its minimum and its
// maximum, and the window manager or compositor drops the resize edges,
// greys out maximize and clamps a drag to them. The toolkit's own frame
// asks [SurfaceSizing] for the same answer, so a window it draws offers no
// resize band either.

// Sizing says whether the desktop may resize a window.
//
// The zero value is [SizingResizable], so a window that says nothing is
// resizable — which is what every window was before this was read.
type Sizing uint8

const (
	// SizingResizable is the default: the user resizes the window, down to
	// WindowOptions.MinWidth / MinHeight and up to MaxWidth / MaxHeight.
	SizingResizable Sizing = iota
	// SizingFixed pins the window to the size it has. The window system is
	// told its minimum and its maximum are the same
	// (xdg_toplevel.set_min_size / set_max_size, WM_NORMAL_HINTS
	// PMinSize | PMaxSize), which is how a desktop knows to drop the
	// resize edges and grey maximize out; an interactive resize
	// (StartSystemResize) is refused, and the toolkit's frame offers no
	// resize band.
	//
	// The *application* may still resize it: Resize moves the pin with the
	// window. A player that folds into a compact shape is two fixed sizes,
	// not a resizable window.
	SizingFixed
)

func (s Sizing) String() string {
	if s == SizingFixed {
		return "fixed"
	}
	return "resizable"
}

// SizeLimits is what the window system is told about how large a window may
// be, in the logical pixels [WindowOptions] speaks. A zero maximum is no
// maximum.
type SizeLimits struct {
	MinWidth, MinHeight int
	MaxWidth, MaxHeight int
}

// Fixed reports whether the limits pin the window to one size.
func (l SizeLimits) Fixed() bool {
	return l.MinWidth > 0 && l.MinWidth == l.MaxWidth && l.MinHeight > 0 && l.MinHeight == l.MaxHeight
}

// limitsFor is the window's limits at size w by h: the options' own, or the
// window's size on both ends when it is fixed.
func limitsFor(sizing Sizing, opts WindowOptions, w, h int) SizeLimits {
	if sizing == SizingFixed {
		return SizeLimits{MinWidth: max(w, 1), MinHeight: max(h, 1), MaxWidth: max(w, 1), MaxHeight: max(h, 1)}
	}
	return SizeLimits{
		MinWidth:  max(opts.MinWidth, 0),
		MinHeight: max(opts.MinHeight, 0),
		MaxWidth:  max(opts.MaxWidth, 0),
		MaxHeight: max(opts.MaxHeight, 0),
	}
}

// dropResizeCaps takes the actions a fixed window does not have out of what
// the desktop said it can do. Maximizing is a resize, so a window that may
// not be resized may not be maximized either — and a caption button that
// cannot work should not be drawn (widgets.WindowControls.Shown).
func dropResizeCaps(c WMCaps, sizing Sizing) WMCaps {
	if sizing != SizingFixed {
		return c
	}
	return (c | CapKnown) &^ CapMaximize
}

// SizingSurface is an optional Surface capability: the window's resize
// policy and the limits the window system has been told about. All three
// Linux backends implement it.
type SizingSurface interface {
	// Sizing is the policy in effect.
	Sizing() Sizing
	// SetSizing changes it and re-states the limits to the window system.
	SetSizing(s Sizing)
	// SizeLimits is what the window system was last told.
	SizeLimits() SizeLimits
}

// SurfaceSizing is s's resize policy (resizable when s cannot say).
func SurfaceSizing(s Surface) Sizing {
	if v, ok := s.(SizingSurface); ok {
		return v.Sizing()
	}
	return SizingResizable
}

// SetSurfaceSizing changes s's resize policy. It reports whether s could.
func SetSurfaceSizing(s Surface, sz Sizing) bool {
	v, ok := s.(SizingSurface)
	if !ok {
		return false
	}
	v.SetSizing(sz)
	return true
}

// SurfaceSizeLimits is what s told the window system (zero when it cannot
// say).
func SurfaceSizeLimits(s Surface) SizeLimits {
	if v, ok := s.(SizingSurface); ok {
		return v.SizeLimits()
	}
	return SizeLimits{}
}
