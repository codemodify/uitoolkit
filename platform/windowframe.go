package platform

import (
	"strings"

	"github.com/codemodify/paintengine2d"
)

// The window-frame seam: one interface for everything a top-level window's
// frame needs from the window system, and one value that says which of it
// the window system will actually do.
//
// # Why one interface and one value
//
// This area used to be nine interfaces — FrameSurface, Lowerer,
// AxisMaximizer, AboveSurface, ShadeSurface, GlassSurface, DesktopSurface,
// DecorationPaletteSurface, IconSurface — that a caller found by type
// assertion. Three backends implemented them and the assertion answered
// "yes" for six of the nine on all three, so it carried no information;
// for the other three it answered a question the caller had to ask twice,
// because the interface it had just found also had a *Supported() method
// that could still say no (AboveSurface.KeepAboveSupported,
// GlassSurface.BlurBehindSupported,
// DecorationPaletteSurface.DecorationPaletteSupported). A missing method
// is the wrong way to say "this desktop will not do that": it is decided
// when the backend is compiled, it cannot change while the window is up —
// and a window manager can be replaced under a running window, KWin drops
// blur when desktop effects go off, and a compositor may withhold minimize
// per window — and a caller that forgets the assertion gets silence rather
// than a compiler error.
//
// So: one [WindowFrame] every backend implements in full, and one
// [FrameCaps] it answers with, asked afresh every time. A backend that
// cannot do something says so in data. The layer above reads the data and
// degrades on purpose — a caption button that cannot work is not drawn
// (widgets.WindowControls.Shown), a menu entry that cannot work is not
// listed — instead of drawing a control that quietly does nothing.

// FrameCaps is what the window system will do for one window's frame
// *now*. It folds two things the caller never had reason to tell apart:
// what the backend has code for (xdg-shell has no keep-above; Wayland has
// no lower), and what the desktop grants this window at this moment
// (xdg_toplevel.wm_capabilities, X11's _NET_WM_ALLOWED_ACTIONS, a fixed
// window that may not be maximized because it may not be resized).
//
// It is read from [WindowFrame.Caps] every time rather than cached.
type FrameCaps uint32

const (
	// FrameMove: the desktop runs an interactive move from a button press
	// the application hands it ([WindowFrame.StartMove]).
	FrameMove FrameCaps = 1 << iota
	// FrameResize: the same for a resize from an edge or a corner.
	FrameResize
	// FrameMenu: the desktop has a window menu of its own to show.
	FrameMenu
	// FrameMinimize, FrameMaximize, FrameFullscreen: the desktop grants
	// this window those states.
	FrameMinimize
	FrameMaximize
	FrameFullscreen
	// FrameMaximizeAxis: the window can be maximized one way only
	// (X11's _NET_WM_STATE_MAXIMIZED_VERT / _HORZ). A desktop with
	// FrameMaximize but not this one maximizes both ways or not at all,
	// which is all xdg-shell can do.
	FrameMaximizeAxis
	// FrameLower: the window can be put below the others.
	FrameLower
	// FrameKeepAbove: the desktop can keep the window above the others.
	FrameKeepAbove
	// FrameShade: the window's height can be pinned while it is rolled up
	// to its title bar, so the desktop does not clamp it back to the
	// minimum height the window states.
	FrameShade
	// FrameBlurBehind: the desktop blurs what lies behind the window, so a
	// translucent window is glass over the desktop rather than over its
	// own pixels.
	FrameBlurBehind
	// FramePalette: the desktop's own frame takes a colour scheme per
	// window.
	FramePalette
	// FrameIcon: the window can carry an icon of its own for the title
	// bar, the switcher and the task bar.
	FrameIcon
	// FrameClientFrame: a frame the toolkit draws works well here — the
	// desktop moves and resizes windows on request and is not a tiling
	// manager. The Auto decoration policy picks a client frame only then.
	FrameClientFrame
)

// Has reports whether every capability in x is present.
func (c FrameCaps) Has(x FrameCaps) bool { return c&x == x }

// frameDesktopCaps are the four capabilities a desktop grants or withholds
// per window. Until it has said, a backend assumes all four: a Wayland
// compositor need never send xdg_toplevel.wm_capabilities, and an X11
// window manager need never set _NET_WM_ALLOWED_ACTIONS, so a caption
// button that waited for permission would never appear at all.
//
// The assumption belongs in the backend, which knows whether its desktop
// has spoken. It used to be a CapKnown bit on the value itself, which made
// every reader carry a tri-state — "yes", "no", "nobody has said, so yes"
// — for a distinction only the backend could act on.
const frameDesktopCaps = FrameMenu | FrameMinimize | FrameMaximize | FrameFullscreen

