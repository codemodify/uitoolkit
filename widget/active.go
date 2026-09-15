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
		focused = h.Focus() == view
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
