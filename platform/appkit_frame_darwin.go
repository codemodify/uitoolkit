//go:build darwin && cgo

package platform

/*
#include <stdlib.h>
#include "appkit_darwin.h"
*/
import "C"

import (
	"unsafe"

	"github.com/codemodify/paintengine2d"
)

// [WindowFrame] for AppKit.
//
// The interesting half of this file is what macOS does *not* do, and
// saying so honestly rather than pretending. A capability withheld is
// the toolkit drawing its own answer or hiding a control that cannot
// work; a capability claimed and then not delivered is a button that
// does nothing. So: no window menu, no shade, no per-axis maximize, no
// colour scheme, and no interactive resize — AppKit has no public API
// to begin one from an edge, which is exactly why FrameSystemResizeBand
// exists.
//
// And the half macOS does for you: the shadow and the resize band, both
// outside the window and both the compositor's. A backend that reserved
// a margin for a shadow here would get two of them.

func (s *akSurface) FrameCaps() FrameCaps {
	if s == nil || s.win == nil || s.Closed() {
		return 0
	}
	c := FrameMove | FrameMinimize | FrameMaximize | FrameFullscreen |
		FrameKeepAbove | FrameLower | FrameIcon | FrameClientFrame |
		// -addChildWindow: is core AppKit. No FrameSkipTaskbar: the Dock
		// lists applications, not windows, so there is no per-window entry
		// for a satellite panel to be left out of.
		FrameOwner |
		FrameSystemShadow | FrameSystemResizeBand
	// Withheld, and each for its own reason:
	//
	//   FrameResize       no API to start one; AppKit's own band does it
	//   FrameMenu         macOS has no window menu to show
	//   FrameMaximizeAxis zoom is both ways or neither
	//   FrameShade        window shade went out with Mac OS 9
	//   FramePalette      a KDE colour scheme means nothing here
	//   FrameBlurBehind   NSVisualEffectView is a view, not a window
	//                     property, and the frame seam has no way to
	//                     put one behind a buffer the toolkit paints
	return dropResizeCaps(c, s.sizing)
}

func (s *akSurface) Decorations() Decorations { return s.decor }

func (s *akSurface) RequestDecorations(d Decorations) {
	if s == nil || s.win == nil || s.Closed() {
		return
	}
	if d == DecorationsAuto {
		d = DecorationsServer
	}
	if d == s.decor {
		return
	}
	s.decor = d
	// Server is AppKit's titled frame; client and none are both a
	// borderless window — the difference between them is what the
	// toolkit paints into it, which is not this layer's business.
	borderless := 0
	if d != DecorationsServer {
		borderless = 1
	}
	C.uitk_ak_set_decorations(s.win, C.int(borderless))
	s.push(Event{Kind: EventDecorations, Decor: d})
}

func (s *akSurface) WindowState() WindowState {
	if s == nil || s.win == nil || s.Closed() {
		return WindowState{}
	}
	return WindowState{
		Maximized:  C.uitk_ak_zoomed(s.win) != 0,
		Fullscreen: C.uitk_ak_fullscreen(s.win) != 0,
		Activated:  C.uitk_ak_active(s.win) != 0,
		Minimized:  C.uitk_ak_miniaturized(s.win) != 0,
		Suspended:  C.uitk_ak_miniaturized(s.win) != 0,
		// Read back from the window rather than remembered from the
		// last request: this is the state a caption's pin draws itself
		// from, and [Window.ToggleKeepAbove] is SetKeepAbove(!this).
		// Leaving it out does not merely fail to light the button up —
		// it makes every press mean "on", so the window can be pinned
		// and never released.
		KeepAbove: C.uitk_ak_above(s.win) != 0,
	}
}

// SetFrame records what the toolkit's frame asks for.
//
// Almost none of it applies. The margin and the input band are for a
// shadow the client draws, and macOS draws the shadow itself
// (FrameSystemShadow) — so the surface does *not* grow here, unlike on
// Wayland, and Size() stays the content area. The corner radii are
// AppKit's too on a titled window; on a borderless one the toolkit
// paints them into a buffer that already has alpha, which
// uitk_ak_set_decorations arranged when the window went borderless.
//
// It is kept so Frame() can answer, and so that a look which asks for
// alpha gets a window that can show it.
func (s *akSurface) SetFrame(f Frame) {
	if s == nil {
		return
	}
	s.frame = f
}

