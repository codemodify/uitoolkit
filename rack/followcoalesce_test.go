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

// settle delivers whatever the desktop has to say and then runs the pass
// those events asked for.
//
// Two pumps, because a move does not run a pass where it arrives: it
// *posts* one, so a burst of configures is one pass and not one each, and
// the loop runs what is posted at the top of its next turn (PumpOnce runs
// the posted work first, then drains the events). A real application never
// waits for that turn — Post wakes the loop — but a test that pumps once
// has only delivered the news.
func settle(a *app.Application) {
	a.PumpOnce()
	a.PumpOnce()
}

// **A burst of moves is one pass.** A drag delivers a configure for every
// step; four arriving in one turn of the loop used to be four passes, each
// walking every pane and asking the window system to move every satellite
// again — and the first three placements were overwritten by the fourth
// before anything was painted.
func TestABurstOfMovesIsOnePass(t *testing.T) {
	a, d, main, _, _, si, _ := coalesceRig(t)
	o := main.Surface().(*platform.Offscreen)
	passes := 0
	d.Moved = func() { passes++ }

	was := d.Rack.Pane(si).Box
	for _, at := range [][2]int{{10, 10}, {20, 20}, {30, 30}, {40, 40}} {
		o.SimulateDesktopMove(at[0], at[1])
	}
	// The four are delivered in this turn. One pass is owed and none has
	// run: the satellites are still where they were, because a burst is
	// answered once, after it.
	a.PumpOnce()
	if !d.owed {
		t.Error("the moves ran their passes where they arrived instead of asking for one")
	}
	if b := d.Rack.Pane(si).Box; b != was {
		t.Errorf("the satellite moved to %v during the burst; want it answered once, after it", b)
	}
	a.PumpOnce()
	if d.owed {
		t.Error("the pass that was owed did not run")
	}
	if passes != 1 {
		t.Errorf("four moves in one turn ran %d passes, want one", passes)
	}
	// And the one pass put the satellite under where the window came to
	// rest, which is the only placement that was ever going to be seen.
	if b := d.Rack.Pane(si).Box; b.X != 40 || b.Y != 140 {
		t.Errorf("the satellite is at %d,%d; want 40,140", b.X, b.Y)
	}
}

// **A move request the window manager never honours is reconciled when
// the patience runs out.** The rack allows a quarter of a second for a
// window to arrive where it was asked to go — and when it never does,
// nothing looked again: no move happened, so no event came, and the rack
// kept the geometry nobody honoured until the application's own pulse or
// an explicit Follow. A player with no pulse waited for ever.
func TestARefusedMoveIsReconciledWhenThePatienceRunsOut(t *testing.T) {
	a, d, main, sat, _, si, _ := coalesceRig(t)
	// The real clock: the wake-up is a real timer and the deadline is a
	// real quarter of a second.
	d.SetClock(nil)
	so := sat.Surface().(*platform.Offscreen)
	so.SetMoveHonoured(false)
	was := d.Rack.Pane(si).Box

	// The main window is dragged away. The satellite is asked to follow
	// and the window manager takes the request and does nothing.
	main.Surface().(*platform.Offscreen).SimulateDesktopMove(600, 600)
	settle(a)
	if x, y, _ := sat.Position(); x != was.X || y != was.Y {
		t.Fatalf("the satellite moved to %d,%d; this test needs a window manager that refuses", x, y)
	}

	// Nothing else happens: no event, no Follow call, no application
	// pulse. The rack has to come back by itself.
	deadline := time.Now().Add(3 * time.Second)
	for {
		a.PumpOnce()
		if d.Rack.Pane(si).Box.X == was.X && d.Rack.Pane(si).To == -1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the rack still has the satellite at %v bonded to %d after the patience ran out",
				d.Rack.Pane(si).Box, d.Rack.Pane(si).To)
		}
		time.Sleep(5 * time.Millisecond)
	}
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
	settle(a)
	if !d.again {
		t.Fatal("a move that arrived during a pass was dropped rather than noted")
	}
	d.inFollow = false

	// The next callback runs the pass that was owed.
	o.SimulateDesktopMove(300, 300)
	settle(a)
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
