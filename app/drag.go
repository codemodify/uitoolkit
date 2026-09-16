package app

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// dragGesture is a press that could become a drag: where it landed and
// what it landed on. Every press arms one — which component would start a
// drag is only asked once the pointer has moved past the threshold, so a
// plain click costs nothing.
type dragGesture struct {
	armed bool
	pos   paintengine2d.Point
	c     widget.Component
}

// dragRun is a drag this application started: what it carries, the window
// it began in, and how it is being moved. The desktop moves it when the
// surface can (which is what lets it reach another application); a drag
// marked local, and one on a backend with no drag of its own, the toolkit
// moves itself between its own windows.
type dragRun struct {
	d     *widget.Drag
	win   *Window
	local bool
	ended bool
	// over is the component a local drag is above now, and the window it
	// is in: its highlight has to be taken back when the drag leaves.
	over    widget.Component
	overWin *Window
	// tear is the window half of a tear-off drag, torn the window it
	// opened (nil until it has one: a desktop that cannot carry a window
	// makes it at the drop instead), and dropped whether the pointer was
	// let go rather than the drag called off — which is the difference
	// between a torn-off window left on the desktop and one that never
	// should have existed (tearoff.go).
	tear    *widget.TearOff
	torn    widget.TearOffWindow
	dropped bool
}

// StartDrag begins a drag of d. It implements [widget.DragHost], so a
// component starts one with widget.StartDrag(c, d) — and anything a press
// arms comes through here too. It reports false when no drag could start:
// nothing to offer, or one is already running (a desktop has one pointer,
// so it can only carry one drag).
func (w *Window) StartDrag(d *widget.Drag) bool { return w.startDrag(d, nil) }

// startDrag begins a drag, with the window half of a tear-off when there
// is one (tearoff.go).
func (w *Window) startDrag(d *widget.Drag, tear *widget.TearOff) bool {
	if w == nil || w.Closed() || d == nil || len(d.Types) == 0 || w.app == nil {
		return false
	}
	if w.app.drag != nil {
		return false
	}
	w.dragArm = dragGesture{}
	run := &dragRun{d: d, win: w, tear: tear}
	if !d.Local {
		if ds, ok := w.surf.(platform.DragSurface); ok {
			w.app.drag = run
			if ds.StartDrag(platform.DragPayload{
				Types:     d.Types,
				Data:      d.Read,
				Icon:      d.Image,
				Hotspot:   d.Hotspot,
				Actions:   d.Allowed(),
				Preferred: d.Preferred,
				// A drag that may carry a window has to say so before it
				// starts: Wayland's toplevel-drag object can be made at
				// no other moment.
				Toplevel: tear != nil,
			}) {
				// The desktop has the pointer now: the release, and every
				// move until the drop, go to it and not to the widget the
				// press landed on.
				w.releasePointer()
				run.carry()
				return true
			}
			w.app.drag = nil
		}
	}
	// A drag marked local, or a backend with none of its own (offscreen):
	// the toolkit moves it between this application's windows, which is
	// all reordering a list ever needed.
	run.local = true
	w.app.drag = run
	w.releasePointer()
	run.carry()
	return true
}

// CancelDrag stops the drag this window started, as Escape does. The
// source is told nothing was taken.
func (w *Window) CancelDrag() {
	run := w.dragRun()
	if run == nil {
		return
	}
	if !run.local {
		if ds, ok := w.surf.(platform.DragSurface); ok {
			ds.CancelDrag()
			return // the backend answers with EventDragEnd
		}
	}
	run.leave()
	run.finish(platform.DragNone)
}

// Dragging reports whether this application is dragging something out of
// one of its windows now.
func (w *Window) Dragging() bool { return w.dragRun() != nil }

// dragRun is the drag running in this application, if any. It is kept on
// the application, not the window: a drag started in one window is
// dropped on another, and both have to find it.
func (w *Window) dragRun() *dragRun {
	if w == nil || w.app == nil {
		return nil
	}
	return w.app.drag
}

// armDrag remembers a press that could become a drag.
func (w *Window) armDrag(c widget.Component, pos paintengine2d.Point) {
	w.dragArm = dragGesture{armed: c != nil, pos: pos, c: c}
}

// dragPointerMove is the pointer moving while a drag is armed or running.
// It reports whether it took the event: a running drag owns the pointer,
// and a press that becomes a drag must not also reach the widget as a
// move (a list would extend its selection under the drag).
func (w *Window) dragPointerMove(ev platform.Event) bool {
	if run := w.dragRun(); run != nil {
		if run.local {
			run.motion(w, ev.Pos)
		}
		// A drag the desktop moves gives this window no pointer at all
		// until it ends; anything that arrives is not the widget's.
		return true
	}
	if !w.dragArm.armed {
		return false
	}
	if near(ev.Pos, w.dragArm.pos, w.dragThreshold()) {
		return false
	}
	c, pos := w.dragArm.c, w.dragArm.pos
	w.dragArm = dragGesture{}
	return w.startDragFrom(c, pos)
}