// WindowFrame is a top-level window's frame seam: the decoration
// negotiation, the window's state, the frame the toolkit draws, and every
// request that hands a gesture or a state change back to the desktop.
//
// Moving, resizing, snapping and the window menu stay the desktop's: the
// toolkit asks, from the user's button press, and the desktop does it.
//
// **Every backend implements all of it.** A backend that cannot do
// something leaves its bit out of [WindowFrame.Caps] and its method
// returns false; it does not omit the method. That is the difference
// between a boundary a second platform can be held to and one it can quietly
// fall short of: a Win32 backend that forgets keep-above fails to compile,
// and one that cannot do it says so where the caption button can read it.
//
// Every request reports whether it was made — not whether it succeeded.
// The desktop has the last word on every state here, and the answer comes
// back as an [EventWindowState]: a window manager that refuses says so by
// never setting the state, which is why [WindowState] and not the last
// request is what the caption paints.
type WindowFrame interface {
	// Caps is what the window system will do for this window now.
	Caps() FrameCaps

	// Decorations is the mode in effect after negotiation: the desktop may
	// answer another mode than the one asked for (KWin draws no frame at
	// all for a full-screen window, GNOME draws none ever).
	Decorations() Decorations
	// RequestDecorations asks for mode d (Auto counts as Server). The
	// answer arrives as EventDecorations when it differs from the current
	// mode.
	RequestDecorations(d Decorations)
	// WindowState is the window's current state (EventWindowState reports
	// every change).
	WindowState() WindowState

	// SetFrame tells the window system about the frame the toolkit draws
	// (see [Frame]): the invisible margin, the resize band inside it, the
	// corner radii and whether the buffer needs alpha. The surface grows
	// by the margin at once — Size() is the buffer, the visible window is
	// Size() less the margin — and the window system is told with the next
	// Present, in the same commit as the buffer.
	SetFrame(f Frame)
	// Frame is the frame last set.
	Frame() Frame

	// StartMove hands the button press being handled to the desktop for an
	// interactive move (Qt's startSystemMove). It must be called while the
	// button is still down; false means nothing started. The pointer then
	// belongs to the desktop until the gesture ends: the release never
	// reaches the window.
	StartMove() bool
	// StartResize is StartMove for a resize from edges.
	StartResize(edges Edges) bool
	// ShowMenu asks the desktop for its window menu at p (surface device
	// pixels); false means it has none, and the toolkit shows its own.
	ShowMenu(p paintengine2d.Point) bool

	// Minimize iconifies the window, keeping it: unlike Hide, the taskbar
	// or the overview brings it back.
	Minimize() bool
	// SetMaximized maximizes the window both ways, or restores it.
	SetMaximized(on bool) bool
	// MaximizeAxis maximizes or restores one way only (FrameMaximizeAxis).
	MaximizeAxis(vertical bool) bool
	// SetFullscreen puts the window full screen, or takes it back.
	SetFullscreen(on bool) bool
	// SetKeepAbove asks the desktop to keep the window above the others,
	// or to stop (FrameKeepAbove).
	SetKeepAbove(on bool) bool
	// Lower puts the window below the others (FrameLower).
	Lower() bool
	// SetShadedHeight pins the window's height to h logical pixels while
	// it is rolled up to its title bar; 0 releases the pin and restores
	// the window's own limits (FrameShade).
	SetShadedHeight(h int) bool

	// SetPalette names the colour-scheme file the desktop's own frame
	// paints the window with, an absolute path; "" goes back to the
	// desktop's colours (FramePalette).
	SetPalette(path string) bool
	// SetIcon gives the window its icon as square images at the sizes the
	// app has; the desktop picks the one it shows. None goes back to the
	// desktop's default (FrameIcon).
	SetIcon(images []*paintengine2d.Image) bool
}

// FrameOf is s's frame seam. **It is never nil.** A surface with no frame
// of its own — a popup, a layer surface, a backend that has not written
// this part yet — answers with one that can do nothing: Caps() is zero,
// every request returns false, WindowState is the zero state and
// Decorations is Server.
//
// That is deliberate. The caller's question is "what will the window
// system do for this window", and it should have one answer, read from
// data, rather than two — an interface that might not be there, and then a
// capability inside it that might say no. A caller that forgets to look at
// Caps() now draws a control that returns false the first time it is used,
// which a test catches; a caller that forgot the old type assertion drew a
// control that did nothing at all, which nothing caught.
func FrameOf(s Surface) WindowFrame {
	if s == nil {
		return noFrame{}
	}
	if f, ok := s.(WindowFrame); ok {
		return f
	}
	return noFrame{}
}

// FrameCapsOf is what the window system will do for s's frame now (nothing,
// for a surface with no frame seam).
func FrameCapsOf(s Surface) FrameCaps { return FrameOf(s).Caps() }

// noFrame is the frame seam of a surface that has none: it can do nothing
// and says so.
type noFrame struct{}

func (noFrame) Caps() FrameCaps                     { return 0 }
func (noFrame) Decorations() Decorations            { return DecorationsServer }
func (noFrame) RequestDecorations(Decorations)      {}
func (noFrame) WindowState() WindowState            { return WindowState{} }
func (noFrame) SetFrame(Frame)                      {}
func (noFrame) Frame() Frame                        { return Frame{} }
func (noFrame) StartMove() bool                     { return false }
func (noFrame) StartResize(Edges) bool              { return false }
func (noFrame) ShowMenu(paintengine2d.Point) bool   { return false }
func (noFrame) Minimize() bool                      { return false }
func (noFrame) SetMaximized(bool) bool              { return false }
func (noFrame) MaximizeAxis(bool) bool              { return false }
func (noFrame) SetFullscreen(bool) bool             { return false }
func (noFrame) SetKeepAbove(bool) bool              { return false }
func (noFrame) Lower() bool                         { return false }
func (noFrame) SetShadedHeight(int) bool            { return false }
func (noFrame) SetPalette(string) bool              { return false }
func (noFrame) SetIcon([]*paintengine2d.Image) bool { return false }

// frameCapNames are the capability names, smallest bit first.
var frameCapNames = []string{
	"move", "resize", "menu", "minimize", "maximize", "fullscreen",
	"maximize-axis", "lower", "keep-above", "shade", "blur-behind",
	"palette", "icon", "client-frame",
}

// String lists the capabilities present, in bit order, space separated
// ("none" for none). It is what a diagnostic prints and what the tour
// sample shows, so a user can see what their desktop will and will not do
// without knowing a protocol.
func (c FrameCaps) String() string {
	if c == 0 {
		return "none"
	}
	out := make([]string, 0, len(frameCapNames))
	for i, name := range frameCapNames {
		if c&(1<<uint(i)) != 0 {
			out = append(out, name)
		}
	}
	return strings.Join(out, " ")
}
