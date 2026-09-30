package app

import (
	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/widget"
)

// The lock keys, which a passphrase prompt needs and nothing else does.
//
// A right passphrase refused is nearly always Caps Lock, and a field
// that holds a secret cannot work that out for itself: the only other
// way to know is to look at the case of what was typed, which means
// reading the secret to do it. So the state comes from the keyboard
// instead, where it is a fact about the hardware rather than about the
// value.

// LockKeys is the state of Caps Lock and Num Lock as of the last event
// that carried modifiers.
//
// A window that has just opened and seen no key yet answers from the
// backend where it can ask ([platform.LockKeysSurface]) — X11 and
// Windows can, Wayland learns it from the compositor's first modifiers
// event and macOS from the first event of any kind — so a prompt can
// warn *before* the first keystroke, which is the whole point.
func (w *Window) LockKeys() (caps, num bool) {
	if w == nil {
		return false, false
	}
	if !w.lockKnown {
		if s, ok := w.surf.(platform.LockKeysSurface); ok {
			c, n := s.LockKeys()
			return c, n
		}
	}
	return w.lockCaps, w.lockNum
}

// OnLockKeys is called when Caps Lock or Num Lock changes, whether or
// not the key reached a widget — pressing Caps Lock is not a key any
// field wants, and it is exactly the key a prompt has to hear.
//
// It fires on a real change only, not on every keystroke.
//
// It is one callback a window, and a second replaces the first. A widget
// that wants the state should implement [widget.LockKeysWatcher]
// instead, which every component in the window gets; this is for the
// window's own code.
func (w *Window) OnLockKeys(fn func(caps, num bool)) {
	if w == nil {
		return
	}
	w.onLockKeys = fn
}

// noteLockKeys records the lock state an event carried and reports a
// change. Every event with modifiers goes through it, which is what
// makes the state current without polling anything.
func (w *Window) noteLockKeys(mods platform.Modifiers) {
	caps, num := mods.CapsLock(), mods.NumLock()
	if w.lockKnown && caps == w.lockCaps && num == w.lockNum {
		return
	}
	first := !w.lockKnown
	w.lockCaps, w.lockNum, w.lockKnown = caps, num, true
	// Every watcher in the window, then the window's own callback. A
	// widget that shows the state is one of many and cannot own the
	// single callback, which is why both exist.
	//
	// Watchers hear the first event too, because for them it is not "the
	// state became known" but "here is the state" — a field drawing a
	// Caps Lock mark has nothing until it is told, and a window that has
	// been open with the lock on since before it was shown has no change
	// to report. OnLockKeys keeps its documented meaning and fires on a
	// change only: it is the window's own callback, and a program that
	// wanted the state at start-up asked for it.
	widget.NotifyLockKeys(caps, num, w.notifyRoots()...)
	if !first && w.onLockKeys != nil {
		w.onLockKeys(caps, num)
	}
}
