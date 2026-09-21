package players

// Windows that stick to one another.
//
// A compact player of this era was not one window: it was a main strip with
// an equaliser under it and a playlist under that, and the three behaved as
// one object. Drag the main strip and the other two came along; drag one of
// them away and it came loose; drag it back to within a few pixels of an
// edge and it snapped flush again.
//
// All of that is arithmetic on four numbers a window, so it lives here,
// away from any window at all, and is tested without opening one. What the
// apps add is the two halves a model cannot have: asking the desktop where
// a window is, and telling it to put one somewhere. Only one of the two
// desktops this toolkit runs on will do the second (see Rack.Apply in each
// app, and docs/players.md).

// Box is a window's outline on the desktop, in the logical pixels
// app.Window's Position, Size and Move speak, so a box's far edge is its
// neighbour's near one at any display scale. Integers, not the toolkit's
// float rects: a snap that lands half a pixel out is a bug, and a type
// that cannot express half a pixel cannot have it.
type Box struct{ X, Y, W, H int }

// Right and Bottom are the far edges, exclusive.
func (b Box) Right() int  { return b.X + b.W }
func (b Box) Bottom() int { return b.Y + b.H }

// Empty reports a box with no area.
func (b Box) Empty() bool { return b.W <= 0 || b.H <= 0 }

// Moved is the box translated by dx, dy.
func (b Box) Moved(dx, dy int) Box { b.X += dx; b.Y += dy; return b }

// At is the box moved to x, y.
func (b Box) At(x, y int) Box { b.X, b.Y = x, y; return b }

// Side is the edge of an anchor that a pane is stuck to.
type Side uint8

// The four edges, and "not stuck to anything".
const (
	SideNone Side = iota
	SideTop
	SideBottom
	SideLeft
	SideRight
)

// String names the side, for a status line and for a test's failure
// message.
func (s Side) String() string {
	switch s {
	case SideTop:
		return "top"
	case SideBottom:
		return "bottom"
	case SideLeft:
		return "left"
	case SideRight:
		return "right"
	}
	return "loose"
}

// Vertical reports whether the pane sits above or below its anchor, so it
// slides along x.
func (s Side) Vertical() bool { return s == SideTop || s == SideBottom }

// Bond is how one pane is stuck to another: which edge, and how far along
// that edge it sits. Along is measured from the anchor's leading corner —
// its left for a top or bottom bond, its top for a left or right one — so
// a bond survives the anchor being resized on the other axis.
type Bond struct {
	Side  Side
	Along int
}

// Stuck reports whether the bond attaches to anything.
func (b Bond) Stuck() bool { return b.Side != SideNone }

// Place is where a pane of size w by h sits when it is bonded to anchor.
func (b Bond) Place(anchor Box, w, h int) Box {
	switch b.Side {
	case SideTop:
		return Box{X: anchor.X + b.Along, Y: anchor.Y - h, W: w, H: h}
	case SideBottom:
		return Box{X: anchor.X + b.Along, Y: anchor.Bottom(), W: w, H: h}
	case SideLeft:
		return Box{X: anchor.X - w, Y: anchor.Y + b.Along, W: w, H: h}
	case SideRight:
		return Box{X: anchor.Right(), Y: anchor.Y + b.Along, W: w, H: h}
	}
	return Box{X: anchor.X, Y: anchor.Y, W: w, H: h}
}

// Snap is the bond a pane at sat would make with anchor if it were let go
// there, and whether it makes one at all.
//
// A pane snaps when the edge that would meet the anchor comes within reach
// of it *and* the two overlap on the other axis — a playlist a screen away
// horizontally is not under the main window however close its top edge is
// to the main window's bottom. Once a side is found, the position along
// that side snaps to flush at either end, which is what makes a stack of
// three windows line up rather than sit in a staircase.
//
// Where two sides both qualify — a small pane at a corner — the nearer one
// wins, and a tie goes to the vertical, because a stack is what these are
// nearly always used for.
func Snap(anchor, sat Box, reach int) (Bond, bool) {
	if anchor.Empty() || sat.Empty() || reach <= 0 {
		return Bond{}, false
	}
	overlapX := sat.X < anchor.Right()+reach && sat.Right() > anchor.X-reach
	overlapY := sat.Y < anchor.Bottom()+reach && sat.Bottom() > anchor.Y-reach

	best, bestGap := Bond{}, reach+1
	try := func(gap int, side Side) {
		if gap < 0 {
			gap = -gap
		}
		if gap > reach || gap >= bestGap {
			return
		}
		bestGap, best.Side = gap, side
	}
	if overlapX {
		try(sat.Y-anchor.Bottom(), SideBottom)
		try(sat.Bottom()-anchor.Y, SideTop)
	}
	if overlapY {
		try(sat.X-anchor.Right(), SideRight)
		try(sat.Right()-anchor.X, SideLeft)
	}
	if best.Side == SideNone {
		return Bond{}, false
	}
	if best.Side.Vertical() {
		best.Along = snapAlong(sat.X-anchor.X, anchor.W, sat.W, reach)
	} else {
		best.Along = snapAlong(sat.Y-anchor.Y, anchor.H, sat.H, reach)
	}
	return best, true
}

