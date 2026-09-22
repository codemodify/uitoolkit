package widget

import "time"

// Timers is implemented by hosts that can run a callback later on the UI
// thread (app.Window). Widgets use it for auto-repeat (scroll arrows,
// spinners) and animation (indeterminate progress, pulsing default
// buttons).
type Timers interface {
	AfterFunc(d time.Duration, fn func()) (stop func())
}

// After runs fn on c's UI thread after d. The returned stop cancels a
// pending call; it is safe to call more than once. Without a Timers host
// (tests, detached widgets) nothing is scheduled and stop is a no-op.
func After(c Component, d time.Duration, fn func()) (stop func()) {
	if c == nil || fn == nil {
		return func() {}
	}
	if t, ok := c.Host().(Timers); ok && t != nil {
		return t.AfterFunc(d, fn)
	}
	return func() {}
}

// Exposed reports whether any of c can be seen: c and every ancestor are
// visible, and c's box survives being clipped by each ancestor's in turn —
// a scroll view's viewport clips its content, so a widget scrolled out of
// sight is not exposed. An animation uses it to stop asking for frames
// nobody would see.
func Exposed(c Component) bool {
	if c == nil || !c.Visible() {
		return false
	}
	r := c.LocalBounds()
	for {
		if r.Empty() {
			return false
		}
		r = r.Translate(c.Bounds().Min)
		p := c.Parent()
		if p == nil {
			return true
		}
		if !p.Visible() {
			return false
		}
		r = r.Intersect(p.LocalBounds())
		c = p
	}
}
