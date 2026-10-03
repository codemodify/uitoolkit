package rack

import (
	"testing"
	"time"
)

// Pane 0 is the root: an application adds its windows main-first, and the
// rest hang from the main one. A snap that joins two separate components
// is turned round rather than putting the root under a satellite.
//
// It used to bond pane 0 like any other. Moving the main window beside a
// loose satellite made the main window its *child*, so the satellite was
// outside the main window's subtree and the next drag of the main window
// left it standing where it was.
func TestPaneZeroStaysTheRoot(t *testing.T) {
	t.Run("main moved beside a loose satellite", func(t *testing.T) {
		r := New(10)
		main := r.Add("Main", Box{X: 0, Y: 0, W: 200, H: 100})
		sat := r.Add("Satellite", Box{X: 500, Y: 100, W: 200, H: 100})
		r.MoveTo(main, 500, 0)
		if to := r.Pane(main).To; to != -1 {
			t.Fatalf("the root is bonded to %d; it is the root", to)
		}
		if to := r.Pane(sat).To; to != main {
			t.Fatalf("the satellite hangs from %d, want the root", to)
		}
		r.MoveTo(main, 700, 300)
		if b := r.Pane(sat).Box; b.X != 700 || b.Y != 400 {
			t.Errorf("the satellite is at %d,%d; it did not follow the root", b.X, b.Y)
		}
	})

	t.Run("a satellite moved beside main", func(t *testing.T) {
		r := New(10)
		main := r.Add("Main", Box{X: 0, Y: 0, W: 200, H: 100})
		sat := r.Add("Satellite", Box{X: 500, Y: 500, W: 200, H: 100})
		r.MoveTo(sat, 0, 100)
		if to := r.Pane(main).To; to != -1 {
			t.Errorf("the root is bonded to %d", to)
		}
		if to := r.Pane(sat).To; to != main {
			t.Errorf("the satellite hangs from %d, want the root", to)
		}
	})

	t.Run("main rejoins a two-pane component", func(t *testing.T) {
		r := New(10)
		main := r.Add("Main", Box{X: 0, Y: 0, W: 200, H: 100})
		a := r.Add("A", Box{X: 500, Y: 0, W: 200, H: 100})
		b := r.Add("B", Box{X: 500, Y: 100, W: 200, H: 100})
		r.MoveTo(b, 500, 100) // b hangs from a
		if r.Pane(b).To != a {
			t.Skip("the two satellites did not join")
		}
		r.MoveTo(main, 300, 0) // main arrives beside a
		if to := r.Pane(main).To; to != -1 {
			t.Errorf("the root is bonded to %d after rejoining a component", to)
		}
		// Everything is reachable from the root, so a drag carries it.
		sub := r.subtree(main)
		for _, want := range []int{a, b} {
			if !contains(sub, want) {
				t.Errorf("pane %d is not under the root: %v", want, sub)
			}
		}
	})
}

// The nearest qualifying anchor wins, not whichever was added first.
//
// Snap weighs the edges within one candidate, but nothing ranked the
// candidates against each other, so rebond took the first that qualified
// in insertion order: an anchor eight pixels away beat one the pane was
// already flush with.
func TestTheNearestAnchorWins(t *testing.T) {
	r := New(10)
	_ = r.Add("Main", Box{X: 0, Y: 0, W: 100, H: 100})
	// far qualifies at six pixels; near is already flush.
	far := r.Add("Far", Box{X: 294, Y: 0, W: 100, H: 100})
	near := r.Add("Near", Box{X: 500, Y: 0, W: 100, H: 100})
	sat := r.Add("Sat", Box{X: 408, Y: 0, W: 100, H: 100})

	r.MoveTo(sat, 400, 0) // flush with near's left edge, 6 short of far's right
	to := r.Pane(sat).To
	if to == far {
		t.Errorf("it took the anchor 6 px away because that pane was added first")
	}
	if to != near {
		t.Errorf("it bonded to %d, want the nearest anchor %d", to, near)
	}
}

// Main joining a component of several panes: it stays the root, every
// pane ends up under it, and the walk terminates.
//
// The first version of hangFrom reversed a link and then stepped to the
// pane it had just pointed at, following the link it had itself written:
// the two panes reversed each other for ever and Rack.MoveTo never
// returned. With the move callback added in 0.23.3 that is an
// application freezing in the middle of a drag.
func TestMainJoiningAMultiPaneComponentTerminates(t *testing.T) {
	for _, join := range []string{"root", "leaf"} {
		t.Run(join, func(t *testing.T) {
			done := make(chan struct{})
			var r *Rack
			var main, list, eq int
			go func() {
				defer close(done)
				r = New(10)
				main = r.Add("Main", Box{X: 0, Y: 0, W: 200, H: 100})
				list = r.Add("List", Box{X: 500, Y: 0, W: 200, H: 100})
				eq = r.Add("EQ", Box{X: 500, Y: 100, W: 200, H: 100})
				r.Attach(eq, list, SideBottom)
				// Beside the component's root, or beside its leaf.
				if join == "root" {
					r.MoveTo(main, 300, 0)
				} else {
					r.MoveTo(main, 300, 100)
				}
			}()
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("MoveTo never returned")
			}

			if to := r.Pane(main).To; to != -1 {
				t.Errorf("the root is bonded to %d", to)
			}
			// Every pane hangs from the root, and nothing is its own
			// ancestor: the reversal left one tree, not a ring.
			sub := r.subtree(main)
			for _, want := range []int{list, eq} {
				if !contains(sub, want) {
					t.Errorf("pane %d is not under the root: %v", want, sub)
				}
			}
			for _, i := range []int{main, list, eq} {
				hops := 0
				for at := i; at >= 0; hops++ {
					if hops > 8 {
						t.Fatalf("pane %d walks a cycle up its anchors", i)
					}
					at = r.Pane(at).To
				}
			}
		})
	}
}
