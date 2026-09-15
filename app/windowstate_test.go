package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

func stateWindow(t *testing.T) (*Application, *Window, *platform.Offscreen) {
	t.Helper()
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 300, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	o, ok := w.Surface().(*platform.Offscreen)
	if !ok {
		t.Fatalf("surface %T", w.Surface())
	}
	return a, w, o
}

// The desktop's activated state, not keyboard focus, drives the active
// look once the desktop reports it.
func TestWindowStateDrivesActiveLook(t *testing.T) {
	a, w, o := stateWindow(t)
	w.SetContent(widgets.NewLabel("x"))
	a.PumpOnce()
	if !w.Active() {
		t.Fatal("a window starts active")
	}
	// Before any state, focus events decide (offscreen tests rely on it).
	w.Inject(platform.Event{Kind: platform.EventFocusOut})
	a.PumpOnce()
	if w.Active() {
		t.Fatal("focus out without a desktop state")
	}
	w.Inject(platform.Event{Kind: platform.EventFocusIn})
	a.PumpOnce()
	if !w.Active() {
		t.Fatal("focus in")
	}
	o.SimulateWindowState(platform.WindowState{Activated: false, Tiled: platform.EdgeLeft})
	a.PumpOnce()
	if w.Active() {
		t.Fatal("not activated by the desktop")
	}
	if w.WindowState().Tiled != platform.EdgeLeft {
		t.Fatalf("state %+v", w.WindowState())
	}
	// Keyboard focus no longer overrides the desktop.
	w.Inject(platform.Event{Kind: platform.EventFocusIn})
	a.PumpOnce()
	if w.Active() {
		t.Fatal("focus in overrode the desktop's activated state")
	}
	o.SimulateWindowState(platform.WindowState{Activated: true})
	a.PumpOnce()
	if !w.Active() {
		t.Fatal("activated")
	}
}

func TestToggleMaximizeAndMinimize(t *testing.T) {
	a, w, o := stateWindow(t)
	w.SetContent(widgets.NewLabel("x"))
	a.PumpOnce()
	w.ToggleMaximize()
	a.PumpOnce()
	if !w.WindowState().Maximized {
		t.Fatal("maximized")
	}
	w.ToggleMaximize()
	a.PumpOnce()
	if w.WindowState().Maximized {
		t.Fatal("restored")
	}
	w.Minimize()
	calls := o.FrameCalls()
	if len(calls.Maximizes) != 2 || !calls.Maximizes[0] || calls.Maximizes[1] || calls.Minimizes != 1 {
		t.Fatalf("calls %+v", calls)
	}
}

// A suspended window neither blinks its caret nor wakes the loop for it.
func TestSuspendedWindowPausesCaret(t *testing.T) {
	a, w, o := stateWindow(t)
	f := widgets.NewTextField("hello", "", nil)
	w.SetContent(widgets.NewColumn(f))
	a.PumpOnce()
	w.RequestFocus(f)
	if !w.wantsBlink() || !a.anyCaret() {
		t.Fatal("a focused field blinks")
	}
	o.SimulateWindowState(platform.WindowState{Suspended: true, Activated: true})
	a.PumpOnce()
	if w.wantsBlink() || a.anyCaret() {
		t.Fatal("a suspended window blinks")
	}
	blink := w.CaretBlink()
	w.toggleBlink()
	if w.CaretBlink() != blink {
		t.Fatal("toggled while suspended")
	}
	o.SimulateWindowState(platform.WindowState{Activated: true})
	a.PumpOnce()
	if !w.wantsBlink() {
		t.Fatal("blinks again once visible")
	}
}
