package players

import (
	"github.com/codemodify/uitoolkit/app"
)

// Desk is a Rack with real windows in it.
//
// rack.go is the arithmetic and this is the half that touches a desktop, and
// the split is not tidiness: only one of the two desktops this toolkit runs
// on will do what a rack needs.
//
// A window that keeps another one stuck to it has to know where it is and be
// able to put the other one somewhere. X11 clients place their own windows,
// so both work there, and the offscreen backend implements them too, which
// is how every one of the snapping tests runs without a compositor. A
// **Wayland toplevel has no position**: a client is never told where its
// windows are and cannot ask for one, and that is the protocol rather than
// an omission in this toolkit. So on Wayland the rack still snaps — the
// arithmetic is the same and the model is the same — but the windows cannot
// be made to follow, and each player says so in its own interface rather
// than pretending they did.
//
// (The protocol's own answer for the one case it cares about is
// xdg-toplevel-drag-v1, which carries a window under the pointer during a
// drag; the toolkit uses it for tear-off. It cannot place a window that is
// not being dragged, which is what a rack is.)

// Desk drives a rack of windows.
type Desk struct {
	Rack *Rack
	wins []*app.Window
	// want is where we last asked the desktop to put each window. A
	// position that differs from it is the *user's* doing — a drag, a
	// snap by the window manager — and that is what tells the two apart
	// without watching for a drag we cannot see.
	want []Box
	// pending counts the looks since we asked the desktop to move a window
	// and it has not arrived yet. It is the one piece of hysteresis in
	// here and it is load-bearing: a move is a *request*, X11 answers it a
	// frame or two later, and without this Follow reads the old position
	// back on the very next look, decides the user must have dragged the
	// window there, and takes it out of the stack it was just put into.
	pending []int
	// Moved runs after Follow has moved anything, so a player can repaint
	// the line that says where its panes are.
	Moved func()
}

// movePatience is how many looks a window gets to arrive where it was asked
// to go before the rack gives up and takes the desktop's answer instead. A
// window manager that refuses a move — a tiled window, a maximized one —
// would otherwise be argued with forever.
const movePatience = 25

// NewDesk is an empty desk whose panes snap within reach pixels.
func NewDesk(reach int) *Desk { return &Desk{Rack: NewRack(reach)} }

// Add puts a window in the rack at wherever the desktop has it, and returns
// its index. The first window added is the one the others hang from.
func (d *Desk) Add(name string, w *app.Window) int {
	i := d.Rack.Add(name, d.boxOf(w))
	d.wins = append(d.wins, w)
	d.want = append(d.want, d.Rack.Pane(i).Box)
	d.pending = append(d.pending, 0)
	return i
}

// Window is the window of pane i, or nil.
func (d *Desk) Window(i int) *app.Window {
	if d == nil || i < 0 || i >= len(d.wins) {
		return nil
	}
	return d.wins[i]
}

// Places reports whether this desktop lets the client put its windows where
// the rack says. It is false on Wayland, where nothing below is possible and
// every player has a line in its interface that says so.
func (d *Desk) Places() bool {
	for _, w := range d.wins {
		if w == nil || w.Closed() {
			continue
		}
		if !w.CanMove() {
			return false
		}
	}
	return len(d.wins) > 0
}

// Follow reads where the desktop has put every window, lets a window the
// user moved find a new bond, and puts whatever hangs from it back under it.
// It reports whether it moved anything.
//
// It is called on a timer rather than on an event because neither backend
// tells an app that another of its windows moved: X11 sends the
// ConfigureNotify to the window that moved, and there is no "one of your
// toplevels changed" at all. A timer that reads four integers per window is
// the honest way to do this from outside the window manager, and it is what
// every player that did this ran.
func (d *Desk) Follow() bool {
	if d == nil || len(d.wins) == 0 {
		return false
	}
	changed := false
	for i, w := range d.wins {
		p := d.Rack.Pane(i)
		if p == nil || w == nil || w.Closed() {
			continue
		}
		box := d.boxOf(w)
		if box.Empty() {
			continue
		}
		if box.W != p.Box.W || box.H != p.Box.H {
			d.Rack.Resize(i, box.W, box.H)
			changed = true
		}
		// Only a position we did not ask for is the user's.
		if box.X == d.want[i].X && box.Y == d.want[i].Y {
			d.pending[i] = 0
			continue
		}
		if d.pending[i] > 0 {
			// Asked for, not arrived. Wait.
			d.pending[i]--
			continue
		}
		d.Rack.MoveTo(i, box.X, box.Y)
		changed = true
	}
	if !changed {
		return false
	}
	d.Apply()
	if d.Moved != nil {
		d.Moved()
	}
	return true
}

