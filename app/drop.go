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

// dragMotion follows a drag over the window, telling the target under it
// (and the one it left) for their highlight, and answering the drag's
// source with what a drop here would do. The answer is not optional:
// XDND's source waits for an XdndStatus for every XdndPosition, and on
// Wayland an offer that is never accepted drops nothing.
func (w *Window) dragMotion(ev platform.Event) {
	t, c := w.dropTarget(ev.Pos, ev.Mimes)
	if c != w.dropOver {
		w.dragLeave()
		w.dropOver = c
	}
	if h, ok := c.(widget.DropHover); ok {
		h.DragOver(local(c, ev.Pos))
	}
	w.acceptDrag(t, c, ev)
}

// acceptDrag tells the drag's source whether this window takes a drop
// where the pointer is and what it would do with it, and remembers the
// action for the drop itself.
func (w *Window) acceptDrag(t widget.DropTarget, c widget.Component, ev platform.Event) {
	w.dropAction = platform.DragNone
	mime, allowed := "", platform.DragNone
	if t != nil {
		allowed = allowedDropActions(c, offeredActions(ev))
		if a := platform.NegotiateDragAction(offeredActions(ev), ev.Action, allowed); a != platform.DragNone {
			w.dropAction = a
			mime = widget.PickDropMime(t.DropTypes(), ev.Mimes)
		}
	}
	if n, ok := w.surf.(platform.DropNegotiator); ok {
		n.AcceptDrag(mime, allowed, w.dropAction)
	}
}

// offeredActions is what the drag's source allows. A source that names
// none means a copy: every drag did before the protocols grew actions.
func offeredActions(ev platform.Event) platform.DragAction {
	if ev.Actions == platform.DragNone {
		return platform.DragCopy
	}
	return ev.Actions
}

func (w *Window) dragLeave() {
	if h, ok := w.dropOver.(widget.DropHover); ok {
		h.DragLeave()
	}
	w.dropOver = nil
	w.dropAction = platform.DragNone
}

// drop hands a drop to the target under it, reading the data in the best
// type both sides know, and tells the source whether it was taken.
func (w *Window) drop(ev platform.Event) {
	w.dragLeave()
	rcv, _ := w.surf.(platform.DropReceiver)
	t, c := w.dropTarget(ev.Pos, ev.Mimes)
	if t == nil {
		w.finishDrop(rcv, "", false, platform.DragNone)
		return
	}
	action := dropActionFor(c, offeredActions(ev), ev.Action)
	mime := widget.PickDropMime(t.DropTypes(), ev.Mimes)
	if action == platform.DragNone {
		w.finishDrop(rcv, "", false, platform.DragNone)
		return
	}
	// A drag this application started is served from the drag itself:
	// the bytes never go through the desktop, and the payload — which
	// has no type at all — reaches the target with them.
	if run := w.dragRun(); run != nil && !run.ended {
		data, _ := run.d.Read(mime)
		taken := t.Drop(run.d.DropEvent(local(c, ev.Pos), mime, data, action))
		w.finishDrop(rcv, mime, taken, action)
		return
	}
	if rcv == nil {
		w.finishDrop(rcv, "", false, platform.DragNone)
		return
	}
	data, ok := rcv.ReceiveDrop(mime)
	e := widget.NewDropEvent(local(c, ev.Pos), mime, data)
	e.Action = action
	taken := ok && t.Drop(e)
	w.finishDrop(rcv, mime, taken, action)
}

// finishDrop closes the protocol with the drag's source: the drop was
// taken, with the action that ran, or it was not. The action goes first
// because that is what the source is told the target did (XDND sends it
// in XdndFinished, Wayland in wl_data_source.action).
func (w *Window) finishDrop(rcv platform.DropReceiver, mime string, taken bool, action platform.DragAction) {
	if !taken {
		mime, action = "", platform.DragNone
	}
	w.dropAction = action
	if n, ok := w.surf.(platform.DropNegotiator); ok {
		n.AcceptDrag(mime, action, action)
	}
	if rcv != nil {
		rcv.FinishDrop(taken)
	}
}