func (s *akSurface) Frame() Frame { return s.frame }

// StartMove hands the press being handled to AppKit.
//
// performWindowDragWithEvent: needs the event, and takes it from
// [NSApp currentEvent] — which is right only because this is called
// from inside the handling of that press, on the thread that is
// handling it. That is the same requirement the Wayland and X11
// backends have (a serial from the press), stated differently.
func (s *akSurface) StartMove() bool {
	if s == nil || s.win == nil || s.Closed() {
		return false
	}
	return C.uitk_ak_start_move(s.win) != 0
}

// StartResize is refused: macOS has no public API to begin a resize
// from an edge, and needs none — a resizable NSWindow already has
// invisible resize borders AppKit manages, which is what
// FrameSystemResizeBand tells the toolkit.
func (s *akSurface) StartResize(Edges) bool { return false }

// ShowMenu is refused: macOS windows have no window menu, so the
// toolkit shows its own.
func (s *akSurface) ShowMenu(paintengine2d.Point) bool { return false }

func (s *akSurface) Minimize() bool {
	if !s.FrameCaps().Has(FrameMinimize) {
		return false
	}
	C.uitk_ak_miniaturize(s.win)
	return true
}

func (s *akSurface) SetMaximized(on bool) bool {
	if !s.FrameCaps().Has(FrameMaximize) {
		return false
	}
	return C.uitk_ak_set_zoomed(s.win, cbool(on)) != 0
}

// MaximizeAxis is refused: zoom is both ways or neither.
func (s *akSurface) MaximizeAxis(bool) bool { return false }

func (s *akSurface) SetFullscreen(on bool) bool {
	if !s.FrameCaps().Has(FrameFullscreen) {
		return false
	}
	C.uitk_ak_set_fullscreen(s.win, cbool(on))
	return true
}

func (s *akSurface) SetKeepAbove(on bool) bool {
	if !s.FrameCaps().Has(FrameKeepAbove) {
		return false
	}
	C.uitk_ak_set_level(s.win, cbool(on))
	// The desktop is never asked again, so the toolkit is told here.
	// Windows does the same in its own SetKeepAbove; AppKit has no
	// notification for a level change to hang it on.
	s.push(Event{Kind: EventWindowState, State: s.WindowState()})
	return true
}

func (s *akSurface) Lower() bool {
	if !s.FrameCaps().Has(FrameLower) {
		return false
	}
	C.uitk_ak_order_back(s.win)
	return true
}

// SetShadedHeight is refused: the Mac has not rolled a window up to its
// title bar since Mac OS 9, and there is nothing to pin.
func (s *akSurface) SetShadedHeight(int) bool { return false }

// SetPalette is refused: the colour scheme it names is a KDE file, and
// AppKit's frame takes its colours from the system appearance.
func (s *akSurface) SetPalette(string) bool { return false }

// SetIcon sets the **application's** icon — the Dock tile — from the
// largest image given.
//
// macOS has no per-window icon: a window's title bar shows one only for
// a document, and that one is the document's file. So the honest
// mapping of "this is the icon for my window" is the Dock tile, which
// is what a caller asking for it wants to see, and it is worth having:
// a Go binary that is not in an .app bundle otherwise gets the blank
// generic tile. The cost is that the last window to ask wins, which for
// the one-icon-per-application case every caller actually has is no
// cost at all.
func (s *akSurface) SetIcon(images []*paintengine2d.Image) bool {
	if !s.FrameCaps().Has(FrameIcon) {
		return false
	}
	img := pickLargest(images)
	if img == nil || len(img.Pix) == 0 {
		C.uitk_ak_set_app_icon(nil, 0, 0)
		return true
	}
	C.uitk_ak_set_app_icon((*C.uchar)(unsafe.Pointer(&img.Pix[0])),
		C.int(img.Width), C.int(img.Height))
	return true
}

// pickLargest is the biggest square image in the list; nil when there is
// none. The Dock draws at 128 points and more on a Retina display, so
// the largest is always the right one to hand it — unlike Windows,
// which wants an exact size per slot.
func pickLargest(images []*paintengine2d.Image) *paintengine2d.Image {
	var best *paintengine2d.Image
	for _, im := range images {
		if im == nil || im.Width <= 0 || im.Width != im.Height {
			continue
		}
		if best == nil || im.Width > best.Width {
			best = im
		}
	}
	return best
}

func cbool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

var _ WindowFrame = (*akSurface)(nil)
