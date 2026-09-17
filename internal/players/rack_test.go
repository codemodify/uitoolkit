package players

import "testing"

// The snapping geometry, without a window anywhere near it.

func TestAPaneSnapsFlushUnderTheOneAboveIt(t *testing.T) {
	main := Box{X: 200, Y: 100, W: 275, H: 116}
	// Let go three pixels low and four to the right of flush.
	sat := Box{X: 204, Y: 219, W: 275, H: 116}
	bond, ok := Snap(main, sat, 10)
	if !ok {
		t.Fatal("no snap within reach")
	}
	if bond.Side != SideBottom {
		t.Errorf("side %v, want bottom", bond.Side)
	}
	if bond.Along != 0 {
		t.Errorf("along %d, want 0 (flush left)", bond.Along)
	}
	got := bond.Place(main, sat.W, sat.H)
	want := Box{X: 200, Y: 216, W: 275, H: 116}
	if got != want {
		t.Errorf("placed at %+v, want %+v", got, want)
	}
}

func TestAPaneFarAwayDoesNotSnap(t *testing.T) {
	main := Box{X: 200, Y: 100, W: 275, H: 116}
	for _, sat := range []Box{
		{X: 204, Y: 260, W: 275, H: 116},  // too far below
		{X: 900, Y: 216, W: 275, H: 116},  // flush below but a screen away
		{X: 204, Y: -400, W: 275, H: 116}, // above and away
	} {
		if _, ok := Snap(main, sat, 10); ok {
			t.Errorf("%+v snapped and should not have", sat)
		}
	}
}

func TestAPaneSnapsFlushToTheFarEndToo(t *testing.T) {
	main := Box{X: 0, Y: 0, W: 400, H: 100}
	// A narrow pane let go near the right-hand end of the edge.
	sat := Box{X: 297, Y: 103, W: 100, H: 60}
	bond, ok := Snap(main, sat, 10)
	if !ok {
		t.Fatal("no snap")
	}
	if bond.Along != 300 {
		t.Errorf("along %d, want 300 (flush right)", bond.Along)
	}
}

func TestTheNearerEdgeWinsAtACorner(t *testing.T) {
	main := Box{X: 0, Y: 0, W: 300, H: 200}
	// Beside the right edge (2 px) and below the bottom (8 px).
	sat := Box{X: 302, Y: 208, W: 120, H: 80}
	bond, ok := Snap(main, sat, 12)
	if !ok {
		t.Fatal("no snap")
	}
	if bond.Side != SideRight {
		t.Errorf("side %v, want right — it is nearer", bond.Side)
	}
}

// The whole point of the rack: drag the main strip and the stack follows.
func TestDraggingTheMainStripCarriesTheStack(t *testing.T) {
	r := NewRack(10)
	main := r.Add("main", Box{X: 100, Y: 100, W: 275, H: 116})
	eq := r.Add("equaliser", Box{X: 100, Y: 216, W: 275, H: 116})
	pl := r.Add("playlist", Box{X: 100, Y: 332, W: 275, H: 232})
	// Snap them into a stack the way a drag would.
	r.MoveTo(eq, 104, 219)
	r.MoveTo(pl, 102, 337)
	if p := r.Pane(eq); p.To != main || p.Bond.Side != SideBottom {
		t.Fatalf("equaliser bonded to %d on the %v", p.To, p.Bond.Side)
	}
	if p := r.Pane(pl); p.To != eq || p.Bond.Side != SideBottom {
		t.Fatalf("playlist bonded to %d on the %v", p.To, p.Bond.Side)
	}

	moved := r.MoveTo(main, 400, 300)
	if len(moved) != 3 {
		t.Errorf("moving the main strip moved %d panes, want 3", len(moved))
	}
	want := []Box{
		{X: 400, Y: 300, W: 275, H: 116},
		{X: 400, Y: 416, W: 275, H: 116},
		{X: 400, Y: 532, W: 275, H: 232},
	}
	for i, w := range want {
		if got := r.Pane(i).Box; got != w {
			t.Errorf("%s at %+v, want %+v", r.Pane(i).Name, got, w)
		}
	}
}

