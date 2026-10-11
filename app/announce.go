package app

// Announce says text to assistive technology, politely: it is spoken
// after whatever is being read now, and nothing is cut off.
//
// It is for what no control holds and no focus move would speak — a
// passing note that something was copied, a background job that
// finished, a vault that locked itself. A screen reader hears about the
// keyboard and about what changes under it, so a message that appears in
// a corner while the focus stays where it was reaches nobody unless the
// window says it. This is the toolkit's word for ARIA's polite live
// region, and [widgets.ShowToast] says every note through it.
//
// Nothing is spoken when no assistive technology is listening: the
// announcements are held for the adapter that drains them (the AT-SPI
// bridge, after each frame) and dropped when there is none. Say what a
// person would want read, not what a log would want written.
func (w *Window) Announce(text string) {
	if w == nil || w.app == nil || text == "" {
		return
	}
	w.app.announce(text)
}

// announce queues one announcement for the accessibility adapter.
//
// The cap is a backstop, not a policy: a window that announces in a loop
// with nothing draining them must not grow a queue for ever, and the
// oldest go first because the newest is what somebody would want to
// hear.
const maxAnnouncements = 32

func (a *Application) announce(text string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.announcements = append(a.announcements, text)
	if n := len(a.announcements) - maxAnnouncements; n > 0 {
		a.announcements = a.announcements[n:]
	}
}

// takeAnnouncements is what has been said since the last call, and
// empties the queue. The accessibility adapter calls it after each frame;
// with no adapter running nothing calls it, and the queue is dropped at
// its cap.
func (a *Application) takeAnnouncements() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.announcements) == 0 {
		return nil
	}
	out := a.announcements
	a.announcements = nil
	return out
}