// snapAlong pulls an offset flush to the near or the far end of the edge
// it runs along, and otherwise leaves it where it is.
func snapAlong(along, anchorLen, satLen, reach int) int {
	if abs(along) <= reach {
		return 0
	}
	if far := anchorLen - satLen; abs(along-far) <= reach {
		return far
	}
	return along
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ---- the rack ---------------------------------------------------------------

// Pane is one window in a rack: where it is, and what it is stuck to.
type Pane struct {
	// Name is what the app calls the window, and what a test reads.
	Name string
	Box  Box
	// To is the index of the pane this one is stuck to, or -1.
	To   int
	Bond Bond
	// Shown is whether the window is open. A hidden pane keeps its bond
	// so that closing the playlist and opening it again puts it back
	// where it was, which is what these players did.
	Shown bool
}

// Rack is a group of windows that snap to one another. Pane 0 is the one
// the others hang from: it is never stuck to anything, which is what makes
// the graph a forest and stops two windows chasing each other.
type Rack struct {
	// Reach is how close an edge has to come before it snaps, in the same
	// pixels the boxes are in.
	Reach int
	panes []*Pane
}

// NewRack is an empty rack with a snapping distance.
func NewRack(reach int) *Rack {
	if reach <= 0 {
		reach = DefaultReach
	}
	return &Rack{Reach: reach}
}

// DefaultReach is how close is close enough, in logical pixels. Ten is
// about a fingertip's worth of slop at 1× and grows with the display,
// since a box is in the logical pixels a window's position is.
const DefaultReach = 10

// Add puts a pane in the rack and returns its index.
func (r *Rack) Add(name string, b Box) int {
	r.panes = append(r.panes, &Pane{Name: name, Box: b, To: -1, Shown: true})
	return len(r.panes) - 1
}

// Len is how many panes there are.
func (r *Rack) Len() int { return len(r.panes) }

// Pane is the i'th pane, or nil.
func (r *Rack) Pane(i int) *Pane {
	if r == nil || i < 0 || i >= len(r.panes) {
		return nil
	}
	return r.panes[i]
}

// Find is the pane with a name, and its index, or nil and -1.
func (r *Rack) Find(name string) (*Pane, int) {
	for i, p := range r.panes {
		if p.Name == name {
			return p, i
		}
	}
	return nil, -1
}

// MoveTo drags pane i to x, y: the pane and everything hanging from it
// move together, and then the pane looks for a new bond among the panes it
// did not carry. It reports the panes whose boxes changed, so a caller can
// tell the desktop about exactly those windows.
func (r *Rack) MoveTo(i, x, y int) []int {
	p := r.Pane(i)
	if p == nil {
		return nil
	}
	dx, dy := x-p.Box.X, y-p.Box.Y
	carried := r.subtree(i)
	moved := r.shift(carried, dx, dy)
	r.rebond(i, carried)
	moved = append(moved, r.reflow(i)...)
	return dedupe(moved)
}

// Nudge moves pane i by a delta, which is what a keyboard does.
func (r *Rack) Nudge(i, dx, dy int) []int {
	p := r.Pane(i)
	if p == nil {
		return nil
	}
	return r.MoveTo(i, p.Box.X+dx, p.Box.Y+dy)
}

// Resize gives pane i a new size and puts whatever hangs from it back
// where the new size says it goes.
func (r *Rack) Resize(i, w, h int) []int {
	p := r.Pane(i)
	if p == nil || (p.Box.W == w && p.Box.H == h) {
		return nil
	}
	p.Box.W, p.Box.H = w, h
	moved := []int{i}
	if p.To >= 0 {
		p.Box = p.Bond.Place(r.panes[p.To].Box, w, h)
	}
	return dedupe(append(moved, r.reflow(i)...))
}

// Detach takes pane i off whatever it was stuck to, leaving it where it
// is. A window dragged away with the pointer detaches itself; this is for
// the control that says so.
func (r *Rack) Detach(i int) {
	if p := r.Pane(i); p != nil {
		p.To, p.Bond = -1, Bond{}
	}
}

// Attach sticks pane i to pane to on a side, flush at the leading corner,
// and moves it there. It refuses a bond that would make a cycle.
func (r *Rack) Attach(i, to int, side Side) []int {
	p, anchor := r.Pane(i), r.Pane(to)
	if p == nil || anchor == nil || i == to || r.descends(to, i) {
		return nil
	}
	p.To, p.Bond = to, Bond{Side: side}
	p.Box = p.Bond.Place(anchor.Box, p.Box.W, p.Box.H)
	return dedupe(append([]int{i}, r.reflow(i)...))
}

// Show and hide a pane. A hidden pane keeps its bond and its box, so it
// comes back where it went.
func (r *Rack) Show(i int, on bool) {
	if p := r.Pane(i); p != nil {
		p.Shown = on
	}
}

// Boxes is every pane's box, in index order — the whole state of the rack
// in the form a test compares.
func (r *Rack) Boxes() []Box {
	out := make([]Box, len(r.panes))
	for i, p := range r.panes {
		out[i] = p.Box
	}
	return out
}

// Bounds is the box every pane fits inside, which is what a headless test
// needs to draw the whole rack on one canvas.
func (r *Rack) Bounds() Box {
	var out Box
	first := true
	for _, p := range r.panes {
		if !p.Shown || p.Box.Empty() {
			continue
		}
		if first {
			out, first = p.Box, false
			continue
		}
		x0, y0 := minInt(out.X, p.Box.X), minInt(out.Y, p.Box.Y)
		x1, y1 := maxInt(out.Right(), p.Box.Right()), maxInt(out.Bottom(), p.Box.Bottom())
		out = Box{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
	}
	return out
}

// rebond looks for a bond for pane i among the panes it is not carrying.
// The nearest qualifying anchor wins; none leaves the pane loose.
func (r *Rack) rebond(i int, carried []int) {
	p := r.Pane(i)
	if p == nil {
		return
	}
	p.To, p.Bond = -1, Bond{}
	for j, other := range r.panes {
		if j == i || !other.Shown || contains(carried, j) {
			continue
		}
		bond, ok := Snap(other.Box, p.Box, r.Reach)
		if !ok {
			continue
		}
		p.To, p.Bond = j, bond
		p.Box = bond.Place(other.Box, p.Box.W, p.Box.H)
		return
	}
}

// reflow puts everything hanging from pane i back where its bond says, and
// returns what moved. It walks the tree breadth first, so a pane under a
// pane under the main strip is placed after its own anchor.
func (r *Rack) reflow(i int) []int {
	var moved []int
	queue := []int{i}
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		for j, p := range r.panes {
			if p.To != at {
				continue
			}
			box := p.Bond.Place(r.panes[at].Box, p.Box.W, p.Box.H)
			if box != p.Box {
				p.Box = box
				moved = append(moved, j)
			}
			queue = append(queue, j)
		}
	}
	return moved
}

// shift translates a set of panes and reports them.
func (r *Rack) shift(idx []int, dx, dy int) []int {
	if dx == 0 && dy == 0 {
		return nil
	}
	for _, i := range idx {
		r.panes[i].Box = r.panes[i].Box.Moved(dx, dy)
	}
	return append([]int(nil), idx...)
}

// subtree is pane i and everything that hangs from it, i first.
func (r *Rack) subtree(i int) []int {
	out := []int{i}
	for at := 0; at < len(out); at++ {
		for j, p := range r.panes {
			if p.To == out[at] && !contains(out, j) {
				out = append(out, j)
			}
		}
	}
	return out
}

// descends reports whether pane i hangs from pane root, which is what a
// new bond must not do backwards.
func (r *Rack) descends(i, root int) bool { return contains(r.subtree(root), i) }

func contains(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func dedupe(s []int) []int {
	var out []int
	for _, v := range s {
		if !contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
