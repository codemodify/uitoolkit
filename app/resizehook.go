package app

// OnResize runs fn on the UI goroutine whenever the window is about to be
// laid out at a logical size it has not been laid out at before: the first
// layout, the desktop resizing or maximizing it, a [Window.SetSize], a
// display scale that changes what the window measures. width and height
// are [Window.Size]'s logical pixels.
//
// It runs before the layout that size brings, so content can change its
// proportions there and be arranged at the new size in the same frame (a
// sidebar that keeps its share of the window, as Settings' theme browser
// does). fn may change the content, or call [Window.SetContent]; the layout
// that follows takes whatever the window then holds. Resizes the window
// system sends while a layout is pending coalesce into the one call.
//
// Qt's resizeEvent and GTK's size-allocate are the same seam. remove
// unregisters fn.
func (w *Window) OnResize(fn func(width, height int)) (remove func()) {
	if w == nil || fn == nil {
		return func() {}
	}
	p := &fn
	w.resizeHooks = append(w.resizeHooks, p)
	return func() {
		for i, q := range w.resizeHooks {
			if q == p {
				w.resizeHooks = append(w.resizeHooks[:i:i], w.resizeHooks[i+1:]...)
				return
			}
		}
	}
}

// noteResize runs the OnResize hooks when the size the window is about to
// be laid out at is not the one they last heard.
func (w *Window) noteResize() {
	lw, lh := w.Size()
	if lw < 1 || lh < 1 || [2]int{lw, lh} == w.resizeSeen {
		return
	}
	w.resizeSeen = [2]int{lw, lh}
	for _, fn := range append([]*func(int, int){}, w.resizeHooks...) {
		(*fn)(lw, lh)
	}
}
