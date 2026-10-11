package widgets

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
)

// toastStack is the window's layer of notes: it holds them, lays them out
// against the bottom edge and takes itself down when the last one goes.
//
// It is the layer rather than each note, because the notes have to agree
// about where they are: two notes that each placed itself would land on
// top of each other. An application never sees it — [ShowToast] makes one
// when it needs one — and it is the reason stacking is the toolkit's
// problem and not every caller's.
type toastStack struct {
	widget.Base
}

// toastGap is between two notes, and toastInset between a note and the
// edges of the content, as 1x design lengths.
const (
	toastGap   = 8
	toastInset = 16
	// toastMaxW is the widest a note grows before its text wraps: a line
	// of about sixty characters, which is the width a sentence is read
	// at rather than the width the window happens to be.
	toastMaxW = 420
)

// toastStackFor is the stack on from's window, making one if the window
// has no notes up. It is nil when there is no window to put notes on.
func toastStackFor(from widget.Component) *toastStack {
	if s, ok := widget.ToastLayerOf(from).(*toastStack); ok {
		return s
	}
	s := &toastStack{}
	s.Init(s)
	if !widget.ShowToastLayer(from, s) {
		return nil
	}
	return s
}

// add puts a note at the top of the pile and starts it.
func (s *toastStack) add(t *Toast) {
	t.done = s.remove
	s.Add(t)
	// The host reaches the note through Add, which is what its timers and
	// its look come from, so the note can only be started after it.
	t.appear()
	s.relayout()
}

// remove takes a finished note off, and the layer with it when it was the
// last one: an empty layer would go on being laid out and painted for
// the rest of the window's life.
func (s *toastStack) remove(t *Toast) {
	s.Remove(t)
	if len(s.Children()) == 0 {
		widget.DismissToastLayer(s)
		return
	}
	s.relayout()
}

// clear takes every note down at once, without a fade: the caller said
// now.
func (s *toastStack) clear() {
	for _, c := range s.notes() {
		c.done = nil
		c.going = true
	}
	s.ClearChildren()
	widget.DismissToastLayer(s)
}

// notes are the stack's notes, oldest first.
func (s *toastStack) notes() []*Toast {
	out := make([]*Toast, 0, len(s.Children()))
	for _, c := range s.Children() {
		if t, ok := c.(*Toast); ok {
			out = append(out, t)
		}
	}
	return out
}

func (s *toastStack) relayout() {
	s.RequestLayout()
	s.Invalidate()
}

// Measure takes whatever it is offered: the layer is the content's box,
// and the notes are laid out inside it.
func (s *toastStack) Measure(c layout.Constraints) paintengine2d.Point {
	return c.Constrain(paintengine2d.Pt(c.MaxW, c.MaxH))
}

// Arrange stacks the notes up from the bottom edge, newest nearest it.
//
// Newest nearest the edge is where the eye already is — it is the one
// that just arrived — and it is what every desktop does with the corner
// its notifications come out of.
func (s *toastStack) Arrange(r paintengine2d.Rect) {
	s.SetBounds(r)
	lk := s.Look()
	inset := style.Dip(lk, toastInset)
	gap := style.Dip(lk, toastGap)
	box := paintengine2d.XYWH(0, 0, r.Dx(), r.Dy())
	maxW := min(box.Dx()-inset*2, style.Dip(lk, toastMaxW))
	if maxW <= 0 {
		maxW = box.Dx()
	}
	y := box.Max.Y - inset
	notes := s.notes()
	for i := len(notes) - 1; i >= 0; i-- {
		t := notes[i]
		sz := t.Measure(layout.Loose(maxW, box.Dy()))
		x := box.Min.X + (box.Dx()-sz.X)*0.5
		t.Arrange(paintengine2d.XYWH(x, y-sz.Y, sz.X, sz.Y))
		y -= sz.Y + gap
	}
}

// HitTest is nil for the layer as well as for each note: nothing on it is
// ever pressed, so the pointer never has to find out what is up there.
func (s *toastStack) HitTest(paintengine2d.Point) widget.Component { return nil }
