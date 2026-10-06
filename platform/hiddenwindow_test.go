package platform

import "testing"

// A window the application opens hidden.
//
// WindowOptions had no way to say it, and Surface.Visible answers the
// surface's *current* visibility rather than whether a new window will appear
// on its first frame: a fresh surface has not been mapped yet, so Visible()
// is false whether it is about to show or not. An application guarding its
// Hide with `if win.Visible()` therefore skipped it, and the panel appeared on
// the next presentation — so a player tracked the desired visibility itself
// and called Hide unconditionally.

func TestAHiddenWindowStaysHiddenThroughItsFirstPresent(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 200, Height: 120, Hidden: true, Headless: true})
	defer o.Close()
	if o.Visible() {
		t.Error("a window asked for hidden reports itself visible")
	}
	if err := o.Present(nil); err != nil {
		t.Fatalf("present: %v", err)
	}
	if o.Visible() {
		t.Error("the first present showed a window asked for hidden")
	}
	// And Show is still the way to show it.
	o.Show()
	if !o.Visible() {
		t.Error("Show did not show it")
	}
}

// Without the option a window is visible once it has been presented, which is
// what every window did before and must still do.
func TestAnOrdinaryWindowShowsItself(t *testing.T) {
	o := NewOffscreen(WindowOptions{Width: 200, Height: 120, Headless: true})
	defer o.Close()
	if err := o.Present(nil); err != nil {
		t.Fatalf("present: %v", err)
	}
	if !o.Visible() {
		t.Error("an ordinary window is not visible after its first present")
	}
}
