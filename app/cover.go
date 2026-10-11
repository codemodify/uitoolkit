package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// SetCover lays c over the whole window inside its frame — the title bar
// as well as the content — and takes it down again with nil.
//
// It is the layer nothing else could be: an overlay is laid over the
// content alone, because a modal dialog dimming its own title bar would
// be wrong (the window can still be moved and closed, as with a macOS
// sheet); a popup is a surface of its own on Wayland and X11, with a
// shadow and a window system that slides it about; and every component
// is clipped to its own box. So there was no way to draw over a window
// as one picture, which is what a transition, a lock screen or a splash
// is. A card that turns the window over to its back has to turn the
// title bar with it, or it is not the window that turned.
//
// What it covers, it owns. The pointer and the keyboard go to the cover
// and to nothing under it — you cannot click what you cannot see, and
// that includes the title bar's own controls — while the frame around it
// stays the window's: a framed window can still be resized from its
// edges and moved by whatever the desktop offers, and the resize band in
// the shadow margin is outside the cover's box. Popups and tooltips stay
// above it, so a cover that holds a widget with a menu still works.
//
// It does not dim, paint or inset anything of itself: the cover is
// exactly what c paints in the box it is given. A cover that is meant to
// look like the window under it can paint the look's own chrome —
// [style.DrawDecorationOf] draws a caption band's ground, which is what
// the toolkit's own frame paints with.
//
// Escape does not take it down. A dialog and a popup are the user's to
// dismiss; a cover is the application's, and a lock screen the user can
// press Escape out of is not a lock screen.
func (w *Window) SetCover(c widget.Component) {
	if w == nil || w.cover == c {
		return
	}
	old := w.cover
	w.cover = c
	if c != nil {
		c.SetHost(w)
	}
	if old != nil && old != c {
		if d, ok := old.(widget.Dismisser); ok {
			d.Dismissed()
		}
	}
	// The focus may have been inside what just went away, or has to leave
	// what is now behind the cover.
	w.dropDeadRefs()
	if c != nil && w.laid {
		w.arrangeCover()
	}
	w.laid = false
	w.fullInvalidate()
}

// Cover is the layer over the window, or nil.
func (w *Window) Cover() widget.Component {
	if w == nil {
		return nil
	}
	return w.cover
}

// coverBox is the window inside its frame: the title bar and the content
// as one box. For a window whose frame is the desktop's it is the whole
// surface, which is the same thing — everything the application owns.
func (w *Window) coverBox() paintengine2d.Rect {
	if g := w.geom; g.framed {
		return g.inner
	}
	ww, hh := w.surf.Size()
	return paintengine2d.XYWH(0, 0, float32(ww), float32(hh))
}

// arrangeCover gives the cover the window's own box.
func (w *Window) arrangeCover() {
	if w.cover == nil {
		return
	}
	b := w.coverBox()
	_ = w.cover.Measure(layout.Tight(b.Dx(), b.Dy()))
	w.cover.Arrange(b)
}

// containingLayer is the layer the keyboard may not leave: the cover
// where one is up, else the modal dialog on top. Nothing outside it can
// be typed into, because nothing outside it can be seen or reached.
func (w *Window) containingLayer() widget.Component {
	if w.cover != nil {
		return w.cover
	}
	return w.overlay
}