// dragPointerUp is the button going up. It reports whether a running
// local drag took it — that release is the drop.
func (w *Window) dragPointerUp(ev platform.Event) bool {
	w.dragArm = dragGesture{}
	run := w.dragRun()
	if run == nil || !run.local {
		return run != nil
	}
	run.drop(w, ev.Pos)
	return true
}

// dragKey cancels a local drag on Escape, and swallows every other key
// while one runs. A drag the desktop moves gets no keys here at all:
// Wayland's compositor cancels it itself, and the X11 source cancels on
// its own keyboard grab.
func (w *Window) dragKey(ev platform.Event) bool {
	run := w.dragRun()
	if run == nil || !run.local {
		return false
	}
	if ev.Key == platform.KeyEscape {
		run.leave()
		run.finish(platform.DragNone)
	}
	return true
}

// dragEnded is the desktop reporting how a drag this window started
// finished: the action the target performed, or none when it was
// cancelled or refused, and whether the pointer was let go at all.
func (w *Window) dragEnded(action platform.DragAction, dropped bool) {
	if run := w.dragRun(); run != nil {
		run.dropped = run.dropped || dropped
		run.finish(action)
	}
}

// startDragFrom asks the pressed component, then its ancestors, what a
// drag from pos carries. Walking up is what lets a row start the drag its
// list defines, without every row knowing about drags.
func (w *Window) startDragFrom(c widget.Component, pos paintengine2d.Point) bool {
	for ; c != nil; c = c.Parent() {
		src, ok := c.(widget.DragSource)
		if !ok || !c.Enabled() {
			continue
		}
		d := src.DragAt(local(c, pos))
		if d == nil {
			continue
		}
		if d.Source == nil {
			d.Source = c
		}
		return w.StartDrag(d)
	}
	return false
}

// motion follows a local drag over this application's windows: the target
// under the pointer is told, and the one the drag left, and the pointer
// takes the shape of what a drop would do.
func (r *dragRun) motion(w *Window, pos paintengine2d.Point) {
	t, c := w.dropTarget(pos, r.d.Types)
	if c != r.over || w != r.overWin {
		r.leave()
		r.over, r.overWin = c, w
	}
	action, mime := platform.DragNone, ""
	if t != nil {
		action = dropActionFor(c, r.d.Allowed(), r.d.Preferred)
		mime = widget.PickDropMime(t.DropTypes(), r.d.Types)
	}
	hoverDrag(c, local(c, pos), mime)
	w.SetCursor(platform.DragCursor(action))
}

// leave takes back the highlight of the target a local drag was over.
func (r *dragRun) leave() {
	if h, ok := r.over.(widget.DropHover); ok {
		h.DragLeave()
	}
	r.over, r.overWin = nil, nil
}

// drop finishes a local drag on the target under the pointer and tells
// the source what became of it.
func (r *dragRun) drop(w *Window, pos paintengine2d.Point) {
	r.leave()
	r.dropped = true
	action := platform.DragNone
	if t, c := w.dropTarget(pos, r.d.Types); t != nil {
		act := dropActionFor(c, r.d.Allowed(), r.d.Preferred)
		if act != platform.DragNone {
			mime := widget.PickDropMime(t.DropTypes(), r.d.Types)
			data, _ := r.d.Read(mime)
			if t.Drop(r.d.DropEvent(local(c, pos), mime, data, act)) {
				action = act
			}
		}
	}
	r.finish(action)
}

// finish ends the drag once and tells its source the action that ran, and
// a tear-off what became of its window.
func (r *dragRun) finish(action platform.DragAction) {
	if r == nil || r.ended {
		return
	}
	r.ended = true
	if r.win != nil {
		if r.win.app != nil && r.win.app.drag == r {
			r.win.app.drag = nil
		}
		r.win.SetCursor(platform.CursorDefault)
	}
	if r.d != nil && r.d.Done != nil {
		r.d.Done(action)
	}
	r.finishTear(action)
}

// allowedDropActions is every action the component under the pointer
// would take: what it says when it has an opinion ([widget.DropActions]),
// a copy otherwise — the action every target is assumed to manage.
func allowedDropActions(c widget.Component, offered platform.DragAction) platform.DragAction {
	if da, ok := c.(widget.DropActions); ok {
		return da.DropActionFor(offered)
	}
	return platform.DragCopy
}

// dropActionFor is the one action a drop on c would perform.
func dropActionFor(c widget.Component, offered, preferred platform.DragAction) platform.DragAction {
	return platform.NegotiateDragAction(offered, preferred, allowedDropActions(c, offered))
}
