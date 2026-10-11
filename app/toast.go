package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/widget"
)

// SetToastLayer puts c over the window's content as its layer of passing
// notes ([widgets.ShowToast] is what an application calls; this is the
// seam under it). nil takes the layer down.
//
// It is over everything the window shows and under its popups, and it
// takes no input at all: the pointer goes through it to the content, and
// so does the keyboard. A note nobody can click is the point — it sits
// where the eye goes without taking the place over, and it goes by
// itself. A cover ([Window.SetCover]) is the other way round, and the
// difference between the two is exactly whether what is laid over the
// window is to be used or only read.
//
// The layer gets the *content's* box, not the whole window: a note
// belongs over the pages, not over the title bar, which is the window's
// own row and is still to be read and used while a note is up.
func (w *Window) SetToastLayer(c widget.Component) {
	if w == nil || w.toasts == c {
		return
	}
	old := w.toasts
	w.toasts = c
	if c != nil {
		c.SetHost(w)
	}
	if old != nil && old != c {
		if d, ok := old.(widget.Dismisser); ok {
			d.Dismissed()
		}
	}
	if c != nil && w.laid {
		w.arrangeToasts()
	}
	w.laid = false
	w.fullInvalidate()
}

// ToastLayer is the layer of notes, or nil.
func (w *Window) ToastLayer() widget.Component {
	if w == nil {
		return nil
	}
	return w.toasts
}

// toastBox is where notes go: the content's box.
func (w *Window) toastBox() paintengine2d.Rect {
	if g := w.geom; !g.content.Empty() {
		return g.content
	}
	ww, hh := w.surf.Size()
	return paintengine2d.XYWH(0, 0, float32(ww), float32(hh))
}

func (w *Window) arrangeToasts() {
	if w.toasts == nil {
		return
	}
	b := w.toastBox()
	_ = w.toasts.Measure(layout.Tight(b.Dx(), b.Dy()))
	w.toasts.Arrange(b)
}
