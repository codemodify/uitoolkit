package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// Window.Active said whether a window has the focus; nothing said when that
// changed. setActive repainted the window and called nothing else, so an
// application whose interface follows the focus — a mail client showing its
// logo in a seal while unread mail waits — had to check on every paint, which
// works only while the toolkit happens to repaint on the change.

func activeWindow(t *testing.T) (*Application, *Window) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("something"))
	a.PumpOnce()
	return a, w
}

func TestOnActiveChangeReportsTakingAndLosingTheFocus(t *testing.T) {
	_, w := activeWindow(t)
	var got []bool
	w.OnActiveChange(func(active bool) { got = append(got, active) })

	// Whatever the window starts as, installing the callback says nothing.
	if len(got) != 0 {
		t.Fatalf("installing the callback reported %v", got)
	}
	was := w.Active()
	w.setActive(!was)
	w.setActive(was)
	if len(got) != 2 || got[0] != !was || got[1] != was {
		t.Fatalf("reported %v, want [%v %v]", got, !was, was)
	}
	// And what it is told matches what the window says it is.
	w.OnActiveChange(func(active bool) {
		if active != w.Active() {
			t.Errorf("told active=%v while Window.Active() says %v", active, w.Active())
		}
	})
	w.setActive(!was)
}

// Only on the change: the same state twice is one event, so an application
// does not have to remember what it was last told.
func TestOnActiveChangeIsOnlyCalledOnAChange(t *testing.T) {
	_, w := activeWindow(t)
	was := w.Active()
	n := 0
	w.OnActiveChange(func(bool) { n++ })
	w.setActive(was)
	w.setActive(was)
	if n != 0 {
		t.Errorf("repeating the state it already had called back %d times", n)
	}
	w.setActive(!was)
	w.setActive(!was)
	if n != 1 {
		t.Errorf("changing once and repeating it called back %d times, want 1", n)
	}
}

// The desktop's own focus events reach it, which is the path that matters:
// an application installs the callback and hears about alt-tab.
func TestOnActiveChangeFollowsTheDesktopsFocusEvents(t *testing.T) {
	_, w := activeWindow(t)
	if w.stateKnown {
		t.Skip("this backend reports an activated state, so focus events are not the path")
	}
	var got []bool
	w.OnActiveChange(func(active bool) { got = append(got, active) })
	w.dispatch(platform.Event{Kind: platform.EventFocusOut})
	w.dispatch(platform.Event{Kind: platform.EventFocusIn})
	if len(got) != 2 || got[0] || !got[1] {
		t.Errorf("focus out then in reported %v, want [false true]", got)
	}
}
