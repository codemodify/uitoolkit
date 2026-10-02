package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// A focus-in carries no modifiers on any backend, so it says nothing about
// the lock keys and must not be taken to say they are off.
//
// A passphrase prompt opened with Caps Lock on read the lock correctly from
// the server, then the focus-in arrived and the window wrote down "off" and
// told every watcher, so the field's warning mark appeared and went again
// before the first keystroke. The mark came back only on the next key —
// by which time the passphrase it was meant to warn about had been typed.
func TestAFocusInDoesNotSayTheLocksAreOff(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("x"))
	a.PumpOnce()

	// A key tells the window Caps Lock is on, the way the first real
	// event would.
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Mods: platform.ModCapsLock})
	if caps, _ := w.LockKeys(); !caps {
		t.Fatal("a key with Caps Lock did not set the state")
	}

	// A focus-in with no modifiers on it must leave that alone.
	w.dispatch(platform.Event{Kind: platform.EventFocusIn})
	if caps, _ := w.LockKeys(); !caps {
		t.Error("a focus-in with no modifiers turned Caps Lock off")
	}

	// A real key saying it is off is still believed.
	w.dispatch(platform.Event{Kind: platform.EventKeyDown})
	if caps, _ := w.LockKeys(); caps {
		t.Error("a key without Caps Lock did not clear it")
	}
}
