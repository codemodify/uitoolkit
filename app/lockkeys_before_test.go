package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widget"
	"github.com/codemodify/uitoolkit/widgets"
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

// A window learns the lock state it *opened* in, from the window system
// saying so rather than from a keystroke.
//
// On Wayland the compositor sends wl_keyboard.modifiers right after
// enter, before any key — and nothing carried it into the window. The
// window asked its surface during the first layout, which on Wayland is
// before the surface is even mapped, was answered "off", and heard
// nothing more until the first keystroke. For a passphrase prompt that
// is the whole case: the warning exists for the first passphrase.
func TestAWindowLearnsTheLockStateItOpenedIn(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 300, Height: 120, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	w.SetContent(widgets.NewLabel("passphrase"))
	a.PumpOnce()

	// A watcher, which is what a SecretField's warning mark is. The
	// window's own OnLockKeys fires on a *change* by documented design,
	// and the state a window opens in is not a change — watchers are the
	// ones told about the first report.
	watch := &lockWatch{}
	watch.Init(watch)
	w.SetContent(watch)
	a.PumpOnce()
	if caps, _ := w.LockKeys(); caps {
		t.Fatal("Caps Lock is on before anything said so")
	}

	// The window system says so, with no key pressed.
	w.dispatch(platform.Event{Kind: platform.EventLockKeys, Mods: platform.ModCapsLock})

	if caps, _ := w.LockKeys(); !caps {
		t.Error("the window did not take the lock state the window system reported")
	}
	// The watcher hears it, which is what draws the mark on the field.
	if !watch.sawCaps {
		t.Error("no watcher was told, so nothing can show the warning")
	}
}

// lockWatch is a component that shows the lock state, as a SecretField's
// Caps Lock mark does.
type lockWatch struct {
	widget.Base
	sawCaps bool
}

func (l *lockWatch) LockKeysChanged(caps, num bool) {
	if caps {
		l.sawCaps = true
	}
}
