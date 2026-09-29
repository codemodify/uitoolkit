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

// FocusWatcher is implemented by containers that paint differently when
// the focus moves in or out of them, rather than onto them: a dock panel's
// title bar marks the panel the keyboard is in, whichever of its widgets
// actually holds the focus. FocusGained and FocusLost only reach the two
// components at either end of the move, so a container needs this instead.
type FocusWatcher interface {
	// FocusMoved is called after the host focus changed; now is the
	// component that holds it, or nil.
	FocusMoved(now Component)
}

// NotifyFocusMoved tells every FocusWatcher under roots that the focus is
// now on c. The host calls it after each move.
func NotifyFocusMoved(now Component, roots ...Component) {
	for _, r := range roots {
		if r == nil {
			continue
		}
		Walk(r, func(n Component) {
			if w, ok := n.(FocusWatcher); ok {
				w.FocusMoved(now)
			}
		})
	}
}

// LockKeysWatcher is implemented by components that show the state of
// Caps Lock or Num Lock: a passphrase field's "Caps Lock is on" mark,
// most of all.
//
// A right passphrase refused is nearly always Caps Lock, and a field
// holding a secret cannot work that out for itself — the only other way
// is to read the secret. The window's own OnLockKeys is one callback, so
// a widget that wanted the state had to take the window's callback and
// hand it back, and two widgets in one window could not both follow it.
// This is the same shape as [FocusWatcher]: every component under the
// window's roots that implements it hears every change.
type LockKeysWatcher interface {
	// LockKeysChanged is called when Caps Lock or Num Lock changes,
	// whether or not the key reached a widget — pressing Caps Lock is not
	// a key any field wants, and it is exactly the key a prompt has to
	// hear.
	LockKeysChanged(caps, num bool)
}

// NotifyLockKeys tells every LockKeysWatcher under roots about the lock
// state. The host calls it on each change.
func NotifyLockKeys(caps, num bool, roots ...Component) {
	for _, r := range roots {
		if r == nil {
			continue
		}
		Walk(r, func(n Component) {
			if w, ok := n.(LockKeysWatcher); ok {
				w.LockKeysChanged(caps, num)
			}
		})
	}
}

// Revealer is implemented by scrolling containers that can bring a
// descendant into view (widgets.ScrollView).
type Revealer interface {
	Reveal(c Component)
}

// RevealFocus scrolls every enclosing Revealer so c is visible — Tab and
// mnemonics used to move focus to controls below the fold of a ScrollView
// without showing them.
func RevealFocus(c Component) {
	if c == nil {
		return
	}
	for p := c.Parent(); p != nil; p = p.Parent() {
		if r, ok := p.(Revealer); ok {
			r.Reveal(c)
		}
	}
}
