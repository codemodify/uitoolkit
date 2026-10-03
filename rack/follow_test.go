package rack

import (
	"testing"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
)

// Satellites follow the desktop's drag when the desktop says so, instead
// of waiting for the application's next tick.
//
// Nothing told anyone a window had moved: the backend cached the position
// and emitted no event, so a Desk could only be driven by polling Follow
// on a timer. A player calling it from its 70 ms playback pulse gets at
// best fourteen updates a second, and the panels visibly trail the window
// being dragged.
func TestSatellitesFollowOnTheDesktopsWord(t *testing.T) {
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
	mi := d.Add("Main", main)
	si := d.Add("Sat", sat)
	a.PumpOnce()

	if !main.SendsMoveEvents() {
		t.Skip("this backend does not report moves")
	}
	// Snap the satellite under Main, the way a user drag would.
	d.Rack.MoveTo(si, 0, 100)
	if d.Rack.Pane(si).To != mi {
		t.Fatalf("the satellite did not bond to Main: To=%d", d.Rack.Pane(si).To)
	}

	// The desktop drags Main. No Follow call from the application.
	o := main.Surface().(*platform.Offscreen)
	o.SimulateDesktopMove(300, 300)
	a.PumpOnce()

	if b := d.Rack.Pane(si).Box; b.X != 300 || b.Y != 400 {
		t.Errorf("the satellite is at %d,%d after a drag the desktop reported; want 300,400", b.X, b.Y)
	}
}
