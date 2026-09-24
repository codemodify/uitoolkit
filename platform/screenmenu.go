package platform

// Where a menu opened at a point on the screen goes.
//
// A tray menu is the one menu the toolkit places against the *screen*
// rather than against a window: the host names a point in root
// coordinates (org.kde.StatusNotifierItem.ContextMenu) and the menu
// belongs next to the icon that was clicked. Everything else the toolkit
// opens is a popup of a window and is placed by [SolvePopup] (X11,
// offscreen) or by the compositor (Wayland, xdg_positioner).
//
// A menu placed at a point still has to obey the screen, and two reports
// from a real KDE Wayland session said it did not:
//
//   - the menu covered the tray icon it was opened from, because the
//     point became the menu's top-left corner;
//   - its submenus were off the screen, because nothing constrained the
//     surface — which is measured to hold the parent menu *and* its
//     widest cascade — against the monitor it landed on.
//
// [SolveScreenMenu] is the single answer to "where does this menu go",
// used by the X11 and the Wayland paths alike so the two cannot drift. It
// is [SolvePopup] with the anchor the protocol never sends filled in: the
// same flip/slide/shrink rules, the same vocabulary, one implementation.

// ScreenMenuPointerGap is the distance a menu keeps from the point it was
// opened at, in *device* pixels at scale 1.
//
// The reason it exists at all: SNI gives a point and never the tray
// icon's rectangle, so a menu whose corner is exactly on that point sits
// on top of the icon (and, on a touchpad click, under the pointer). A
// real menu is placed beside its anchor rectangle; with no rectangle to
// be had, the honest approximation is a small gap around the point. A
// cursor is roughly 24 device pixels across in the usual themes, so a
// gap of that order clears both the pointer and a tray icon of ordinary
// size without pushing the menu visibly away from what it belongs to.
const ScreenMenuPointerGap = 24

// ScreenMenuPlacement is the placement of a menu opened at a point of the
// screen: an anchor box of ScreenMenuPointerGap around the point, the
// menu hanging down and to the right of it, free to flip to the other
// side, slide back in, and finally shrink.
//
// x, y and gap are in the same units as the menu's size and the screen
// rectangle it will be constrained to — logical pixels of the desktop for
// every caller today.
func ScreenMenuPlacement(x, y, w, h, gap int) PopupPlacement {
	if gap < 0 {
		gap = 0
	}
	return PopupPlacement{
		// The anchor is the point grown by the gap in every direction.
		// Hanging off its bottom-right corner puts the menu gap pixels
		// below and right of the point; flipping puts it gap pixels
		// above or left of it, never on it. This is the rectangle the
		// SNI protocol does not send us, guessed.
		Anchor:     FrameRect{X: x - gap, Y: y - gap, W: 2 * gap, H: 2 * gap},
		AnchorEdge: EdgeBottom | EdgeRight,
		Gravity:    EdgeBottom | EdgeRight,
		W:          w, H: h,
		Adjust: AdjustFlipX | AdjustFlipY | AdjustSlideX | AdjustSlideY | AdjustResizeX | AdjustResizeY,
	}
}

// SolveScreenMenu is where a menu of w x h opened at the screen point
// x, y goes, constrained to screen: flipped to the other side of the
// point where it does not fit, slid back onto the screen where flipping
// does not help, and shrunk only when it is larger than the screen
// itself. The result never covers the point (see ScreenMenuPointerGap)
// and never has a negative size.
//
// The rectangle to pass as w x h is the *whole* menu surface — for the
// tray menu, the parent menu plus its widest cascade chain, because the
// submenus open inside that surface. Constraining the parent alone is
// what put the submenus off the edge of the screen.
//
// screen is the monitor's rectangle in the same coordinates (see
// [ScreenRectAt]); an empty one constrains nothing, which is what a
// backend that cannot say leaves the caller with.
func SolveScreenMenu(x, y, w, h int, screen FrameRect, gap int) FrameRect {
	return SolvePopup(ScreenMenuPlacement(x, y, w, h, gap), screen)
}
