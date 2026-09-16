package platform

import "github.com/codemodify/paintengine2d"

// offscreenDrag is a drag started on an offscreen surface. It stands in
// for the desktop that would carry a real one: the test moves it, drops
// it, and says what the target did with it, and the surface keeps the
// state a backend keeps — what is offered, and what the window last
// answered about a drop.
type offscreenDrag struct {
	payload DragPayload
	running bool
	// mime and action are the window's last answer to the drag over it
	// (DropNegotiator): the type it would read and what it would do.
	// A backend sends these to the source, so a test can read them to
	// see what the source would have been shown.
	mime    string
	allowed DragAction
	action  DragAction
	// ended is the action the drag finished with, once it has, and
	// dropped whether the pointer was let go rather than the drag called
	// off.
	ended   DragAction
	dropped bool
	// carries is whether this desktop can move a window with the drag
	// (SetDragsToplevels); win is the window one is carrying now and dx,
	// dy where in it the pointer sits.
	carries bool
	win     *Offscreen
	dx, dy  int
}

// StartDrag implements [DragSurface]: it begins a drag whose data the
// test reads back with [Offscreen.DragData]. Only one runs at a time, as
// on a real desktop.
func (o *Offscreen) StartDrag(p DragPayload) bool {
	if o.dragOut.running || len(p.Types) == 0 {
		return false
	}
	if p.Actions == DragNone {
		p.Actions = DragCopy
	}
	o.dragOut = offscreenDrag{payload: p, running: true, carries: o.dragOut.carries}
	return true
}

// CancelDrag implements [DragSurface]: the drag ends with nothing taken
// and the pointer never let go, as Escape ends a real one.
func (o *Offscreen) CancelDrag() { o.endDrag(DragNone, false) }

// Dragging implements [DragSurface].
func (o *Offscreen) Dragging() bool { return o.dragOut.running }

// SetDragsToplevels says whether this desktop can carry a window under a
// drag (it can, until a test says otherwise). Turning it off is the
// compositor without xdg-toplevel-drag-v1, where a tear-off runs as a
// plain drag and makes its window at the drop.
func (o *Offscreen) SetDragsToplevels(v bool) { o.dragOut.carries = v }

// DragsToplevels implements [ToplevelDragSurface].
func (o *Offscreen) DragsToplevels() bool { return o.dragOut.carries }

// AttachToplevel implements [ToplevelDragSurface]: win follows the
// pointer for the rest of the drag, dx, dy inside it under the pointer.
func (o *Offscreen) AttachToplevel(win Surface, dx, dy int) bool {
	w, ok := win.(*Offscreen)
	if !o.dragOut.running || !o.dragOut.carries || !ok || w == nil || w.closed {
		return false
	}
	o.dragOut.win, o.dragOut.dx, o.dragOut.dy = w, dx, dy
	return true
}

// DragToplevel is the window the running drag carries, and where in it
// the pointer sits.
func (o *Offscreen) DragToplevel() (Surface, int, int) {
	if o.dragOut.win == nil {
		return nil, 0, 0
	}
	return o.dragOut.win, o.dragOut.dx, o.dragOut.dy
}

// carryToplevel puts the window the drag carries under the pointer, as a
// compositor moving it does. pos is in this surface's own coordinates.
func (o *Offscreen) carryToplevel(pos paintengine2d.Point) {
	if o.dragOut.win == nil {
		return
	}
	o.dragOut.win.Move(o.posX+int(pos.X)-o.dragOut.dx, o.posY+int(pos.Y)-o.dragOut.dy)
}

// AcceptDrag implements [DropNegotiator]: it records what the window says
// it would do with the drag over it, which is what a real backend would
// send back to the source.
func (o *Offscreen) AcceptDrag(mime string, allowed, a DragAction) {
	o.dragOut.mime, o.dragOut.allowed, o.dragOut.action = mime, allowed, a
}

// DragAllowed is every action the window said a drop where the pointer
// is could perform, which is what a compositor picks from.
func (o *Offscreen) DragAllowed() DragAction { return o.dragOut.allowed }

