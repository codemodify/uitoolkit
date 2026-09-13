package widget

// MarkKeyboardFocus records keyboard-originated focus (Tab, arrows, mnemonics).
func MarkKeyboardFocus(c Component) {
	type hint interface{ MarkKeyboardFocus() }
	if v, ok := c.(hint); ok {
		v.MarkKeyboardFocus()
	}
}

// MarkPointerFocus records pointer-originated focus (mouse / touch).
func MarkPointerFocus(c Component) {
	type hint interface{ MarkPointerFocus() }
	if v, ok := c.(hint); ok {
		v.MarkPointerFocus()
	}
}

// FocusOwner is the host's currently focused component, or nil.
func FocusOwner(from Component) Component {
	if from == nil {
		return nil
	}
	h := from.Host()
	if h == nil {
		return nil
	}
	return h.Focus()
}

// FocusWithin reports whether the host focus is root or one of its
// descendants. Popups and overlays use it so they only move focus when they
// actually hold it — never stealing it from whatever the user just clicked.
func FocusWithin(root Component) bool {
	if root == nil {
		return false
	}
	f := FocusOwner(root)
	if f == nil {
		return false
	}
	return Contains(root, f)
}

// FirstFocusable is the first tab-order candidate in root's subtree, or nil.
func FirstFocusable(root Component) Component {
	list := Focusables(root)
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

// FocusFirstIn moves focus to the first focusable inside root (keyboard
// origin, so the focus ring shows). Reports whether focus moved.
func FocusFirstIn(root Component) bool {
	if root == nil {
		return false
	}
	h := root.Host()
	if h == nil {
		return false
	}
	target := FirstFocusable(root)
	if target == nil {
		return false
	}
	if h.Focus() == target {
		return true
	}
	h.RequestFocus(target)
	MarkKeyboardFocus(target)
	return true
}

// RestoreFocus hands focus back to prev, but only while scope still owns it.
// A popup or overlay calls this as it is dismissed so the anchor (menu bar,
// combo box, the field that opened a dialog) becomes focused again instead of
// leaving a detached node as the host focus. keyboard marks the restored
// component as keyboard-focused so the ring stays visible for key-driven
// dismissals. Reports whether focus moved.
func RestoreFocus(scope, prev Component, keyboard bool) bool {
	if scope == nil || prev == nil || prev == scope {
		return false
	}
	if !FocusWithin(scope) {
		return false
	}
	h := scope.Host()
	if h == nil {
		return false
	}
	h.RequestFocus(prev)
	if keyboard {
		MarkKeyboardFocus(prev)
	}
	return true
}

// LiveUnder reports whether c is still reachable from any of roots. Popup
// cascades are followed, since a submenu is a sibling of its parent popup
// rather than a child of it.
func LiveUnder(c Component, roots ...Component) bool {
	if c == nil {
		return false
	}
	for _, r := range roots {
		if r == nil {
			continue
		}
		found := false
		WalkCascade(r, func(n Component) {
			if !found && Contains(n, c) {
				found = true
			}
		})
		if found {
			return true
		}
	}
	return false
}

// ClearFocusOutside drops the host focus when it is not reachable from any
// live root (content swap, dismissed overlay or popup, removed subtree). The
// app window calls it after changing a layer so a detached component cannot
// keep receiving keys. Reports whether focus was cleared.
func ClearFocusOutside(h Host, roots ...Component) bool {
	if h == nil {
		return false
	}
	f := h.Focus()
	if f == nil {
		return false
	}
	if LiveUnder(f, roots...) {
		return false
	}
	h.RequestFocus(nil)
	return true
}

// KeyTarget is the component that may receive a key or text event: while a
// modal overlay is mounted, nothing outside it may. Returns nil when the event
// must be dropped.
func KeyTarget(focus, overlay Component) Component {
	if overlay != nil && !LiveUnder(focus, overlay) {
		return nil
	}
	return focus
}
