package widget

import (
	"github.com/codemodify/paintengine2d"
	"github.com/codemodify/uitoolkit/layout"
	"github.com/codemodify/uitoolkit/style"
)

// Slots lays components out on the named slots of a skin's fixed layout
// (style/skin_layout.go): the app says which of its components goes in
// which slot, and the skin says where each slot is.
//
// It is a placement table and nothing else. It never adds, removes, hides
// or reorders a component, so the tab order is still the order the app
// added its components in, the accessibility tree is still the app's, and a
// component in a slot is the same component — its keys, its name, its focus
// ring — that it was under an ordinary layout. A component whose slot the
// skin leaves out is handed back by Arrange for the app to deal with, never
// dropped by the skin: whether a control exists is the app's decision.
//
//	s := widget.NewSlots("minim.strip").
//		Bind("play", play).
//		Bind("stop", stop)
//	if s.Available(lk) {
//		for _, c := range s.Arrange(lk, box) { … the skin has no place for c }
//	}
//
// A look that is not a skin, or a skin without the layout, has no slots at
// all (Available answers false), and the app lays itself out its own way —
// which is what keeps a panel laid out by a skin a partial override like
// everything else a skin does.
type Slots struct {
	layout string
	binds  []slotBind
}

type slotBind struct {
	slot string
	c    Component
}

// NewSlots is an empty binding table for the skin layout of that name.
func NewSlots(layout string) *Slots { return &Slots{layout: layout} }

// Layout is the name of the skin layout the table binds to.
func (s *Slots) Layout() string { return s.layout }

// Bind puts c in the named slot, and returns the table so bindings chain.
// Binding a slot again replaces its component.
func (s *Slots) Bind(slot string, c Component) *Slots {
	for i := range s.binds {
		if s.binds[i].slot == slot {
			s.binds[i].c = c
			return s
		}
	}
	s.binds = append(s.binds, slotBind{slot: slot, c: c})
	return s
}

// Slot is the component bound to the named slot, or nil.
func (s *Slots) Slot(slot string) Component {
	for _, b := range s.binds {
		if b.slot == slot {
			return b.c
		}
	}
	return nil
}

// Available reports whether lk's skin has the layout at all.
func (s *Slots) Available(lk style.LookAndFeel) bool {
	_, ok := style.SkinLayoutOf(lk, s.layout)
	return ok
}

// Measure is the size the layout was drawn for, at lk's scale and within
// c, and whether lk has the layout.
func (s *Slots) Measure(lk style.LookAndFeel, c layout.Constraints) (paintengine2d.Point, bool) {
	sz, ok := style.SkinLayoutSize(lk, s.layout)
	if !ok {
		return paintengine2d.Point{}, false
	}
	return c.Constrain(sz), true
}

// Rect is where the named slot is inside box: a place the app paints into
// itself — a clock's digits, a lamp — as well as one a component sits in.
func (s *Slots) Rect(lk style.LookAndFeel, box paintengine2d.Rect, slot string) (paintengine2d.Rect, bool) {
	return style.SkinSlotRect(lk, s.layout, slot, box)
}

// Arrange puts every bound component on its slot inside box, box being in
// the coordinates the components are arranged in (their parent's local
// ones). The components the skin has no slot for are arranged into an empty
// rect and returned, in binding order: the skin has said nothing about
// them, and it is the app's to hide them or to place them itself.
//
// With no layout in lk at all nothing is arranged and every component is
// returned.
func (s *Slots) Arrange(lk style.LookAndFeel, box paintengine2d.Rect) (unplaced []Component) {
	if !s.Available(lk) {
		for _, b := range s.binds {
			unplaced = append(unplaced, b.c)
		}
		return unplaced
	}
	for _, b := range s.binds {
		if b.c == nil {
			continue
		}
		r, ok := style.SkinSlotRect(lk, s.layout, b.slot, box)
		if !ok {
			b.c.Arrange(paintengine2d.Rect{})
			unplaced = append(unplaced, b.c)
			continue
		}
		b.c.Arrange(r)
	}
	return unplaced
}
