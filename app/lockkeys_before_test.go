package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
)

// Every backend that can be asked for the lock keys is asked, so a prompt
// opened with Caps Lock on can say so before a single key is typed —
// which is the one case the warning exists for.
//
// X11 and Windows could already. Wayland was reported as unable to, in
// v0.23.2's own notes, on the grounds that the state arrives with the
// first key. It does not: the compositor sends wl_keyboard.modifiers
// right after wl_keyboard.enter, the backend has folded it into its xkb
// state since long before that release, and it simply told nobody. macOS
// can be asked at any time through NSEvent.modifierFlags.
func TestEveryBackendThatCanBeAskedForTheLocksIs(t *testing.T) {
	// The seam itself: a surface that answers it is used, and one that
	// does not falls back to what the window last learnt from an event.
	a := New(Options{Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 100, Height: 60, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	a.PumpOnce()
	// No event has carried modifiers yet. A window whose surface cannot
	// be asked answers false rather than guessing.
	if caps, _ := w.LockKeys(); caps {
		t.Error("a window that has seen nothing reports Caps Lock on")
	}
	// And once an event carries them, that is what it reports.
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Mods: platform.ModCapsLock})
	if caps, _ := w.LockKeys(); !caps {
		t.Error("a key carrying Caps Lock did not set the state")
	}
}
