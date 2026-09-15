package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// dropTarget is the component under pos that takes one of the offered
// types: the one hit, or its nearest ancestor that does.
func (w *Window) dropTarget(pos paintengine2d.Point, offered []string) (widget.DropTarget, widget.Component) {
	for c := w.hitContent(pos); c != nil; c = c.Parent() {
		if t, ok := c.(widget.DropTarget); ok && c.Enabled() && widget.PickDropMime(t.DropTypes(), offered) != "" {
			return t, c
		}
	}
	return nil, nil
}

// dragMotion follows a drag from another app over the window, telling the
// target under it (and the one it left) for their highlight.
func (w *Window) dragMotion(ev platform.Event) {
	_, c := w.dropTarget(ev.Pos, ev.Mimes)
	if c != w.dropOver {
		w.dragLeave()
		w.dropOver = c
	}
	if h, ok := c.(widget.DropHover); ok {
		h.DragOver(local(c, ev.Pos))
	}
}

func (w *Window) dragLeave() {
	if h, ok := w.dropOver.(widget.DropHover); ok {
		h.DragLeave()
	}
	w.dropOver = nil
}

// drop hands a drop from another app to the target under it, reading the
// data in the best type both sides know, and tells the source whether it
// was taken.
func (w *Window) drop(ev platform.Event) {
	w.dragLeave()
	rcv, ok := w.surf.(platform.DropReceiver)
	if !ok {
		return
	}
	t, c := w.dropTarget(ev.Pos, ev.Mimes)
	if t == nil {
		rcv.FinishDrop(false)
		return
	}
	mime := widget.PickDropMime(t.DropTypes(), ev.Mimes)
	data, ok := rcv.ReceiveDrop(mime)
	taken := ok && t.Drop(widget.NewDropEvent(local(c, ev.Pos), mime, data))
	rcv.FinishDrop(taken)
}
