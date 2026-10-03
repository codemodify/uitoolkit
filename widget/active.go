package widget

import "github.com/codemodify/uitoolkit/style"

// ActiveHost is a host that knows whether its window has keyboard focus.
type ActiveHost interface {
	Active() bool
}

// WindowActive reports whether c's window has keyboard focus. Hosts that
// cannot tell (offscreen windows) count as active.
func WindowActive(c Component) bool {
	if c == nil {
		return true
	}
	if h, ok := c.Host().(ActiveHost); ok {
		return h.Active()
	}
	return true
}

// ItemState is the look state of one row of an item view: selected,
// hovered, the current row of a focused view (Focused), and Inactive /
// Backdrop when the view or its window does not have focus.
func ItemState(view Component, selected, hovered, current bool) style.ControlState {
	return itemState(view, selected, hovered, current)
}

// RowItemState is ItemState for row index i, which also marks odd rows
// Alternate for striped looks.
func RowItemState(view Component, i int, selected, hovered, current bool) style.ControlState {
	st := itemState(view, selected, hovered, current)
	if i%2 == 1 {
		st |= style.StateAlternate
	}
	return st
}

func itemState(view Component, selected, hovered, current bool) style.ControlState {
	st := style.RowState(selected, hovered)
	if view == nil {
		return st
	}
	if !view.Enabled() {
		st |= style.StateDisabled
	}
	active := WindowActive(view)
	if !active {
		st |= style.StateBackdrop
	}
	focused := false
	if h := view.Host(); h != nil {
		f := h.Focus()
		// The identity Init established, not the receiver.
		//
		// A view embedded in an outer component — the commonest way to
		// extend a ListView — passes its *inner* receiver here, while
		// the host's focus is the outer component that Init was called
		// with. The comparison then never matched: the rows of a focused
		// list drew as unfocused and inactive, and an application had to
		// work out the focus itself to paint its own rows correctly.
		focused = f == view || f == selfOf(view)
	}
	if focused && active {
		if current {
			st |= style.StateFocused
		}
	} else {
		st |= style.StateInactive
	}
	return st
}

// selfOf is the component Init was given, which is the one the rest of
// the tree knows: for a view embedded in a wrapper, the wrapper.
func selfOf(c Component) Component {
	if s, ok := c.(interface{ me() Component }); ok {
		return s.me()
	}
	return c
}
