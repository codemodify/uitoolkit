package app

import (
	"testing"

	"github.com/codemodify/uitoolkit/platform"
	"github.com/codemodify/uitoolkit/style"
	"github.com/codemodify/uitoolkit/widgets"
)

// A focus-in does not read the event's own modifiers, and does ask the
// keyboard.
//
// A focus-in carries no modifiers on any backend, and reading that empty set
// as "every lock is off" is how this was got wrong: a passphrase prompt
// opened with Caps Lock on read the lock correctly from the server, then the
// focus-in arrived and the window wrote down "off" and told every watcher, so
// the field's warning mark appeared and went again before the first
// keystroke, and came back only on the next key — by which time the
// passphrase it was meant to warn about had been typed.
//
// Asking the *keyboard* at a focus-in is the other half, and the one this
// needed: the lock keys belong to the user and not to this window, so a lock
// turned on while another program had the focus is on when the focus comes
// back, and nothing said so. A prompt went on showing what it knew when it
// last saw a key, an hour and another program ago.
func TestAFocusInAsksTheKeyboardAndNotTheEvent(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	o := w.Surface().(*platform.Offscreen)
	w.SetContent(widgets.NewLabel("x"))
	a.PumpOnce()

	// The keyboard has Caps Lock on, and a key says so — the way the first
	// real event would.
	o.SetLockKeys(true, false)
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Mods: platform.ModCapsLock})
	if caps, _ := w.LockKeys(); !caps {
		t.Fatal("a key with Caps Lock did not set the state")
	}

	// A focus-in must not read its own empty modifiers as "off".
	w.dispatch(platform.Event{Kind: platform.EventFocusIn})
	if caps, _ := w.LockKeys(); !caps {
		t.Error("a focus-in with no modifiers turned Caps Lock off")
	}

	// The lock is changed in another program, and the focus comes back.
	var told []bool
	w.OnLockKeys(func(caps, _ bool) { told = append(told, caps) })
	o.SetLockKeys(false, false)
	w.dispatch(platform.Event{Kind: platform.EventFocusOut})
	w.dispatch(platform.Event{Kind: platform.EventFocusIn})
	if caps, _ := w.LockKeys(); caps {
		t.Error("the window still believes a lock that was turned off elsewhere")
	}
	if len(told) != 1 || told[0] {
		t.Errorf("the watchers were told %v, want one call saying it is off", told)
	}

	// And a real key saying it is off is still believed.
	o.SetLockKeys(true, false)
	w.dispatch(platform.Event{Kind: platform.EventKeyDown, Mods: platform.ModCapsLock})
	w.dispatch(platform.Event{Kind: platform.EventKeyDown})
	if caps, _ := w.LockKeys(); caps {
		t.Error("a key without Caps Lock did not clear it")
	}
}

// A backend that cannot be asked is not asked.
//
// Wayland knows only what the compositor last told it and macOS learns the
// modifiers from the next event of any kind; both leave
// platform.LockKeysSurface out and say what they know by pushing
// EventLockKeys instead. Asking one of them would turn "nobody has said yet"
// into "every lock is off", which is the bug above in another coat.
//
// The surface is wrapped rather than simulated, because the offscreen one
// *can* be asked: what is under test is the branch taken for the backends
// that cannot, and a test that used a backend that can would be testing
// nothing. Embedding the interface is what hides LockKeys — a method set
// comes from the embedded *type*, and platform.Surface has no LockKeys in it.
type cannotBeAsked struct{ platform.Surface }

func TestAFocusInDoesNotAskABackendThatCannotAnswer(t *testing.T) {
	a := New(Options{Look: style.DarkLook(), Headless: true})
	w, err := a.NewWindow(platform.WindowOptions{Width: 400, Height: 200, Headless: true})
	if err != nil {
		t.Fatal(err)
	}
	o := w.Surface().(*platform.Offscreen)
	w.SetContent(widgets.NewLabel("x"))
	a.PumpOnce()

	// The window system says the lock is on, as Wayland's modifiers event
	// does, while the keyboard this process could ask says otherwise.
	w.dispatch(platform.Event{Kind: platform.EventLockKeys, Mods: platform.ModCapsLock})
	if caps, _ := w.LockKeys(); !caps {
		t.Fatal("the window system's word about the lock was not taken")
	}
	o.SetLockKeys(false, false)

	real := w.surf
	w.surf = cannotBeAsked{real}
	w.refreshLockKeys()
	w.surf = real

	if caps, _ := w.LockKeys(); !caps {
		t.Error("a backend that cannot be asked was asked, and its silence read as off")
	}
}
