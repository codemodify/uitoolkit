package platform

// MoveEventSurface is a surface that reports its own movement
// ([EventMove]).
//
// X11 and Windows and macOS can: the window system tells the client where
// it put the window, and the backend passes that on. Wayland cannot — a
// toplevel has no position in that protocol and a client is never told
// one — so an application that wants to know should ask before it builds
// a design around being told.
//
// This answers whether *this surface* will report it, which is a question
// about the backend rather than about the window.
type MoveEventSurface interface {
	SendsMoveEvents() bool
}

// SendsMoveEvents reports whether s will send [EventMove] when the
// desktop moves it. A surface that says nothing is taken not to.
func SendsMoveEvents(s Surface) bool {
	m, ok := s.(MoveEventSurface)
	return ok && m.SendsMoveEvents()
}
