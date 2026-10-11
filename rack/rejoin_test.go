package rack

import "testing"

// parents is every pane's anchor, which is the whole shape of the rack.
func parents(r *Rack) []int {
	out := make([]int, r.Len())
	for i := range out {
		out[i] = r.Pane(i).To
	}
	return out
}

// roots is how many panes hang from nothing. There is exactly one: pane 0.
func roots(r *Rack) int {
	n := 0
	for _, to := range parents(r) {
		if to < 0 {
			n++
		}
	}
	return n
}

// chainRig is the main window and a foreign chain of n panes, standing
// well clear of it: pane 1 is the chain's root and each pane after it
// hangs from the one before, running to the right.
func chainRig(t *testing.T, n int) (*Rack, []int) {
	t.Helper()
	r := New(10)
	r.Add("Main", Box{X: 0, Y: 0, W: 200, H: 100})
	chain := make([]int, 0, n)
	for k := 0; k < n; k++ {
		i := r.Add(string(rune('A'+k)), Box{X: 1000 + 200*k, Y: 0, W: 200, H: 100})
		chain = append(chain, i)
		if k > 0 {
			r.MoveTo(i, 1000+200*k, 0)
		}
	}
	for k := 1; k < n; k++ {
		if to := r.Pane(chain[k]).To; to != chain[k-1] {
			t.Fatalf("the fixture's pane %d hangs from %d, want %d", chain[k], to, chain[k-1])
		}
	}
	if roots(r) != 2 {
		t.Fatalf("the fixture has %d roots, want the main window and the chain's", roots(r))
	}
	return r, chain
}

// **Joining a chain of three or more panes must keep every link.** The
// path from the pane that was joined up to its own root is reversed so
// the main window ends up the root of everything — and the reversal used
// to clear the link it had just written, so the middle pane came out with
// no anchor at all: a second root beside the first, and that part of the
// group stopped following the main window. A two-pane chain has one link
// and no middle, which is why it passed.
func TestMainRejoiningALongChainKeepsEveryLink(t *testing.T) {
	for _, n := range []int{2, 3, 4, 5} {
		r, chain := chainRig(t, n)
		last := chain[len(chain)-1]
		end := r.Pane(last).Box

		// The main window is dragged against the far end of the chain.
		r.MoveTo(0, end.Right(), 0)

		if to := r.Pane(0).To; to != -1 {
			t.Errorf("%d panes: the main window hangs from %d; it is the root", n, to)
		}
		if got := roots(r); got != 1 {
			t.Errorf("%d panes: %d roots (%v), want the main window alone", n, got, parents(r))
		}
		// Every pane of the chain is still in one piece under the main
		// window, and in the order the reversal should have left them.
		if to := r.Pane(last).To; to != 0 {
			t.Errorf("%d panes: the pane that was joined hangs from %d, want the main window", n, to)
		}
		for k := len(chain) - 2; k >= 0; k-- {
			if to := r.Pane(chain[k]).To; to != chain[k+1] {
				t.Errorf("%d panes: pane %d hangs from %d, want %d (%v)",
					n, chain[k], to, chain[k+1], parents(r))
			}
		}
		// And the whole group follows the main window, which is what the
		// links are for.
		was := make([]Box, r.Len())
		for i := range was {
			was[i] = r.Pane(i).Box
		}
		r.MoveTo(0, r.Pane(0).Box.X+40, r.Pane(0).Box.Y+25)
		for i := range was {
			b := r.Pane(i).Box
			if b.X != was[i].X+40 || b.Y != was[i].Y+25 {
				t.Errorf("%d panes: pane %d went from %v to %v, want it to follow by 40,25",
					n, i, was[i], b)
			}
		}
	}
}

// **The pane that was dragged closes the gap.** A bond is taken from as
// far as Reach away, so there is usually a gap — and placing the group's
// side by the bond moved a window nobody had touched: the dragged window
// stopped where it was dropped and the stationary group jumped the last
// two pixels to meet it.
func TestTheDraggedPaneClosesTheGapNotTheGroup(t *testing.T) {
	for _, n := range []int{1, 2, 3} {
		r, chain := chainRig(t, n)
		stood := make([]Box, 0, n)
		for _, i := range chain {
			stood = append(stood, r.Pane(i).Box)
		}
		end := r.Pane(chain[len(chain)-1]).Box

		// Dropped two pixels short of the chain's end, which is inside
		// Reach, so it snaps.
		const gap = 2
		r.MoveTo(0, end.Right()+gap, 0)

		for k, i := range chain {
			if b := r.Pane(i).Box; b != stood[k] {
				t.Errorf("%d panes: pane %d moved from %v to %v; it was standing still", n, i, stood[k], b)
			}
		}
		if b := r.Pane(0).Box; b.X != end.Right() {
			t.Errorf("%d panes: the dragged pane is at %d, want it flush at %d", n, b.X, end.Right())
		}
	}
}

// The exact-edge case, which was already right, stays right: dropped
// flush, nothing moves at all.
func TestAnExactEdgeJoinMovesNothing(t *testing.T) {
	r, chain := chainRig(t, 3)
	stood := make([]Box, r.Len())
	for i := range stood {
		stood[i] = r.Pane(i).Box
	}
	end := r.Pane(chain[len(chain)-1]).Box
	r.MoveTo(0, end.Right(), 0)
	for _, i := range chain {
		if b := r.Pane(i).Box; b != stood[i] {
			t.Errorf("pane %d moved from %v to %v", i, stood[i], b)
		}
	}
	if b := r.Pane(0).Box; b.X != end.Right() || b.Y != 0 {
		t.Errorf("the dragged pane is at %d,%d, want %d,0", b.X, b.Y, end.Right())
	}
}