// DragAccepted is the window's last answer about a drag over it: the type
// it would read it as, and what it would do. An empty type means it takes
// nothing where the pointer is.
func (o *Offscreen) DragAccepted() (string, DragAction) {
	return o.dragOut.mime, o.dragOut.action
}

// DragOffer is the running drag's payload, and whether one is running.
func (o *Offscreen) DragOffer() (DragPayload, bool) {
	return o.dragOut.payload, o.dragOut.running
}

// DragData reads the running drag's data as mime, the way a target that
// took the drop would.
func (o *Offscreen) DragData(mime string) ([]byte, bool) {
	if !o.dragOut.running || o.dragOut.payload.Data == nil {
		return nil, false
	}
	return o.dragOut.payload.Data(mime)
}

// DragIcon is the picture the running drag would show under the pointer,
// and where in it the pointer sits.
func (o *Offscreen) DragIcon() (*paintengine2d.Image, paintengine2d.Point) {
	return o.dragOut.payload.Icon, o.dragOut.payload.Hotspot
}

// DragEnded is the action the last drag finished with: DragNone when it
// was cancelled or nothing took it.
func (o *Offscreen) DragEnded() DragAction { return o.dragOut.ended }

// SimulateDragOver moves the running drag over this surface at pos, as a
// compositor delivering the drag back to the window it started in does.
// The window answers it, and [Offscreen.DragAccepted] is that answer.
func (o *Offscreen) SimulateDragOver(pos paintengine2d.Point) {
	if !o.dragOut.running {
		return
	}
	o.carryToplevel(pos)
	o.Inject(o.dragEvent(EventDragMotion, pos))
}

// SimulateDragLeave takes the running drag off this surface.
func (o *Offscreen) SimulateDragLeave() {
	if !o.dragOut.running {
		return
	}
	o.Inject(Event{Kind: EventDragLeave})
}

// SimulateDragDrop drops the running drag on this surface at pos. The
// drag ends when the window finishes the drop, with the action it
// performed — the round trip a backend makes, so a test sees the source
// told what the target did.
func (o *Offscreen) SimulateDragDrop(pos paintengine2d.Point) {
	if !o.dragOut.running {
		return
	}
	o.carryToplevel(pos)
	o.dragOut.dropped = true
	o.Inject(o.dragEvent(EventDrop, pos))
}

// SimulateDragRelease is the pointer let go where nothing takes the drop:
// the drag is over, with nothing performed, but it was dropped and not
// called off — which is what leaves a torn-off window on the desktop.
func (o *Offscreen) SimulateDragRelease() {
	if !o.dragOut.running {
		return
	}
	o.endDrag(DragNone, true)
}

// dropFinished ends a drag of this surface's own once the window has
// finished the drop it made: the source hears the action the target
// performed, as wl_data_source.dnd_finished and XdndFinished report it.
func (o *Offscreen) dropFinished(taken bool) {
	if !o.dragOut.running {
		return
	}
	a := o.dragOut.action
	if !taken {
		a = DragNone
	}
	o.endDrag(a, true)
}

// dragEvent is one drag event carrying what the running drag offers.
func (o *Offscreen) dragEvent(kind EventKind, pos paintengine2d.Point) Event {
	p := o.dragOut.payload
	return Event{
		Kind:    kind,
		Pos:     pos,
		Mimes:   p.Types,
		Actions: p.Actions,
		Action:  p.Preferred,
	}
}

// EndDrag ends the running drag with the action the target performed, as
// a backend does when the desktop reports the drop finished.
func (o *Offscreen) EndDrag(a DragAction) { o.endDrag(a, true) }

// endDrag ends the drag, saying whether the pointer was let go (a drop,
// whatever came of it) or the drag was called off.
func (o *Offscreen) endDrag(a DragAction, dropped bool) {
	if !o.dragOut.running {
		return
	}
	o.dragOut.running = false
	o.dragOut.ended = a
	o.dragOut.dropped = dropped
	o.dragOut.win = nil
	o.Inject(Event{Kind: EventDragEnd, Action: a, Dropped: dropped})
}

// DragDropped reports whether the last drag ended with the pointer let go
// rather than called off.
func (o *Offscreen) DragDropped() bool { return o.dragOut.dropped }
