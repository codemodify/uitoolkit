package app

// OnActiveChange is called when the window takes or loses the keyboard
// focus, with whether it now has it.
//
// [Window.Active] has always said *whether* a window has the focus, and
// nothing said *when* that changed: `setActive` repaints the window and
// called nothing else. So an application whose interface depends on the
// focus — a mail client that shows its logo in a seal while unread mail
// waits and puts the plain logo back when the window is looked at — had to
// check `widget.WindowActive` on every paint, which works only for as long
// as the toolkit happens to repaint on the change, and costs a check per
// frame to find out about an event that happens twice a minute.
//
// It is one callback per window, like [Window.OnMove] and
// [Window.OnLockKeys]: the window's own code, not a subscription.
//
// It is called on the change and not on the state, so an application does
// not have to remember what it was last told. The first call is the first
// change after it is installed: a window that already has the focus does
// not call back for having it, and [Window.Active] is there to ask.
//
// Qt's QEvent::WindowActivate, GTK's notify::is-active, Win32's
// WM_ACTIVATE.
func (w *Window) OnActiveChange(fn func(active bool)) {
	if w == nil {
		return
	}
	w.onActive = fn
}