// Dragging a pane in the middle of the stack carries what hangs from it
// and leaves what it hung from where it was.
func TestDraggingAPaneCarriesOnlyWhatHangsFromIt(t *testing.T) {
	r := NewRack(10)
	main := r.Add("main", Box{X: 0, Y: 0, W: 200, H: 100})
	eq := r.Add("equaliser", Box{X: 0, Y: 100, W: 200, H: 100})
	pl := r.Add("playlist", Box{X: 0, Y: 200, W: 200, H: 100})
	r.MoveTo(eq, 0, 100)
	r.MoveTo(pl, 0, 200)

	r.MoveTo(eq, 600, 500)
	if got := r.Pane(main).Box; got != (Box{X: 0, Y: 0, W: 200, H: 100}) {
		t.Errorf("the main strip moved to %+v", got)
	}
	if got := r.Pane(eq).Box; got != (Box{X: 600, Y: 500, W: 200, H: 100}) {
		t.Errorf("the equaliser is at %+v", got)
	}
	if got := r.Pane(pl).Box; got != (Box{X: 600, Y: 600, W: 200, H: 100}) {
		t.Errorf("the playlist did not follow: %+v", got)
	}
	if r.Pane(eq).To != -1 {
		t.Errorf("the equaliser is still bonded to %d", r.Pane(eq).To)
	}
	if r.Pane(pl).To != eq {
		t.Errorf("the playlist came loose: bonded to %d", r.Pane(pl).To)
	}
}

// A pane cannot end up stuck to something that hangs from it, or the two
// would chase each other forever.
func TestAPaneNeverBondsToItsOwnDescendant(t *testing.T) {
	r := NewRack(10)
	a := r.Add("a", Box{X: 0, Y: 0, W: 100, H: 100})
	b := r.Add("b", Box{X: 0, Y: 100, W: 100, H: 100})
	r.MoveTo(b, 0, 100)
	if r.Pane(b).To != a {
		t.Fatal("b did not stick to a")
	}
	// Dragging a to sit under b must leave a loose, not bonded to b.
	r.MoveTo(a, 0, 200)
	if r.Pane(a).To != -1 {
		t.Errorf("a bonded to %d, which hangs from a", r.Pane(a).To)
	}
	if r.Pane(b).Box.Y != 300 {
		t.Errorf("b did not come along: %+v", r.Pane(b).Box)
	}
}

// Resizing the anchor keeps the stack flush under it.
func TestResizingTheAnchorMovesWhatHangsFromIt(t *testing.T) {
	r := NewRack(10)
	r.Add("main", Box{X: 50, Y: 50, W: 400, H: 200})
	pl := r.Add("playlist", Box{X: 50, Y: 250, W: 400, H: 300})
	r.MoveTo(pl, 50, 250)
	r.Resize(0, 400, 260)
	if got := r.Pane(pl).Box.Y; got != 310 {
		t.Errorf("the playlist is at y=%d, want 310", got)
	}
}

// Detach and Attach are the controls, not the drag.
func TestAttachAndDetach(t *testing.T) {
	r := NewRack(10)
	r.Add("main", Box{X: 10, Y: 10, W: 300, H: 120})
	pl := r.Add("playlist", Box{X: 900, Y: 700, W: 300, H: 200})
	r.Attach(pl, 0, SideBottom)
	if got := r.Pane(pl).Box; got != (Box{X: 10, Y: 130, W: 300, H: 200}) {
		t.Errorf("attached at %+v", got)
	}
	r.Attach(pl, 0, SideRight)
	if got := r.Pane(pl).Box; got != (Box{X: 310, Y: 10, W: 300, H: 200}) {
		t.Errorf("re-attached at %+v", got)
	}
	r.Detach(pl)
	r.MoveTo(0, 500, 500)
	if got := r.Pane(pl).Box; got.X != 310 || got.Y != 10 {
		t.Errorf("a detached pane followed the anchor to %+v", got)
	}
}

func TestBoundsCoversEveryShownPane(t *testing.T) {
	r := NewRack(10)
	r.Add("main", Box{X: 100, Y: 100, W: 200, H: 100})
	pl := r.Add("playlist", Box{X: 100, Y: 200, W: 260, H: 150})
	r.MoveTo(pl, 100, 200)
	if got, want := r.Bounds(), (Box{X: 100, Y: 100, W: 260, H: 250}); got != want {
		t.Errorf("bounds %+v, want %+v", got, want)
	}
	r.Show(pl, false)
	if got, want := r.Bounds(), (Box{X: 100, Y: 100, W: 200, H: 100}); got != want {
		t.Errorf("bounds with the playlist hidden %+v, want %+v", got, want)
	}
}
