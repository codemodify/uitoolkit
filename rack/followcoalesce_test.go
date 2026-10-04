package rack

import (
	"testing"
	"time"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// A Desk whose windows report their moves, with the clock in the test's hand.
func coalesceRig(t *testing.T) (*app.Application, *Desk, *app.Window, *app.Window, int, int, *time.Time) {
	t.Helper()
	a := app.New(app.Options{Look: style.DarkLook(), Headless: true})
	mk := func(x, y int) *app.Window {
		w, err := a.NewWindow(platform.WindowOptions{
			X: x, Y: y, Width: 200, Height: 100, Headless: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	main, sat := mk(0, 0), mk(0, 100)
	d := NewDesk(10)
	now := time.Unix(1000, 0)
	d.SetClock(func() time.Time { return now })
	mi := d.Add("Main", main)
	si := d.Add("Sat", sat)
	a.PumpOnce()
	if !main.SendsMoveEvents() {
		t.Skip("this backend does not report moves")
	}
	d.Rack.MoveTo(si, 0, 100)
	if d.Rack.Pane(si).To != mi {
		t.Fatalf("the satellite did not bond to Main: To=%d", d.Rack.Pane(si).To)
	}
	return a, d, main, sat, mi, si, &now
}

// The last move of a drag is the one that says where the window came to
// rest, and it used to be the one lost.
//
// A pass over the panes moves the satellites, and each satellite reports its
// own move — so a flag said "a pass is running" and threw those away, which
// was right. It threw away the *desktop's* moves too: a configure arriving
// while a pass was running went nowhere at all, so the satellites were left
// following a position the window had already left, until something else
// moved it.
func TestAMoveDuringAPassIsNotLost(t *testing.T) {
	a, d, main, _, _, si, _ := coalesceRig(t)
	o := main.Surface().(*platform.Offscreen)

	// A pass is running, and the desktop says Main moved again. This is
	// what a satellite's own move callback does from inside Follow.
	d.inFollow = true
	o.SimulateDesktopMove(300, 300)
	a.PumpOnce()
	if !d.again {
		t.Fatal("a move that arrived during a pass was dropped rather than noted")
	}
	d.inFollow = false

	// The next callback runs the pass that was owed.
	o.SimulateDesktopMove(300, 300)
	a.PumpOnce()
	if b := d.Rack.Pane(si).Box; b.X != 300 || b.Y != 400 {
		t.Errorf("the satellite is at %d,%d; want 300,400", b.X, b.Y)
	}
}

// Patience is a length of time, not a number of looks.
//
// It used to be a count, from when an application chose how often to call
// Follow. Once the desktop began reporting moves the count was spent by *the
// user*: every event is a look, so a fast drag used the whole allowance in a
// fraction of a second and a window that paused was waited for far too long.
// Either way the patience meant something different on every machine.
func TestPatienceIsMeasuredInTime(t *testing.T) {
	a, d, _, sat, _, si, now := coalesceRig(t)

	// The rack has asked the satellite to move and is inside the time it
	// allows for the answer; the desktop has put it somewhere else.
	was := d.Rack.Pane(si).Box
	d.until[si] = now.Add(movePatience)
	so := sat.Surface().(*platform.Offscreen)
	so.SimulateDesktopMove(700, 700)
	a.PumpOnce()

	// However many looks it takes, the rack goes on waiting while the time
	// stands still — a count of 25 would have been spent four times over —
	// because a move is a request and the answer it is waiting for may
	// still be in flight.
	for i := 0; i < 100; i++ {
		d.Follow()
	}
	if got := d.Rack.Pane(si).Box; got.X != was.X || got.Y != was.Y {
		t.Errorf("the rack took the desktop's answer (%d,%d) while still inside its time", got.X, got.Y)
	}

	// And the time passing ends the wait, which is the other half: a rack
	// that waited for ever would argue with a window manager that refuses.
	*now = now.Add(movePatience + time.Millisecond)
	d.Follow()
	if got := d.Rack.Pane(si).Box; got.X != 700 || got.Y != 700 {
		t.Errorf("the rack is at %d,%d after the wait ran out; want the desktop's 700,700", got.X, got.Y)
	}
}

// The clock is the Desk's own, so a test need not wait out a real quarter of
// a second — and an application that wants its rack on a simulated clock can
// say so.
func TestADeskTellsTheTimeItIsGiven(t *testing.T) {
	d := NewDesk(10)
	at := time.Unix(5000, 0)
	d.SetClock(func() time.Time { return at })
	if got := d.clock(); !got.Equal(at) {
		t.Errorf("the desk reads %v, want the clock it was given", got)
	}
	d.SetClock(nil)
	if d.clock().Equal(at) {
		t.Error("a nil clock did not put the real one back")
	}
}
