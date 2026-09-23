package rack_test

import (
	"testing"

	"github.com/codemodify/uitoolkit/app"
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/rack"
	"github.com/codemodify/uitoolkit/style"
)

// The package from outside, as an application sees it.

// The geometry with no window anywhere near it: an app names its panes,
// drags one within reach of another and reads back where they all are.
func TestARackSnapsFromOutsideThePackage(t *testing.T) {
	r := rack.New(rack.DefaultReach)
	r.Add("main", rack.Box{X: 100, Y: 100, W: 275, H: 116})
	list := r.Add("playlist", rack.Box{X: 800, Y: 600, W: 275, H: 232})

	// Let go a few pixels shy of flush under the main window.
	r.MoveTo(list, 104, 219)
	pane := r.Pane(list)
	if pane.To != 0 || pane.Bond.Side != rack.SideBottom {
		t.Fatalf("the playlist bonded to %d on the %v", pane.To, pane.Bond.Side)
	}
	if got, want := pane.Box, (rack.Box{X: 100, Y: 216, W: 275, H: 232}); got != want {
		t.Errorf("snapped to %+v, want %+v", got, want)
	}

	// And the stack travels with the window it hangs from.
	r.MoveTo(0, 400, 300)
	if got, want := r.Pane(list).Box, (rack.Box{X: 400, Y: 416, W: 275, H: 232}); got != want {
		t.Errorf("the playlist followed to %+v, want %+v", got, want)
	}
	if got, want := r.Bounds(), (rack.Box{X: 400, Y: 300, W: 275, H: 348}); got != want {
		t.Errorf("bounds %+v, want %+v", got, want)
	}
}

// The desk half of the rack, with real windows in it. The offscreen backend
// implements both halves a rack needs — where a window is, and putting one
// somewhere — so this runs without a compositor, and it is the same code
// that runs on X11.
func TestTheDeskMovesRealWindows(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := app.New(app.Options{Headless: true, Look: style.DarkLook(), Scale: 1, DisableLookWatch: true})
	open := func(name string, w, h int) *app.Window {
		win, err := a.NewWindow(platform.WindowOptions{
			Title: name, Width: w, Height: h, Headless: true,
			Decorations: platform.DecorationsClient,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { win.Close() })
		return win
	}
	main, sat := open("main", 300, 120), open("playlist", 300, 200)
	a.PumpOnce()

	d := rack.NewDesk(10)
	if got := d.Places(); got {
		t.Fatal("an empty desk claims it can place windows")
	}
	iMain := d.Add("main", main)
	iSat := d.Add("playlist", sat)
	if !d.Places() {
		t.Fatal("the offscreen backend should place windows")
	}

	d.Attach(iSat, iMain, rack.SideBottom)
	mainBox := d.Rack.Pane(iMain).Box
	if got := d.Rack.Pane(iSat).Box; got.X != mainBox.X || got.Y != mainBox.Bottom() {
		t.Fatalf("attached at %+v, want flush under %+v", got, mainBox)
	}
	if x, y, ok := sat.Position(); !ok || x != mainBox.X || y != mainBox.Bottom() {
		t.Errorf("the window is at %d,%d; the rack says %+v", x, y, d.Rack.Pane(iSat).Box)
	}

	// The desktop moves the main window: the satellite follows on the next
	// look, and Follow says it moved something.
	main.Move(500, 400)
	a.PumpOnce()
	if !d.Follow() {
		t.Fatal("Follow saw nothing after the main window moved")
	}
	if got := d.Rack.Pane(iSat).Box; got.X != 500 || got.Y != 400+120 {
		t.Errorf("the satellite is at %+v, want 500,%d", got, 400+120)
	}
	if x, y, ok := sat.Position(); !ok || x != 500 || y != 520 {
		t.Errorf("the satellite's window is at %d,%d", x, y)
	}
	// And a second look with nothing moved reports nothing, so a player's
	// timer does not repaint every frame for no reason.
	if d.Follow() {
		t.Error("Follow moved something when nothing had changed")
	}

	// Hiding keeps the bond, and showing puts the window back where it was
	// before it is mapped, so it never appears in the wrong place first.
	d.Show(iSat, false)
	if d.Rack.Pane(iSat).To != iMain {
		t.Error("hiding the satellite dropped its bond")
	}
	main.Move(120, 90)
	a.PumpOnce()
	d.Follow()
	d.Show(iSat, true)
	if x, y, ok := sat.Position(); !ok || x != 120 || y != 90+120 {
		t.Errorf("the satellite came back at %d,%d, want 120,%d", x, y, 90+120)
	}
}
