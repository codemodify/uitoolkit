package platform

// SendsMoveEvents: the simulated desktop reports a move, so anything
// built on [EventMove] can be tested without one ([MoveEventSurface]).
func (o *Offscreen) SendsMoveEvents() bool { return true }

// SimulateDesktopMove is the desktop moving the window — a drag by the
// user, a window manager placing it — as opposed to [Offscreen.Move],
// which is the application asking. It records the new position and sends
// [EventMove], which is what a real backend does when it is told where
// its window went.
func (o *Offscreen) SimulateDesktopMove(x, y int) {
	if o == nil || o.closed {
		return
	}
	o.posX, o.posY = DevicePosition(x, o.Scale()), DevicePosition(y, o.Scale())
	o.queue = append(o.queue, Event{Kind: EventMove, Width: x, Height: y})
}
