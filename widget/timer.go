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