// Adopt takes the box the desktop has actually given every window into the
// rack, leaving the bonds alone, and reports whether every position is
// known.
//
// It is the thing a rack needs at startup and the reason a stack cannot
// simply be built when the windows are made. A window manager places a
// window when it *maps* it, and the application is not told where until a
// configure comes back afterwards — so every box in a fresh rack is at the
// origin, and a stack arranged then is arranged around a window that is not
// there yet. The apps call this on their clock until it answers yes, and
// then attach their panes once.
func (d *Desk) Adopt() bool {
	known := len(d.wins) > 0
	for i, w := range d.wins {
		p := d.Rack.Pane(i)
		if p == nil || w == nil || w.Closed() {
			continue
		}
		x, y, ok := w.Position()
		if !ok {
			known = false
			continue
		}
		// The rack's boxes are the logical pixels a position is in.
		ww, hh := w.Size()
		p.Box = Box{X: x, Y: y, W: ww, H: hh}
		d.want[i] = p.Box
		d.pending[i] = 0
	}
	return known
}

// Apply puts every window where the rack says it goes, and skips the ones
// already there. A desktop that will not place windows does nothing here and
// reports it, which is what Places is for.
func (d *Desk) Apply() {
	for i, w := range d.wins {
		p := d.Rack.Pane(i)
		if p == nil || w == nil || w.Closed() {
			continue
		}
		if p.Box.X == d.want[i].X && p.Box.Y == d.want[i].Y {
			d.want[i] = p.Box
			continue
		}
		if w.Move(p.Box.X, p.Box.Y) {
			d.want[i] = p.Box
			d.pending[i] = d.patience(w, p.Box)
		}
	}
}

// patience is how many looks a window gets to arrive where it was just
// asked to go: none at all when it is already there. A backend that places
// a window as the call is made — the offscreen one, and an X11 server that
// is not busy — needs no hysteresis, and giving it some would make a real
// drag half a second later look like a move nobody asked for.
func (d *Desk) patience(w *app.Window, want Box) int {
	if x, y, ok := w.Position(); ok && x == want.X && y == want.Y {
		return 0
	}
	return movePatience
}

// Attach sticks pane i to pane to on a side and moves it there — the
// control rather than the drag, which is how a player's "dock the playlist"
// menu item works.
func (d *Desk) Attach(i, to int, side Side) {
	d.Rack.Attach(i, to, side)
	d.Apply()
}

// Show opens or hides a pane's window and keeps the rack in step. A hidden
// pane keeps its bond, so the playlist comes back where it went.
func (d *Desk) Show(i int, on bool) {
	w := d.Window(i)
	if w == nil {
		return
	}
	d.Rack.Show(i, on)
	if !on {
		w.Hide()
		return
	}
	// Place it before showing it, so it does not appear at the desktop's
	// idea of where it goes and then jump to the rack's.
	if p := d.Rack.Pane(i); p != nil {
		if w.Move(p.Box.X, p.Box.Y) {
			d.want[i] = p.Box
			d.pending[i] = d.patience(w, p.Box)
		}
	}
	w.Show()
}

// boxOf is where a window is and how big it is, in the logical pixels the rack
// speaks. A window the desktop will not tell us about — every Wayland one —
// keeps the box the rack already had, so the model stays coherent and only
// the *following* is lost.
func (d *Desk) boxOf(w *app.Window) Box {
	if w == nil || w.Closed() {
		return Box{}
	}
	ww, hh := w.Size()
	x, y, ok := w.Position()
	if !ok {
		if p, i := d.paneFor(w); p != nil {
			_ = i
			return Box{X: p.Box.X, Y: p.Box.Y, W: ww, H: hh}
		}
		return Box{W: ww, H: hh}
	}
	return Box{X: x, Y: y, W: ww, H: hh}
}

// paneFor is the pane a window belongs to.
func (d *Desk) paneFor(w *app.Window) (*Pane, int) {
	for i, x := range d.wins {
		if x == w {
			return d.Rack.Pane(i), i
		}
	}
	return nil, -1
}
