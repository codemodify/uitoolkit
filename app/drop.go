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
	mime := ""
	if t != nil {
		mime = widget.PickDropMime(t.DropTypes(), ev.Mimes)
	}
	hoverDrag(c, local(c, ev.Pos), mime)
	w.acceptDrag(t, c, ev)
}

// hoverDrag tells the component under the pointer that a drag is over it,
// with the type a drop there would be read in where it asks for that
// ([widget.DropHoverMime]): a tab strip marks where a torn-off tab would
// land and where a file would land differently.
func hoverDrag(c widget.Component, pos paintengine2d.Point, mime string) {
	switch h := c.(type) {
	case widget.DropHoverMime:
		h.DragOverMime(pos, mime)
	case widget.DropHover:
		h.DragOver(pos)
	}
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
			// Say that the user may be asked, where asking is the
			// target's to offer: a Wayland compositor answers with ask
			// only for a target that named it (dropask.go).
			if wantsDropAsk(ev, allowed) {
				allowed |= platform.DragAsk
			}
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
	// The drag left the choice to the user: the menu goes up at the drop
	// point and the drop waits for it. Nothing has been read or handed
	// over yet, so a dismissed menu is a clean refusal (dropask.go).
	if choices := dropAsks(ev, allowedDropActions(c, offeredActions(ev))); choices != platform.DragNone {
		asked := w.askDropAction(ev.Pos, choices, func(a platform.DragAction) {
			if a == platform.DragNone {
				w.finishDrop(rcv, "", false, platform.DragNone)
				return
			}
			w.deliverDrop(rcv, t, c, ev, mime, a)
		})
		if asked {
			return
		}
	}
	w.deliverDrop(rcv, t, c, ev, mime, action)
}

// deliverDrop hands the drop to the target with the action settled, reads
// the data it takes, and closes the protocol with the source.
func (w *Window) deliverDrop(rcv platform.DropReceiver, t widget.DropTarget, c widget.Component, ev platform.Event, mime string, action platform.DragAction) {
	// A drag this application started is served from the drag itself:
	// the bytes never go through the desktop, and the payload — which
	// has no type at all — reaches the target with them.
	if run := w.dragRun(); run != nil && !run.ended {
		data, _ := run.d.Read(mime)
		taken := t.Drop(run.d.DropEvent(local(c, ev.Pos), mime, data, action))
		w.finishDrop(rcv, mime, taken, action)
		if run.win == nil || run.win.Closed() {
			// Taking the drop closed the window the drag came from: a
			// floating dock panel docked back into this one, which is
			// what the protocol asks a client to do with a window it
			// snaps into another (xdg-toplevel-drag). That window will
			// never report the end of the drag, so the drop is its end.
			run.dropped = true
			if !taken {
				action = platform.DragNone
			}
			run.finish(action)
		}
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
