package rack

import (
	"testing"
	"time"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
)

// reconciled waits for the rack to notice that a move it asked for never
// happened: the pane comes back to where the window actually is and
// stops being bonded. Nothing calls Follow, and the application has no
// pulse of its own — a wake-up the Desk arranged is the only thing that
// can run a pass.
func reconciled(t *testing.T, a *app.Application, d *Desk, i int, within time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(within)
	for {
		a.PumpOnce()
		if d.Rack.Pane(i).To == -1 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// **A satellite shown again after being moved behind the rack's back is
// reconciled.** Desk.Show places the window before showing it and sets
// the patience for that request — and armed nothing to look when the
// patience ran out, so a window manager that refused the move left the
// satellite bonded to a position it was not in until the application
// called Follow itself. A music player found it with the window hidden,
// the desktop moved under it, and the move refused.
func TestAReshownSatelliteIsReconciledWithoutBeingAsked(t *testing.T) {
	a, d, _, sat, _, si, _ := coalesceRig(t)
	d.SetClock(nil)
	so := sat.Surface().(*platform.Offscreen)

	d.Show(si, false)
	a.PumpOnce()
	// The desktop puts it somewhere else while it is hidden, says
	// nothing about it, and will refuse to move it back.
	so.SimulateDesktopMove(700, 700)
	so.SetMoveHonoured(false)
	d.Show(si, true)

	if !reconciled(t, a, d, si, 3*time.Second) {
		x, y, _ := sat.Position()
		t.Errorf("the satellite is at %d,%d and the rack still has it at %v bonded to %d",
			x, y, d.Rack.Pane(si).Box, d.Rack.Pane(si).To)
	}
}

// **Closing one satellite does not strand the others.** The wake-up that
// reconciles a refused move was a timer on the window that asked for it,
// and a timer dies with its window: two satellites refusing, then the
// later one closed before its patience ran out, took the only wake with
// it and left the first bonded to a position it was never in. A music
// player closed a window and watched the other one stick.
func TestClosingOneSatelliteDoesNotStrandAnother(t *testing.T) {
	a, d, main, sat, mi, si, _ := coalesceRig(t)
	d.SetClock(nil)

	// A second satellite, bonded like the first.
	sat2, err := a.NewWindow(platform.WindowOptions{X: 0, Y: 200, Width: 200, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	si2 := d.Add("Sat2", sat2)
	a.PumpOnce()
	d.Rack.MoveTo(si2, 0, 200)
	if d.Rack.Pane(si2).To != mi && d.Rack.Pane(si2).To != si {
		t.Skipf("the second satellite bonded to %d, which this rig did not arrange", d.Rack.Pane(si2).To)
	}

	// Both refuse to move, and the main window is dragged away.
	sat.Surface().(*platform.Offscreen).SetMoveHonoured(false)
	sat2.Surface().(*platform.Offscreen).SetMoveHonoured(false)
	main.Surface().(*platform.Offscreen).SimulateDesktopMove(600, 600)
	settle(a)

	// The later one goes away before anybody's patience runs out, which
	// is what used to take the wake-up with it.
	sat2.Close()
	a.PumpOnce()

	if !reconciled(t, a, d, si, 3*time.Second) {
		t.Errorf("the first satellite is still bonded to %d at %v after the other one closed",
			d.Rack.Pane(si).To, d.Rack.Pane(si).Box)
	}
}

// Two more ways to lose the only wake-up, both without an event behind
// them: a *direct* Apply and a *direct* Show. The pass re-arms whatever
// is still outstanding, but a pass has to run — and nothing asks for
// one when the application drives the rack itself. A music player
// reported both after the event-driven cases were repaired.
func TestADirectRequestSurvivesItsSatelliteClosing(t *testing.T) {
	rig := func(t *testing.T) (*app.Application, *Desk, int, int, *app.Window) {
		t.Helper()
		a, d, _, sat, mi, si, _ := coalesceRig(t)
		d.SetClock(nil)
		sat2, err := a.NewWindow(platform.WindowOptions{X: 0, Y: 200, Width: 200, Height: 100, Headless: true})
		if err != nil {
			t.Fatal(err)
		}
		si2 := d.Add("Sat2", sat2)
		a.PumpOnce()
		d.Rack.MoveTo(si2, 0, 200)
		if d.Rack.Pane(si).To != mi {
			t.Skipf("the first satellite bonded to %d, which this rig did not arrange", d.Rack.Pane(si).To)
		}
		sat.Surface().(*platform.Offscreen).SetMoveHonoured(false)
		sat2.Surface().(*platform.Offscreen).SetMoveHonoured(false)
		return a, d, si, si2, sat2
	}

	t.Run("a direct Apply", func(t *testing.T) {
		a, d, si, _, sat2 := rig(t)
		// The application moves the group itself and asks for the
		// placement: no desktop event anywhere in this.
		d.Rack.MoveTo(0, 600, 700)
		d.Apply()
		sat2.Close()
		a.PumpOnce()
		if !reconciled(t, a, d, si, 3*time.Second) {
			t.Errorf("after a direct Apply the satellite is still bonded to %d at %v",
				d.Rack.Pane(si).To, d.Rack.Pane(si).Box)
		}
	})

	t.Run("a direct Show of each", func(t *testing.T) {
		a, d, si, si2, sat2 := rig(t)
		d.Show(si, false)
		d.Show(si2, false)
		a.PumpOnce()
		d.Window(si).Surface().(*platform.Offscreen).SimulateDesktopMove(700, 700)
		sat2.Surface().(*platform.Offscreen).SimulateDesktopMove(700, 800)
		d.Show(si, true)
		d.Show(si2, true)
		sat2.Close()
		a.PumpOnce()
		if !reconciled(t, a, d, si, 3*time.Second) {
			t.Errorf("after a direct Show the satellite is still bonded to %d at %v",
				d.Rack.Pane(si).To, d.Rack.Pane(si).Box)
		}
	})
}

// **A later request does not postpone an earlier deadline.** The wake
// was "one at a time, replaced by each new request", and the
// replacement asked nothing about *when*: ask to move one window, then
// another a moment later, and the second request's patience — which
// runs from the moment it was made — cancelled the wake the first one
// was waiting on. The first window's deadline passed with nobody
// looking and it stayed bonded to a position it was never in until the
// later one ran out as well. The same symptom as a lost wake, arriving
// late rather than never, which is what the music player still saw
// after the lost-wake paths were closed (item 10).
//
// The later deadline is handed to [Desk.wakeAt] rather than waited for.
// It is the one road every request takes, and a test that produced a
// deadline this much later by making a second request would have to
// sleep for the difference and then measure the margin that was left:
// a second and three quarters of real time to watch a window not
// reconcile.
func TestALaterRequestDoesNotPostponeAnEarlierDeadline(t *testing.T) {
	a, d, main, sat, mi, si, _ := coalesceRig(t)
	d.SetClock(nil)

	sat2, err := a.NewWindow(platform.WindowOptions{X: 0, Y: 200, Width: 200, Height: 100, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	si2 := d.Add("Sat2", sat2)
	a.PumpOnce()
	d.Rack.MoveTo(si2, 0, 200)
	if d.Rack.Pane(si).To != mi {
		t.Skipf("the first satellite bonded to %d, which this rig did not arrange", d.Rack.Pane(si).To)
	}

	// The first satellite refuses, and the drag gives it a deadline a
	// quarter of a second out with a wake armed for it.
	sat.Surface().(*platform.Offscreen).SetMoveHonoured(false)
	main.Surface().(*platform.Offscreen).SimulateDesktopMove(600, 600)
	settle(a)
	if d.until[si].IsZero() {
		t.Skip("the first satellite's move was honoured, so there is no deadline to postpone")
	}

	// And now the other one is asked to move, with its own patience
	// running from here.
	d.wakeAt(d.Window(si2), time.Now().Add(2*time.Second))

	if !reconciled(t, a, d, si, time.Second) {
		t.Errorf("the first satellite is still bonded to %d at %v a second on, with its own deadline a quarter of a second out",
			d.Rack.Pane(si).To, d.Rack.Pane(si).Box)
	}
}
