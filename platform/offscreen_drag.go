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
	mime   string
	action DragAction
	// ended is the action the drag finished with, once it has.
	ended DragAction
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
	o.dragOut = offscreenDrag{payload: p, running: true}
	return true
}

// CancelDrag implements [DragSurface]: the drag ends with nothing taken,
// as Escape ends a real one.
func (o *Offscreen) CancelDrag() { o.EndDrag(DragNone) }

// Dragging implements [DragSurface].
func (o *Offscreen) Dragging() bool { return o.dragOut.running }

// AcceptDrag implements [DropNegotiator]: it records what the window says
// it would do with the drag over it, which is what a real backend would
// send back to the source.
func (o *Offscreen) AcceptDrag(mime string, a DragAction) {
	o.dragOut.mime, o.dragOut.action = mime, a
}

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
	o.Inject(o.dragEvent(EventDrop, pos))
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
	o.EndDrag(a)
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
func (o *Offscreen) EndDrag(a DragAction) {
	if !o.dragOut.running {
		return
	}
	o.dragOut.running = false
	o.dragOut.ended = a
	o.Inject(Event{Kind: EventDragEnd, Action: a})
}
