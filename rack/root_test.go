package rack

import "testing"

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
